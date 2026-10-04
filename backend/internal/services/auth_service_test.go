package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

// fakeSender records the last code "emailed" to each address.
type fakeSender struct {
	codes map[string]string
	err   error
}

func (f *fakeSender) SendLoginCode(_ context.Context, email, code string) error {
	if f.err != nil {
		return f.err
	}
	f.codes[email] = code
	return nil
}

// setupAuthService uses real repositories on in-memory SQLite so expiry,
// attempt counting and single use are exercised against actual queries.
func setupAuthService(t *testing.T) (*AuthService, *fakeSender, *repositories.GormUserRepository, *time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.LoginCode{}); err != nil {
		t.Fatal(err)
	}

	userRepo := repositories.NewGormUserRepository(db)
	sender := &fakeSender{codes: map[string]string{}}
	service := NewAuthService(userRepo, repositories.NewGormLoginCodeRepository(db), sender, "test-secret")

	clock := time.Now()
	service.now = func() time.Time { return clock }
	return service, sender, userRepo, &clock
}

func wrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestAuthService_RequestAndVerify(t *testing.T) {
	ctx := context.Background()

	t.Run("existing user signs in, email matched case-insensitively", func(t *testing.T) {
		service, sender, userRepo, _ := setupAuthService(t)
		assert.NoError(t, userRepo.Create(ctx, &models.User{Username: "jane", Email: "Jane@Example.com", FullName: "Jane"}))

		assert.NoError(t, service.RequestCode(ctx, "  jane@example.COM "))
		code := sender.codes["jane@example.com"]
		assert.Len(t, code, 6)

		result, err := service.VerifyCode(ctx, "JANE@example.com", code)
		assert.NoError(t, err)
		assert.False(t, result.NewAccount)
		assert.Equal(t, "jane", result.User.Username)
	})

	t.Run("unknown email is reported as a new account", func(t *testing.T) {
		service, sender, _, _ := setupAuthService(t)
		assert.NoError(t, service.RequestCode(ctx, "new@example.com"))

		result, err := service.VerifyCode(ctx, "new@example.com", sender.codes["new@example.com"])
		assert.NoError(t, err)
		assert.True(t, result.NewAccount)
		assert.Equal(t, "new@example.com", result.Email)
		assert.Nil(t, result.User)
	})

	t.Run("code works only once", func(t *testing.T) {
		service, sender, _, _ := setupAuthService(t)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
		code := sender.codes["a@example.com"]

		_, err := service.VerifyCode(ctx, "a@example.com", code)
		assert.NoError(t, err)
		_, err = service.VerifyCode(ctx, "a@example.com", code)
		assert.ErrorIs(t, err, ErrInvalidCode)
	})

	t.Run("code expires after 10 minutes", func(t *testing.T) {
		service, sender, _, clock := setupAuthService(t)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))

		*clock = clock.Add(loginCodeTTL + time.Second)
		_, err := service.VerifyCode(ctx, "a@example.com", sender.codes["a@example.com"])
		assert.ErrorIs(t, err, ErrInvalidCode)
	})

	t.Run("code is dead after too many wrong guesses, even the right one", func(t *testing.T) {
		service, sender, _, _ := setupAuthService(t)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
		code := sender.codes["a@example.com"]

		for i := 0; i < loginCodeMaxAttempts; i++ {
			_, err := service.VerifyCode(ctx, "a@example.com", wrongCode(code))
			assert.ErrorIs(t, err, ErrInvalidCode)
		}
		_, err := service.VerifyCode(ctx, "a@example.com", code)
		assert.ErrorIs(t, err, ErrInvalidCode)
	})

	t.Run("requesting a new code invalidates the previous one", func(t *testing.T) {
		service, sender, _, clock := setupAuthService(t)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
		first := sender.codes["a@example.com"]

		*clock = clock.Add(time.Second)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
		second := sender.codes["a@example.com"]

		if first != second {
			_, err := service.VerifyCode(ctx, "a@example.com", first)
			assert.ErrorIs(t, err, ErrInvalidCode)
		}
		_, err := service.VerifyCode(ctx, "a@example.com", second)
		assert.NoError(t, err)
	})

	t.Run("code for one email does not work for another", func(t *testing.T) {
		service, sender, _, _ := setupAuthService(t)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
		assert.NoError(t, service.RequestCode(ctx, "b@example.com"))

		_, err := service.VerifyCode(ctx, "b@example.com", sender.codes["a@example.com"])
		if sender.codes["a@example.com"] != sender.codes["b@example.com"] {
			assert.ErrorIs(t, err, ErrInvalidCode)
		}
	})

	t.Run("rate limited after 5 requests per hour", func(t *testing.T) {
		service, _, _, clock := setupAuthService(t)
		for i := 0; i < loginCodeMaxPerHour; i++ {
			*clock = clock.Add(time.Second)
			assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
		}
		*clock = clock.Add(time.Second)
		assert.ErrorIs(t, service.RequestCode(ctx, "a@example.com"), ErrTooManyRequests)

		*clock = clock.Add(time.Hour)
		assert.NoError(t, service.RequestCode(ctx, "a@example.com"))
	})

	t.Run("send failure is returned", func(t *testing.T) {
		service, sender, _, _ := setupAuthService(t)
		sender.err = errors.New("smtp down")
		assert.EqualError(t, service.RequestCode(ctx, "a@example.com"), "smtp down")
	})
}

func TestAuthService_CompleteSignup(t *testing.T) {
	ctx := context.Background()

	t.Run("creates user with username from email", func(t *testing.T) {
		service, _, userRepo, _ := setupAuthService(t)

		user, err := service.CompleteSignup(ctx, "John.Doe+tasks@Example.com", "  John Doe ")
		assert.NoError(t, err)
		assert.Equal(t, "johndoetasks", user.Username)
		assert.Equal(t, "john.doe+tasks@example.com", user.Email)
		assert.Equal(t, "John Doe", user.FullName)

		stored, err := userRepo.GetByEmail(ctx, "john.doe+tasks@example.com")
		assert.NoError(t, err)
		assert.Equal(t, "", stored.Password)
	})

	t.Run("adds a suffix when the username is taken", func(t *testing.T) {
		service, _, userRepo, _ := setupAuthService(t)
		assert.NoError(t, userRepo.Create(ctx, &models.User{Username: "sam", Email: "sam@other.com"}))

		user, err := service.CompleteSignup(ctx, "sam@example.com", "Sam")
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(user.Username, "sam"))
		assert.Len(t, user.Username, len("sam")+4)
	})

	t.Run("rejects an email that already has an account", func(t *testing.T) {
		service, _, userRepo, _ := setupAuthService(t)
		assert.NoError(t, userRepo.Create(ctx, &models.User{Username: "sam", Email: "sam@example.com"}))

		_, err := service.CompleteSignup(ctx, "SAM@example.com", "Sam")
		assert.ErrorIs(t, err, ErrEmailExists)
	})

	t.Run("requires a full name", func(t *testing.T) {
		service, _, _, _ := setupAuthService(t)
		_, err := service.CompleteSignup(ctx, "a@example.com", "   ")
		assert.ErrorIs(t, err, ErrFullNameRequired)
	})
}
