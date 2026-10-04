package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"myguy/internal/models"
)

func TestAssignIfOpenAndDeclinePending(t *testing.T) {
	ctx := context.Background()
	db, err := setupTestDB()
	mustNoError(t, err)
	tasks := NewGormTaskRepository(db)
	apps := NewGormApplicationRepository(db)

	task := &models.Task{Title: "Paint fence", CreatedBy: 1, Status: "open", Fee: 100}
	mustNoError(t, tasks.Create(ctx, task))
	accepted := &models.Application{TaskID: task.ID, ApplicantID: 2, ProposedFee: 80, Status: "pending"}
	other := &models.Application{TaskID: task.ID, ApplicantID: 3, ProposedFee: 90, Status: "pending"}
	alreadyDeclined := &models.Application{TaskID: task.ID, ApplicantID: 4, Status: "declined"}
	for _, a := range []*models.Application{accepted, other, alreadyDeclined} {
		mustNoError(t, apps.Create(ctx, a))
	}

	ok, err := tasks.AssignIfOpen(ctx, task.ID, 2, 80)
	mustNoError(t, err)
	assert.True(t, ok)

	// A second acceptance must lose: the task is no longer open.
	ok, err = tasks.AssignIfOpen(ctx, task.ID, 3, 90)
	mustNoError(t, err)
	assert.False(t, ok)

	got, err := tasks.GetByID(ctx, task.ID)
	mustNoError(t, err)
	assert.Equal(t, "in_progress", got.Status)
	assert.Equal(t, uint(2), *got.AssignedTo)
	assert.Equal(t, 80.0, got.Fee)

	mustNoError(t, apps.DeclinePending(ctx, task.ID, accepted.ID))
	list, err := apps.ListByTask(ctx, task.ID)
	mustNoError(t, err)
	status := map[uint]string{}
	for _, a := range list {
		status[a.ApplicantID] = a.Status
	}
	assert.Equal(t, "pending", status[2], "accepted application is untouched by the bulk decline")
	assert.Equal(t, "declined", status[3])
	assert.Equal(t, "declined", status[4])
}

func TestDeclinePendingWithNoExceptionDeclinesAll(t *testing.T) {
	ctx := context.Background()
	db, err := setupTestDB()
	mustNoError(t, err)
	apps := NewGormApplicationRepository(db)

	for _, applicant := range []uint{2, 3} {
		mustNoError(t, apps.Create(ctx, &models.Application{TaskID: 1, ApplicantID: applicant, Status: "pending"}))
	}
	mustNoError(t, apps.Create(ctx, &models.Application{TaskID: 2, ApplicantID: 4, Status: "pending"}))

	mustNoError(t, apps.DeclinePending(ctx, 1, 0))

	for _, task := range []uint{1, 2} {
		list, err := apps.ListByTask(ctx, task)
		mustNoError(t, err)
		for _, a := range list {
			want := map[uint]string{1: "declined", 2: "pending"}[task]
			assert.Equal(t, want, a.Status, "task %d applicant %d", task, a.ApplicantID)
		}
	}
}
