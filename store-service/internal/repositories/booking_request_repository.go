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

func (r *bookingRequestRepository) UpdateChatNotificationStatus(bookingID uint, notified bool, attempts int) error {
	return r.db.Model(&models.BookingRequest{}).
		Where("id = ?", bookingID).
		Updates(map[string]interface{}{
			"chat_notified":             notified,
			"notification_attempts":     attempts,
			"last_notification_attempt": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (r *bookingRequestRepository) IncrementNotificationAttempts(bookingID uint) error {
	return r.db.Model(&models.BookingRequest{}).
		Where("id = ?", bookingID).
		Updates(map[string]interface{}{
			"notification_attempts":     gorm.Expr("notification_attempts + 1"),
			"last_notification_attempt": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (r *bookingRequestRepository) UpdateBuyerRating(id uint, rating int, review string) error {
	return r.db.Model(&models.BookingRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"buyer_rating": rating,
			"buyer_review": review,
		}).Error
}

func (r *bookingRequestRepository) UpdateSellerRating(id uint, rating int, review string) error {
	return r.db.Model(&models.BookingRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"seller_rating": rating,
			"seller_review": review,
		}).Error
}

// GetRatingsReceived returns bookings in which userID was rated: as the seller
// (buyer_rating) or as the buyer (seller_rating). Newest first.
func (r *bookingRequestRepository) GetRatingsReceived(userID uint) ([]models.BookingRequest, error) {
	var requests []models.BookingRequest
	err := r.db.Preload("Item").
		Joins("JOIN store_items ON store_items.id = booking_requests.item_id").
		Where("(store_items.seller_id = ? AND booking_requests.buyer_rating IS NOT NULL) OR "+
			"(booking_requests.requester_id = ? AND booking_requests.seller_rating IS NOT NULL)", userID, userID).
		Order("booking_requests.updated_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetRatingsInvolving returns bookings with a rating in which userID was the
// seller or the buyer: ratings they gave or received. Newest first.
func (r *bookingRequestRepository) GetRatingsInvolving(userID uint) ([]models.BookingRequest, error) {
	var requests []models.BookingRequest
	err := r.db.Preload("Item").
		Joins("JOIN store_items ON store_items.id = booking_requests.item_id").
		Where("(store_items.seller_id = ? OR booking_requests.requester_id = ?) AND "+
			"(booking_requests.buyer_rating IS NOT NULL OR booking_requests.seller_rating IS NOT NULL)", userID, userID).
		Order("booking_requests.updated_at DESC").
		Find(&requests).Error
	return requests, err
}
