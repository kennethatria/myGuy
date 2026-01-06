package services

import (
	"store-service/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
