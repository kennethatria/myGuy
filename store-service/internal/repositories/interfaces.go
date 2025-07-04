package repositories

import (
	"store-service/internal/models"
)

type StoreItemRepository interface {
	Create(item *models.StoreItem) error
	GetByID(id uint) (*models.StoreItem, error)
	GetAll(filter models.StoreItemFilter) ([]models.StoreItem, int64, error)
	Update(item *models.StoreItem) error
	Delete(id uint) error
	GetBySellerID(sellerID uint) ([]models.StoreItem, error)
	GetByBuyerID(buyerID uint) ([]models.StoreItem, error)
	UpdateStatus(id uint, status string) error
	MarkAsSold(id uint, buyerID uint) error
	ExpireOldBidItems() error
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
	GetByItemAndRequester(itemID uint, requesterID uint) (*models.BookingRequest, error)
	GetByRequesterID(requesterID uint) ([]models.BookingRequest, error)
	UpdateStatus(id uint, status string) error
	Delete(id uint) error
}

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	UpsertFromJWT(userID uint, username, email, name string) (*models.User, error)
}