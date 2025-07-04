package repositories

import (
	"store-service/internal/models"

	"gorm.io/gorm"
)

type bookingRequestRepository struct {
	db *gorm.DB
}

func NewBookingRequestRepository(db *gorm.DB) BookingRequestRepository {
	return &bookingRequestRepository{db: db}
}

func (r *bookingRequestRepository) Create(request *models.BookingRequest) error {
	return r.db.Create(request).Error
}

func (r *bookingRequestRepository) GetByID(id uint) (*models.BookingRequest, error) {
	var request models.BookingRequest
	err := r.db.Preload("Item").Preload("Requester").First(&request, id).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *bookingRequestRepository) GetByItemID(itemID uint) (*models.BookingRequest, error) {
	var request models.BookingRequest
	err := r.db.Preload("Item").Preload("Requester").Where("item_id = ?", itemID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *bookingRequestRepository) GetAllByItemID(itemID uint) ([]models.BookingRequest, error) {
	var requests []models.BookingRequest
	err := r.db.Preload("Item").Preload("Requester").Where("item_id = ?", itemID).Find(&requests).Error
	return requests, err
}

func (r *bookingRequestRepository) GetByItemAndRequester(itemID uint, requesterID uint) (*models.BookingRequest, error) {
	var request models.BookingRequest
	err := r.db.Preload("Item").Preload("Requester").Where("item_id = ? AND requester_id = ?", itemID, requesterID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *bookingRequestRepository) GetByRequesterID(requesterID uint) ([]models.BookingRequest, error) {
	var requests []models.BookingRequest
	err := r.db.Preload("Item").Preload("Requester").Where("requester_id = ?", requesterID).Find(&requests).Error
	return requests, err
}

func (r *bookingRequestRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.BookingRequest{}).Where("id = ?", id).Update("status", status).Error
}

func (r *bookingRequestRepository) Delete(id uint) error {
	return r.db.Delete(&models.BookingRequest{}, id).Error
}