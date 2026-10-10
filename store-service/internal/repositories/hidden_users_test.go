package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"store-service/internal/models"
)

// Blocked accounts' listings and requests stay off the boards
func TestHiddenUsersLeaveTheBoards(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)
	items := NewStoreItemRepository(db)
	requests := NewItemRequestRepository(db)
	for _, seller := range []uint{1, 2, 3} {
		require.NoError(t, items.Create(&models.StoreItem{Title: "Lamp", SellerID: seller, PriceType: "fixed", Status: "active"}))
		require.NoError(t, requests.Create(&models.ItemRequest{Title: "Desk wanted", RequesterID: seller, Status: "active"}))
	}

	listed, total, err := items.GetAll(models.StoreItemFilter{HiddenUserIDs: []uint{2, 3}, Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, uint(1), listed[0].SellerID)

	wanted, total, err := requests.GetAll(models.ItemRequestFilter{HiddenUserIDs: []uint{1}, Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	for _, r := range wanted {
		assert.NotEqual(t, uint(1), r.RequesterID)
	}

	// Nobody hidden: everything
	_, total, err = items.GetAll(models.StoreItemFilter{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
}
