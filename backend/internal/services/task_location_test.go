package services

import (
	"context"
	"errors"
	"testing"

	"myguy/internal/models"
	"myguy/internal/proximity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// recordingLocator captures location calls instead of making them.
type recordingLocator struct {
	saved   []string
	deleted []string
}

func (r *recordingLocator) Save(kind string, id uint, at proximity.Location) {
	r.saved = append(r.saved, kind)
}

func (r *recordingLocator) Delete(kind string, id uint) {
	r.deleted = append(r.deleted, kind)
}

func TestTaskLocations(t *testing.T) {
	ctx := context.Background()
	at := &proximity.Location{Lat: 0.35, Lng: 32.585}

	t.Run("a new gig's location is saved when given", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		locator := &recordingLocator{}
		service.WithLocator(locator)
		taskRepo.On("Create", ctx, mock.Anything).Return(nil)

		_, err := service.CreateTask(ctx, CreateTaskInput{Title: "Paint my fence", Description: "White paint", CreatedBy: 1, Location: at})
		assert.NoError(t, err)
		_, err = service.CreateTask(ctx, CreateTaskInput{Title: "Mow my lawn", Description: "Small garden", CreatedBy: 1})
		assert.NoError(t, err)

		assert.Equal(t, []string{"task"}, locator.saved, "only the gig that shared one")
	})

	t.Run("no location saved when the gig wasn't created", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		locator := &recordingLocator{}
		service.WithLocator(locator)
		taskRepo.On("Create", ctx, mock.Anything).Return(errors.New("db down"))

		_, err := service.CreateTask(ctx, CreateTaskInput{Title: "Paint my fence", Description: "White paint", CreatedBy: 1, Location: at})
		assert.Error(t, err)
		assert.Empty(t, locator.saved)
	})

	t.Run("deleting a gig deletes its location", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		locator := &recordingLocator{}
		service.WithLocator(locator)
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1}, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)
		taskRepo.On("Delete", ctx, uint(1)).Return(nil)

		assert.NoError(t, service.DeleteTask(ctx, 1, 1))
		assert.Equal(t, []string{"task"}, locator.deleted)
	})

	t.Run("SaveLocation ignores a missing location; nil locator is a no-op", func(t *testing.T) {
		service, _, _ := setupTaskService()
		service.WithLocator(nil)
		service.SaveLocation(1, nil)
		service.SaveLocation(1, at) // noop locator: nothing to assert, mustn't panic

		locator := &recordingLocator{}
		service.WithLocator(locator)
		service.SaveLocation(1, nil)
		assert.Empty(t, locator.saved)
	})
}
