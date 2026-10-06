package services

import (
	"store-service/internal/models"
	"store-service/internal/proximity"
)

type StoreServiceInterface interface {
	CreateItem(userID uint, req models.CreateStoreItemRequest) (*models.StoreItem, error)
	GetItem(id uint) (*models.StoreItem, error)
	GetItems(filter models.StoreItemFilter) ([]models.StoreItem, int64, error)
	GetItemsNear(filter models.StoreItemFilter, at proximity.Location) ([]models.StoreItem, int64, error)
	TagItemDistances(items []models.StoreItem, at proximity.Location)
	UpdateItem(id uint, userID uint, req models.UpdateStoreItemRequest) (*models.StoreItem, error)
	DeleteItem(id uint, userID uint) error
	RepostItem(id uint, userID uint) (*models.StoreItem, error)
	PlaceBid(itemID uint, userID uint, req models.CreateBidRequest) (*models.Bid, error)
	GetItemBids(itemID uint) ([]models.Bid, error)
	AcceptBid(itemID uint, bidID uint, sellerID uint) error
	PurchaseItem(itemID uint, buyerID uint) error
	GetUserListings(userID uint) ([]models.StoreItem, error)
	GetUserPurchases(userID uint) ([]models.StoreItem, error)
	GetUserBids(userID uint) ([]models.Bid, error)
	CreateBookingRequest(itemID uint, requesterID uint, message string) (*models.BookingRequest, error)
	GetBookingRequestByItem(itemID uint, userID uint) (*models.BookingRequest, error)
	GetAllBookingRequestsByItem(itemID uint, userID uint) ([]models.BookingRequest, error)
	ApproveBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error)
	RejectBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error)
	GetUserBookingRequests(userID uint) ([]models.BookingRequest, error)
	GetUserRatings(userID uint) ([]models.BookingRating, error)
	GetMyRatings(userID uint) ([]models.BookingRating, error)
	ConfirmItemReceived(requestID uint, buyerID uint) (*models.BookingRequest, error)
	ConfirmDelivery(requestID uint, sellerID uint) (*models.BookingRequest, error)
	SubmitBuyerRating(requestID uint, buyerID uint, rating int, review string) (*models.BookingRequest, error)
	SubmitSellerRating(requestID uint, sellerID uint, rating int, review string) (*models.BookingRequest, error)
}