package repositories

import (
	"store-service/internal/models"
	"time"
)

type StoreItemRepository interface {
	Create(item *models.StoreItem) error
	GetByID(id uint) (*models.StoreItem, error)
	GetByIDForUpdate(id uint) (*models.StoreItem, error)
	GetAll(filter models.StoreItemFilter) ([]models.StoreItem, int64, error)
	ListIDs(filter models.StoreItemFilter) ([]uint, error)
	GetByIDs(ids []uint) ([]models.StoreItem, error)
	Update(item *models.StoreItem) error
	Delete(id uint) error
	GetBySellerID(sellerID uint) ([]models.StoreItem, error)
	GetByBuyerID(buyerID uint) ([]models.StoreItem, error)
	UpdateStatus(id uint, status string) error
	MarkAsSold(id uint, buyerID uint) error
	ExpireUnanswered(now time.Time) (int64, error)
	StartMissingDeadlines(deadline time.Time) (int64, error)
}

type ItemRequestRepository interface {
	Create(request *models.ItemRequest) error
	GetByID(id uint) (*models.ItemRequest, error)
	GetAll(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error)
	ListIDs(filter models.ItemRequestFilter) ([]uint, error)
	GetByIDs(ids []uint) ([]models.ItemRequest, error)
	GetByRequesterID(requesterID uint) ([]models.ItemRequest, error)
	Update(request *models.ItemRequest) error
	Delete(id uint) error
	MarkFulfilled(id uint, itemID uint) (bool, error)
	Reopen(id uint, deadline time.Time) (bool, error)
	ExpireUnanswered(now time.Time) (int64, error)
}

type BidRepository interface {
	Create(bid *models.Bid) error
	GetByID(id uint) (*models.Bid, error)
	GetByItemID(itemID uint) ([]models.Bid, error)
	GetByBidderID(bidderID uint) ([]models.Bid, error)
	GetHighestBidForItem(itemID uint) (*models.Bid, error)
	UpdateBidStatus(id uint, status string) error
	MarkOutbidBids(itemID uint, winningBidID uint) error
	GetActiveBidsForItem(itemID uint) ([]models.Bid, error)
}

type BookingRequestRepository interface {
	Create(request *models.BookingRequest) error
	GetByID(id uint) (*models.BookingRequest, error)
	GetByItemID(itemID uint) (*models.BookingRequest, error)
	GetAllByItemID(itemID uint) ([]models.BookingRequest, error)
	GetByItemAndRequester(itemID uint, requesterID uint) (*models.BookingRequest, error)
	GetByRequesterID(requesterID uint) ([]models.BookingRequest, error)
	UpdateStatus(id uint, status string) error
	Delete(id uint) error
	UpdateChatNotificationStatus(bookingID uint, notified bool, attempts int) error
	IncrementNotificationAttempts(bookingID uint) error
	UpdateBuyerRating(id uint, rating int, review string) error
	UpdateSellerRating(id uint, rating int, review string) error
	GetRatingsReceived(userID uint) ([]models.BookingRequest, error)
	GetRatingsInvolving(userID uint) ([]models.BookingRequest, error)
	ListMatched() ([]models.BookingRequest, error)
}

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	UpsertFromJWT(userID uint, username, email, name string) (*models.User, error)
	UpdateRating(userID uint, newRating float64) error
}