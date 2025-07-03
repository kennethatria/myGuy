package repositories

import (
	"store-service/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.StoreItem{}, &models.ItemImage{}, &models.Bid{}, &models.BookingRequest{}, &models.User{})
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

	return db, nil
}

func TestStoreItemRepository_Create(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	t.Run("successful create item", func(t *testing.T) {
		item := &models.StoreItem{
			Title:       "Test Item",
			Description: "Test Description",
			SellerID:    1,
			PriceType:   "fixed",
			FixedPrice:  100.0,
			Category:    "electronics",
			Condition:   "new",
			Status:      "active",
		}

		err := repo.Create(item)

		assert.NoError(t, err)
		assert.NotZero(t, item.ID)
		assert.NotZero(t, item.CreatedAt)
	})

	t.Run("create item with images", func(t *testing.T) {
		item := &models.StoreItem{
			Title:       "Item with Images",
			Description: "Test Description",
			SellerID:    1,
			PriceType:   "fixed",
			FixedPrice:  150.0,
			Category:    "electronics",
			Condition:   "new",
			Status:      "active",
			Images: []models.ItemImage{
				{URL: "image1.jpg", Order: 0},
				{URL: "image2.jpg", Order: 1},
			},
		}

		err := repo.Create(item)

		assert.NoError(t, err)
		assert.NotZero(t, item.ID)
		assert.Len(t, item.Images, 2)
		
		// Verify images were created with correct item ID
		for _, img := range item.Images {
			assert.Equal(t, item.ID, img.ItemID)
		}
	})

	t.Run("create bidding item", func(t *testing.T) {
		bidDeadline := time.Now().Add(24 * time.Hour)
		item := &models.StoreItem{
			Title:           "Auction Item",
			Description:     "Test Auction",
			SellerID:        2,
			PriceType:       "bidding",
			StartingBid:     50.0,
			MinBidIncrement: 5.0,
			BidDeadline:     &bidDeadline,
			Category:        "collectibles",
			Condition:       "good",
			Status:          "active",
		}

		err := repo.Create(item)

		assert.NoError(t, err)
		assert.NotZero(t, item.ID)
		assert.Equal(t, "bidding", item.PriceType)
		assert.Equal(t, 50.0, item.StartingBid)
		assert.Equal(t, 5.0, item.MinBidIncrement)
		assert.NotNil(t, item.BidDeadline)
	})
}

func TestStoreItemRepository_GetByID(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test item
	testItem := &models.StoreItem{
		Title:       "Test Item",
		Description: "Test Description",
		SellerID:    1,
		PriceType:   "fixed",
		FixedPrice:  100.0,
		Category:    "electronics",
		Condition:   "new",
		Status:      "active",
		Images: []models.ItemImage{
			{URL: "image1.jpg", Order: 0},
			{URL: "image2.jpg", Order: 1},
		},
	}
	db.Create(testItem)

	// Create test bids
	testBids := []models.Bid{
		{ItemID: testItem.ID, BidderID: 2, Amount: 110.0, Status: "active"},
		{ItemID: testItem.ID, BidderID: 3, Amount: 105.0, Status: "outbid"},
	}
	for _, bid := range testBids {
		db.Create(&bid)
	}

	t.Run("successful get by ID", func(t *testing.T) {
		item, err := repo.GetByID(testItem.ID)

		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, testItem.Title, item.Title)
		assert.Equal(t, testItem.Description, item.Description)
		assert.Equal(t, testItem.SellerID, item.SellerID)
		assert.NotNil(t, item.Seller)
		assert.Len(t, item.Images, 2)
		assert.Len(t, item.Bids, 1) // Only active bids
		
		// Check image ordering
		assert.Equal(t, 0, item.Images[0].Order)
		assert.Equal(t, 1, item.Images[1].Order)
	})

	t.Run("item not found", func(t *testing.T) {
		item, err := repo.GetByID(9999)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestStoreItemRepository_GetAll(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test items
	testItems := []models.StoreItem{
		{
			Title:       "Electronics Item",
			Description: "A great electronic device",
			SellerID:    1,
			PriceType:   "fixed",
			FixedPrice:  100.0,
			Category:    "electronics",
			Condition:   "new",
			Status:      "active",
		},
		{
			Title:       "Book Item",
			Description: "An interesting book",
			SellerID:    2,
			PriceType:   "fixed",
			FixedPrice:  25.0,
			Category:    "books",
			Condition:   "good",
			Status:      "active",
		},
		{
			Title:       "Auction Item",
			Description: "Auction description",
			SellerID:    1,
			PriceType:   "bidding",
			StartingBid: 50.0,
			CurrentBid:  75.0,
			Category:    "collectibles",
			Condition:   "fair",
			Status:      "active",
		},
		{
			Title:       "Sold Item",
			Description: "This item is sold",
			SellerID:    3,
			PriceType:   "fixed",
			FixedPrice:  200.0,
			Category:    "electronics",
			Condition:   "new",
			Status:      "sold",
		},
	}

	for _, item := range testItems {
		db.Create(&item)
	}

	t.Run("get all items without filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Page:    1,
			PerPage: 10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 3) // Only active items by default
		assert.Equal(t, int64(3), total)
	})

	t.Run("search filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Search:  "electronics",
			Page:    1,
			PerPage: 10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.Contains(t, items[0].Title, "Electronics")
	})

	t.Run("category filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Category: "electronics",
			Page:     1,
			PerPage:  10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "electronics", items[0].Category)
	})

	t.Run("price type filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			PriceType: "bidding",
			Page:      1,
			PerPage:   10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "bidding", items[0].PriceType)
	})

	t.Run("condition filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Condition: "new",
			Page:      1,
			PerPage:   10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "new", items[0].Condition)
	})

	t.Run("status filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Status:  "sold",
			Page:    1,
			PerPage: 10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "sold", items[0].Status)
	})

	t.Run("seller ID filter", func(t *testing.T) {
		filter := models.StoreItemFilter{
			SellerID: 1,
			Page:     1,
			PerPage:  10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, int64(2), total)
		for _, item := range items {
			assert.Equal(t, uint(1), item.SellerID)
		}
	})

	t.Run("price range filter for fixed price", func(t *testing.T) {
		filter := models.StoreItemFilter{
			PriceType: "fixed",
			MinPrice:  50.0,
			MaxPrice:  150.0,
			Page:      1,
			PerPage:   10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.True(t, items[0].FixedPrice >= 50.0 && items[0].FixedPrice <= 150.0)
	})

	t.Run("price range filter for bidding", func(t *testing.T) {
		filter := models.StoreItemFilter{
			PriceType: "bidding",
			MinPrice:  70.0,
			MaxPrice:  100.0,
			Page:      1,
			PerPage:   10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "bidding", items[0].PriceType)
		assert.True(t, items[0].CurrentBid >= 70.0 && items[0].CurrentBid <= 100.0)
	})

	t.Run("sort by price", func(t *testing.T) {
		filter := models.StoreItemFilter{
			SortBy:    "price",
			SortOrder: "asc",
			Page:      1,
			PerPage:   10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 3)
		assert.Equal(t, int64(3), total)
		
		// Check ascending order
		assert.True(t, items[0].FixedPrice <= items[1].FixedPrice || items[0].CurrentBid <= items[1].CurrentBid)
	})

	t.Run("sort by title", func(t *testing.T) {
		filter := models.StoreItemFilter{
			SortBy:    "title",
			SortOrder: "asc",
			Page:      1,
			PerPage:   10,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 3)
		assert.Equal(t, int64(3), total)
		
		// Check that items are sorted
		assert.True(t, items[0].Title <= items[1].Title)
	})

	t.Run("pagination", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Page:    1,
			PerPage: 2,
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, int64(3), total)
		
		// Test second page
		filter.Page = 2
		items, total, err = repo.GetAll(filter)
		
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(3), total)
	})

	t.Run("default pagination", func(t *testing.T) {
		filter := models.StoreItemFilter{
			Page:    0, // Invalid page
			PerPage: 0, // Invalid per page
		}

		items, total, err := repo.GetAll(filter)

		assert.NoError(t, err)
		assert.Len(t, items, 3)
		assert.Equal(t, int64(3), total)
	})
}

func TestStoreItemRepository_Update(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test item
	testItem := &models.StoreItem{
		Title:       "Original Title",
		Description: "Original Description",
		SellerID:    1,
		PriceType:   "fixed",
		FixedPrice:  100.0,
		Category:    "electronics",
		Condition:   "new",
		Status:      "active",
	}
	db.Create(testItem)

	t.Run("successful update", func(t *testing.T) {
		testItem.Title = "Updated Title"
		testItem.Description = "Updated Description"
		testItem.FixedPrice = 150.0

		err := repo.Update(testItem)

		assert.NoError(t, err)
		
		// Verify the update
		var updated models.StoreItem
		db.First(&updated, testItem.ID)
		assert.Equal(t, "Updated Title", updated.Title)
		assert.Equal(t, "Updated Description", updated.Description)
		assert.Equal(t, 150.0, updated.FixedPrice)
	})

	t.Run("update with images", func(t *testing.T) {
		testItem.Images = []models.ItemImage{
			{URL: "new_image1.jpg", Order: 0},
			{URL: "new_image2.jpg", Order: 1},
		}

		err := repo.Update(testItem)

		assert.NoError(t, err)
		
		// Verify images were updated
		var updated models.StoreItem
		db.Preload("Images").First(&updated, testItem.ID)
		assert.Len(t, updated.Images, 2)
	})
}

func TestStoreItemRepository_Delete(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test item
	testItem := &models.StoreItem{
		Title:    "Test Item",
		SellerID: 1,
		Status:   "active",
	}
	db.Create(testItem)

	t.Run("successful delete", func(t *testing.T) {
		err := repo.Delete(testItem.ID)

		assert.NoError(t, err)
		
		// Verify the item is deleted (soft delete)
		var deleted models.StoreItem
		result := db.First(&deleted, testItem.ID)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})

	t.Run("delete non-existent item", func(t *testing.T) {
		err := repo.Delete(9999)

		// GORM doesn't return error for deleting non-existent records
		assert.NoError(t, err)
	})
}

func TestStoreItemRepository_GetBySellerID(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test items
	testItems := []models.StoreItem{
		{Title: "Item 1", SellerID: 1, Status: "active"},
		{Title: "Item 2", SellerID: 1, Status: "sold"},
		{Title: "Item 3", SellerID: 2, Status: "active"},
	}

	for _, item := range testItems {
		db.Create(&item)
	}

	t.Run("successful get by seller ID", func(t *testing.T) {
		items, err := repo.GetBySellerID(1)

		assert.NoError(t, err)
		assert.Len(t, items, 2)
		for _, item := range items {
			assert.Equal(t, uint(1), item.SellerID)
		}
	})

	t.Run("no items for seller", func(t *testing.T) {
		items, err := repo.GetBySellerID(999)

		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestStoreItemRepository_GetByBuyerID(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test items
	buyerID := uint(2)
	testItems := []models.StoreItem{
		{Title: "Purchased Item 1", SellerID: 1, BuyerID: &buyerID, Status: "sold"},
		{Title: "Purchased Item 2", SellerID: 3, BuyerID: &buyerID, Status: "sold"},
		{Title: "Not Purchased", SellerID: 1, Status: "active"},
	}

	for _, item := range testItems {
		db.Create(&item)
	}

	t.Run("successful get by buyer ID", func(t *testing.T) {
		items, err := repo.GetByBuyerID(2)

		assert.NoError(t, err)
		assert.Len(t, items, 2)
		for _, item := range items {
			assert.Equal(t, uint(2), *item.BuyerID)
		}
	})

	t.Run("no purchases for buyer", func(t *testing.T) {
		items, err := repo.GetByBuyerID(999)

		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestStoreItemRepository_UpdateStatus(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test item
	testItem := &models.StoreItem{
		Title:    "Test Item",
		SellerID: 1,
		Status:   "active",
	}
	db.Create(testItem)

	t.Run("successful status update", func(t *testing.T) {
		err := repo.UpdateStatus(testItem.ID, "sold")

		assert.NoError(t, err)
		
		// Verify the status update
		var updated models.StoreItem
		db.First(&updated, testItem.ID)
		assert.Equal(t, "sold", updated.Status)
	})

	t.Run("update non-existent item", func(t *testing.T) {
		err := repo.UpdateStatus(9999, "expired")

		// GORM doesn't return error for updating non-existent records
		assert.NoError(t, err)
	})
}

func TestStoreItemRepository_MarkAsSold(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test item
	testItem := &models.StoreItem{
		Title:    "Test Item",
		SellerID: 1,
		Status:   "active",
	}
	db.Create(testItem)

	t.Run("successful mark as sold", func(t *testing.T) {
		buyerID := uint(2)
		err := repo.MarkAsSold(testItem.ID, buyerID)

		assert.NoError(t, err)
		
		// Verify the item is marked as sold
		var updated models.StoreItem
		db.First(&updated, testItem.ID)
		assert.Equal(t, "sold", updated.Status)
		assert.Equal(t, buyerID, *updated.BuyerID)
		assert.NotNil(t, updated.SoldAt)
	})

	t.Run("mark non-existent item as sold", func(t *testing.T) {
		err := repo.MarkAsSold(9999, 2)

		// GORM doesn't return error for updating non-existent records
		assert.NoError(t, err)
	})
}

func TestStoreItemRepository_ExpireOldBidItems(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	
	repo := NewStoreItemRepository(db)

	// Create test items
	pastDeadline := time.Now().Add(-1 * time.Hour)
	futureDeadline := time.Now().Add(1 * time.Hour)
	
	testItems := []models.StoreItem{
		{
			Title:       "Expired Auction",
			SellerID:    1,
			PriceType:   "bidding",
			BidDeadline: &pastDeadline,
			Status:      "active",
		},
		{
			Title:       "Active Auction",
			SellerID:    1,
			PriceType:   "bidding",
			BidDeadline: &futureDeadline,
			Status:      "active",
		},
		{
			Title:     "Fixed Price Item",
			SellerID:  1,
			PriceType: "fixed",
			Status:    "active",
		},
	}

	for _, item := range testItems {
		db.Create(&item)
	}

	t.Run("successful expire old bid items", func(t *testing.T) {
		err := repo.ExpireOldBidItems()

		assert.NoError(t, err)
		
		// Verify only the expired auction is marked as expired
		var items []models.StoreItem
		db.Find(&items)
		
		expiredCount := 0
		activeCount := 0
		
		for _, item := range items {
			if item.Status == "expired" {
				expiredCount++
				assert.Equal(t, "bidding", item.PriceType)
				assert.True(t, item.BidDeadline.Before(time.Now()))
			} else if item.Status == "active" {
				activeCount++
			}
		}
		
		assert.Equal(t, 1, expiredCount)
		assert.Equal(t, 2, activeCount)
	})
}