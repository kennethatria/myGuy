package repositories

import (
	"store-service/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBidTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.StoreItem{}, &models.ItemImage{}, &models.Bid{}, &models.User{})
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
		{ID: 1, Title: "Auction Item 1", SellerID: 1, PriceType: "bidding", StartingBid: 100.0, Status: "active"},
		{ID: 2, Title: "Auction Item 2", SellerID: 2, PriceType: "bidding", StartingBid: 50.0, Status: "active"},
	}
	
	for _, item := range items {
		db.Create(&item)
	}

	return db, nil
}

func TestBidRepository_Create(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	t.Run("successful create bid", func(t *testing.T) {
		bid := &models.Bid{
			ItemID:   1,
			BidderID: 2,
			Amount:   110.0,
			Message:  "My first bid",
			Status:   "active",
		}

		err := repo.Create(bid)

		assert.NoError(t, err)
		assert.NotZero(t, bid.ID)
		assert.NotZero(t, bid.CreatedAt)
	})

	t.Run("create bid without message", func(t *testing.T) {
		bid := &models.Bid{
			ItemID:   1,
			BidderID: 3,
			Amount:   115.0,
			Status:   "active",
		}

		err := repo.Create(bid)

		assert.NoError(t, err)
		assert.NotZero(t, bid.ID)
		assert.Empty(t, bid.Message)
	})

	t.Run("create multiple bids for same item", func(t *testing.T) {
		bids := []models.Bid{
			{ItemID: 1, BidderID: 2, Amount: 120.0, Status: "active"},
			{ItemID: 1, BidderID: 3, Amount: 125.0, Status: "active"},
		}

		for _, bid := range bids {
			err := repo.Create(&bid)
			assert.NoError(t, err)
			assert.NotZero(t, bid.ID)
		}
	})
}

func TestBidRepository_GetByID(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bid
	testBid := &models.Bid{
		ItemID:   1,
		BidderID: 2,
		Amount:   110.0,
		Message:  "Test bid",
		Status:   "active",
	}
	db.Create(testBid)

	t.Run("successful get by ID", func(t *testing.T) {
		bid, err := repo.GetByID(testBid.ID)

		assert.NoError(t, err)
		assert.NotNil(t, bid)
		assert.Equal(t, testBid.ItemID, bid.ItemID)
		assert.Equal(t, testBid.BidderID, bid.BidderID)
		assert.Equal(t, testBid.Amount, bid.Amount)
		assert.Equal(t, testBid.Message, bid.Message)
		assert.Equal(t, testBid.Status, bid.Status)
	})

	t.Run("bid not found", func(t *testing.T) {
		bid, err := repo.GetByID(9999)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestBidRepository_GetByItemID(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bids
	testBids := []models.Bid{
		{ItemID: 1, BidderID: 2, Amount: 110.0, Status: "active"},
		{ItemID: 1, BidderID: 3, Amount: 120.0, Status: "active"},
		{ItemID: 1, BidderID: 2, Amount: 105.0, Status: "outbid"},
		{ItemID: 2, BidderID: 3, Amount: 60.0, Status: "active"},
	}

	for _, bid := range testBids {
		db.Create(&bid)
	}

	t.Run("successful get by item ID", func(t *testing.T) {
		bids, err := repo.GetByItemID(1)

		assert.NoError(t, err)
		assert.Len(t, bids, 3)
		
		// Check that all bids belong to item 1
		for _, bid := range bids {
			assert.Equal(t, uint(1), bid.ItemID)
		}
		
		// Check that bids are ordered by amount DESC
		assert.True(t, bids[0].Amount >= bids[1].Amount)
		assert.True(t, bids[1].Amount >= bids[2].Amount)
	})

	t.Run("no bids for item", func(t *testing.T) {
		bids, err := repo.GetByItemID(999)

		assert.NoError(t, err)
		assert.Empty(t, bids)
	})

	t.Run("get bids for different item", func(t *testing.T) {
		bids, err := repo.GetByItemID(2)

		assert.NoError(t, err)
		assert.Len(t, bids, 1)
		assert.Equal(t, uint(2), bids[0].ItemID)
	})
}

func TestBidRepository_GetByBidderID(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bids
	testBids := []models.Bid{
		{ItemID: 1, BidderID: 2, Amount: 110.0, Status: "active"},
		{ItemID: 2, BidderID: 2, Amount: 60.0, Status: "active"},
		{ItemID: 1, BidderID: 3, Amount: 120.0, Status: "active"},
	}

	for _, bid := range testBids {
		db.Create(&bid)
	}

	t.Run("successful get by bidder ID", func(t *testing.T) {
		bids, err := repo.GetByBidderID(2)

		assert.NoError(t, err)
		assert.Len(t, bids, 2)
		
		// Check that all bids belong to bidder 2
		for _, bid := range bids {
			assert.Equal(t, uint(2), bid.BidderID)
		}
	})

	t.Run("no bids for bidder", func(t *testing.T) {
		bids, err := repo.GetByBidderID(999)

		assert.NoError(t, err)
		assert.Empty(t, bids)
	})

	t.Run("get bids for different bidder", func(t *testing.T) {
		bids, err := repo.GetByBidderID(3)

		assert.NoError(t, err)
		assert.Len(t, bids, 1)
		assert.Equal(t, uint(3), bids[0].BidderID)
	})
}

func TestBidRepository_GetHighestBidForItem(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bids
	testBids := []models.Bid{
		{ItemID: 1, BidderID: 2, Amount: 110.0, Status: "active"},
		{ItemID: 1, BidderID: 3, Amount: 120.0, Status: "active"},
		{ItemID: 1, BidderID: 2, Amount: 105.0, Status: "outbid"},
		{ItemID: 2, BidderID: 3, Amount: 60.0, Status: "active"},
	}

	for _, bid := range testBids {
		db.Create(&bid)
	}

	t.Run("successful get highest bid", func(t *testing.T) {
		bid, err := repo.GetHighestBidForItem(1)

		assert.NoError(t, err)
		assert.NotNil(t, bid)
		assert.Equal(t, uint(1), bid.ItemID)
		assert.Equal(t, 120.0, bid.Amount)
		assert.Equal(t, "active", bid.Status)
	})

	t.Run("no active bids for item", func(t *testing.T) {
		// Mark all bids for item 1 as outbid
		db.Model(&models.Bid{}).Where("item_id = ?", 1).Update("status", "outbid")
		
		bid, err := repo.GetHighestBidForItem(1)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("no bids for item", func(t *testing.T) {
		bid, err := repo.GetHighestBidForItem(999)

		assert.Error(t, err)
		assert.Nil(t, bid)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestBidRepository_UpdateBidStatus(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bid
	testBid := &models.Bid{
		ItemID:   1,
		BidderID: 2,
		Amount:   110.0,
		Status:   "active",
	}
	db.Create(testBid)

	t.Run("successful status update", func(t *testing.T) {
		err := repo.UpdateBidStatus(testBid.ID, "won")

		assert.NoError(t, err)
		
		// Verify the status update
		var updated models.Bid
		db.First(&updated, testBid.ID)
		assert.Equal(t, "won", updated.Status)
	})

	t.Run("update to outbid status", func(t *testing.T) {
		err := repo.UpdateBidStatus(testBid.ID, "outbid")

		assert.NoError(t, err)
		
		// Verify the status update
		var updated models.Bid
		db.First(&updated, testBid.ID)
		assert.Equal(t, "outbid", updated.Status)
	})

	t.Run("update non-existent bid", func(t *testing.T) {
		err := repo.UpdateBidStatus(9999, "won")

		// GORM doesn't return error for updating non-existent records
		assert.NoError(t, err)
	})
}

func TestBidRepository_MarkOutbidBids(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bids
	testBids := []models.Bid{
		{ItemID: 1, BidderID: 2, Amount: 110.0, Status: "active"},
		{ItemID: 1, BidderID: 3, Amount: 115.0, Status: "active"},
		{ItemID: 1, BidderID: 2, Amount: 120.0, Status: "active"}, // This will be the winning bid
		{ItemID: 2, BidderID: 3, Amount: 60.0, Status: "active"},  // Different item
	}

	for _, bid := range testBids {
		db.Create(&bid)
	}

	winningBidID := testBids[2].ID

	t.Run("successful mark outbid bids", func(t *testing.T) {
		err := repo.MarkOutbidBids(1, winningBidID)

		assert.NoError(t, err)
		
		// Verify that other bids for item 1 are marked as outbid
		var bids []models.Bid
		db.Where("item_id = ?", 1).Find(&bids)
		
		outbidCount := 0
		activeCount := 0
		
		for _, bid := range bids {
			if bid.ID == winningBidID {
				assert.Equal(t, "active", bid.Status)
				activeCount++
			} else {
				assert.Equal(t, "outbid", bid.Status)
				outbidCount++
			}
		}
		
		assert.Equal(t, 2, outbidCount)
		assert.Equal(t, 1, activeCount)
		
		// Verify that bids for other items are not affected
		var otherBids []models.Bid
		db.Where("item_id = ?", 2).Find(&otherBids)
		assert.Len(t, otherBids, 1)
		assert.Equal(t, "active", otherBids[0].Status)
	})

	t.Run("mark outbid for non-existent item", func(t *testing.T) {
		err := repo.MarkOutbidBids(999, 1)

		// GORM doesn't return error for updating non-existent records
		assert.NoError(t, err)
	})

	t.Run("mark outbid with non-existent winning bid", func(t *testing.T) {
		err := repo.MarkOutbidBids(1, 9999)

		assert.NoError(t, err)
		
		// All bids for item 1 should be marked as outbid
		var bids []models.Bid
		db.Where("item_id = ?", 1).Find(&bids)
		
		for _, bid := range bids {
			assert.Equal(t, "outbid", bid.Status)
		}
	})
}

func TestBidRepository_GetActiveBidsForItem(t *testing.T) {
	db, err := setupBidTestDB()
	assert.NoError(t, err)
	
	repo := NewBidRepository(db)

	// Create test bids
	testBids := []models.Bid{
		{ItemID: 1, BidderID: 2, Amount: 110.0, Status: "active"},
		{ItemID: 1, BidderID: 3, Amount: 120.0, Status: "active"},
		{ItemID: 1, BidderID: 2, Amount: 105.0, Status: "outbid"},
		{ItemID: 1, BidderID: 3, Amount: 115.0, Status: "won"},
		{ItemID: 2, BidderID: 3, Amount: 60.0, Status: "active"},
	}

	for _, bid := range testBids {
		db.Create(&bid)
	}

	t.Run("successful get active bids", func(t *testing.T) {
		bids, err := repo.GetActiveBidsForItem(1)

		assert.NoError(t, err)
		assert.Len(t, bids, 2)
		
		// Check that all returned bids are active
		for _, bid := range bids {
			assert.Equal(t, uint(1), bid.ItemID)
			assert.Equal(t, "active", bid.Status)
		}
		
		// Check that bids are ordered by amount DESC
		assert.True(t, bids[0].Amount >= bids[1].Amount)
	})

	t.Run("no active bids for item", func(t *testing.T) {
		// Mark all active bids for item 1 as outbid
		db.Model(&models.Bid{}).Where("item_id = ? AND status = ?", 1, "active").Update("status", "outbid")
		
		bids, err := repo.GetActiveBidsForItem(1)

		assert.NoError(t, err)
		assert.Empty(t, bids)
	})

	t.Run("no bids for item", func(t *testing.T) {
		bids, err := repo.GetActiveBidsForItem(999)

		assert.NoError(t, err)
		assert.Empty(t, bids)
	})

	t.Run("get active bids for different item", func(t *testing.T) {
		bids, err := repo.GetActiveBidsForItem(2)

		assert.NoError(t, err)
		assert.Len(t, bids, 1)
		assert.Equal(t, uint(2), bids[0].ItemID)
		assert.Equal(t, "active", bids[0].Status)
	})
}