package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"store-service/internal/models"
)

func TestGetRatingsReceived(t *testing.T) {
	db, err := setupBookingTestDB()
	if err != nil {
		t.Fatal(err)
	}
	repo := NewBookingRequestRepository(db)
	rating := func(v int) *int { return &v }

	// Items 1 and 3 are sold by user 1; item 2 by user 2 (setupBookingTestDB).
	bookings := []models.BookingRequest{
		{ItemID: 1, RequesterID: 2, Status: "completed", BuyerRating: rating(5), BuyerReview: "great seller"}, // user 1 rated as seller
		{ItemID: 2, RequesterID: 1, Status: "completed", SellerRating: rating(4), SellerReview: "good buyer"}, // user 1 rated as buyer
		{ItemID: 2, RequesterID: 1, Status: "completed", BuyerRating: rating(2)},                             // user 1 rated user 2: not received
		{ItemID: 1, RequesterID: 3, Status: "completed"},                                                     // no rating yet
	}
	for i := range bookings {
		if err := repo.Create(&bookings[i]); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.GetRatingsReceived(1)
	assert.NoError(t, err)
	ids := []uint{}
	for _, b := range got {
		ids = append(ids, b.ID)
	}
	assert.ElementsMatch(t, []uint{bookings[0].ID, bookings[1].ID}, ids)
	for _, b := range got {
		assert.NotNil(t, b.Item, "item is preloaded for its title and seller")
	}
}
