package services

import (
	"fmt"
	"store-service/internal/contacts"
	"store-service/internal/models"
	"store-service/internal/proximity"
	"store-service/internal/repositories"
	"strings"
	"time"

	"gorm.io/gorm"
)

// A listing is a short sticky note, like a gig (backend task_service.go):
// it stays on the board for ListingLifetime unless someone bids or asks to
// book. Auctions close when the note comes down.
const (
	ListingLifetime  = 24 * time.Hour
	headlineMaxWords = 5
	bodyMaxWords     = 20
	headlineMaxChars = 60
	bodyMaxChars     = 200
)

// Listing text errors: the request is fine, the words need changing.
var (
	ErrHeadlineRequired = NewUserError("headline is required")
	ErrBodyRequired     = NewUserError("note is required")
	ErrHeadlineTooLong  = NewUserError(fmt.Sprintf("headline can be at most %d words", headlineMaxWords))
	ErrBodyTooLong      = NewUserError(fmt.Sprintf("note can be at most %d words", bodyMaxWords))
	ErrContactDetails   = NewUserError("remove phone numbers, emails, links and handles; you can share them in chat once the seller approves your booking")
	ErrListingExpired   = NewUserError("this listing has expired")
	ErrRequestClosed    = NewUserError("this request is no longer open")
	ErrOwnRequest       = NewUserError("you can't list an item for your own request")
	ErrInvalidLocation  = NewUserError(proximity.ErrInvalidLocation.Error())
)

// Locator keeps the rough location of posts in the proximity service.
// Implementations must not block or fail the caller (it is best effort).
type Locator interface {
	Save(kind string, id uint, at proximity.Location)
	Delete(kind string, id uint)
}

type noopLocator struct{}

func (noopLocator) Save(string, uint, proximity.Location) {}
func (noopLocator) Delete(string, uint)                   {}

// parseLocation reads an optional rough location from a create request.
func parseLocation(lat, lng *float64) (*proximity.Location, error) {
	at, err := proximity.Parse(lat, lng)
	if err != nil {
		return nil, ErrInvalidLocation
	}
	return at, nil
}

// validateListingText enforces the sticky-note limits and keeps contact
// details off public listings. It returns the trimmed headline and note.
func validateListingText(title, description string) (string, string, error) {
	title, description = strings.TrimSpace(title), strings.TrimSpace(description)
	switch {
	case title == "":
		return "", "", ErrHeadlineRequired
	case description == "":
		return "", "", ErrBodyRequired
	case len(strings.Fields(title)) > headlineMaxWords || len(title) > headlineMaxChars:
		return "", "", ErrHeadlineTooLong
	case len(strings.Fields(description)) > bodyMaxWords || len(description) > bodyMaxChars:
		return "", "", ErrBodyTooLong
	case contacts.Contains(title) || contacts.Contains(description):
		return "", "", ErrContactDetails
	}
	return title, description, nil
}

type StoreService struct {
	db              *gorm.DB
	itemRepo        repositories.StoreItemRepository
	bidRepo         repositories.BidRepository
	bookingRepo     repositories.BookingRequestRepository
	userRepo        repositories.UserRepository
	requestRepo     repositories.ItemRequestRepository
	chat            ChatNotifier
	locator         Locator
}

func NewStoreService(db *gorm.DB, itemRepo repositories.StoreItemRepository, bidRepo repositories.BidRepository, bookingRepo repositories.BookingRequestRepository, userRepo repositories.UserRepository) *StoreService {
	return &StoreService{
		db:          db,
		itemRepo:    itemRepo,
		bidRepo:     bidRepo,
		bookingRepo: bookingRepo,
		userRepo:    userRepo,
		chat:        noopChatNotifier{},
		locator:     noopLocator{},
	}
}

// WithLocator saves listings' rough locations through locator (nil = don't).
func (s *StoreService) WithLocator(locator Locator) *StoreService {
	if locator != nil {
		s.locator = locator
	}
	return s
}

// WithRequests lets listings answer requests: requests is where they live,
// chat tells a requester when a seller lists something for them (nil = no
// messages).
func (s *StoreService) WithRequests(requests repositories.ItemRequestRepository, chat ChatNotifier) *StoreService {
	s.requestRepo = requests
	if chat != nil {
		s.chat = chat
	}
	return s
}

func (s *StoreService) CreateItem(userID uint, req models.CreateStoreItemRequest) (*models.StoreItem, error) {
	title, description, err := validateListingText(req.Title, req.Description)
	if err != nil {
		return nil, err
	}
	at, err := parseLocation(req.Lat, req.Lng)
	if err != nil {
		return nil, err
	}

	// A listing made for a request must answer someone else's open one
	var answers *models.ItemRequest
	if req.RequestID != nil {
		if s.requestRepo == nil {
			return nil, ErrRequestClosed
		}
		answers, err = s.requestRepo.GetByID(*req.RequestID)
		if err != nil || answers.Status != "active" {
			return nil, ErrRequestClosed
		}
		if answers.RequesterID == userID {
			return nil, ErrOwnRequest
		}
	}

	// A price is optional (0 means agree it in chat); auctions need a start
	if req.PriceType == "" {
		req.PriceType = "fixed"
	}
	if req.PriceType == "fixed" && req.FixedPrice < 0 {
		return nil, NewUserError("price can't be negative")
	}
	if req.PriceType == "bidding" {
		if req.StartingBid <= 0 {
			return nil, NewUserError("starting bid must be greater than 0")
		}
		if req.MinBidIncrement <= 0 {
			req.MinBidIncrement = 1.0 // Default increment
		}
	}

	item := &models.StoreItem{
		Title:           title,
		Description:     description,
		SellerID:        userID,
		PriceType:       req.PriceType,
		FixedPrice:      req.FixedPrice,
		StartingBid:     req.StartingBid,
		MinBidIncrement: req.MinBidIncrement,
		Category:        req.Category,
		Condition:       req.Condition,
		Location:        req.Location,
		ShippingInfo:    req.ShippingInfo,
		Status:          "active",
		RequestID:       req.RequestID,
	}
	startListing(item)

	// Create image records
	for i, imageURL := range req.Images {
		item.Images = append(item.Images, models.ItemImage{
			URL:   imageURL,
			Order: i,
		})
	}

	if err := s.itemRepo.Create(item); err != nil {
		return nil, err
	}
	if at != nil {
		s.locator.Save("item", item.ID, *at)
	}

	if answers != nil {
		s.chat.StoreMessage(item.ID, userID, answers.RequesterID,
			fmt.Sprintf("I listed \"%s\" for your request \"%s\". Open it to book it.", item.Title, answers.Title))
	}

	return item, nil
}

// startListing puts item on the board for a fresh ListingLifetime; an
// auction's bidding runs for the same time.
func startListing(item *models.StoreItem) {
	deadline := time.Now().UTC().Add(ListingLifetime)
	item.Deadline = &deadline
	if item.PriceType == "bidding" {
		item.BidDeadline = &deadline
	}
}

// hasReactions reports whether anyone bid on or asked to book item; such a
// listing stays up past its deadline so the seller can deal.
func (s *StoreService) hasReactions(item *models.StoreItem) (bool, error) {
	if item.BidCount > 0 || len(item.Bids) > 0 {
		return true, nil
	}
	requests, err := s.bookingRepo.GetAllByItemID(item.ID)
	if err != nil {
		return false, err
	}
	return len(requests) > 0, nil
}

// pastDeadline reports whether item's note has come off the board for new
// people: its deadline passed and nobody reacted in time.
func (s *StoreService) pastDeadline(item *models.StoreItem) (bool, error) {
	if item.Deadline == nil || time.Now().Before(*item.Deadline) {
		return false, nil
	}
	reacted, err := s.hasReactions(item)
	return !reacted, err
}

// RepostItem puts an expired listing back on the board for a fresh
// ListingLifetime.
func (s *StoreService) RepostItem(id uint, userID uint) (*models.StoreItem, error) {
	item, err := s.itemRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if item.SellerID != userID {
		return nil, NewUserError("unauthorized: you can only repost your own items")
	}
	if item.Status != "expired" {
		return nil, NewUserError("only an expired listing can be reposted")
	}

	item.Status = "active"
	startListing(item)
	if err := s.itemRepo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

// ExpireStaleItems takes listings that reached their deadline without a bid
// or booking request off the board, returning how many it changed.
func (s *StoreService) ExpireStaleItems() (int64, error) {
	return s.itemRepo.ExpireUnanswered(time.Now().UTC())
}

func (s *StoreService) GetItem(id uint) (*models.StoreItem, error) {
	return s.itemRepo.GetByID(id)
}

func (s *StoreService) GetItems(filter models.StoreItemFilter) ([]models.StoreItem, int64, error) {
	return s.itemRepo.GetAll(filter)
}

func (s *StoreService) UpdateItem(id uint, userID uint, req models.UpdateStoreItemRequest) (*models.StoreItem, error) {
	item, err := s.itemRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if item.SellerID != userID {
		return nil, NewUserError("unauthorized: you can only update your own items")
	}

	if item.Status != "active" {
		return nil, NewUserError("cannot update item that is not active")
	}

	// Update fields; the note must still fit the board
	if req.Title != "" || req.Description != "" {
		title, description := item.Title, item.Description
		if req.Title != "" {
			title = req.Title
		}
		if req.Description != "" {
			description = req.Description
		}
		if item.Title, item.Description, err = validateListingText(title, description); err != nil {
			return nil, err
		}
	}
	if req.Category != "" {
		item.Category = req.Category
	}
	if len(req.Images) > 0 {
		var images []models.ItemImage
		for i, url := range req.Images {
			images = append(images, models.ItemImage{
				URL:   url,
				Order: i,
			})
		}
		item.Images = images
	}
	if req.Condition != "" {
		item.Condition = req.Condition
	}
	if req.Location != "" {
		item.Location = req.Location
	}
	if req.ShippingInfo != "" {
		item.ShippingInfo = req.ShippingInfo
	}

	err = s.itemRepo.Update(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *StoreService) DeleteItem(id uint, userID uint) error {
	item, err := s.itemRepo.GetByID(id)
	if err != nil {
		return err
	}

	if item.SellerID != userID {
		return NewUserError("unauthorized: you can only delete your own items")
	}

	if item.Status != "active" && item.Status != "expired" {
		return NewUserError("only a live or expired listing can be removed")
	}

	if err := s.itemRepo.Delete(id); err != nil {
		return err
	}
	s.locator.Delete("item", id)
	return nil
}

func (s *StoreService) PlaceBid(itemID uint, userID uint, req models.CreateBidRequest) (*models.Bid, error) {
	// Function to execute the bidding logic
	placeBidLogic := func(itemRepo repositories.StoreItemRepository, bidRepo repositories.BidRepository) (*models.Bid, error) {
		// Use GetByIDForUpdate if available (not nil DB), otherwise fallback to GetByID (for tests)
		var item *models.StoreItem
		var err error
		
		// Ideally we should always use GetByIDForUpdate, but for tests mocking might be easier if we check
		// However, standardizing on GetByIDForUpdate in the interface makes it clean.
		item, err = itemRepo.GetByIDForUpdate(itemID)
		if err != nil {
			return nil, err
		}

		if item.PriceType != "bidding" {
			return nil, NewUserError("this item is not available for bidding")
		}

		if item.Status != "active" {
			return nil, NewUserError("item is not active")
		}

		if item.SellerID == userID {
			return nil, NewUserError("you cannot bid on your own item")
		}

		// Bidding closes with the note; the seller can still accept a bid
		if item.BidDeadline != nil && time.Now().After(*item.BidDeadline) {
			return nil, NewUserError("bidding has ended for this item")
		}

		if contacts.Contains(req.Message) {
			return nil, ErrContactDetails
		}

		// Check minimum bid amount
		minBid := item.StartingBid
		if item.CurrentBid > 0 {
			minBid = item.CurrentBid + item.MinBidIncrement
		}

		if req.Amount < minBid {
			return nil, NewUserError("bid amount must be at least $" + formatPrice(minBid))
		}

		// Create bid
		bid := &models.Bid{
			ItemID:   itemID,
			BidderID: userID,
			Amount:   req.Amount,
			Message:  req.Message,
			Status:   "active",
		}

		err = bidRepo.Create(bid)
		if err != nil {
			return nil, err
		}

		// Update item's current bid
		item.CurrentBid = req.Amount
		err = itemRepo.Update(item)
		if err != nil {
			return nil, err
		}

		// Mark other bids as outbid
		_ = bidRepo.MarkOutbidBids(itemID, bid.ID)

		return bid, nil
	}

	// If no DB (e.g. testing), just run logic
	if s.db == nil {
		return placeBidLogic(s.itemRepo, s.bidRepo)
	}

	// Run in transaction
	var bid *models.Bid
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txItemRepo := repositories.NewStoreItemRepository(tx)
		txBidRepo := repositories.NewBidRepository(tx)
		
		var err error
		bid, err = placeBidLogic(txItemRepo, txBidRepo)
		return err
	})

	if err != nil {
		return nil, err
	}

	return bid, nil
}

func (s *StoreService) GetItemBids(itemID uint) ([]models.Bid, error) {
	return s.bidRepo.GetByItemID(itemID)
}

func (s *StoreService) AcceptBid(itemID uint, bidID uint, sellerID uint) error {
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return err
	}

	if item.SellerID != sellerID {
		return NewUserError("unauthorized: only the seller can accept bids")
	}

	if item.Status != "active" {
		return NewUserError("item is not active")
	}

	bid, err := s.bidRepo.GetByID(bidID)
	if err != nil {
		return err
	}

	if bid.ItemID != itemID {
		return NewUserError("bid does not belong to this item")
	}

	// Mark item as sold
	err = s.itemRepo.MarkAsSold(itemID, bid.BidderID)
	if err != nil {
		return err
	}

	// Mark winning bid
	err = s.bidRepo.UpdateBidStatus(bidID, "won")
	if err != nil {
		return err
	}

	// Mark other bids as outbid
	_ = s.bidRepo.MarkOutbidBids(itemID, bidID)

	return nil
}

func (s *StoreService) PurchaseItem(itemID uint, buyerID uint) error {
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return err
	}

	if item.PriceType != "fixed" {
		return NewUserError("this item is only available through bidding")
	}

	if item.Status != "active" {
		return NewUserError("item is not available for purchase")
	}

	if item.SellerID == buyerID {
		return NewUserError("you cannot purchase your own item")
	}

	// Mark item as sold
	return s.itemRepo.MarkAsSold(itemID, buyerID)
}

func (s *StoreService) GetUserListings(userID uint) ([]models.StoreItem, error) {
	return s.itemRepo.GetBySellerID(userID)
}

func (s *StoreService) GetUserPurchases(userID uint) ([]models.StoreItem, error) {
	return s.itemRepo.GetByBuyerID(userID)
}

func (s *StoreService) GetUserBids(userID uint) ([]models.Bid, error) {
	return s.bidRepo.GetByBidderID(userID)
}

// Booking Request methods
func (s *StoreService) CreateBookingRequest(itemID uint, requesterID uint, message string) (*models.BookingRequest, error) {
	// Check if item exists and is active
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return nil, err
	}
	if item.Status != "active" {
		return nil, NewUserError("item is not available for booking")
	}
	if item.SellerID == requesterID {
		return nil, NewUserError("cannot book your own item")
	}
	if contacts.Contains(message) {
		return nil, ErrContactDetails
	}
	// Checked here too, so the gap between expiry runs can't let a late
	// request in
	expired, err := s.pastDeadline(item)
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, ErrListingExpired
	}

	// Check if user already has a booking request for this item
	existing, err := s.bookingRepo.GetByItemAndRequester(itemID, requesterID)
	if err == nil && existing != nil {
		return nil, NewUserError("you already have a booking request for this item")
	}

	bookingRequest := &models.BookingRequest{
		ItemID:      itemID,
		RequesterID: requesterID,
		Status:      "pending",
		Message:     message,
	}

	err = s.bookingRepo.Create(bookingRequest)
	if err != nil {
		return nil, err
	}

	// Return with preloaded data
	bookingWithData, err := s.bookingRepo.GetByID(bookingRequest.ID)
	if err != nil {
		return nil, err
	}

	// Notify chat service asynchronously (don't block on this)
	go NotifyChatServiceAboutBooking(bookingWithData, item, s.bookingRepo)

	return bookingWithData, nil
}

func (s *StoreService) GetBookingRequestByItem(itemID uint, userID uint) (*models.BookingRequest, error) {
	// First check if user is the item owner or requester
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return nil, err
	}

	var bookingRequest *models.BookingRequest

	if item.SellerID == userID {
		// User is the owner, get any booking request for this item
		bookingRequest, err = s.bookingRepo.GetByItemID(itemID)
	} else {
		// User is potentially a requester, get their specific request
		bookingRequest, err = s.bookingRepo.GetByItemAndRequester(itemID, userID)
	}

	if err != nil {
		return nil, err
	}

	return bookingRequest, nil
}

func (s *StoreService) GetAllBookingRequestsByItem(itemID uint, userID uint) ([]models.BookingRequest, error) {
	// First check if user is the item owner
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return nil, err
	}

	if item.SellerID != userID {
		return nil, NewUserError("unauthorized: you are not the owner of this item")
	}

	// Get all booking requests for this item
	bookingRequests, err := s.bookingRepo.GetAllByItemID(itemID)
	if err != nil {
		return nil, err
	}

	return bookingRequests, nil
}

func (s *StoreService) ApproveBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error) {
	// Get the booking request
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}

	// Verify the owner is actually the item owner
	if request.Item.SellerID != ownerID {
		return nil, NewUserError("unauthorized: you are not the owner of this item")
	}

	if request.Status != "pending" {
		return nil, NewUserError("booking request is not pending")
	}

	// Check if any other booking for this item is already approved
	allRequests, err := s.bookingRepo.GetAllByItemID(request.ItemID)
	if err != nil {
		return nil, err
	}

	for _, req := range allRequests {
		if req.Status == "approved" {
			return nil, NewUserError("another booking is already approved for this item")
		}
	}

	// Update status to approved
	err = s.bookingRepo.UpdateStatus(requestID, "approved")
	if err != nil {
		return nil, err
	}

	s.fulfilRequest(request)

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

// fulfilRequest closes the request a listing answered once the requester's
// own booking on it is approved. Best effort: the booking already stands.
func (s *StoreService) fulfilRequest(booking *models.BookingRequest) {
	if s.requestRepo == nil || booking.Item == nil || booking.Item.RequestID == nil {
		return
	}
	wanted, err := s.requestRepo.GetByID(*booking.Item.RequestID)
	if err != nil || wanted.RequesterID != booking.RequesterID {
		return
	}
	if _, err := s.requestRepo.MarkFulfilled(wanted.ID, booking.ItemID); err != nil {
		fmt.Printf("Error closing request %d: %v\n", wanted.ID, err)
	}
}

func (s *StoreService) RejectBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error) {
	// Get the booking request
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}

	// Verify the owner is actually the item owner
	if request.Item.SellerID != ownerID {
		return nil, NewUserError("unauthorized: you are not the owner of this item")
	}

	if request.Status != "pending" {
		return nil, NewUserError("booking request is not pending")
	}

	// Update status to rejected
	err = s.bookingRepo.UpdateStatus(requestID, "rejected")
	if err != nil {
		return nil, err
	}

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

func (s *StoreService) GetUserBookingRequests(userID uint) ([]models.BookingRequest, error) {
	return s.bookingRepo.GetByRequesterID(userID)
}

// GetUserRatings lists every store rating userID has received, as a seller
// or as a buyer.
func (s *StoreService) GetUserRatings(userID uint) ([]models.ReceivedRating, error) {
	requests, err := s.bookingRepo.GetRatingsReceived(userID)
	if err != nil {
		return nil, err
	}

	ratings := make([]models.ReceivedRating, 0, len(requests))
	for _, req := range requests {
		var sellerID uint
		var itemTitle string
		if req.Item != nil {
			sellerID = req.Item.SellerID
			itemTitle = req.Item.Title
		}

		if req.BuyerRating != nil && sellerID == userID {
			ratings = append(ratings, models.ReceivedRating{
				BookingID: req.ID, ItemID: req.ItemID, ItemTitle: itemTitle,
				RaterID: req.RequesterID, RatedAs: "seller",
				Rating: *req.BuyerRating, Review: req.BuyerReview, RatedAt: req.UpdatedAt,
			})
		}
		if req.SellerRating != nil && req.RequesterID == userID {
			ratings = append(ratings, models.ReceivedRating{
				BookingID: req.ID, ItemID: req.ItemID, ItemTitle: itemTitle,
				RaterID: sellerID, RatedAs: "buyer",
				Rating: *req.SellerRating, Review: req.SellerReview, RatedAt: req.UpdatedAt,
			})
		}
	}
	return ratings, nil
}

func (s *StoreService) ConfirmItemReceived(requestID uint, buyerID uint) (*models.BookingRequest, error) {
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, NewUserError("booking request not found")
	}

	// Only the requester (buyer) can confirm receipt
	if request.RequesterID != buyerID {
		return nil, NewUserError("only the buyer can confirm receipt")
	}

	// Must be in approved status
	if request.Status != "approved" {
		return nil, NewUserError("booking must be approved before confirming receipt")
	}

	err = s.bookingRepo.UpdateStatus(requestID, "item_received")
	if err != nil {
		return nil, err
	}

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

func (s *StoreService) ConfirmDelivery(requestID uint, sellerID uint) (*models.BookingRequest, error) {
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, NewUserError("booking request not found")
	}

	// Get item to verify seller
	item, err := s.itemRepo.GetByID(request.ItemID)
	if err != nil {
		return nil, err
	}

	// Only the item owner (seller) can confirm delivery
	if item.SellerID != sellerID {
		return nil, NewUserError("only the seller can confirm delivery")
	}

	// Must be in item_received status
	if request.Status != "item_received" {
		return nil, NewUserError("buyer must confirm receipt before seller can confirm delivery")
	}

	// Update booking status
	err = s.bookingRepo.UpdateStatus(requestID, "completed")
	if err != nil {
		return nil, err
	}

	// Mark the item as sold
	err = s.itemRepo.MarkAsSold(item.ID, request.RequesterID)
	if err != nil {
		// Log error but don't fail the request since booking is already completed
		fmt.Printf("Error marking item %d as sold: %v\n", item.ID, err)
	}

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

func (s *StoreService) SubmitBuyerRating(requestID uint, buyerID uint, rating int, review string) (*models.BookingRequest, error) {
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, NewUserError("booking request not found")
	}

	// Only the requester (buyer) can submit this rating
	if request.RequesterID != buyerID {
		return nil, NewUserError("only the buyer can rate the seller")
	}

	// Must be in completed status
	if request.Status != "completed" {
		return nil, NewUserError("booking must be completed before rating")
	}

	// Check if already rated
	if request.BuyerRating != nil {
		return nil, NewUserError("buyer has already rated this transaction")
	}

	// Update booking with rating
	err = s.bookingRepo.UpdateBuyerRating(requestID, rating, review)
	if err != nil {
		return nil, err
	}

	// Update seller's overall rating
	if request.Item != nil {
		err = s.userRepo.UpdateRating(request.Item.SellerID, float64(rating))
		if err != nil {
			return nil, err
		}
	}

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

func (s *StoreService) SubmitSellerRating(requestID uint, sellerID uint, rating int, review string) (*models.BookingRequest, error) {
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, NewUserError("booking request not found")
	}

	// Get item to verify seller
	item, err := s.itemRepo.GetByID(request.ItemID)
	if err != nil {
		return nil, err
	}

	// Only the item owner (seller) can submit this rating
	if item.SellerID != sellerID {
		return nil, NewUserError("only the seller can rate the buyer")
	}

	// Must be in completed status
	if request.Status != "completed" {
		return nil, NewUserError("booking must be completed before rating")
	}

	// Check if already rated
	if request.SellerRating != nil {
		return nil, NewUserError("seller has already rated this transaction")
	}

	// Update booking with rating
	err = s.bookingRepo.UpdateSellerRating(requestID, rating, review)
	if err != nil {
		return nil, err
	}

	// Update buyer's overall rating
	err = s.userRepo.UpdateRating(request.RequesterID, float64(rating))
	if err != nil {
		return nil, err
	}

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

// Helper function to format price
func formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}