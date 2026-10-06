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
	err := r.db.Preload("Seller").Preload("Request").Preload("Images", func(db *gorm.DB) *gorm.DB {
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

// filtered applies every StoreItemFilter condition (not sorting or paging).
func (r *storeItemRepository) filtered(filter models.StoreItemFilter) *gorm.DB {
	query := r.db.Model(&models.StoreItem{})

	// Apply filters
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		// SQLite doesn't support ILIKE, use LIKE with LOWER for case-insensitive search
		query = query.Where("LOWER(title) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)", searchPattern, searchPattern)
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
		// By default the board: items still for sale, and reserved ones (shown
		// as reserved, no longer bookable)
		query = query.Where("status IN ?", []string{"active", "reserved"})
	}

	if filter.SellerID > 0 {
		query = query.Where("seller_id = ?", filter.SellerID)
	}

	if filter.ExcludeSellerID > 0 {
		query = query.Where("seller_id <> ?", filter.ExcludeSellerID)
	}

	if filter.RequestID > 0 {
		query = query.Where("request_id = ?", filter.RequestID)
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

	return query
}

func (r *storeItemRepository) GetAll(filter models.StoreItemFilter) ([]models.StoreItem, int64, error) {
	var items []models.StoreItem
	var totalCount int64

	query := r.filtered(filter)

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
	case "deadline":
		query = query.Order(fmt.Sprintf("deadline %s", sortOrder))
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

// ListIDs returns the ids of every listing matching filter, newest first,
// ignoring sorting and paging.
func (r *storeItemRepository) ListIDs(filter models.StoreItemFilter) ([]uint, error) {
	var ids []uint
	err := r.filtered(filter).Order("created_at DESC").Order("id DESC").Pluck("id", &ids).Error
	return ids, err
}

// GetByIDs loads listings with their photos and seller, in no particular order.
func (r *storeItemRepository) GetByIDs(ids []uint) ([]models.StoreItem, error) {
	var items []models.StoreItem
	if len(ids) == 0 {
		return items, nil
	}
	err := r.db.Where("id IN ?", ids).Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("\"order\" ASC")
	}).Preload("Seller").Find(&items).Error
	return items, err
}

func (r *storeItemRepository) Update(item *models.StoreItem) error {
	return r.db.Save(item).Error
}

func (r *storeItemRepository) Delete(id uint) error {
	return r.db.Delete(&models.StoreItem{}, id).Error
}

func (r *storeItemRepository) GetBySellerID(sellerID uint) ([]models.StoreItem, error) {
	var items []models.StoreItem
	err := r.db.Where("seller_id = ?", sellerID).Order("created_at DESC").Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("\"order\" ASC")
	}).Find(&items).Error
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

// ExpireUnanswered takes listings whose deadline passed before anyone bid
// or asked to book off the board, in one statement, returning how many
// changed. A listing with a reaction stays up so the seller can deal.
func (r *storeItemRepository) ExpireUnanswered(now time.Time) (int64, error) {
	res := r.db.Model(&models.StoreItem{}).
		Where("status = ? AND deadline < ?", "active", now).
		Where("NOT EXISTS (SELECT 1 FROM bids WHERE bids.item_id = store_items.id AND bids.deleted_at IS NULL)").
		Where("NOT EXISTS (SELECT 1 FROM booking_requests WHERE booking_requests.item_id = store_items.id AND booking_requests.deleted_at IS NULL)").
		Update("status", "expired")
	return res.RowsAffected, res.Error
}

// StartMissingDeadlines gives active listings that predate deadlines one,
// so they come off the board like any other note.
func (r *storeItemRepository) StartMissingDeadlines(deadline time.Time) (int64, error) {
	res := r.db.Model(&models.StoreItem{}).
		Where("status = ? AND deadline IS NULL", "active").
		Updates(map[string]interface{}{"deadline": deadline})
	return res.RowsAffected, res.Error
}
