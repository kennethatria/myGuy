package repositories

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"myguy/internal/models"
)

// Runs only against a real PostgreSQL (SQLite does not enforce foreign keys):
//
//	TEST_POSTGRES_DSN="postgres://user:pass@localhost:5432/db?sslmode=disable" go test ./internal/repositories/ -run Postgres
func setupPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	mustNoError(t, err)
	// Same models and order as cmd/api/main.go, so constraints match production.
	mustNoError(t, db.AutoMigrate(&models.User{}, &models.Task{}, &models.Application{}, &models.Review{}, &models.LoginCode{}))
	t.Cleanup(func() {
		db.Exec("TRUNCATE reviews, applications, tasks, users RESTART IDENTITY CASCADE")
	})
	return db
}

func TestPostgres_DeleteTaskWithApplications(t *testing.T) {
	ctx := context.Background()
	db := setupPostgres(t)

	owner := &models.User{Username: "owner", Email: "owner@example.com"}
	applicant := &models.User{Username: "applicant", Email: "applicant@example.com"}
	mustNoError(t, db.Create(owner).Error)
	mustNoError(t, db.Create(applicant).Error)

	tasks := NewGormTaskRepository(db)
	task := &models.Task{Title: "Fix sink", CreatedBy: owner.ID, Status: "open"}
	mustNoError(t, tasks.Create(ctx, task))
	mustNoError(t, NewGormApplicationRepository(db).Create(ctx, &models.Application{TaskID: task.ID, ApplicantID: applicant.ID, Status: "pending"}))

	assert.NoError(t, tasks.Delete(ctx, task.ID))

	var remaining int64
	db.Model(&models.Application{}).Where("task_id = ?", task.ID).Count(&remaining)
	assert.Zero(t, remaining, "applications are removed with their task")
}

func TestPostgres_RatingsFollowReviews(t *testing.T) {
	ctx := context.Background()
	db := setupPostgres(t)
	users := NewGormUserRepository(db)

	owner := &models.User{Username: "owner", Email: "owner@example.com"}
	worker := &models.User{Username: "worker", Email: "worker@example.com"}
	mustNoError(t, users.Create(ctx, owner))
	mustNoError(t, users.Create(ctx, worker))
	for i, rating := range []int{5, 2} {
		task := &models.Task{Title: "job", CreatedBy: owner.ID, AssignedTo: &worker.ID, Status: "completed"}
		mustNoError(t, db.Create(task).Error)
		mustNoError(t, db.Create(&models.Review{TaskID: task.ID, ReviewerID: owner.ID, ReviewedUserID: worker.ID, Rating: rating, Comment: string(rune('a' + i))}).Error)
	}

	mustNoError(t, users.RecalculateAllRatings(ctx))
	got, _ := users.GetByID(ctx, worker.ID)
	assert.Equal(t, 3.5, got.AverageRating)

	mustNoError(t, users.UpdateRating(ctx, worker.ID))
	got, _ = users.GetByID(ctx, worker.ID)
	assert.Equal(t, 3.5, got.AverageRating)
}
