package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"store-service/internal/models"
)

func TestHandlerReportsCounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.StoreItem{}, &models.ItemRequest{}, &models.BookingRequest{}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * 24 * time.Hour)
	items := []models.StoreItem{
		{Title: "lamp", SellerID: 1, Status: "active"},
		{Title: "chair", SellerID: 2, Status: "sold", CreatedAt: old},
		{Title: "gone", SellerID: 3, Status: "active"},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&items[2]).Error; err != nil { // removed listings don't count
		t.Fatal(err)
	}
	if err := db.Create(&models.ItemRequest{Title: "bike", RequesterID: 4, Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.BookingRequest{ItemID: items[0].ID, RequesterID: 4, Status: "pending"}).Error; err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	Handler(db).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		"myguy_store_active_accounts_7d 2",
		`myguy_listings{status="active"} 1`,
		`myguy_listings{status="sold"} 1`,
		`myguy_requests{status="active"} 1`,
		`myguy_bookings{status="pending"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}
