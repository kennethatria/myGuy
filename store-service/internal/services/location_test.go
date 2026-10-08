package services

import (
	"errors"
	"testing"

	"store-service/internal/models"
	"store-service/internal/proximity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func fp(v float64) *float64 { return &v }

func TestListingLocations(t *testing.T) {
	t.Run("saved on create when given, refused when invalid", func(t *testing.T) {
		service, itemRepo, _ := setupService()
		locator := &recordingLocator{}
		service.WithLocator(locator)
		itemRepo.On("Create", mock.Anything).Return(nil)

		_, err := service.CreateItem(1, models.CreateStoreItemRequest{Title: "Lamp", Description: "Brass desk lamp", Lat: fp(0.3476), Lng: fp(32.5842)})
		assert.NoError(t, err)
		_, err = service.CreateItem(1, models.CreateStoreItemRequest{Title: "Sofa", Description: "Grey"})
		assert.NoError(t, err)
		_, err = service.CreateItem(1, models.CreateStoreItemRequest{Title: "Fan", Description: "Standing", Lat: fp(0.35)})
		assert.ErrorIs(t, err, ErrInvalidLocation)
		assert.True(t, IsUserError(err))

		assert.Equal(t, []string{"item"}, locator.saved)
	})

	t.Run("nothing saved when the listing wasn't created", func(t *testing.T) {
		service, itemRepo, _ := setupService()
		locator := &recordingLocator{}
		service.WithLocator(locator)
		itemRepo.On("Create", mock.Anything).Return(errors.New("db down"))

		_, err := service.CreateItem(1, models.CreateStoreItemRequest{Title: "Lamp", Description: "Brass", Lat: fp(0.35), Lng: fp(32.58)})
		assert.Error(t, err)
		assert.Empty(t, locator.saved)
	})

	t.Run("deleted with the listing", func(t *testing.T) {
		service, itemRepo, bookingRepo := setupService()
		locator := &recordingLocator{}
		service.WithLocator(locator).WithLocator(nil) // nil keeps the current one
		itemRepo.On("GetByID", uint(1)).Return(&models.StoreItem{ID: 1, SellerID: 1, Status: "expired"}, nil)
		bookingRepo.On("GetAllByItemID", uint(1)).Return([]models.BookingRequest{}, nil)
		itemRepo.On("Delete", uint(1)).Return(nil)

		assert.NoError(t, service.DeleteItem(1, 1))
		assert.Equal(t, []string{"item"}, locator.deleted)
	})
}

func TestRequestLocations(t *testing.T) {
	service, requests, _ := setupRequestService()
	locator := &recordingLocator{}
	service.WithLocator(locator).WithLocator(nil)
	requests.On("Create", mock.Anything).Return(nil)
	requests.On("GetByID", uint(3)).Return(&models.ItemRequest{ID: 3, RequesterID: 1, Status: "active"}, nil)
	requests.On("Delete", uint(3)).Return(nil)

	_, err := service.CreateRequest(1, models.CreateItemRequestRequest{Title: "Printer wanted", Description: "Any laser", Lat: fp(-0.33), Lng: fp(31.73)})
	assert.NoError(t, err)
	_, err = service.CreateRequest(1, models.CreateItemRequestRequest{Title: "Desk wanted", Description: "Small"})
	assert.NoError(t, err)
	_, err = service.CreateRequest(1, models.CreateItemRequestRequest{Title: "Desk wanted", Description: "Small", Lat: fp(100), Lng: fp(0)})
	assert.ErrorIs(t, err, ErrInvalidLocation)
	assert.NoError(t, service.DeleteRequest(3, 1))

	assert.Equal(t, []string{"request"}, locator.saved)
	assert.Equal(t, []string{"request"}, locator.deleted)
}
