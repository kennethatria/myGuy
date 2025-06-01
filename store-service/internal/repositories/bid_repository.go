package repositories

import (
	"store-service/internal/models"

	"gorm.io/gorm"
)

type bidRepository struct {
	db *gorm.DB
}

func NewBidRepository(db *gorm.DB) BidRepository {
	return &bidRepository{db: db}
}

func (r *bidRepository) Create(bid *models.Bid) error {
	return r.db.Create(bid).Error
}

func (r *bidRepository) GetByID(id uint) (*models.Bid, error) {
	var bid models.Bid
	err := r.db.First(&bid, id).Error
	if err != nil {
		return nil, err
	}
	return &bid, nil
}

func (r *bidRepository) GetByItemID(itemID uint) ([]models.Bid, error) {
	var bids []models.Bid
	err := r.db.Where("item_id = ?", itemID).Order("amount DESC").Find(&bids).Error
	return bids, err
}

func (r *bidRepository) GetByBidderID(bidderID uint) ([]models.Bid, error) {
	var bids []models.Bid
	err := r.db.Where("bidder_id = ?", bidderID).Find(&bids).Error
	return bids, err
}

func (r *bidRepository) GetHighestBidForItem(itemID uint) (*models.Bid, error) {
	var bid models.Bid
	err := r.db.Where("item_id = ? AND status = ?", itemID, "active").Order("amount DESC").First(&bid).Error
	if err != nil {
		return nil, err
	}
	return &bid, nil
}

func (r *bidRepository) UpdateBidStatus(id uint, status string) error {
	return r.db.Model(&models.Bid{}).Where("id = ?", id).Update("status", status).Error
}

func (r *bidRepository) MarkOutbidBids(itemID uint, winningBidID uint) error {
	return r.db.Model(&models.Bid{}).
		Where("item_id = ? AND id != ? AND status = ?", itemID, winningBidID, "active").
		Update("status", "outbid").Error
}

func (r *bidRepository) GetActiveBidsForItem(itemID uint) ([]models.Bid, error) {
	var bids []models.Bid
	err := r.db.Where("item_id = ? AND status = ?", itemID, "active").Order("amount DESC").Find(&bids).Error
	return bids, err
}