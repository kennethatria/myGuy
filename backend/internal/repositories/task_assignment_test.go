package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"myguy/internal/models"
)

func TestAssignIfOpenAndDeclinePending(t *testing.T) {
	ctx := context.Background()
	db, err := setupTestDB()
	mustNoError(t, err)
	tasks := NewGormTaskRepository(db)
	apps := NewGormApplicationRepository(db)

	task := &models.Task{Title: "Paint fence", CreatedBy: 1, Status: "open"}
	mustNoError(t, tasks.Create(ctx, task))
	accepted := &models.Application{TaskID: task.ID, ApplicantID: 2, ProposedFee: 80, Status: "pending"}
	other := &models.Application{TaskID: task.ID, ApplicantID: 3, ProposedFee: 90, Status: "pending"}
	alreadyDeclined := &models.Application{TaskID: task.ID, ApplicantID: 4, Status: "declined"}
	for _, a := range []*models.Application{accepted, other, alreadyDeclined} {
		mustNoError(t, apps.Create(ctx, a))
	}

	ok, err := tasks.AssignIfOpen(ctx, task.ID, 2)
	mustNoError(t, err)
	assert.True(t, ok)

	// A second acceptance must lose: the task is no longer open.
	ok, err = tasks.AssignIfOpen(ctx, task.ID, 3)
	mustNoError(t, err)
	assert.False(t, ok)

	got, err := tasks.GetByID(ctx, task.ID)
	mustNoError(t, err)
	assert.Equal(t, "in_progress", got.Status)
	assert.Equal(t, uint(2), *got.AssignedTo)

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

func TestExpireUnanswered(t *testing.T) {
	ctx := context.Background()
	db, err := setupTestDB()
	mustNoError(t, err)
	tasks := NewGormTaskRepository(db)
	apps := NewGormApplicationRepository(db)
	now := time.Now().UTC()

	stale := &models.Task{Title: "Nobody came", CreatedBy: 1, Status: "open", Deadline: now.Add(-time.Minute)}
	answered := &models.Task{Title: "Someone applied", CreatedBy: 1, Status: "open", Deadline: now.Add(-time.Minute)}
	fresh := &models.Task{Title: "Still live", CreatedBy: 1, Status: "open", Deadline: now.Add(time.Hour)}
	assigned := &models.Task{Title: "Underway", CreatedBy: 1, Status: "in_progress", Deadline: now.Add(-time.Minute)}
	for _, task := range []*models.Task{stale, answered, fresh, assigned} {
		mustNoError(t, tasks.Create(ctx, task))
	}
	// Even a declined application counts as a reaction.
	mustNoError(t, apps.Create(ctx, &models.Application{TaskID: answered.ID, ApplicantID: 2, Status: "declined"}))

	n, err := tasks.ExpireUnanswered(ctx, now)
	mustNoError(t, err)
	assert.Equal(t, int64(1), n)

	for task, want := range map[*models.Task]string{stale: "expired", answered: "open", fresh: "open", assigned: "in_progress"} {
		got, err := tasks.GetByID(ctx, task.ID)
		mustNoError(t, err)
		assert.Equal(t, want, got.Status, task.Title)
	}
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
