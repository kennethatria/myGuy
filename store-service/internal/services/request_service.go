package services

import (
	"context"
	"log"
	"store-service/internal/models"
	"store-service/internal/proximity"
	"store-service/internal/repositories"
	"time"
)

// RequestServiceInterface is what the request handlers need.
type RequestServiceInterface interface {
	CreateRequest(userID uint, req models.CreateItemRequestRequest) (*models.ItemRequest, error)
	GetRequest(id uint) (*models.ItemRequest, error)
	GetRequests(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error)
	GetRequestsNear(ctx context.Context, filter models.ItemRequestFilter, at proximity.Location) ([]models.ItemRequest, int64, error)
	TagRequestDistances(ctx context.Context, requests []models.ItemRequest, at proximity.Location)
	GetRequestListings(ctx context.Context, id uint) ([]models.StoreItem, error)
	GetUserRequests(userID uint) ([]models.ItemRequest, error)
	RepostRequest(id uint, userID uint) (*models.ItemRequest, error)
	UpdateRequest(id uint, userID uint, req models.UpdateItemRequestRequest) (*models.ItemRequest, error)
	DeleteRequest(id uint, userID uint) error
}

// RequestService handles "wanted" notes. They follow the listing rules: a
// short note, no contact details, 24 hours on the board unless a seller
// lists something for it.
type RequestService struct {
	requestRepo repositories.ItemRequestRepository
	itemRepo    repositories.StoreItemRepository
	locator     Locator
	distancer   Distancer // nil: no distance sorting or tags
}

func NewRequestService(requestRepo repositories.ItemRequestRepository, itemRepo repositories.StoreItemRepository) *RequestService {
	return &RequestService{requestRepo: requestRepo, itemRepo: itemRepo, locator: noopLocator{}}
}

// WithDistancer turns on distance sorting and tags (nil leaves them off).
func (s *RequestService) WithDistancer(distancer Distancer) *RequestService {
	s.distancer = distancer
	return s
}

// WithLocator saves requests' rough locations through locator (nil = don't).
func (s *RequestService) WithLocator(locator Locator) *RequestService {
	if locator != nil {
		s.locator = locator
	}
	return s
}

func (s *RequestService) CreateRequest(userID uint, req models.CreateItemRequestRequest) (*models.ItemRequest, error) {
	title, description, err := validateListingText(req.Title, req.Description)
	if err != nil {
		return nil, err
	}
	at, err := parseLocation(req.Lat, req.Lng)
	if err != nil {
		return nil, err
	}
	request := &models.ItemRequest{
		Title:       title,
		Description: description,
		RequesterID: userID,
		Status:      "active",
	}
	startRequest(request)
	if err := s.requestRepo.Create(request); err != nil {
		return nil, err
	}
	if at != nil {
		s.locator.Save("request", request.ID, *at)
	}
	return request, nil
}

// startRequest puts a request on the board for a fresh ListingLifetime.
func startRequest(request *models.ItemRequest) {
	deadline := time.Now().UTC().Add(ListingLifetime)
	request.Deadline = &deadline
}

func (s *RequestService) GetRequest(id uint) (*models.ItemRequest, error) {
	return s.requestRepo.GetByID(id)
}

func (s *RequestService) GetRequests(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error) {
	return s.requestRepo.GetAll(filter)
}

// GetRequestsNear is GetRequests ordered by distance from at, the way
// StoreService.GetItemsNear orders listings, with the same fallback.
func (s *RequestService) GetRequestsNear(ctx context.Context, filter models.ItemRequestFilter, at proximity.Location) ([]models.ItemRequest, int64, error) {
	if s.distancer == nil {
		return s.GetRequests(filter)
	}
	ids, err := s.requestRepo.ListIDs(filter)
	if err != nil {
		return nil, 0, err
	}
	buckets, err := s.distancer.Distances(ctx, "request", at, ids)
	if err != nil {
		log.Printf("WARNING: distance sort unavailable, showing newest first: %v", err)
		return s.GetRequests(filter)
	}
	rankByDistance(ids, buckets)
	pageIDs := pageOf(ids, filter.Page, filter.PerPage)

	loaded, err := s.requestRepo.GetByIDs(pageIDs)
	if err != nil {
		return nil, 0, err
	}
	byID := make(map[uint]models.ItemRequest, len(loaded))
	for _, request := range loaded {
		byID[request.ID] = request
	}
	requests := make([]models.ItemRequest, 0, len(pageIDs))
	for _, id := range pageIDs {
		if request, ok := byID[id]; ok {
			request.Distance = tagFor(buckets, id)
			requests = append(requests, request)
		}
	}
	return requests, int64(len(ids)), nil
}

// TagRequestDistances adds a rough distance tag from at to each request
// that has a location. Best effort.
func (s *RequestService) TagRequestDistances(ctx context.Context, requests []models.ItemRequest, at proximity.Location) {
	if s.distancer == nil || len(requests) == 0 {
		return
	}
	ids := make([]uint, len(requests))
	for i, request := range requests {
		ids[i] = request.ID
	}
	buckets, err := s.distancer.Distances(ctx, "request", at, ids)
	if err != nil {
		log.Printf("WARNING: distance tags unavailable: %v", err)
		return
	}
	for i := range requests {
		requests[i].Distance = tagFor(buckets, requests[i].ID)
	}
}

// GetRequestListings lists the live listings sellers made for a request,
// nearest to the requester first (tagged with that distance) when both have
// a location; otherwise newest first.
func (s *RequestService) GetRequestListings(ctx context.Context, id uint) ([]models.StoreItem, error) {
	items, _, err := s.itemRepo.GetAll(models.StoreItemFilter{RequestID: id, Status: "active", PerPage: 100})
	if err != nil || s.distancer == nil || len(items) == 0 {
		return items, err
	}
	ids := make([]uint, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	buckets, derr := s.distancer.DistancesFrom(ctx, "item", "request", id, ids)
	if derr != nil {
		log.Printf("WARNING: distance sort for request %d unavailable: %v", id, derr)
		return items, nil
	}
	rankByDistance(ids, buckets)
	byID := make(map[uint]models.StoreItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	ranked := make([]models.StoreItem, len(ids))
	for i, itemID := range ids {
		ranked[i] = byID[itemID]
		ranked[i].Distance = tagFor(buckets, itemID)
	}
	return ranked, nil
}

func (s *RequestService) GetUserRequests(userID uint) ([]models.ItemRequest, error) {
	return s.requestRepo.GetByRequesterID(userID)
}

// RepostRequest puts an expired request back on the board for 24 hours.
func (s *RequestService) RepostRequest(id uint, userID uint) (*models.ItemRequest, error) {
	request, err := s.requestRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if request.RequesterID != userID {
		return nil, NewUserError("unauthorized: you can only repost your own requests")
	}
	if request.Status != "expired" {
		return nil, NewUserError("only an expired request can be reposted")
	}
	request.Status = "active"
	startRequest(request)
	if err := s.requestRepo.Update(request); err != nil {
		return nil, err
	}
	return request, nil
}

// UpdateRequest changes the headline and note of the requester's live
// request, under the same rules as posting it. Listings already made for it
// stay linked, as with a listing edited while bookings wait.
func (s *RequestService) UpdateRequest(id uint, userID uint, req models.UpdateItemRequestRequest) (*models.ItemRequest, error) {
	request, err := s.requestRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if request.RequesterID != userID {
		return nil, NewUserError("unauthorized: you can only edit your own requests")
	}
	if request.Status != "active" {
		return nil, NewUserError("only a live request can be edited")
	}
	title, description, err := validateListingText(req.Title, req.Description)
	if err != nil {
		return nil, err
	}
	request.Title, request.Description = title, description
	if err := s.requestRepo.Update(request); err != nil {
		return nil, err
	}
	return request, nil
}

// DeleteRequest removes the requester's live or expired request. Listings
// made for it stay up; they are ordinary listings too.
func (s *RequestService) DeleteRequest(id uint, userID uint) error {
	request, err := s.requestRepo.GetByID(id)
	if err != nil {
		return err
	}
	if request.RequesterID != userID {
		return NewUserError("unauthorized: you can only remove your own requests")
	}
	if request.Status != "active" && request.Status != "expired" {
		return NewUserError("only a live or expired request can be removed")
	}
	if err := s.requestRepo.Delete(id); err != nil {
		return err
	}
	s.locator.Delete("request", id)
	return nil
}

// ExpireStaleRequests takes requests that got no listing within their 24
// hours off the board, returning how many it changed.
func (s *RequestService) ExpireStaleRequests() (int64, error) {
	return s.requestRepo.ExpireUnanswered(time.Now().UTC())
}
