package repositories

import (
	"fmt"
	"store-service/internal/models"
	"time"

	"gorm.io/gorm"
)

type itemRequestRepository struct {
	db *gorm.DB
}

func NewItemRequestRepository(db *gorm.DB) ItemRequestRepository {
	return &itemRequestRepository{db: db}
}

func (r *itemRequestRepository) Create(request *models.ItemRequest) error {
	return r.db.Create(request).Error
}

func (r *itemRequestRepository) GetByID(id uint) (*models.ItemRequest, error) {
	var request models.ItemRequest
	if err := r.db.Preload("Requester").First(&request, id).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *itemRequestRepository) GetAll(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error) {
	var requests []models.ItemRequest
	var total int64

	status := filter.Status
	if status == "" {
		status = "active"
	}
	query := r.db.Model(&models.ItemRequest{}).Where("status = ?", status)
	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		query = query.Where("LOWER(title) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)", pattern, pattern)
	}
	if filter.ExcludeRequesterID > 0 {
		query = query.Where("requester_id <> ?", filter.ExcludeRequesterID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "DESC"
	if filter.SortOrder == "asc" {
		order = "ASC"
	}
	switch filter.SortBy {
	case "deadline":
		query = query.Order(fmt.Sprintf("deadline %s", order))
	default:
		query = query.Order(fmt.Sprintf("created_at %s", order))
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	err := query.Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage).
		Preload("Requester").Find(&requests).Error
	return requests, total, err
}

func (r *itemRequestRepository) GetByRequesterID(requesterID uint) ([]models.ItemRequest, error) {
	var requests []models.ItemRequest
	err := r.db.Where("requester_id = ?", requesterID).Order("created_at DESC").Find(&requests).Error
	return requests, err
}

func (r *itemRequestRepository) Update(request *models.ItemRequest) error {
	return r.db.Save(request).Error
}

func (r *itemRequestRepository) Delete(id uint) error {
	return r.db.Delete(&models.ItemRequest{}, id).Error
}

// MarkFulfilled closes an active request, recording the listing that met it.
// It reports whether the request was still active.
func (r *itemRequestRepository) MarkFulfilled(id uint, itemID uint) (bool, error) {
	res := r.db.Model(&models.ItemRequest{}).Where("id = ? AND status = ?", id, "active").
		Updates(map[string]interface{}{"status": "fulfilled", "fulfilled_item_id": itemID})
	return res.RowsAffected > 0, res.Error
}

// ExpireUnanswered takes requests whose deadline passed before any seller
// listed something for them off the board, returning how many changed.
func (r *itemRequestRepository) ExpireUnanswered(now time.Time) (int64, error) {
	res := r.db.Model(&models.ItemRequest{}).
		Where("status = ? AND deadline < ?", "active", now).
		Where("NOT EXISTS (SELECT 1 FROM store_items WHERE store_items.request_id = item_requests.id AND store_items.deleted_at IS NULL)").
		Update("status", "expired")
	return res.RowsAffected, res.Error
}
