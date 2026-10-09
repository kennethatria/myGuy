package services

import (
	"context"
	"errors"
	"store-service/internal/models"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockItemRequestRepository struct {
	mock.Mock
}

func (m *MockItemRequestRepository) Create(request *models.ItemRequest) error {
	return m.Called(request).Error(0)
}

func (m *MockItemRequestRepository) GetByID(id uint) (*models.ItemRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ItemRequest), args.Error(1)
}

func (m *MockItemRequestRepository) GetAll(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.ItemRequest), args.Get(1).(int64), args.Error(2)
}

func (m *MockItemRequestRepository) ListIDs(filter models.ItemRequestFilter) ([]uint, error) {
	args := m.Called(filter)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockItemRequestRepository) GetByIDs(ids []uint) ([]models.ItemRequest, error) {
	args := m.Called(ids)
	return args.Get(0).([]models.ItemRequest), args.Error(1)
}

func (m *MockItemRequestRepository) GetByRequesterID(requesterID uint) ([]models.ItemRequest, error) {
	args := m.Called(requesterID)
	return args.Get(0).([]models.ItemRequest), args.Error(1)
}

func (m *MockItemRequestRepository) Update(request *models.ItemRequest) error {
	return m.Called(request).Error(0)
}

func (m *MockItemRequestRepository) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockItemRequestRepository) MarkFulfilled(id uint, itemID uint) (bool, error) {
	args := m.Called(id, itemID)
	return args.Bool(0), args.Error(1)
}

func (m *MockItemRequestRepository) Reopen(id uint, deadline time.Time) (bool, error) {
	args := m.Called(id, deadline)
	return args.Bool(0), args.Error(1)
}

func (m *MockItemRequestRepository) ExpireUnanswered(now time.Time) (int64, error) {
	args := m.Called(now)
	return args.Get(0).(int64), args.Error(1)
}

// fakeChat records store messages instead of posting them
type fakeChat struct {
	mu     sync.Mutex
	sent   []string
	closed   []uint // bookings closed in chat
	answered []uint // requests told a listing answers them
	// pairs recorded as matched, and an error to answer with
	unlocked  [][3]uint
	unlockErr error
}

func (f *fakeChat) Unlock(itemID, sellerID, buyerID uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unlocked = append(f.unlocked, [3]uint{itemID, sellerID, buyerID})
	return f.unlockErr
}

func (f *fakeChat) BookingClosed(bookingID, sellerID uint, note string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, bookingID)
	f.sent = append(f.sent, note)
}

func (f *fakeChat) RequestAnswered(itemID, sellerID, requesterID, requestID uint, content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, content)
	f.answered = append(f.answered, requestID)
}

func setupRequestService() (*RequestService, *MockItemRequestRepository, *MockStoreItemRepository) {
	requests := new(MockItemRequestRepository)
	items := new(MockStoreItemRepository)
	return NewRequestService(requests, items), requests, items
}

func TestCreateRequest(t *testing.T) {
	t.Run("a short note goes up for 24 hours", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		requests.On("Create", mock.AnythingOfType("*models.ItemRequest")).Return(nil)

		got, err := service.CreateRequest(1, models.CreateItemRequestRequest{Title: " Printer wanted ", Description: "Any laser printer"})

		assert.NoError(t, err)
		assert.Equal(t, "Printer wanted", got.Title)
		assert.Equal(t, "active", got.Status)
		assert.Equal(t, uint(1), got.RequesterID)
		assert.WithinDuration(t, time.Now().Add(ListingLifetime), *got.Deadline, time.Minute)
	})

	t.Run("note rules apply", func(t *testing.T) {
		service, _, _ := setupRequestService()
		_, err := service.CreateRequest(1, models.CreateItemRequestRequest{Title: "Printer", Description: strings.Repeat("word ", 21)})
		assert.ErrorIs(t, err, ErrBodyTooLong)
		_, err = service.CreateRequest(1, models.CreateItemRequestRequest{Title: "Printer", Description: "call 0772 123 456"})
		assert.ErrorIs(t, err, ErrContactDetails)
	})

	t.Run("repository error", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		requests.On("Create", mock.Anything).Return(errors.New("db down"))
		_, err := service.CreateRequest(1, models.CreateItemRequestRequest{Title: "Printer", Description: "Any"})
		assert.ErrorContains(t, err, "db down")
	})
}

func TestRequestReads(t *testing.T) {
	service, requests, items := setupRequestService()
	filter := models.ItemRequestFilter{Search: "printer"}
	requests.On("GetByID", uint(1)).Return(&models.ItemRequest{ID: 1}, nil)
	requests.On("GetAll", filter).Return([]models.ItemRequest{{ID: 1}}, int64(1), nil)
	requests.On("GetByRequesterID", uint(4)).Return([]models.ItemRequest{{ID: 2}}, nil)
	items.On("GetAll", models.StoreItemFilter{RequestID: 1, Status: "active", PerPage: 100}).Return([]models.StoreItem{{ID: 9}}, int64(1), nil)

	got, err := service.GetRequest(1)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), got.ID)
	list, total, err := service.GetRequests(filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
	mine, err := service.GetUserRequests(4)
	assert.NoError(t, err)
	assert.Len(t, mine, 1)
	offers, err := service.GetRequestListings(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, offers, 1)
}

func TestRepostRequest(t *testing.T) {
	t.Run("expired request goes back up", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		old := time.Now().Add(-time.Hour)
		r := &models.ItemRequest{ID: 1, RequesterID: 1, Status: "expired", Deadline: &old}
		requests.On("GetByID", uint(1)).Return(r, nil)
		requests.On("Update", r).Return(nil)

		got, err := service.RepostRequest(1, 1)

		assert.NoError(t, err)
		assert.Equal(t, "active", got.Status)
		assert.True(t, got.Deadline.After(time.Now()))
	})

	t.Run("refused for others and live requests", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		requests.On("GetByID", uint(1)).Return(&models.ItemRequest{ID: 1, RequesterID: 2, Status: "expired"}, nil)
		requests.On("GetByID", uint(2)).Return(&models.ItemRequest{ID: 2, RequesterID: 1, Status: "active"}, nil)
		requests.On("GetByID", uint(3)).Return(nil, gorm.ErrRecordNotFound)

		_, err := service.RepostRequest(1, 1)
		assert.ErrorContains(t, err, "unauthorized")
		_, err = service.RepostRequest(2, 1)
		assert.ErrorContains(t, err, "only an expired request")
		_, err = service.RepostRequest(3, 1)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestUpdateRequest(t *testing.T) {
	t.Run("live request takes the new note", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		r := &models.ItemRequest{ID: 1, RequesterID: 1, Status: "active", Title: "Printer", Description: "Any"}
		requests.On("GetByID", uint(1)).Return(r, nil)
		requests.On("Update", r).Return(nil)

		got, err := service.UpdateRequest(1, 1, models.UpdateItemRequestRequest{Title: " Laser printer ", Description: "For a small office"})

		assert.NoError(t, err)
		assert.Equal(t, "Laser printer", got.Title)
		assert.Equal(t, "For a small office", got.Description)
	})

	t.Run("same rules as posting", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		requests.On("GetByID", uint(1)).Return(&models.ItemRequest{ID: 1, RequesterID: 1, Status: "active"}, nil)

		_, err := service.UpdateRequest(1, 1, models.UpdateItemRequestRequest{Title: "Printer", Description: "call 0772 123456"})
		assert.Error(t, err)
		_, err = service.UpdateRequest(1, 1, models.UpdateItemRequestRequest{Title: "One two three four five six"})
		assert.Error(t, err)
		requests.AssertNotCalled(t, "Update", mock.Anything)
	})

	t.Run("refused for others and closed requests", func(t *testing.T) {
		service, requests, _ := setupRequestService()
		requests.On("GetByID", uint(1)).Return(&models.ItemRequest{ID: 1, RequesterID: 2, Status: "active"}, nil)
		requests.On("GetByID", uint(2)).Return(&models.ItemRequest{ID: 2, RequesterID: 1, Status: "fulfilled"}, nil)
		requests.On("GetByID", uint(3)).Return(nil, gorm.ErrRecordNotFound)
		edit := models.UpdateItemRequestRequest{Title: "Printer"}

		_, err := service.UpdateRequest(1, 1, edit)
		assert.ErrorContains(t, err, "unauthorized")
		_, err = service.UpdateRequest(2, 1, edit)
		assert.ErrorContains(t, err, "only a live request")
		_, err = service.UpdateRequest(3, 1, edit)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestDeleteRequest(t *testing.T) {
	service, requests, _ := setupRequestService()
	requests.On("GetByID", uint(1)).Return(&models.ItemRequest{ID: 1, RequesterID: 1, Status: "expired"}, nil)
	requests.On("GetByID", uint(2)).Return(&models.ItemRequest{ID: 2, RequesterID: 2, Status: "active"}, nil)
	requests.On("GetByID", uint(3)).Return(&models.ItemRequest{ID: 3, RequesterID: 1, Status: "fulfilled"}, nil)
	requests.On("GetByID", uint(4)).Return(nil, gorm.ErrRecordNotFound)
	requests.On("Delete", uint(1)).Return(nil)

	assert.NoError(t, service.DeleteRequest(1, 1))
	assert.ErrorContains(t, service.DeleteRequest(2, 1), "unauthorized")
	assert.ErrorContains(t, service.DeleteRequest(3, 1), "only a live or expired")
	assert.Error(t, service.DeleteRequest(4, 1))
}

func TestExpireStaleRequests(t *testing.T) {
	service, requests, _ := setupRequestService()
	requests.On("ExpireUnanswered", mock.AnythingOfType("time.Time")).Return(int64(2), nil)
	n, err := service.ExpireStaleRequests()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestCreateItem_ForRequest(t *testing.T) {
	newReq := func(requestID uint) models.CreateStoreItemRequest {
		return models.CreateStoreItemRequest{Title: "HP laser printer", Description: "Works well", RequestID: &requestID}
	}

	t.Run("links the listing and tells the requester", func(t *testing.T) {
		service, itemRepo, _ := setupService()
		requests, chat := new(MockItemRequestRepository), &fakeChat{}
		service.WithRequests(requests, chat)
		requests.On("GetByID", uint(5)).Return(&models.ItemRequest{ID: 5, RequesterID: 2, Status: "active", Title: "Printer wanted"}, nil)
		itemRepo.On("Create", mock.AnythingOfType("*models.StoreItem")).Return(nil)

		item, err := service.CreateItem(1, newReq(5))

		assert.NoError(t, err)
		assert.Equal(t, uint(5), *item.RequestID)
		assert.Equal(t, []string{`🎁 I listed "HP laser printer" for your request "Printer wanted".`}, chat.sent)
		assert.Equal(t, []uint{5}, chat.answered)
	})

	t.Run("refused for a closed, missing or own request", func(t *testing.T) {
		service, _, _ := setupService()
		requests := new(MockItemRequestRepository)
		service.WithRequests(requests, nil)
		requests.On("GetByID", uint(5)).Return(&models.ItemRequest{ID: 5, RequesterID: 2, Status: "fulfilled"}, nil)
		requests.On("GetByID", uint(6)).Return(nil, gorm.ErrRecordNotFound)
		requests.On("GetByID", uint(7)).Return(&models.ItemRequest{ID: 7, RequesterID: 1, Status: "active"}, nil)

		_, err := service.CreateItem(1, newReq(5))
		assert.ErrorIs(t, err, ErrRequestClosed)
		_, err = service.CreateItem(1, newReq(6))
		assert.ErrorIs(t, err, ErrRequestClosed)
		_, err = service.CreateItem(1, newReq(7))
		assert.ErrorIs(t, err, ErrOwnRequest)
	})

	t.Run("refused when requests aren't wired up", func(t *testing.T) {
		service, _, _ := setupService()
		_, err := service.CreateItem(1, newReq(5))
		assert.ErrorIs(t, err, ErrRequestClosed)
	})
}

func TestApproveBookingRequest_FulfilsRequest(t *testing.T) {
	requestID := uint(5)
	booking := func(status string, requester uint) *models.BookingRequest {
		return &models.BookingRequest{ID: 1, ItemID: 9, RequesterID: requester, Status: status,
			Item: &models.StoreItem{ID: 9, SellerID: 1, RequestID: &requestID}}
	}

	t.Run("the requester's approved booking closes their request", func(t *testing.T) {
		service, itemRepo, bookingRepo := setupService()
		requests := new(MockItemRequestRepository)
		service.WithRequests(requests, nil)
		itemRepo.On("UpdateStatus", uint(9), "reserved").Return(nil)
		bookingRepo.On("GetByID", uint(1)).Return(booking("pending", 2), nil).Once()
		bookingRepo.On("GetAllByItemID", uint(9)).Return([]models.BookingRequest{}, nil)
		bookingRepo.On("UpdateStatus", uint(1), "approved").Return(nil)
		bookingRepo.On("GetByID", uint(1)).Return(booking("approved", 2), nil).Once()
		requests.On("GetByID", requestID).Return(&models.ItemRequest{ID: requestID, RequesterID: 2, Status: "active"}, nil)
		requests.On("MarkFulfilled", requestID, uint(9)).Return(true, nil)

		_, err := service.ApproveBookingRequest(1, 1)

		assert.NoError(t, err)
		requests.AssertExpectations(t)
	})

	t.Run("someone else's booking leaves the request open", func(t *testing.T) {
		service, itemRepo, bookingRepo := setupService()
		requests := new(MockItemRequestRepository)
		service.WithRequests(requests, nil)
		itemRepo.On("UpdateStatus", uint(9), "reserved").Return(nil)
		bookingRepo.On("GetByID", uint(1)).Return(booking("pending", 3), nil).Once()
		bookingRepo.On("GetAllByItemID", uint(9)).Return([]models.BookingRequest{}, nil)
		bookingRepo.On("UpdateStatus", uint(1), "approved").Return(nil)
		bookingRepo.On("GetByID", uint(1)).Return(booking("approved", 3), nil).Once()
		requests.On("GetByID", requestID).Return(&models.ItemRequest{ID: requestID, RequesterID: 2, Status: "active"}, nil)

		_, err := service.ApproveBookingRequest(1, 1)

		assert.NoError(t, err)
		requests.AssertNotCalled(t, "MarkFulfilled", mock.Anything, mock.Anything)
	})
}
