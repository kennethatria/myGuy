package services

import (
	"errors"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"time"
)

// RequestServiceInterface is what the request handlers need.
type RequestServiceInterface interface {
	CreateRequest(userID uint, req models.CreateItemRequestRequest) (*models.ItemRequest, error)
	GetRequest(id uint) (*models.ItemRequest, error)
	GetRequests(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error)
	GetRequestListings(id uint) ([]models.StoreItem, error)
	GetUserRequests(userID uint) ([]models.ItemRequest, error)
	RepostRequest(id uint, userID uint) (*models.ItemRequest, error)
	DeleteRequest(id uint, userID uint) error
}

// RequestService handles "wanted" notes. They follow the listing rules: a
// short note, no contact details, 24 hours on the board unless a seller
// lists something for it.
type RequestService struct {
	requestRepo repositories.ItemRequestRepository
	itemRepo    repositories.StoreItemRepository
}

func NewRequestService(requestRepo repositories.ItemRequestRepository, itemRepo repositories.StoreItemRepository) *RequestService {
	return &RequestService{requestRepo: requestRepo, itemRepo: itemRepo}
}

func (s *RequestService) CreateRequest(userID uint, req models.CreateItemRequestRequest) (*models.ItemRequest, error) {
	title, description, err := validateListingText(req.Title, req.Description)
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

// GetRequestListings lists the live listings sellers made for a request.
func (s *RequestService) GetRequestListings(id uint) ([]models.StoreItem, error) {
	items, _, err := s.itemRepo.GetAll(models.StoreItemFilter{RequestID: id, Status: "active", PerPage: 100})
	return items, err
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
		return nil, errors.New("unauthorized: you can only repost your own requests")
	}
	if request.Status != "expired" {
		return nil, errors.New("only an expired request can be reposted")
	}
	request.Status = "active"
	startRequest(request)
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
		return errors.New("unauthorized: you can only remove your own requests")
	}
	if request.Status != "active" && request.Status != "expired" {
		return errors.New("only a live or expired request can be removed")
	}
	return s.requestRepo.Delete(id)
}

// ExpireStaleRequests takes requests that got no listing within their 24
// hours off the board, returning how many it changed.
func (s *RequestService) ExpireStaleRequests() (int64, error) {
	return s.requestRepo.ExpireUnanswered(time.Now().UTC())
}
