package repositories

import (
	"testing"
	"time"

	"store-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIDsAndGetByIDs(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)
	items := NewStoreItemRepository(db)
	requests := NewItemRequestRepository(db)

	base := time.Now().Add(-time.Hour)
	var ids []uint
	for i, status := range []string{"active", "sold", "active", "active"} {
		item := &models.StoreItem{Title: "Lamp", SellerID: 2, PriceType: "fixed", Status: status, CreatedAt: base.Add(time.Duration(i) * time.Minute)}
		require.NoError(t, items.Create(item))
		ids = append(ids, item.ID)
	}
	mine := &models.StoreItem{Title: "Mine", SellerID: 1, PriceType: "fixed", Status: "active", CreatedAt: base.Add(time.Hour)}
	require.NoError(t, items.Create(mine))

	got, err := items.ListIDs(models.StoreItemFilter{Status: "active", ExcludeSellerID: 1, Page: 2, PerPage: 1})
	require.NoError(t, err)
	assert.Equal(t, []uint{ids[3], ids[2], ids[0]}, got, "all active matches, newest first, no paging")

	loaded, err := items.GetByIDs([]uint{ids[0], mine.ID})
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	none, err := items.GetByIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, none)

	first := &models.ItemRequest{Title: "Printer", Description: "Any", RequesterID: 2, Status: "active", CreatedAt: base}
	second := &models.ItemRequest{Title: "Desk", Description: "Any", RequesterID: 2, Status: "active", CreatedAt: base.Add(time.Minute)}
	closed := &models.ItemRequest{Title: "Fan", Description: "Any", RequesterID: 2, Status: "fulfilled", CreatedAt: base}
	for _, r := range []*models.ItemRequest{first, second, closed} {
		require.NoError(t, requests.Create(r))
	}
	reqIDs, err := requests.ListIDs(models.ItemRequestFilter{ExcludeRequesterID: 1})
	require.NoError(t, err)
	assert.Equal(t, []uint{second.ID, first.ID}, reqIDs)

	loadedReqs, err := requests.GetByIDs(reqIDs)
	require.NoError(t, err)
	assert.Len(t, loadedReqs, 2)
	assert.NotNil(t, loadedReqs[0].Requester)
	noReqs, err := requests.GetByIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, noReqs)
}
