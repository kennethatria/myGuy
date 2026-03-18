package services

import (
	"errors"
	"store-service/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupServiceWithUserRepo() (*StoreService, *MockStoreItemRepository, *MockBidRepository, *MockBookingRequestRepository, *MockUserRepository) {
	itemRepo := new(MockStoreItemRepository)
	bidRepo := new(MockBidRepository)
	bookingRepo := new(MockBookingRequestRepository)
	userRepo := new(MockUserRepository)
	service := NewStoreService(nil, itemRepo, bidRepo, bookingRepo, userRepo)
	return service, itemRepo, bidRepo, bookingRepo, userRepo
}

func TestBookingWorkflow_Scenarios(t *testing.T) {
	t.Run("Bug_ConfirmDelivery_ShouldMarkItemAsSold", func(t *testing.T) {
		// Setup service and mocks
		service, itemRepo, _, bookingRepo := setupService()

		// Arrange
		sellerID := uint(1)
		buyerID := uint(2)
		itemID := uint(100)
		requestID := uint(50)

		// Mock Item
		item := &models.StoreItem{
			ID:       itemID,
			SellerID: sellerID,
			Status:   "active", // Initially active
		}

		// Mock Booking Request (Ready for completion)
		request := &models.BookingRequest{
			ID:          requestID,
			ItemID:      itemID,
			RequesterID: buyerID,
			Status:      "item_received", // State required for ConfirmDelivery
			Item:        item,
		}

		// Mock Repository Calls
		bookingRepo.On("GetByID", requestID).Return(request, nil)
		itemRepo.On("GetByID", itemID).Return(item, nil)

		// 1. Expect booking status update to "completed"
		bookingRepo.On("UpdateStatus", requestID, "completed").Return(nil)

		// 2. CRITICAL: Expect item status update to "sold"
		// This expectation is likely MISSING in the current implementation, so we expect this test to fail
		// or verify that it IS called if we are fixing it.
		itemRepo.On("MarkAsSold", itemID, buyerID).Return(nil)

		// Return the updated booking for the final GetByID call
		completedRequest := &models.BookingRequest{
			ID:          requestID,
			ItemID:      itemID,
			RequesterID: buyerID,
			Status:      "completed",
		}
		bookingRepo.On("GetByID", requestID).Return(completedRequest, nil)

		// Act
		_, err := service.ConfirmDelivery(requestID, sellerID)

		// Assert
		assert.NoError(t, err)
		itemRepo.AssertExpectations(t)
	})

	t.Run("ApproveBooking_ShouldPreventDoubleBooking", func(t *testing.T) {
		// Setup service and mocks
		service, _, _, bookingRepo := setupService()

		// Arrange
		sellerID := uint(1)
		itemID := uint(100)
		
		// Request 1: The one we are trying to approve
		requestToApprove := &models.BookingRequest{
			ID:          uint(51),
			ItemID:      itemID,
			RequesterID: uint(2),
			Status:      "pending",
			Item: &models.StoreItem{
				ID:       itemID,
				SellerID: sellerID,
			},
		}

		// Request 2: Already approved!
		existingApprovedRequest := models.BookingRequest{
			ID:          uint(52),
			ItemID:      itemID,
			RequesterID: uint(3),
			Status:      "approved",
		}

		bookingRepo.On("GetByID", requestToApprove.ID).Return(requestToApprove, nil)
		
		// We expect the service to check for other approved bookings.
		// If this call is missing, the code doesn't check for double bookings.
		bookingRepo.On("GetAllByItemID", itemID).Return([]models.BookingRequest{
			*requestToApprove,
			existingApprovedRequest,
		}, nil)

		// Act
		_, err := service.ApproveBookingRequest(requestToApprove.ID, sellerID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "another booking is already approved")
		
		// Ensure UpdateStatus was NOT called (because it should have failed before)
		bookingRepo.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything)
	})
}

func TestConfirmItemReceived(t *testing.T) {
	t.Run("successful confirm item received", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()

		buyerID := uint(2)
		requestID := uint(1)

		request := &models.BookingRequest{
			ID:          requestID,
			ItemID:      uint(100),
			RequesterID: buyerID,
			Status:      "approved",
		}
		updatedRequest := &models.BookingRequest{
			ID:          requestID,
			RequesterID: buyerID,
			Status:      "item_received",
		}

		bookingRepo.On("GetByID", requestID).Return(request, nil).Once()
		bookingRepo.On("UpdateStatus", requestID, "item_received").Return(nil)
		bookingRepo.On("GetByID", requestID).Return(updatedRequest, nil).Once()

		result, err := service.ConfirmItemReceived(requestID, buyerID)

		assert.NoError(t, err)
		assert.Equal(t, "item_received", result.Status)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("booking not found", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()

		bookingRepo.On("GetByID", uint(1)).Return(nil, errors.New("record not found"))

		_, err := service.ConfirmItemReceived(1, 2)

		assert.Error(t, err)
		bookingRepo.AssertExpectations(t)
	})

	t.Run("not the buyer", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()

		request := &models.BookingRequest{
			ID:          uint(1),
			RequesterID: uint(3), // Different user
			Status:      "approved",
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		_, err := service.ConfirmItemReceived(1, 2) // userID=2, but RequesterID=3

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only the buyer can confirm receipt")
		bookingRepo.AssertExpectations(t)
	})

	t.Run("booking not in approved status", func(t *testing.T) {
		service, _, _, bookingRepo := setupService()

		request := &models.BookingRequest{
			ID:          uint(1),
			RequesterID: uint(2),
			Status:      "pending", // Not approved
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		_, err := service.ConfirmItemReceived(1, 2)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "booking must be approved before confirming receipt")
		bookingRepo.AssertExpectations(t)
	})
}

func TestSubmitBuyerRating(t *testing.T) {
	t.Run("successful buyer rating", func(t *testing.T) {
		service, _, _, bookingRepo, userRepo := setupServiceWithUserRepo()

		buyerID := uint(2)
		requestID := uint(1)
		sellerID := uint(1)

		item := &models.StoreItem{ID: uint(100), SellerID: sellerID}
		request := &models.BookingRequest{
			ID:          requestID,
			ItemID:      uint(100),
			RequesterID: buyerID,
			Status:      "completed",
			Item:        item,
		}
		updatedRequest := &models.BookingRequest{
			ID:          requestID,
			RequesterID: buyerID,
			Status:      "completed",
		}

		bookingRepo.On("GetByID", requestID).Return(request, nil).Once()
		bookingRepo.On("UpdateBuyerRating", requestID, 5, "Great!").Return(nil)
		userRepo.On("UpdateRating", sellerID, float64(5)).Return(nil)
		bookingRepo.On("GetByID", requestID).Return(updatedRequest, nil).Once()

		result, err := service.SubmitBuyerRating(requestID, buyerID, 5, "Great!")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		bookingRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("not the buyer", func(t *testing.T) {
		service, _, _, bookingRepo, _ := setupServiceWithUserRepo()

		request := &models.BookingRequest{
			ID:          uint(1),
			RequesterID: uint(3), // Different user
			Status:      "completed",
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		_, err := service.SubmitBuyerRating(1, 2, 5, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only the buyer can rate the seller")
		bookingRepo.AssertExpectations(t)
	})

	t.Run("booking not completed", func(t *testing.T) {
		service, _, _, bookingRepo, _ := setupServiceWithUserRepo()

		request := &models.BookingRequest{
			ID:          uint(1),
			RequesterID: uint(2),
			Status:      "approved", // Not completed
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		_, err := service.SubmitBuyerRating(1, 2, 5, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "booking must be completed before rating")
		bookingRepo.AssertExpectations(t)
	})

	t.Run("already rated", func(t *testing.T) {
		service, _, _, bookingRepo, _ := setupServiceWithUserRepo()

		rating := 4
		request := &models.BookingRequest{
			ID:          uint(1),
			RequesterID: uint(2),
			Status:      "completed",
			BuyerRating: &rating, // Already rated
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)

		_, err := service.SubmitBuyerRating(1, 2, 5, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "buyer has already rated this transaction")
		bookingRepo.AssertExpectations(t)
	})
}

func TestSubmitSellerRating(t *testing.T) {
	t.Run("successful seller rating", func(t *testing.T) {
		service, itemRepo, _, bookingRepo, userRepo := setupServiceWithUserRepo()

		sellerID := uint(1)
		buyerID := uint(2)
		requestID := uint(1)
		itemID := uint(100)

		item := &models.StoreItem{ID: itemID, SellerID: sellerID}
		request := &models.BookingRequest{
			ID:          requestID,
			ItemID:      itemID,
			RequesterID: buyerID,
			Status:      "completed",
		}
		updatedRequest := &models.BookingRequest{
			ID:     requestID,
			Status: "completed",
		}

		bookingRepo.On("GetByID", requestID).Return(request, nil).Once()
		itemRepo.On("GetByID", itemID).Return(item, nil)
		bookingRepo.On("UpdateSellerRating", requestID, 4, "Good buyer").Return(nil)
		userRepo.On("UpdateRating", buyerID, float64(4)).Return(nil)
		bookingRepo.On("GetByID", requestID).Return(updatedRequest, nil).Once()

		result, err := service.SubmitSellerRating(requestID, sellerID, 4, "Good buyer")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		bookingRepo.AssertExpectations(t)
		itemRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("not the seller", func(t *testing.T) {
		service, itemRepo, _, bookingRepo, _ := setupServiceWithUserRepo()

		item := &models.StoreItem{ID: uint(100), SellerID: uint(3)} // Different seller
		request := &models.BookingRequest{
			ID:          uint(1),
			ItemID:      uint(100),
			RequesterID: uint(2),
			Status:      "completed",
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)
		itemRepo.On("GetByID", uint(100)).Return(item, nil)

		_, err := service.SubmitSellerRating(1, 1, 5, "") // sellerID=1, but item.SellerID=3

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only the seller can rate the buyer")
		bookingRepo.AssertExpectations(t)
		itemRepo.AssertExpectations(t)
	})

	t.Run("booking not completed", func(t *testing.T) {
		service, itemRepo, _, bookingRepo, _ := setupServiceWithUserRepo()

		item := &models.StoreItem{ID: uint(100), SellerID: uint(1)}
		request := &models.BookingRequest{
			ID:          uint(1),
			ItemID:      uint(100),
			RequesterID: uint(2),
			Status:      "approved", // Not completed
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)
		itemRepo.On("GetByID", uint(100)).Return(item, nil)

		_, err := service.SubmitSellerRating(1, 1, 5, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "booking must be completed before rating")
		bookingRepo.AssertExpectations(t)
		itemRepo.AssertExpectations(t)
	})

	t.Run("already rated", func(t *testing.T) {
		service, itemRepo, _, bookingRepo, _ := setupServiceWithUserRepo()

		rating := 4
		item := &models.StoreItem{ID: uint(100), SellerID: uint(1)}
		request := &models.BookingRequest{
			ID:           uint(1),
			ItemID:       uint(100),
			RequesterID:  uint(2),
			Status:       "completed",
			SellerRating: &rating, // Already rated
		}
		bookingRepo.On("GetByID", uint(1)).Return(request, nil)
		itemRepo.On("GetByID", uint(100)).Return(item, nil)

		_, err := service.SubmitSellerRating(1, 1, 5, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "seller has already rated this transaction")
		bookingRepo.AssertExpectations(t)
		itemRepo.AssertExpectations(t)
	})
}
