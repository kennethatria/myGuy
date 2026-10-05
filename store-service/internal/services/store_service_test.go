package services

import (
	"errors"
	"store-service/internal/models"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Mock repositories
type MockStoreItemRepository struct {
	mock.Mock
}

func (m *MockStoreItemRepository) Create(item *models.StoreItem) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockStoreItemRepository) GetByID(id uint) (*models.StoreItem, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoreItem), args.Error(1)
}

func (m *MockStoreItemRepository) GetByIDForUpdate(id uint) (*models.StoreItem, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoreItem), args.Error(1)
}

func (m *MockStoreItemRepository) GetAll(filter models.StoreItemFilter) ([]models.StoreItem, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.StoreItem), args.Get(1).(int64), args.Error(2)
}

func (m *MockStoreItemRepository) Update(item *models.StoreItem) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockStoreItemRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStoreItemRepository) GetBySellerID(sellerID uint) ([]models.StoreItem, error) {
	args := m.Called(sellerID)
	return args.Get(0).([]models.StoreItem), args.Error(1)
}

func (m *MockStoreItemRepository) GetByBuyerID(buyerID uint) ([]models.StoreItem, error) {
	args := m.Called(buyerID)
	return args.Get(0).([]models.StoreItem), args.Error(1)
}

func (m *MockStoreItemRepository) UpdateStatus(id uint, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockStoreItemRepository) MarkAsSold(id uint, buyerID uint) error {
	args := m.Called(id, buyerID)
	return args.Error(0)
}

func (m *MockStoreItemRepository) ExpireUnanswered(now time.Time) (int64, error) {
	args := m.Called(now)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStoreItemRepository) StartMissingDeadlines(deadline time.Time) (int64, error) {
	args := m.Called(deadline)
	return args.Get(0).(int64), args.Error(1)
}

type MockBidRepository struct {
	mock.Mock
}

func (m *MockBidRepository) Create(bid *models.Bid) error {
	args := m.Called(bid)
	return args.Error(0)
}

func (m *MockBidRepository) GetByID(id uint) (*models.Bid, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bid), args.Error(1)
}

func (m *MockBidRepository) GetByItemID(itemID uint) ([]models.Bid, error) {
	args := m.Called(itemID)
	return args.Get(0).([]models.Bid), args.Error(1)
}

func (m *MockBidRepository) GetByBidderID(bidderID uint) ([]models.Bid, error) {
	args := m.Called(bidderID)
	return args.Get(0).([]models.Bid), args.Error(1)
}

func (m *MockBidRepository) GetHighestBidForItem(itemID uint) (*models.Bid, error) {
	args := m.Called(itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bid), args.Error(1)
}

func (m *MockBidRepository) UpdateBidStatus(id uint, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockBidRepository) MarkOutbidBids(itemID uint, winningBidID uint) error {
	args := m.Called(itemID, winningBidID)
	return args.Error(0)
}

func (m *MockBidRepository) GetActiveBidsForItem(itemID uint) ([]models.Bid, error) {
	args := m.Called(itemID)
	return args.Get(0).([]models.Bid), args.Error(1)
}

type MockBookingRequestRepository struct {
	mock.Mock
}

func (m *MockBookingRequestRepository) Create(request *models.BookingRequest) error {
	args := m.Called(request)
	return args.Error(0)
}

func (m *MockBookingRequestRepository) GetByID(id uint) (*models.BookingRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockBookingRequestRepository) GetByItemID(itemID uint) (*models.BookingRequest, error) {
	args := m.Called(itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockBookingRequestRepository) GetByItemAndRequester(itemID uint, requesterID uint) (*models.BookingRequest, error) {
	args := m.Called(itemID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockBookingRequestRepository) GetRatingsReceived(userID uint) ([]models.BookingRequest, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.BookingRequest), args.Error(1)
}

func (m *MockBookingRequestRepository) GetByRequesterID(requesterID uint) ([]models.BookingRequest, error) {
	args := m.Called(requesterID)
	return args.Get(0).([]models.BookingRequest), args.Error(1)
}

func (m *MockBookingRequestRepository) GetAllByItemID(itemID uint) ([]models.BookingRequest, error) {
	args := m.Called(itemID)
	return args.Get(0).([]models.BookingRequest), args.Error(1)
}

func (m *MockBookingRequestRepository) UpdateStatus(id uint, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockBookingRequestRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockBookingRequestRepository) UpdateChatNotificationStatus(bookingID uint, notified bool, attempts int) error {
	args := m.Called(bookingID, notified, attempts)
	return args.Error(0)
}

func (m *MockBookingRequestRepository) IncrementNotificationAttempts(bookingID uint) error {
	args := m.Called(bookingID)
	return args.Error(0)
}

func (m *MockBookingRequestRepository) UpdateBuyerRating(id uint, rating int, review string) error {
	args := m.Called(id, rating, review)
	return args.Error(0)
}

func (m *MockBookingRequestRepository) UpdateSellerRating(id uint, rating int, review string) error {
	args := m.Called(id, rating, review)
	return args.Error(0)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) UpsertFromJWT(userID uint, username, email, name string) (*models.User, error) {
	args := m.Called(userID, username, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateRating(userID uint, newRating float64) error {
	args := m.Called(userID, newRating)
	return args.Error(0)
}

func setupService() (*StoreService, *MockStoreItemRepository, *MockBidRepository, *MockBookingRequestRepository) {
	itemRepo := new(MockStoreItemRepository)
	bidRepo := new(MockBidRepository)
	bookingRepo := new(MockBookingRequestRepository)
	userRepo := new(MockUserRepository)
	service := NewStoreService(nil, itemRepo, bidRepo, bookingRepo, userRepo)
	return service, itemRepo, bidRepo, bookingRepo
}

func TestCreateItem(t *testing.T) {
	t.Run("successful fixed price item creation", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		req := models.CreateStoreItemRequest{
			Title:       "  Test Item  ",
			Description: "Test Description",
			PriceType:   "fixed",
			FixedPrice:  100.0,
			Category:    "electronics",
			Condition:   "new",
			Images:      []string{"image1.jpg", "image2.jpg"},
		}

		itemRepo.On("Create", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.CreateItem(1, req)

		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "Test Item", item.Title)
		assert.Equal(t, req.Description, item.Description)
		assert.Equal(t, req.PriceType, item.PriceType)
		assert.Equal(t, req.FixedPrice, item.FixedPrice)
		assert.Equal(t, uint(1), item.SellerID)
		assert.Equal(t, "active", item.Status)
		assert.Len(t, item.Images, 2)
		assert.WithinDuration(t, time.Now().Add(ListingLifetime), *item.Deadline, time.Minute)
		assert.Nil(t, item.BidDeadline)
		itemRepo.AssertExpectations(t)
	})

	t.Run("auction closes with the note", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		req := models.CreateStoreItemRequest{
			Title:           "Auction Item",
			Description:     "Test Auction",
			PriceType:       "bidding",
			StartingBid:     50.0,
			MinBidIncrement: 5.0,
			Category:        "electronics",
			Condition:       "good",
		}

		itemRepo.On("Create", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.CreateItem(1, req)

		assert.NoError(t, err)
		assert.Equal(t, req.StartingBid, item.StartingBid)
		assert.Equal(t, req.MinBidIncrement, item.MinBidIncrement)
		assert.NotNil(t, item.Deadline)
		assert.Equal(t, item.Deadline, item.BidDeadline)
		itemRepo.AssertExpectations(t)
	})

	t.Run("note rules", func(t *testing.T) {
		cases := map[string]struct {
			title, description string
			want               error
		}{
			"missing headline":  {"  ", "A lamp", ErrHeadlineRequired},
			"missing note":      {"Lamp", " ", ErrBodyRequired},
			"headline too long": {"one two three four five six", "A lamp", ErrHeadlineTooLong},
			"note too long":     {"Lamp", strings.Repeat("word ", 21), ErrBodyTooLong},
			"phone number":      {"Lamp", "Call 0772 123 456", ErrContactDetails},
			"link in headline":  {"Lamp at shop.ug", "Brass desk lamp", ErrContactDetails},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				service, _, _, _ := setupService()
				item, err := service.CreateItem(1, models.CreateStoreItemRequest{
					Title: tc.title, Description: tc.description,
				})
				assert.ErrorIs(t, err, tc.want)
				assert.Nil(t, item)
			})
		}
	})

	t.Run("just a note: price agreed in chat", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("Create", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.CreateItem(1, models.CreateStoreItemRequest{
			Title: "Kids bike", Description: "Red, fits ages 5-8, collect in Ntinda",
		})

		assert.NoError(t, err)
		assert.Equal(t, "fixed", item.PriceType)
		assert.Zero(t, item.FixedPrice)
		assert.Empty(t, item.Category)
		assert.Empty(t, item.Condition)
	})

	t.Run("negative price", func(t *testing.T) {
		service, _, _, _ := setupService()
		req := models.CreateStoreItemRequest{
			Title:       "Test Item",
			Description: "Test Description",
			PriceType:   "fixed",
			FixedPrice:  -1,
		}

		item, err := service.CreateItem(1, req)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "price can't be negative")
	})

	t.Run("invalid starting bid", func(t *testing.T) {
		service, _, _, _ := setupService()
		req := models.CreateStoreItemRequest{
			Title:       "Test Item",
			Description: "Test Description",
			PriceType:   "bidding",
			StartingBid: 0,
		}

		item, err := service.CreateItem(1, req)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "starting bid must be greater than 0")
	})

	t.Run("default min bid increment", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		req := models.CreateStoreItemRequest{
			Title:       "Test Item",
			Description: "Test Description",
			PriceType:   "bidding",
			StartingBid: 50.0,
		}

		itemRepo.On("Create", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.CreateItem(1, req)

		assert.NoError(t, err)
		assert.Equal(t, 1.0, item.MinBidIncrement)
		itemRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		req := models.CreateStoreItemRequest{
			Title:       "Test Item",
			Description: "Test Description",
			PriceType:   "fixed",
			FixedPrice:  100.0,
		}

		itemRepo.On("Create", mock.AnythingOfType("*models.StoreItem")).Return(errors.New("database error"))

		item, err := service.CreateItem(1, req)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "database error")
		itemRepo.AssertExpectations(t)
	})
}

func TestRepostItem(t *testing.T) {
	t.Run("expired listing goes back up for 24 hours", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		old := time.Now().Add(-2 * time.Hour)
		item := &models.StoreItem{ID: 1, SellerID: 1, PriceType: "bidding", Status: "expired", Deadline: &old, BidDeadline: &old}
		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		itemRepo.On("Update", item).Return(nil)

		got, err := service.RepostItem(1, 1)

		assert.NoError(t, err)
		assert.Equal(t, "active", got.Status)
		assert.WithinDuration(t, time.Now().Add(ListingLifetime), *got.Deadline, time.Minute)
		assert.Equal(t, got.Deadline, got.BidDeadline)
		itemRepo.AssertExpectations(t)
	})

	t.Run("only the seller", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 2, Status: "expired"}, nil)

		_, err := service.RepostItem(1, 1)

		assert.ErrorContains(t, err, "unauthorized")
	})

	t.Run("only an expired listing", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 1, Status: "sold"}, nil)

		_, err := service.RepostItem(1, 1)

		assert.ErrorContains(t, err, "only an expired listing")
	})

	t.Run("missing item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(9)).Return(nil, gorm.ErrRecordNotFound)

		_, err := service.RepostItem(9, 1)

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestExpireStaleItems(t *testing.T) {
	service, itemRepo, _, _ := setupService()
	itemRepo.On("ExpireUnanswered", mock.AnythingOfType("time.Time")).Return(int64(3), nil)

	n, err := service.ExpireStaleItems()

	assert.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestGetItem(t *testing.T) {
	t.Run("successful get item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		expectedItem := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 1,
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(expectedItem, nil)

		item, err := service.GetItem(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedItem, item)
		itemRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		item, err := service.GetItem(999)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})
}

func TestGetItems(t *testing.T) {
	t.Run("successful get items", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		filter := models.StoreItemFilter{
			Search:   "test",
			Category: "electronics",
			Page:     1,
			PerPage:  10,
		}

		expectedItems := []models.StoreItem{
			{ID: 1, Title: "Test Item 1", SellerID: 1, Status: "active"},
			{ID: 2, Title: "Test Item 2", SellerID: 2, Status: "active"},
		}

		itemRepo.On("GetAll", filter).Return(expectedItems, int64(2), nil)

		items, count, err := service.GetItems(filter)

		assert.NoError(t, err)
		assert.Equal(t, expectedItems, items)
		assert.Equal(t, int64(2), count)
		itemRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		filter := models.StoreItemFilter{Page: 1, PerPage: 10}

		itemRepo.On("GetAll", filter).Return([]models.StoreItem{}, int64(0), errors.New("database error"))

		items, count, err := service.GetItems(filter)

		assert.Error(t, err)
		assert.Empty(t, items)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "database error")
		itemRepo.AssertExpectations(t)
	})
}

func TestUpdateItem(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:       1,
			Title:    "Original Title",
			SellerID: 1,
			Status:   "active",
		}

		req := models.UpdateStoreItemRequest{
			Title:       "Updated Title",
			Description: "Updated Description",
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)
		itemRepo.On("Update", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.UpdateItem(1, 1, req)

		assert.NoError(t, err)
		assert.Equal(t, req.Title, item.Title)
		assert.Equal(t, req.Description, item.Description)
		itemRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		req := models.UpdateStoreItemRequest{
			Title: "Updated Title",
		}

		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		item, err := service.UpdateItem(999, 1, req)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("unauthorized update", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:       1,
			Title:    "Original Title",
			SellerID: 2, // Different seller
			Status:   "active",
		}

		req := models.UpdateStoreItemRequest{
			Title: "Updated Title",
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)

		item, err := service.UpdateItem(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "unauthorized")
		itemRepo.AssertExpectations(t)
	})

	t.Run("cannot update inactive item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:       1,
			Title:    "Original Title",
			SellerID: 1,
			Status:   "sold",
		}

		req := models.UpdateStoreItemRequest{
			Title: "Updated Title",
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)

		item, err := service.UpdateItem(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "cannot update item that is not active")
		itemRepo.AssertExpectations(t)
	})

	t.Run("update with images", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:          1,
			Title:       "Original Title",
			Description: "Original note",
			SellerID:    1,
			Status:      "active",
		}

		req := models.UpdateStoreItemRequest{
			Title:  "Updated Title",
			Images: []string{"new_image1.jpg", "new_image2.jpg"},
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)
		itemRepo.On("Update", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.UpdateItem(1, 1, req)

		assert.NoError(t, err)
		assert.Equal(t, req.Title, item.Title)
		assert.Len(t, item.Images, 2)
		itemRepo.AssertExpectations(t)
	})
}

func TestDeleteItem(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 1,
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)
		itemRepo.On("Delete", uint(1)).Return(nil)

		err := service.DeleteItem(1, 1)

		assert.NoError(t, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		err := service.DeleteItem(999, 1)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("unauthorized delete", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 2, // Different seller
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)

		err := service.DeleteItem(1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		itemRepo.AssertExpectations(t)
	})

	t.Run("cannot delete inactive item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		existingItem := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 1,
			Status:   "sold",
		}

		itemRepo.On("GetByID", uint(1)).Return(existingItem, nil)

		err := service.DeleteItem(1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only a live or expired listing can be removed")
		itemRepo.AssertExpectations(t)
	})

	t.Run("expired listing can be removed", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 1, Status: "expired"}, nil)
		itemRepo.On("Delete", uint(1)).Return(nil)

		assert.NoError(t, service.DeleteItem(1, 1))
		itemRepo.AssertExpectations(t)
	})
}

func TestPlaceBid(t *testing.T) {
	t.Run("successful first bid", func(t *testing.T) {
		service, itemRepo, bidRepo, _ := setupService()
		item := &models.StoreItem{
			ID:              1,
			Title:           "Auction Item",
			SellerID:        2,
			PriceType:       "bidding",
			StartingBid:     100.0,
			MinBidIncrement: 5.0,
			CurrentBid:      0,
			Status:          "active",
		}

		req := models.CreateBidRequest{
			Amount:  110.0,
			Message: "My bid",
		}

		expectedBid := &models.Bid{
			ID:       1,
			ItemID:   1,
			BidderID: 1,
			Amount:   110.0,
			Message:  "My bid",
			Status:   "active",
		}

		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)
		bidRepo.On("Create", mock.AnythingOfType("*models.Bid")).Return(nil)
		itemRepo.On("Update", mock.AnythingOfType("*models.StoreItem")).Return(nil)
		bidRepo.On("MarkOutbidBids", uint(1), uint(0)).Return(nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.NoError(t, err)
		assert.NotNil(t, bid)
		assert.Equal(t, expectedBid.Amount, bid.Amount)
		assert.Equal(t, expectedBid.Message, bid.Message)
		itemRepo.AssertExpectations(t)
		bidRepo.AssertExpectations(t)
	})

	t.Run("successful higher bid", func(t *testing.T) {
		service, itemRepo, bidRepo, _ := setupService()
		item := &models.StoreItem{
			ID:              1,
			Title:           "Auction Item",
			SellerID:        2,
			PriceType:       "bidding",
			StartingBid:     100.0,
			MinBidIncrement: 5.0,
			CurrentBid:      110.0,
			Status:          "active",
		}

		req := models.CreateBidRequest{
			Amount: 120.0,
		}

		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)
		bidRepo.On("Create", mock.AnythingOfType("*models.Bid")).Return(nil)
		itemRepo.On("Update", mock.AnythingOfType("*models.StoreItem")).Return(nil)
		bidRepo.On("MarkOutbidBids", uint(1), uint(0)).Return(nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.NoError(t, err)
		assert.NotNil(t, bid)
		assert.Equal(t, 120.0, bid.Amount)
		itemRepo.AssertExpectations(t)
		bidRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		req := models.CreateBidRequest{Amount: 110.0}

		itemRepo.On("GetByIDForUpdate", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		bid, err := service.PlaceBid(999, 1, req)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("not bidding item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:        1,
			Title:     "Fixed Price Item",
			SellerID:  2,
			PriceType: "fixed",
			Status:    "active",
		}

		req := models.CreateBidRequest{Amount: 110.0}

		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Contains(t, err.Error(), "not available for bidding")
		itemRepo.AssertExpectations(t)
	})

	t.Run("inactive item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:        1,
			Title:     "Auction Item",
			SellerID:  2,
			PriceType: "bidding",
			Status:    "sold",
		}

		req := models.CreateBidRequest{Amount: 110.0}

		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Contains(t, err.Error(), "item is not active")
		itemRepo.AssertExpectations(t)
	})

	t.Run("cannot bid on own item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:        1,
			Title:     "Auction Item",
			SellerID:  1, // Same as bidder
			PriceType: "bidding",
			Status:    "active",
		}

		req := models.CreateBidRequest{Amount: 110.0}

		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Contains(t, err.Error(), "cannot bid on your own item")
		itemRepo.AssertExpectations(t)
	})

	t.Run("bid amount too low", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:              1,
			Title:           "Auction Item",
			SellerID:        2,
			PriceType:       "bidding",
			StartingBid:     100.0,
			MinBidIncrement: 5.0,
			CurrentBid:      110.0,
			Status:          "active",
		}

		req := models.CreateBidRequest{Amount: 110.0} // Same as current bid

		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Contains(t, err.Error(), "bid amount must be at least")
		itemRepo.AssertExpectations(t)
	})

	t.Run("bidding ended", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		pastDeadline := time.Now().Add(-1 * time.Hour)
		item := &models.StoreItem{
			ID:              1,
			Title:           "Auction Item",
			SellerID:        2,
			PriceType:       "bidding",
			StartingBid:     100.0,
			MinBidIncrement: 5.0,
			CurrentBid:      0,
			BidDeadline:     &pastDeadline,
			Status:          "active",
		}

		req := models.CreateBidRequest{Amount: 110.0}

		// The listing stays as it is: the seller may still accept a bid
		itemRepo.On("GetByIDForUpdate", uint(1)).Return(item, nil)

		bid, err := service.PlaceBid(1, 1, req)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Contains(t, err.Error(), "bidding has ended")
		itemRepo.AssertExpectations(t)
	})
}

func TestAcceptBid(t *testing.T) {
	t.Run("successful accept bid", func(t *testing.T) {
		service, itemRepo, bidRepo, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Auction Item",
			SellerID: 1,
			Status:   "active",
		}

		bid := &models.Bid{
			ID:       1,
			ItemID:   1,
			BidderID: 2,
			Amount:   110.0,
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bidRepo.On("GetByID", uint(1)).Return(bid, nil)
		itemRepo.On("MarkAsSold", uint(1), uint(2)).Return(nil)
		bidRepo.On("UpdateBidStatus", uint(1), "won").Return(nil)
		bidRepo.On("MarkOutbidBids", uint(1), uint(1)).Return(nil)

		err := service.AcceptBid(1, 1, 1)

		assert.NoError(t, err)
		itemRepo.AssertExpectations(t)
		bidRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		err := service.AcceptBid(999, 1, 1)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("unauthorized seller", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Auction Item",
			SellerID: 2, // Different seller
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		err := service.AcceptBid(1, 1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		itemRepo.AssertExpectations(t)
	})

	t.Run("bid not found", func(t *testing.T) {
		service, itemRepo, bidRepo, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Auction Item",
			SellerID: 1,
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bidRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		err := service.AcceptBid(1, 999, 1)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
		bidRepo.AssertExpectations(t)
	})

	t.Run("bid belongs to different item", func(t *testing.T) {
		service, itemRepo, bidRepo, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Auction Item",
			SellerID: 1,
			Status:   "active",
		}

		bid := &models.Bid{
			ID:       1,
			ItemID:   2, // Different item
			BidderID: 2,
			Amount:   110.0,
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bidRepo.On("GetByID", uint(1)).Return(bid, nil)

		err := service.AcceptBid(1, 1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bid does not belong to this item")
		itemRepo.AssertExpectations(t)
		bidRepo.AssertExpectations(t)
	})
}

func TestPurchaseItem(t *testing.T) {
	t.Run("successful purchase", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:         1,
			Title:      "Fixed Price Item",
			SellerID:   2,
			PriceType:  "fixed",
			FixedPrice: 100.0,
			Status:     "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		itemRepo.On("MarkAsSold", uint(1), uint(1)).Return(nil)

		err := service.PurchaseItem(1, 1)

		assert.NoError(t, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		err := service.PurchaseItem(999, 1)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("bidding item only", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:        1,
			Title:     "Auction Item",
			SellerID:  2,
			PriceType: "bidding",
			Status:    "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		err := service.PurchaseItem(1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available through bidding")
		itemRepo.AssertExpectations(t)
	})

	t.Run("inactive item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:        1,
			Title:     "Fixed Price Item",
			SellerID:  2,
			PriceType: "fixed",
			Status:    "sold",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		err := service.PurchaseItem(1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available for purchase")
		itemRepo.AssertExpectations(t)
	})

	t.Run("cannot purchase own item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:        1,
			Title:     "Fixed Price Item",
			SellerID:  1, // Same as buyer
			PriceType: "fixed",
			Status:    "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		err := service.PurchaseItem(1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot purchase your own item")
		itemRepo.AssertExpectations(t)
	})
}

func TestGetItemBids(t *testing.T) {
	t.Run("successful get item bids", func(t *testing.T) {
		service, _, bidRepo, _ := setupService()
		expectedBids := []models.Bid{
			{ID: 1, ItemID: 1, BidderID: 1, Amount: 150.0, Status: "active"},
			{ID: 2, ItemID: 1, BidderID: 2, Amount: 120.0, Status: "outbid"},
		}

		bidRepo.On("GetByItemID", uint(1)).Return(expectedBids, nil)

		bids, err := service.GetItemBids(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedBids, bids)
		bidRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, _, bidRepo, _ := setupService()
		bidRepo.On("GetByItemID", uint(1)).Return([]models.Bid{}, errors.New("database error"))

		bids, err := service.GetItemBids(1)

		assert.Error(t, err)
		assert.Empty(t, bids)
		assert.Contains(t, err.Error(), "database error")
		bidRepo.AssertExpectations(t)
	})
}

func TestGetUserListings(t *testing.T) {
	t.Run("successful get user listings", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		expectedItems := []models.StoreItem{
			{ID: 1, Title: "My Item 1", SellerID: 1, Status: "active"},
			{ID: 2, Title: "My Item 2", SellerID: 1, Status: "sold"},
		}

		itemRepo.On("GetBySellerID", uint(1)).Return(expectedItems, nil)

		items, err := service.GetUserListings(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedItems, items)
		itemRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetBySellerID", uint(1)).Return([]models.StoreItem{}, errors.New("database error"))

		items, err := service.GetUserListings(1)

		assert.Error(t, err)
		assert.Empty(t, items)
		assert.Contains(t, err.Error(), "database error")
		itemRepo.AssertExpectations(t)
	})
}

func TestGetUserPurchases(t *testing.T) {
	t.Run("successful get user purchases", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		expectedItems := []models.StoreItem{
			{ID: 1, Title: "Purchased Item 1", SellerID: 2, Status: "sold"},
			{ID: 2, Title: "Purchased Item 2", SellerID: 3, Status: "sold"},
		}

		itemRepo.On("GetByBuyerID", uint(1)).Return(expectedItems, nil)

		items, err := service.GetUserPurchases(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedItems, items)
		itemRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByBuyerID", uint(1)).Return([]models.StoreItem{}, errors.New("database error"))

		items, err := service.GetUserPurchases(1)

		assert.Error(t, err)
		assert.Empty(t, items)
		assert.Contains(t, err.Error(), "database error")
		itemRepo.AssertExpectations(t)
	})
}

func TestGetUserBids(t *testing.T) {
	t.Run("successful get user bids", func(t *testing.T) {
		service, _, bidRepo, _ := setupService()
		expectedBids := []models.Bid{
			{ID: 1, ItemID: 1, BidderID: 1, Amount: 150.0, Status: "active"},
			{ID: 2, ItemID: 2, BidderID: 1, Amount: 120.0, Status: "outbid"},
		}

		bidRepo.On("GetByBidderID", uint(1)).Return(expectedBids, nil)

		bids, err := service.GetUserBids(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedBids, bids)
		bidRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, _, bidRepo, _ := setupService()
		bidRepo.On("GetByBidderID", uint(1)).Return([]models.Bid{}, errors.New("database error"))

		bids, err := service.GetUserBids(1)

		assert.Error(t, err)
		assert.Empty(t, bids)
		assert.Contains(t, err.Error(), "database error")
		bidRepo.AssertExpectations(t)
	})
}

func TestCreateBookingRequest_ListingRules(t *testing.T) {
	past := time.Now().Add(-time.Minute)

	t.Run("refused once the note is past its deadline with no reactions", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 2, Status: "active", Deadline: &past}, nil)
		bookingRepo.On("GetAllByItemID", uint(1)).Return([]models.BookingRequest{}, nil)

		request, err := service.CreateBookingRequest(1, 1, "")

		assert.ErrorIs(t, err, ErrListingExpired)
		assert.Nil(t, request)
	})

	t.Run("still open past the deadline once someone reacted", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 2, Status: "active", Deadline: &past}, nil)
		bookingRepo.On("GetAllByItemID", uint(1)).Return([]models.BookingRequest{{ID: 5, ItemID: 1, RequesterID: 3}}, nil)
		bookingRepo.On("GetByItemAndRequester", uint(1), uint(1)).Return(&models.BookingRequest{ID: 6}, nil)

		_, err := service.CreateBookingRequest(1, 1, "")

		// Past the deadline check: refused only as a duplicate
		assert.ErrorContains(t, err, "already have a booking request")
	})

	t.Run("contact details refused before approval", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 2, Status: "active"}, nil)

		_, err := service.CreateBookingRequest(1, 1, "email me at a@b.com")

		assert.ErrorIs(t, err, ErrContactDetails)
	})
}

func TestPlaceBid_ContactDetails(t *testing.T) {
	service, itemRepo, _, _ := setupService()
	itemRepo.On("GetByIDForUpdate", uint(1)).Return(&models.StoreItem{
		ID: 1, SellerID: 2, PriceType: "bidding", Status: "active", StartingBid: 100,
	}, nil)

	bid, err := service.PlaceBid(1, 1, models.CreateBidRequest{Amount: 150, Message: "call +256 772 123456"})

	assert.ErrorIs(t, err, ErrContactDetails)
	assert.Nil(t, bid)
}

func TestUpdateItem_NoteRules(t *testing.T) {
	service, itemRepo, _, _ := setupService()
	itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{
		ID: 1, SellerID: 1, Status: "active", Title: "Lamp", Description: "Brass desk lamp",
	}, nil)

	_, err := service.UpdateItem(1, 1, models.UpdateStoreItemRequest{Description: strings.Repeat("word ", 21)})

	assert.ErrorIs(t, err, ErrBodyTooLong)
}

func TestCreateBookingRequest(t *testing.T) {
	t.Run("successful booking request", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 2,
			Status:   "active",
		}

		expectedRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 1,
			Status:      "pending",
			Message:     "I'd like to book this item",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetByItemAndRequester", uint(1), uint(1)).Return(nil, gorm.ErrRecordNotFound)
		bookingRepo.On("Create", mock.AnythingOfType("*models.BookingRequest")).Return(nil)
		bookingRepo.On("GetByID", uint(0)).Return(expectedRequest, nil)
		bookingRepo.On("IncrementNotificationAttempts", mock.AnythingOfType("uint")).Return(nil).Maybe()

		request, err := service.CreateBookingRequest(1, 1, "I'd like to book this item")

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, expectedRequest.Message, request.Message)
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		request, err := service.CreateBookingRequest(999, 1, "Message")

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("item not available", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 2,
			Status:   "sold",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		request, err := service.CreateBookingRequest(1, 1, "Message")

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Contains(t, err.Error(), "not available for booking")
		itemRepo.AssertExpectations(t)
	})

	t.Run("cannot book own item", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 1, // Same as requester
			Status:   "active",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		request, err := service.CreateBookingRequest(1, 1, "Message")

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Contains(t, err.Error(), "cannot book your own item")
		itemRepo.AssertExpectations(t)
	})

	t.Run("duplicate booking request", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 2,
			Status:   "active",
		}

		existingRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 1,
			Status:      "pending",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetByItemAndRequester", uint(1), uint(1)).Return(existingRequest, nil)

		request, err := service.CreateBookingRequest(1, 1, "Message")

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Contains(t, err.Error(), "already have a booking request")
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})
}

func TestApproveBookingRequest(t *testing.T) {
	t.Run("successful approval", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		request := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 1,
			},
		}
		approvedRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "approved",
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 1,
			},
		}

		bookingRepo.On("GetByID", uint(1)).Return(request, nil).Once()
		bookingRepo.On("GetAllByItemID", uint(1)).Return([]models.BookingRequest{*request}, nil)
		bookingRepo.On("UpdateStatus", uint(1), "approved").Return(nil)
		bookingRepo.On("GetByID", uint(1)).Return(approvedRequest, nil).Once()

		result, err := service.ApproveBookingRequest(1, 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "approved", result.Status)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("request not found", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		bookingRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		result, err := service.ApproveBookingRequest(999, 1)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		request := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 2, // Different owner
			},
		}

		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		result, err := service.ApproveBookingRequest(1, 1)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unauthorized")
		bookingRepo.AssertExpectations(t)
	})

	t.Run("request not pending", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		request := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "approved", // Already approved
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 1,
			},
		}

		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		result, err := service.ApproveBookingRequest(1, 1)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not pending")
		bookingRepo.AssertExpectations(t)
	})
}

func TestRejectBookingRequest(t *testing.T) {
	t.Run("successful rejection", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		request := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 1,
			},
		}
		rejectedRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "rejected",
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 1,
			},
		}

		bookingRepo.On("GetByID", uint(1)).Return(request, nil).Once()
		bookingRepo.On("UpdateStatus", uint(1), "rejected").Return(nil)
		bookingRepo.On("GetByID", uint(1)).Return(rejectedRequest, nil).Once()

		result, err := service.RejectBookingRequest(1, 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "rejected", result.Status)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		request := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
			Item: &models.StoreItem{
				ID:       1,
				SellerID: 2, // Different owner
			},
		}

		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		result, err := service.RejectBookingRequest(1, 1)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unauthorized")
		bookingRepo.AssertExpectations(t)
	})
}

func TestGetBookingRequestByItem(t *testing.T) {
	t.Run("successful get as owner", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			SellerID: 1,
		}

		expectedRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetByItemID", uint(1)).Return(expectedRequest, nil)

		request, err := service.GetBookingRequestByItem(1, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRequest, request)
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("successful get as requester", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			SellerID: 2,
		}

		expectedRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 1,
			Status:      "pending",
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetByItemAndRequester", uint(1), uint(1)).Return(expectedRequest, nil)

		request, err := service.GetBookingRequestByItem(1, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRequest, request)
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		request, err := service.GetBookingRequestByItem(999, 1)

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})
}

func TestGetUserBookingRequests(t *testing.T) {
	t.Run("successful get user booking requests", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		expectedRequests := []models.BookingRequest{
			{ID: 1, ItemID: 1, RequesterID: 1, Status: "pending"},
			{ID: 2, ItemID: 2, RequesterID: 1, Status: "approved"},
		}

		bookingRepo.On("GetByRequesterID", uint(1)).Return(expectedRequests, nil)

		requests, err := service.GetUserBookingRequests(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRequests, requests)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()
		bookingRepo.On("GetByRequesterID", uint(1)).Return([]models.BookingRequest{}, errors.New("database error"))

		requests, err := service.GetUserBookingRequests(1)

		assert.Error(t, err)
		assert.Empty(t, requests)
		assert.Contains(t, err.Error(), "database error")
		bookingRepo.AssertExpectations(t)
	})
}

func TestGetAllBookingRequestsByItem(t *testing.T) {
	t.Run("successful get all requests as owner", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			SellerID: 1,
		}

		expectedRequests := []models.BookingRequest{
			{ID: 1, ItemID: 1, RequesterID: 2, Status: "pending"},
			{ID: 2, ItemID: 1, RequesterID: 3, Status: "approved"},
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetAllByItemID", uint(1)).Return(expectedRequests, nil)

		requests, err := service.GetAllBookingRequestsByItem(1, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRequests, requests)
		assert.Len(t, requests, 2)
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("unauthorized access - not item owner", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		item := &models.StoreItem{
			ID:       1,
			SellerID: 2, // Different owner
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)

		requests, err := service.GetAllBookingRequestsByItem(1, 1)

		assert.Error(t, err)
		assert.Nil(t, requests)
		assert.Contains(t, err.Error(), "unauthorized: you are not the owner of this item")
		itemRepo.AssertExpectations(t)
	})

	t.Run("item not found", func(t *testing.T) {
		service, itemRepo, _, _ := setupService()
		itemRepo.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		requests, err := service.GetAllBookingRequestsByItem(999, 1)

		assert.Error(t, err)
		assert.Nil(t, requests)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("empty booking requests list", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			SellerID: 1,
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetAllByItemID", uint(1)).Return([]models.BookingRequest{}, nil)

		requests, err := service.GetAllBookingRequestsByItem(1, 1)

		assert.NoError(t, err)
		assert.Empty(t, requests)
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, itemRepo, _, bookingRepo := setupService()
		item := &models.StoreItem{
			ID:       1,
			SellerID: 1,
		}

		itemRepo.On("GetByID", uint(1)).Return(item, nil)
		bookingRepo.On("GetAllByItemID", uint(1)).Return([]models.BookingRequest{}, errors.New("database error"))

		requests, err := service.GetAllBookingRequestsByItem(1, 1)

		assert.Error(t, err)
		assert.Empty(t, requests)
		assert.Contains(t, err.Error(), "database error")
		itemRepo.AssertExpectations(t)
		bookingRepo.AssertExpectations(t)
	})
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		price    float64
		expected string
	}{
		{100.0, "100.00"},
		{99.99, "99.99"},
		{0.5, "0.50"},
		{1234.567, "1234.57"},
	}

	for _, tt := range tests {
		result := formatPrice(tt.price)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetUserRatings(t *testing.T) {
	service, _, _, bookingRepo := setupService()
	rating := func(v int) *int { return &v }
	soldBy1 := &models.StoreItem{ID: 1, Title: "Bike", SellerID: 1}
	soldBy2 := &models.StoreItem{ID: 2, Title: "Desk", SellerID: 2}

	bookingRepo.On("GetRatingsReceived", uint(1)).Return([]models.BookingRequest{
		{ID: 10, ItemID: 1, Item: soldBy1, RequesterID: 2, BuyerRating: rating(5), BuyerReview: "great seller"},
		// User 1 bought from user 2: only the seller's rating of user 1 counts,
		// not user 1's own rating of the seller.
		{ID: 11, ItemID: 2, Item: soldBy2, RequesterID: 1, SellerRating: rating(4), SellerReview: "good buyer", BuyerRating: rating(1)},
	}, nil)

	ratings, err := service.GetUserRatings(1)

	assert.NoError(t, err)
	assert.Equal(t, []models.ReceivedRating{
		{BookingID: 10, ItemID: 1, ItemTitle: "Bike", RaterID: 2, RatedAs: "seller", Rating: 5, Review: "great seller"},
		{BookingID: 11, ItemID: 2, ItemTitle: "Desk", RaterID: 2, RatedAs: "buyer", Rating: 4, Review: "good buyer"},
	}, ratings)
}

