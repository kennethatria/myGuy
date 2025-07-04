package repositories

import (
	"store-service/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBookingTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.StoreItem{}, &models.BookingRequest{}, &models.User{})
	if err != nil {
		return nil, err
	}

	// Create test users
	users := []models.User{
		{ID: 1, Username: "user1", Email: "user1@example.com"},
		{ID: 2, Username: "user2", Email: "user2@example.com"},
		{ID: 3, Username: "user3", Email: "user3@example.com"},
	}
	
	for _, user := range users {
		db.Create(&user)
	}

	// Create test items
	items := []models.StoreItem{
		{ID: 1, Title: "Bookable Item 1", SellerID: 1, Status: "active"},
		{ID: 2, Title: "Bookable Item 2", SellerID: 2, Status: "active"},
		{ID: 3, Title: "Sold Item", SellerID: 1, Status: "sold"},
	}
	
	for _, item := range items {
		db.Create(&item)
	}

	return db, nil
}

func TestBookingRequestRepository_Create(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	t.Run("successful create booking request", func(t *testing.T) {
		request := &models.BookingRequest{
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
			Message:     "I'd like to book this item",
		}

		err := repo.Create(request)

		assert.NoError(t, err)
		assert.NotZero(t, request.ID)
		assert.NotZero(t, request.CreatedAt)
	})

	t.Run("create booking request without message", func(t *testing.T) {
		request := &models.BookingRequest{
			ItemID:      2,
			RequesterID: 3,
			Status:      "pending",
		}

		err := repo.Create(request)

		assert.NoError(t, err)
		assert.NotZero(t, request.ID)
		assert.Empty(t, request.Message)
	})

	t.Run("create multiple booking requests", func(t *testing.T) {
		requests := []models.BookingRequest{
			{ItemID: 1, RequesterID: 3, Status: "pending", Message: "Another booking request"},
			{ItemID: 2, RequesterID: 2, Status: "pending", Message: "Yet another request"},
		}

		for _, request := range requests {
			err := repo.Create(&request)
			assert.NoError(t, err)
			assert.NotZero(t, request.ID)
		}
	})
}

func TestBookingRequestRepository_GetByID(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	// Create test booking request
	testRequest := &models.BookingRequest{
		ItemID:      1,
		RequesterID: 2,
		Status:      "pending",
		Message:     "Test booking request",
	}
	db.Create(testRequest)

	t.Run("successful get by ID", func(t *testing.T) {
		request, err := repo.GetByID(testRequest.ID)

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, testRequest.ItemID, request.ItemID)
		assert.Equal(t, testRequest.RequesterID, request.RequesterID)
		assert.Equal(t, testRequest.Status, request.Status)
		assert.Equal(t, testRequest.Message, request.Message)
		assert.NotNil(t, request.Item)
		assert.NotNil(t, request.Requester)
	})

	t.Run("booking request not found", func(t *testing.T) {
		request, err := repo.GetByID(9999)

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestBookingRequestRepository_GetByItemID(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	// Create test booking requests
	testRequests := []models.BookingRequest{
		{ItemID: 1, RequesterID: 2, Status: "pending", Message: "First request"},
		{ItemID: 1, RequesterID: 3, Status: "approved", Message: "Second request"},
		{ItemID: 2, RequesterID: 2, Status: "pending", Message: "Different item"},
	}

	for _, request := range testRequests {
		db.Create(&request)
	}

	t.Run("successful get by item ID", func(t *testing.T) {
		request, err := repo.GetByItemID(1)

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, uint(1), request.ItemID)
		assert.NotNil(t, request.Item)
		assert.NotNil(t, request.Requester)
	})

	t.Run("no booking request for item", func(t *testing.T) {
		request, err := repo.GetByItemID(999)

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("get booking request for different item", func(t *testing.T) {
		request, err := repo.GetByItemID(2)

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, uint(2), request.ItemID)
	})
}

func TestBookingRequestRepository_GetByItemAndRequester(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	// Create test booking requests
	testRequests := []models.BookingRequest{
		{ItemID: 1, RequesterID: 2, Status: "pending", Message: "User 2's request for item 1"},
		{ItemID: 1, RequesterID: 3, Status: "approved", Message: "User 3's request for item 1"},
		{ItemID: 2, RequesterID: 2, Status: "pending", Message: "User 2's request for item 2"},
	}

	for _, request := range testRequests {
		db.Create(&request)
	}

	t.Run("successful get by item and requester", func(t *testing.T) {
		request, err := repo.GetByItemAndRequester(1, 2)

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, uint(1), request.ItemID)
		assert.Equal(t, uint(2), request.RequesterID)
		assert.Equal(t, "pending", request.Status)
		assert.NotNil(t, request.Item)
		assert.NotNil(t, request.Requester)
	})

	t.Run("get different requester for same item", func(t *testing.T) {
		request, err := repo.GetByItemAndRequester(1, 3)

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, uint(1), request.ItemID)
		assert.Equal(t, uint(3), request.RequesterID)
		assert.Equal(t, "approved", request.Status)
	})

	t.Run("get same requester for different item", func(t *testing.T) {
		request, err := repo.GetByItemAndRequester(2, 2)

		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, uint(2), request.ItemID)
		assert.Equal(t, uint(2), request.RequesterID)
	})

	t.Run("no booking request for item and requester combination", func(t *testing.T) {
		request, err := repo.GetByItemAndRequester(999, 2)

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("no booking request for valid item but non-existent requester", func(t *testing.T) {
		request, err := repo.GetByItemAndRequester(1, 999)

		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestBookingRequestRepository_GetByRequesterID(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	// Create test booking requests
	testRequests := []models.BookingRequest{
		{ItemID: 1, RequesterID: 2, Status: "pending", Message: "First request by user 2"},
		{ItemID: 2, RequesterID: 2, Status: "approved", Message: "Second request by user 2"},
		{ItemID: 1, RequesterID: 3, Status: "pending", Message: "Request by user 3"},
	}

	for _, request := range testRequests {
		db.Create(&request)
	}

	t.Run("successful get by requester ID", func(t *testing.T) {
		requests, err := repo.GetByRequesterID(2)

		assert.NoError(t, err)
		assert.Len(t, requests, 2)
		
		// Check that all requests belong to requester 2
		for _, request := range requests {
			assert.Equal(t, uint(2), request.RequesterID)
			assert.NotNil(t, request.Item)
		}
	})

	t.Run("get requests for different requester", func(t *testing.T) {
		requests, err := repo.GetByRequesterID(3)

		assert.NoError(t, err)
		assert.Len(t, requests, 1)
		assert.Equal(t, uint(3), requests[0].RequesterID)
	})

	t.Run("no requests for requester", func(t *testing.T) {
		requests, err := repo.GetByRequesterID(999)

		assert.NoError(t, err)
		assert.Empty(t, requests)
	})

	t.Run("no requests for requester with no booking requests", func(t *testing.T) {
		requests, err := repo.GetByRequesterID(1)

		assert.NoError(t, err)
		assert.Empty(t, requests)
	})
}

func TestBookingRequestRepository_UpdateStatus(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	// Create test booking request
	testRequest := &models.BookingRequest{
		ItemID:      1,
		RequesterID: 2,
		Status:      "pending",
		Message:     "Test booking request",
	}
	db.Create(testRequest)

	t.Run("successful status update to approved", func(t *testing.T) {
		err := repo.UpdateStatus(testRequest.ID, "approved")

		assert.NoError(t, err)
		
		// Verify the status update
		var updated models.BookingRequest
		db.First(&updated, testRequest.ID)
		assert.Equal(t, "approved", updated.Status)
	})

	t.Run("successful status update to rejected", func(t *testing.T) {
		err := repo.UpdateStatus(testRequest.ID, "rejected")

		assert.NoError(t, err)
		
		// Verify the status update
		var updated models.BookingRequest
		db.First(&updated, testRequest.ID)
		assert.Equal(t, "rejected", updated.Status)
	})

	t.Run("update status back to pending", func(t *testing.T) {
		err := repo.UpdateStatus(testRequest.ID, "pending")

		assert.NoError(t, err)
		
		// Verify the status update
		var updated models.BookingRequest
		db.First(&updated, testRequest.ID)
		assert.Equal(t, "pending", updated.Status)
	})

	t.Run("update non-existent booking request", func(t *testing.T) {
		err := repo.UpdateStatus(9999, "approved")

		// GORM doesn't return error for updating non-existent records
		assert.NoError(t, err)
	})
}

func TestBookingRequestRepository_Delete(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	// Create test booking requests
	testRequests := []models.BookingRequest{
		{ItemID: 1, RequesterID: 2, Status: "pending", Message: "Request to be deleted"},
		{ItemID: 2, RequesterID: 3, Status: "approved", Message: "Request to keep"},
	}

	for _, request := range testRequests {
		db.Create(&request)
	}

	t.Run("successful delete", func(t *testing.T) {
		err := repo.Delete(testRequests[0].ID)

		assert.NoError(t, err)
		
		// Verify the request is deleted (soft delete)
		var deleted models.BookingRequest
		result := db.First(&deleted, testRequests[0].ID)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
		
		// Verify other requests are not affected
		var remaining models.BookingRequest
		result = db.First(&remaining, testRequests[1].ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, testRequests[1].Message, remaining.Message)
	})

	t.Run("delete non-existent booking request", func(t *testing.T) {
		err := repo.Delete(9999)

		// GORM doesn't return error for deleting non-existent records
		assert.NoError(t, err)
	})

	t.Run("delete already deleted request", func(t *testing.T) {
		// Try to delete the same request again
		err := repo.Delete(testRequests[0].ID)

		// GORM doesn't return error for deleting already deleted records
		assert.NoError(t, err)
	})
}

func TestBookingRequestRepository_Integration(t *testing.T) {
	db, err := setupBookingTestDB()
	assert.NoError(t, err)
	
	repo := NewBookingRequestRepository(db)

	t.Run("complete booking request workflow", func(t *testing.T) {
		// 1. Create a booking request
		request := &models.BookingRequest{
			ItemID:      1,
			RequesterID: 2,
			Status:      "pending",
			Message:     "I'd like to book this item for the weekend",
		}

		err := repo.Create(request)
		assert.NoError(t, err)
		assert.NotZero(t, request.ID)

		// 2. Get the request by ID
		retrieved, err := repo.GetByID(request.ID)
		assert.NoError(t, err)
		assert.Equal(t, request.ItemID, retrieved.ItemID)
		assert.Equal(t, request.RequesterID, retrieved.RequesterID)
		assert.Equal(t, request.Status, retrieved.Status)
		assert.Equal(t, request.Message, retrieved.Message)

		// 3. Get the request by item and requester
		byItemAndRequester, err := repo.GetByItemAndRequester(1, 2)
		assert.NoError(t, err)
		assert.Equal(t, request.ID, byItemAndRequester.ID)

		// 4. Get all requests by requester
		userRequests, err := repo.GetByRequesterID(2)
		assert.NoError(t, err)
		assert.Len(t, userRequests, 1)
		assert.Equal(t, request.ID, userRequests[0].ID)

		// 5. Approve the request
		err = repo.UpdateStatus(request.ID, "approved")
		assert.NoError(t, err)

		// 6. Verify the status change
		updated, err := repo.GetByID(request.ID)
		assert.NoError(t, err)
		assert.Equal(t, "approved", updated.Status)

		// 7. Get by item ID
		byItem, err := repo.GetByItemID(1)
		assert.NoError(t, err)
		assert.Equal(t, request.ID, byItem.ID)
		assert.Equal(t, "approved", byItem.Status)

		// 8. Clean up - delete the request
		err = repo.Delete(request.ID)
		assert.NoError(t, err)

		// 9. Verify deletion
		_, err = repo.GetByID(request.ID)
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("multiple requests per item", func(t *testing.T) {
		// Create multiple requests for the same item
		requests := []models.BookingRequest{
			{ItemID: 1, RequesterID: 2, Status: "pending", Message: "First request"},
			{ItemID: 1, RequesterID: 3, Status: "pending", Message: "Second request"},
		}

		for _, request := range requests {
			err := repo.Create(&request)
			assert.NoError(t, err)
		}

		// Get requests by each requester
		user2Requests, err := repo.GetByRequesterID(2)
		assert.NoError(t, err)
		assert.Len(t, user2Requests, 1)

		user3Requests, err := repo.GetByRequesterID(3)
		assert.NoError(t, err)
		assert.Len(t, user3Requests, 1)

		// Get by item and specific requester
		request2, err := repo.GetByItemAndRequester(1, 2)
		assert.NoError(t, err)
		assert.Equal(t, uint(2), request2.RequesterID)

		request3, err := repo.GetByItemAndRequester(1, 3)
		assert.NoError(t, err)
		assert.Equal(t, uint(3), request3.RequesterID)

		// Approve one and reject the other
		err = repo.UpdateStatus(requests[0].ID, "approved")
		assert.NoError(t, err)

		err = repo.UpdateStatus(requests[1].ID, "rejected")
		assert.NoError(t, err)

		// Verify status changes
		approved, err := repo.GetByID(requests[0].ID)
		assert.NoError(t, err)
		assert.Equal(t, "approved", approved.Status)

		rejected, err := repo.GetByID(requests[1].ID)
		assert.NoError(t, err)
		assert.Equal(t, "rejected", rejected.Status)
	})

	t.Run("get all requests by item ID", func(t *testing.T) {
		// Create multiple requests for the same item
		requests := []models.BookingRequest{
			{ItemID: 2, RequesterID: 1, Status: "pending", Message: "Request 1"},
			{ItemID: 2, RequesterID: 3, Status: "approved", Message: "Request 2"},
		}

		for _, request := range requests {
			err := repo.Create(&request)
			assert.NoError(t, err)
		}

		// Get all requests for item 2
		allRequests, err := repo.GetAllByItemID(2)
		assert.NoError(t, err)
		assert.Len(t, allRequests, 2)

		// Verify both requests are returned
		requestIDs := make([]uint, 0, len(allRequests))
		for _, req := range allRequests {
			requestIDs = append(requestIDs, req.RequesterID)
		}
		assert.Contains(t, requestIDs, uint(1))
		assert.Contains(t, requestIDs, uint(3))

		// Test with item that has no requests
		noRequests, err := repo.GetAllByItemID(999)
		assert.NoError(t, err)
		assert.Len(t, noRequests, 0)
	})
}