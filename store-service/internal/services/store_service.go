package services

import (
	"errors"
	"fmt"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"time"
)

type StoreService struct {
	itemRepo repositories.StoreItemRepository
	bidRepo  repositories.BidRepository
}

func NewStoreService(itemRepo repositories.StoreItemRepository, bidRepo repositories.BidRepository) *StoreService {
	return &StoreService{
		itemRepo: itemRepo,
		bidRepo:  bidRepo,
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
		Images:          req.Images,
		Condition:       req.Condition,
		Location:        req.Location,
		ShippingInfo:    req.ShippingInfo,
		Status:          "active",
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
		item.Images = req.Images
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
	item, err := s.itemRepo.GetByID(itemID)
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
		_ = s.itemRepo.UpdateStatus(itemID, "expired")
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

	err = s.bidRepo.Create(bid)
	if err != nil {
		return nil, err
	}

	// Update item's current bid
	item.CurrentBid = req.Amount
	err = s.itemRepo.Update(item)
	if err != nil {
		return nil, err
	}

	// Mark other bids as outbid
	_ = s.bidRepo.MarkOutbidBids(itemID, bid.ID)

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

// Helper function to format price
func formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}