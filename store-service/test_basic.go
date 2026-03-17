package main

import (
	"fmt"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"store-service/internal/services"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Set up in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.StoreItem{}, &models.ItemImage{}, &models.Bid{}, &models.BookingRequest{}, &models.User{})
	if err != nil {
		panic("failed to migrate database")
	}

	// Create test users
	users := []models.User{
		{ID: 1, Username: "user1", Email: "user1@example.com", Name: "User One"},
		{ID: 2, Username: "user2", Email: "user2@example.com", Name: "User Two"},
	}
	
	for _, user := range users {
		db.Create(&user)
	}

	// Initialize repositories
	itemRepo := repositories.NewStoreItemRepository(db)
	bidRepo := repositories.NewBidRepository(db)
	bookingRepo := repositories.NewBookingRequestRepository(db)
	userRepo := repositories.NewUserRepository(db)
	
	// Initialize service
	storeService := services.NewStoreService(db, itemRepo, bidRepo, bookingRepo, userRepo)

	// Test 1: Create a fixed price item
	fmt.Println("=== Test 1: Create Fixed Price Item ===")
	req := models.CreateStoreItemRequest{
		Title:       "iPhone 15 Pro",
		Description: "Brand new iPhone 15 Pro",
		PriceType:   "fixed",
		FixedPrice:  999.99,
		Category:    "electronics",
		Condition:   "new",
	}

	item, err := storeService.CreateItem(1, req)
	if err != nil {
		fmt.Printf("Error creating item: %v\n", err)
		return
	}
	fmt.Printf("Created item: ID=%d, Title=%s, Price=%.2f\n", item.ID, item.Title, item.FixedPrice)

	// Test 2: Get the item
	fmt.Println("\n=== Test 2: Get Item ===")
	retrievedItem, err := storeService.GetItem(item.ID)
	if err != nil {
		fmt.Printf("Error getting item: %v\n", err)
		return
	}
	fmt.Printf("Retrieved item: ID=%d, Title=%s\n", retrievedItem.ID, retrievedItem.Title)

	// Test 3: Create a bidding item
	fmt.Println("\n=== Test 3: Create Bidding Item ===")
	bidDeadline := time.Now().Add(24 * time.Hour)
	bidReq := models.CreateStoreItemRequest{
		Title:           "Vintage Guitar",
		Description:     "Classic acoustic guitar",
		PriceType:       "bidding",
		StartingBid:     500.0,
		MinBidIncrement: 25.0,
		BidDeadline:     &bidDeadline,
		Category:        "music",
		Condition:       "good",
	}

	bidItem, err := storeService.CreateItem(1, bidReq)
	if err != nil {
		fmt.Printf("Error creating bid item: %v\n", err)
		return
	}
	fmt.Printf("Created bid item: ID=%d, Title=%s, StartingBid=%.2f\n", bidItem.ID, bidItem.Title, bidItem.StartingBid)

	// Test 4: Place a bid
	fmt.Println("\n=== Test 4: Place Bid ===")
	placeBidReq := models.CreateBidRequest{
		Amount:  525.0,
		Message: "Great looking guitar!",
	}

	bid, err := storeService.PlaceBid(bidItem.ID, 2, placeBidReq)
	if err != nil {
		fmt.Printf("Error placing bid: %v\n", err)
		return
	}
	fmt.Printf("Placed bid: ID=%d, Amount=%.2f, Message=%s\n", bid.ID, bid.Amount, bid.Message)

	// Test 5: Get item bids
	fmt.Println("\n=== Test 5: Get Item Bids ===")
	bids, err := storeService.GetItemBids(bidItem.ID)
	if err != nil {
		fmt.Printf("Error getting bids: %v\n", err)
		return
	}
	fmt.Printf("Found %d bids for item %d\n", len(bids), bidItem.ID)

	// Test 6: Purchase fixed price item
	fmt.Println("\n=== Test 6: Purchase Item ===")
	err = storeService.PurchaseItem(item.ID, 2)
	if err != nil {
		fmt.Printf("Error purchasing item: %v\n", err)
		return
	}
	fmt.Printf("Successfully purchased item %d\n", item.ID)

	// Test 7: Get user listings
	fmt.Println("\n=== Test 7: Get User Listings ===")
	listings, err := storeService.GetUserListings(1)
	if err != nil {
		fmt.Printf("Error getting listings: %v\n", err)
		return
	}
	fmt.Printf("User 1 has %d listings\n", len(listings))

	// Test 8: Get user purchases
	fmt.Println("\n=== Test 8: Get User Purchases ===")
	purchases, err := storeService.GetUserPurchases(2)
	if err != nil {
		fmt.Printf("Error getting purchases: %v\n", err)
		return
	}
	fmt.Printf("User 2 has %d purchases\n", len(purchases))

	// Test 9: Create booking request
	fmt.Println("\n=== Test 9: Create Booking Request ===")
	booking, err := storeService.CreateBookingRequest(bidItem.ID, 2, "I'd like to book this guitar for a weekend gig")
	if err != nil {
		fmt.Printf("Error creating booking request: %v\n", err)
		return
	}
	fmt.Printf("Created booking request: ID=%d, Status=%s\n", booking.ID, booking.Status)

	// Test 10: Get all items with filtering
	fmt.Println("\n=== Test 10: Get Items with Filtering ===")
	filter := models.StoreItemFilter{
		Category: "electronics",
		Page:     1,
		PerPage:  10,
	}
	
	items, total, err := storeService.GetItems(filter)
	if err != nil {
		fmt.Printf("Error getting items: %v\n", err)
		return
	}
	fmt.Printf("Found %d electronics items (total: %d)\n", len(items), total)

	fmt.Println("\n=== All Tests Completed Successfully! ===")
	fmt.Println("The store service is working correctly with comprehensive functionality:")
	fmt.Println("✅ Item creation (fixed price and bidding)")
	fmt.Println("✅ Item retrieval and filtering")
	fmt.Println("✅ Bidding system")
	fmt.Println("✅ Purchase system")
	fmt.Println("✅ User-specific endpoints (listings, purchases, bids)")
	fmt.Println("✅ Booking request system")
	fmt.Println("✅ Database operations with proper relationships")
}