package metrics

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"myguy/internal/models"
)

func TestHandlerReportsCounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Task{}, &models.Application{}); err != nil {
		t.Fatal(err)
	}
	users := []models.User{
		{Username: "a", Email: "a@example.com", Password: "-"},
		{Username: "b", Email: "b@example.com", Password: "-"},
		{Username: "c", Email: "c@example.com", Password: "-"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * 24 * time.Hour)
	tasks := []models.Task{
		{Title: "new", Status: "open", CreatedBy: users[0].ID},
		{Title: "old", Status: "completed", CreatedBy: users[2].ID, CreatedAt: old},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Application{TaskID: tasks[0].ID, ApplicantID: users[1].ID, Status: "pending"}).Error; err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	Handler(db).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)

	for _, want := range []string{
		"myguy_accounts 3",
		"myguy_gig_active_accounts_7d 2",
		`myguy_gigs{status="open"} 1`,
		`myguy_gigs{status="completed"} 1`,
		`myguy_applications{status="pending"} 1`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(string(body), "example.com") {
		t.Error("metrics must not contain emails")
	}
}

func TestHandlerKeepsServingWhenAQueryFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Only users exists: the gig queries fail, the account count still works.
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	Handler(db).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(rec.Body.String(), "myguy_accounts 0") {
		t.Errorf("account count missing:\n%s", rec.Body.String())
	}
}
