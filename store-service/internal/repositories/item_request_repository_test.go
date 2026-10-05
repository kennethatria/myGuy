package repositories

import (
	"store-service/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemRequestRepository(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)
	repo := NewItemRequestRepository(db)
	items := NewStoreItemRepository(db)

	soon := time.Now().Add(time.Hour)
	later := time.Now().Add(20 * time.Hour)
	printer := &models.ItemRequest{Title: "Printer wanted", Description: "Any laser printer", RequesterID: 1, Status: "active", Deadline: &later}
	desk := &models.ItemRequest{Title: "Desk wanted", Description: "Small desk", RequesterID: 2, Status: "active", Deadline: &soon}
	gone := &models.ItemRequest{Title: "Chair wanted", Description: "Any chair", RequesterID: 2, Status: "expired", Deadline: &soon}
	for _, r := range []*models.ItemRequest{printer, desk, gone} {
		require.NoError(t, repo.Create(r))
	}

	t.Run("get by id loads the requester and counts live offers", func(t *testing.T) {
		require.NoError(t, items.Create(&models.StoreItem{Title: "HP printer", SellerID: 2, PriceType: "fixed", Status: "active", RequestID: &printer.ID}))
		require.NoError(t, items.Create(&models.StoreItem{Title: "Old printer", SellerID: 3, PriceType: "fixed", Status: "sold", RequestID: &printer.ID}))

		got, err := repo.GetByID(printer.ID)

		require.NoError(t, err)
		assert.Equal(t, "user1", got.Requester.Username)
		assert.Equal(t, 1, got.OfferCount)
	})

	t.Run("board lists live requests, filters and sorts", func(t *testing.T) {
		all, total, err := repo.GetAll(models.ItemRequestFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, all, 2)

		others, _, err := repo.GetAll(models.ItemRequestFilter{ExcludeRequesterID: 1, SortBy: "deadline", SortOrder: "asc"})
		require.NoError(t, err)
		if assert.Len(t, others, 1) {
			assert.Equal(t, "Desk wanted", others[0].Title)
		}

		found, _, err := repo.GetAll(models.ItemRequestFilter{Search: "LASER"})
		require.NoError(t, err)
		if assert.Len(t, found, 1) {
			assert.Equal(t, "Printer wanted", found[0].Title)
		}

		expired, _, err := repo.GetAll(models.ItemRequestFilter{Status: "expired", Page: 1, PerPage: 5})
		require.NoError(t, err)
		assert.Len(t, expired, 1)
	})

	t.Run("requester's own requests, newest first", func(t *testing.T) {
		mine, err := repo.GetByRequesterID(2)
		require.NoError(t, err)
		assert.Len(t, mine, 2)
	})

	t.Run("expire only unanswered requests past their deadline", func(t *testing.T) {
		past := time.Now().Add(-time.Minute)
		printer.Deadline, desk.Deadline = &past, &past
		require.NoError(t, repo.Update(printer))
		require.NoError(t, repo.Update(desk))

		n, err := repo.ExpireUnanswered(time.Now())

		require.NoError(t, err)
		assert.Equal(t, int64(1), n)
		got, _ := repo.GetByID(printer.ID)
		assert.Equal(t, "active", got.Status, "a listing answered it, so it stays up")
		got, _ = repo.GetByID(desk.ID)
		assert.Equal(t, "expired", got.Status)
	})

	t.Run("mark fulfilled only once", func(t *testing.T) {
		closed, err := repo.MarkFulfilled(printer.ID, 7)
		require.NoError(t, err)
		assert.True(t, closed)
		got, _ := repo.GetByID(printer.ID)
		assert.Equal(t, "fulfilled", got.Status)
		assert.Equal(t, uint(7), *got.FulfilledItemID)

		again, err := repo.MarkFulfilled(printer.ID, 8)
		require.NoError(t, err)
		assert.False(t, again)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, repo.Delete(gone.ID))
		_, err := repo.GetByID(gone.ID)
		assert.Error(t, err)
	})

	t.Run("listings filter by request and load it", func(t *testing.T) {
		list, _, err := items.GetAll(models.StoreItemFilter{RequestID: printer.ID})
		require.NoError(t, err)
		if assert.Len(t, list, 1) {
			assert.Equal(t, "HP printer", list[0].Title)
			got, err := items.GetByID(list[0].ID)
			require.NoError(t, err)
			assert.Equal(t, "Printer wanted", got.Request.Title)
		}
	})
}
