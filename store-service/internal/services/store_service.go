package services

import (
	"errors"
	"fmt"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"time"

	"gorm.io/gorm"
)

type StoreService struct {
	db              *gorm.DB
	itemRepo        repositories.StoreItemRepository
	bidRepo         repositories.BidRepository
	bookingRepo     repositories.BookingRequestRepository
	userRepo        repositories.UserRepository
}

func NewStoreService(db *gorm.DB, itemRepo repositories.StoreItemRepository, bidRepo repositories.BidRepository, bookingRepo repositories.BookingRequestRepository, userRepo repositories.UserRepository) *StoreService {
	return &StoreService{
		db:          db,
		itemRepo:    itemRepo,
		bidRepo:     bidRepo,
		bookingRepo: bookingRepo,
		userRepo:    userRepo,
	}
}

func (s *StoreService) CreateItem(userID uint, req models.CreateStoreItemRequest) (*models.StoreItem, error) {
	// Validate price based on type
	if req.PriceType == "fixed" && req.FixedPrice <= 0 {
		return nil, errors.New("fixed price must be greater than 0")
	}
	if req.PriceType == "bidding" {
		if req.StartingBid <= 0 {
			return nil, errors.New("starting bid must be greater than 0")
		}
		if req.MinBidIncrement <= 0 {
			req.MinBidIncrement = 1.0 // Default increment
		}
		if req.BidDeadline != nil && req.BidDeadline.Before(time.Now()) {
			return nil, errors.New("bid deadline must be in the future")
		}
	}

	item := &models.StoreItem{
		Title:           req.Title,
		Description:     req.Description,
		SellerID:        userID,
		PriceType:       req.PriceType,
		FixedPrice:      req.FixedPrice,
		StartingBid:     req.StartingBid,
		MinBidIncrement: req.MinBidIncrement,
		BidDeadline:     req.BidDeadline,
		Category:        req.Category,
		Condition:       req.Condition,
		Location:        req.Location,
		ShippingInfo:    req.ShippingInfo,
		Status:          "active",
	}
	
	// Create image records
	for i, imageURL := range req.Images {
		item.Images = append(item.Images, models.ItemImage{
			URL:   imageURL,
			Order: i,
		})
	}

	err := s.itemRepo.Create(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *StoreService) GetItem(id uint) (*models.StoreItem, error) {
	return s.itemRepo.GetByID(id)
}

func (s *StoreService) GetItems(filter models.StoreItemFilter) ([]models.StoreItem, int64, error) {
	// Expire old bid items before fetching
	_ = s.itemRepo.ExpireOldBidItems()
	
	return s.itemRepo.GetAll(filter)
}

func (s *StoreService) UpdateItem(id uint, userID uint, req models.UpdateStoreItemRequest) (*models.StoreItem, error) {
	item, err := s.itemRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if item.SellerID != userID {
		return nil, errors.New("unauthorized: you can only update your own items")
	}

	if item.Status != "active" {
		return nil, errors.New("cannot update item that is not active")
	}

	// Update fields
	if req.Title != "" {
		item.Title = req.Title
	}
	if req.Description != "" {
		item.Description = req.Description
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
		return errors.New("unauthorized: you can only delete your own items")
	}

	if item.Status != "active" {
		return errors.New("cannot delete item that is not active")
	}

	return s.itemRepo.Delete(id)
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
			return nil, errors.New("this item is not available for bidding")
		}

		if item.Status != "active" {
			return nil, errors.New("item is not active")
		}

		if item.SellerID == userID {
			return nil, errors.New("you cannot bid on your own item")
		}

		if item.BidDeadline != nil && time.Now().After(*item.BidDeadline) {
			// Mark item as expired
			_ = itemRepo.UpdateStatus(itemID, "expired")
			return nil, errors.New("bidding has ended for this item")
		}

		// Check minimum bid amount
		minBid := item.StartingBid
		if item.CurrentBid > 0 {
			minBid = item.CurrentBid + item.MinBidIncrement
		}

		if req.Amount < minBid {
			return nil, errors.New("bid amount must be at least $" + formatPrice(minBid))
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
		return errors.New("unauthorized: only the seller can accept bids")
	}

	if item.Status != "active" {
		return errors.New("item is not active")
	}

	bid, err := s.bidRepo.GetByID(bidID)
	if err != nil {
		return err
	}

	if bid.ItemID != itemID {
		return errors.New("bid does not belong to this item")
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
		return errors.New("this item is only available through bidding")
	}

	if item.Status != "active" {
		return errors.New("item is not available for purchase")
	}

	if item.SellerID == buyerID {
		return errors.New("you cannot purchase your own item")
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
		return nil, errors.New("item is not available for booking")
	}
	if item.SellerID == requesterID {
		return nil, errors.New("cannot book your own item")
	}

	// Check if user already has a booking request for this item
	existing, err := s.bookingRepo.GetByItemAndRequester(itemID, requesterID)
	if err == nil && existing != nil {
		return nil, errors.New("you already have a booking request for this item")
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
		return nil, errors.New("unauthorized: you are not the owner of this item")
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
		return nil, errors.New("unauthorized: you are not the owner of this item")
	}

	if request.Status != "pending" {
		return nil, errors.New("booking request is not pending")
	}

	// Check if any other booking for this item is already approved
	allRequests, err := s.bookingRepo.GetAllByItemID(request.ItemID)
	if err != nil {
		return nil, err
	}

	for _, req := range allRequests {
		if req.Status == "approved" {
			return nil, errors.New("another booking is already approved for this item")
		}
	}

	// Update status to approved
	err = s.bookingRepo.UpdateStatus(requestID, "approved")
	if err != nil {
		return nil, err
	}

	// Get and return the updated booking request
	return s.bookingRepo.GetByID(requestID)
}

func (s *StoreService) RejectBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error) {
	// Get the booking request
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}

	// Verify the owner is actually the item owner
	if request.Item.SellerID != ownerID {
		return nil, errors.New("unauthorized: you are not the owner of this item")
	}

	if request.Status != "pending" {
		return nil, errors.New("booking request is not pending")
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

func (s *StoreService) ConfirmItemReceived(requestID uint, buyerID uint) (*models.BookingRequest, error) {
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, errors.New("booking request not found")
	}

	// Only the requester (buyer) can confirm receipt
	if request.RequesterID != buyerID {
		return nil, errors.New("only the buyer can confirm receipt")
	}

	// Must be in approved status
	if request.Status != "approved" {
		return nil, errors.New("booking must be approved before confirming receipt")
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
		return nil, errors.New("booking request not found")
	}

	// Get item to verify seller
	item, err := s.itemRepo.GetByID(request.ItemID)
	if err != nil {
		return nil, err
	}

	// Only the item owner (seller) can confirm delivery
	if item.SellerID != sellerID {
		return nil, errors.New("only the seller can confirm delivery")
	}

	// Must be in item_received status
	if request.Status != "item_received" {
		return nil, errors.New("buyer must confirm receipt before seller can confirm delivery")
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
		return nil, errors.New("booking request not found")
	}

	// Only the requester (buyer) can submit this rating
	if request.RequesterID != buyerID {
		return nil, errors.New("only the buyer can rate the seller")
	}

	// Must be in completed status
	if request.Status != "completed" {
		return nil, errors.New("booking must be completed before rating")
	}

	// Check if already rated
	if request.BuyerRating != nil {
		return nil, errors.New("buyer has already rated this transaction")
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
		return nil, errors.New("booking request not found")
	}

	// Get item to verify seller
	item, err := s.itemRepo.GetByID(request.ItemID)
	if err != nil {
		return nil, err
	}

	// Only the item owner (seller) can submit this rating
	if item.SellerID != sellerID {
		return nil, errors.New("only the seller can rate the buyer")
	}

	// Must be in completed status
	if request.Status != "completed" {
		return nil, errors.New("booking must be completed before rating")
	}

	// Check if already rated
	if request.SellerRating != nil {
		return nil, errors.New("seller has already rated this transaction")
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