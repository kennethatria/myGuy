package models

import (
	"time"

	"gorm.io/gorm"
)

type StoreItem struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Title           string         `json:"title" gorm:"not null"`
	Description     string         `json:"description"`
	SellerID        uint           `json:"seller_id" gorm:"not null;index:idx_store_items_seller_id"`
	Seller          *User          `json:"seller,omitempty" gorm:"foreignKey:SellerID"`
	PriceType       string         `json:"price_type" gorm:"not null;index:idx_store_items_price_type"` // "fixed" or "bidding"
	FixedPrice      float64        `json:"fixed_price,omitempty" gorm:"index:idx_store_items_fixed_price"`
	StartingBid     float64        `json:"starting_bid,omitempty"`
	CurrentBid      float64        `json:"current_bid,omitempty"`
	MinBidIncrement float64        `json:"min_bid_increment,omitempty"`
	BidDeadline     *time.Time     `json:"bid_deadline,omitempty" gorm:"index:idx_store_items_bid_deadline"`
	Status          string         `json:"status" gorm:"default:'active';index:idx_store_items_status"` // active, sold, expired, cancelled
	Category        string         `json:"category" gorm:"index:idx_store_items_category"`
	Images          []ItemImage    `json:"images" gorm:"foreignKey:ItemID"`
	Condition       string         `json:"condition" gorm:"index:idx_store_items_condition"` // new, like-new, good, fair, poor
	Location        string         `json:"location"`
	ShippingInfo    string         `json:"shipping_info"`
	BuyerID         *uint          `json:"buyer_id,omitempty" gorm:"index:idx_store_items_buyer_id"`
	SoldAt          *time.Time     `json:"sold_at,omitempty"`
	Bids            []Bid          `json:"bids,omitempty" gorm:"foreignKey:ItemID"`
	BidCount        int            `json:"bid_count" gorm:"-"`
	IsAuction       bool           `json:"is_auction" gorm:"-"`
	Price           float64        `json:"price" gorm:"-"`
	CreatedAt       time.Time      `json:"created_at" gorm:"index:idx_store_items_created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

type ItemImage struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ItemID    uint           `json:"item_id" gorm:"not null;index:idx_item_images_item_id"`
	URL       string         `json:"url" gorm:"not null"`
	Order     int            `json:"order" gorm:"default:0"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type Bid struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ItemID    uint           `json:"item_id" gorm:"not null;index:idx_bids_item_id"`
	Item      StoreItem      `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	BidderID  uint           `json:"bidder_id" gorm:"not null;index:idx_bids_bidder_id"`
	Amount    float64        `json:"amount" gorm:"not null"`
	Message   string         `json:"message"`
	Status    string         `json:"status" gorm:"default:'active';index:idx_bids_status"` // active, outbid, won, cancelled
	CreatedAt time.Time      `json:"created_at" gorm:"index:idx_bids_created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type BookingRequest struct {
	ID                       uint           `json:"id" gorm:"primaryKey"`
	ItemID                   uint           `json:"item_id" gorm:"not null;index:idx_booking_requests_item_id"`
	Item                     *StoreItem     `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	RequesterID              uint           `json:"requester_id" gorm:"not null;index:idx_booking_requests_requester_id"`
	Requester                *User          `json:"requester,omitempty" gorm:"foreignKey:RequesterID"`
	Status                   string         `json:"status" gorm:"default:'pending';index:idx_booking_requests_status"` // pending, approved, rejected, item_received, completed
	Message                  string         `json:"message"`
	BuyerRating              *int           `json:"buyer_rating,omitempty"` // Buyer's rating of seller (1-5)
	BuyerReview              string         `json:"buyer_review,omitempty"`
	SellerRating             *int           `json:"seller_rating,omitempty"` // Seller's rating of buyer (1-5)
	SellerReview             string         `json:"seller_review,omitempty"`
	ChatNotified             bool           `json:"-" gorm:"default:false;index:idx_booking_requests_chat_notified"`
	NotificationAttempts     int            `json:"-" gorm:"default:0"`
	LastNotificationAttempt  *time.Time     `json:"-"`
	CreatedAt                time.Time      `json:"created_at" gorm:"index:idx_booking_requests_created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `json:"-" gorm:"index"`
}

// DTOs for API requests/responses
type CreateStoreItemRequest struct {
	Title           string    `json:"title" binding:"required"`
	Description     string    `json:"description"`
	PriceType       string    `json:"price_type" binding:"required,oneof=fixed bidding"`
	FixedPrice      float64   `json:"fixed_price,omitempty"`
	StartingBid     float64   `json:"starting_bid,omitempty"`
	MinBidIncrement float64   `json:"min_bid_increment,omitempty"`
	BidDeadline     *time.Time `json:"bid_deadline,omitempty"`
	Category        string    `json:"category"`
	Images          []string  `json:"images"`
	Condition       string    `json:"condition" binding:"oneof=new like-new good fair poor"`
	Location        string    `json:"location"`
	ShippingInfo    string    `json:"shipping_info"`
}

type UpdateStoreItemRequest struct {
	Title        string   `json:"title,omitempty"`
	Description  string   `json:"description,omitempty"`
	Category     string   `json:"category,omitempty"`
	Images       []string `json:"images,omitempty"`
	Condition    string   `json:"condition,omitempty"`
	Location     string   `json:"location,omitempty"`
	ShippingInfo string   `json:"shipping_info,omitempty"`
}

type CreateBidRequest struct {
	Amount  float64 `json:"amount" binding:"required"`
	Message string  `json:"message,omitempty"`
}

type CreateBookingRequestRequest struct {
	Message string `json:"message,omitempty"`
}

type SubmitRatingRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Review string `json:"review,omitempty"`
}

type StoreItemFilter struct {
	Search      string
	Category    string
	PriceType   string
	MinPrice    float64
	MaxPrice    float64
	Condition   string
	SellerID    uint
	Status      string
	SortBy      string
	SortOrder   string
	Page        int
	PerPage     int
}

// AfterFind hook to populate computed fields
func (s *StoreItem) AfterFind(tx *gorm.DB) error {
	s.IsAuction = s.PriceType == "bidding"
	if s.IsAuction {
		s.Price = s.CurrentBid
		if s.Price == 0 {
			s.Price = s.StartingBid
		}
	} else {
		s.Price = s.FixedPrice
	}
	
	// Count bids
	var bidCount int64
	tx.Model(&Bid{}).Where("item_id = ?", s.ID).Count(&bidCount)
	s.BidCount = int(bidCount)
	
	return nil
}