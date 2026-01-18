package repositories

import (
	"fmt"
	"store-service/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storeItemRepository struct {
	db *gorm.DB
}

func NewStoreItemRepository(db *gorm.DB) StoreItemRepository {
	return &storeItemRepository{db: db}
}

func (r *storeItemRepository) Create(item *models.StoreItem) error {
	return r.db.Create(item).Error
}

func (r *storeItemRepository) GetByID(id uint) (*models.StoreItem, error) {
	var item models.StoreItem
	err := r.db.Preload("Seller").Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("\"order\" ASC")
	}).Preload("Bids", "status = ?", "active").First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *storeItemRepository) GetByIDForUpdate(id uint) (*models.StoreItem, error) {
	var item models.StoreItem
	err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Seller").Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("\"order\" ASC")
	}).Preload("Bids", "status = ?", "active").First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *storeItemRepository) GetAll(filter models.StoreItemFilter) ([]models.StoreItem, int64, error) {
	var items []models.StoreItem
	var totalCount int64

	query := r.db.Model(&models.StoreItem{})

	// Apply filters
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.PriceType != "" {
		query = query.Where("price_type = ?", filter.PriceType)
	}

	if filter.Condition != "" {
		query = query.Where("condition = ?", filter.Condition)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		// Default to showing only active items unless a specific status is requested
		query = query.Where("status = ?", "active")
	}

	if filter.SellerID > 0 {
		query = query.Where("seller_id = ?", filter.SellerID)
	}

	// Price filtering based on price type
	if filter.MinPrice > 0 || filter.MaxPrice > 0 {
		if filter.PriceType == "fixed" {
			if filter.MinPrice > 0 {
				query = query.Where("fixed_price >= ?", filter.MinPrice)
			}
			if filter.MaxPrice > 0 {
				query = query.Where("fixed_price <= ?", filter.MaxPrice)
			}
		} else if filter.PriceType == "bidding" {
			if filter.MinPrice > 0 {
				query = query.Where("current_bid >= ? OR (current_bid = 0 AND starting_bid >= ?)", filter.MinPrice, filter.MinPrice)
			}
			if filter.MaxPrice > 0 {
				query = query.Where("current_bid <= ? OR (current_bid = 0 AND starting_bid <= ?)", filter.MaxPrice, filter.MaxPrice)
			}
		}
	}

	// Count total records
	query.Count(&totalCount)

	// Sorting
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	switch filter.SortBy {
	case "price":
		query = query.Order(fmt.Sprintf("COALESCE(fixed_price, current_bid, starting_bid) %s", sortOrder))
	case "created_at":
		query = query.Order(fmt.Sprintf("created_at %s", sortOrder))
	case "title":
		query = query.Order(fmt.Sprintf("title %s", sortOrder))
	default:
		query = query.Order("created_at DESC")
	}

	// Pagination
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}

	offset := (filter.Page - 1) * filter.PerPage
	query = query.Offset(offset).Limit(filter.PerPage)

	// Execute query with preloads
	err := query.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("\"order\" ASC")
	}).Preload("Seller").Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, totalCount, nil
}

func (r *storeItemRepository) Update(item *models.StoreItem) error {
	return r.db.Save(item).Error
}

func (r *storeItemRepository) Delete(id uint) error {
	return r.db.Delete(&models.StoreItem{}, id).Error
}

func (r *storeItemRepository) GetBySellerID(sellerID uint) ([]models.StoreItem, error) {
	var items []models.StoreItem
	err := r.db.Where("seller_id = ?", sellerID).Find(&items).Error
	return items, err
}

func (r *storeItemRepository) GetByBuyerID(buyerID uint) ([]models.StoreItem, error) {
	var items []models.StoreItem
	err := r.db.Where("buyer_id = ?", buyerID).Find(&items).Error
	return items, err
}

func (r *storeItemRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.StoreItem{}).Where("id = ?", id).Update("status", status).Error
}

func (r *storeItemRepository) MarkAsSold(id uint, buyerID uint) error {
	now := time.Now()
	return r.db.Model(&models.StoreItem{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":   "sold",
		"buyer_id": buyerID,
		"sold_at":  &now,
	}).Error
}

func (r *storeItemRepository) ExpireOldBidItems() error {
	now := time.Now()
	return r.db.Model(&models.StoreItem{}).
		Where("price_type = ? AND status = ? AND bid_deadline < ?", "bidding", "active", now).
		Update("status", "expired").Error
}