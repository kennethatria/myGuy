package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

const (
	loginCodeTTL         = 10 * time.Minute
	loginCodeMaxAttempts = 5 // wrong guesses before a code is dead
	loginCodeMaxPerHour  = 5 // codes that may be requested per email per hour
	usernameMaxLength    = 20
	usernameMaxTries     = 10
)

var (
	ErrInvalidCode      = errors.New("invalid or expired code")
	ErrTooManyRequests  = errors.New("too many code requests, try again later")
	ErrFullNameRequired = errors.New("full name is required")
)

var usernameUnsafeChars = regexp.MustCompile(`[^a-z0-9]`)

// CodeSender delivers a login code to an email address.
type CodeSender interface {
	SendLoginCode(ctx context.Context, email, code string) error
}

// AuthService implements passwordless sign-in with emailed one-time codes.
// The same flow signs existing users in and lets new users sign up.
type AuthService struct {
	userRepo repositories.UserRepository
	codeRepo repositories.LoginCodeRepository
	sender   CodeSender
	secret   []byte
	now      func() time.Time
}

func NewAuthService(userRepo repositories.UserRepository, codeRepo repositories.LoginCodeRepository, sender CodeSender, secret string) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		codeRepo: codeRepo,
		sender:   sender,
		secret:   []byte(secret),
		now:      time.Now,
	}
}

// VerifyResult is the outcome of a correct code: either an existing user, or
// NewAccount for an email with no account yet (the caller then completes
// sign-up for Email).
type VerifyResult struct {
	Email      string
	User       *models.UserResponse
	NewAccount bool
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// RequestCode emails a fresh code to email, replacing any outstanding one.
// It behaves the same whether or not an account exists.
func (s *AuthService) RequestCode(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	now := s.now()

	recent, err := s.codeRepo.CountSince(ctx, email, now.Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= loginCodeMaxPerHour {
		return ErrTooManyRequests
	}

	code, err := generateCode()
	if err != nil {
		return err
	}
	if err := s.codeRepo.InvalidateActive(ctx, email, now); err != nil {
		return err
	}
	if err := s.codeRepo.Create(ctx, &models.LoginCode{
		Email:     email,
		CodeHash:  s.hashCode(email, code),
		ExpiresAt: now.Add(loginCodeTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}
	return s.sender.SendLoginCode(ctx, email, code)
}

// VerifyCode checks code against the latest active code for email and
// consumes it on success.
func (s *AuthService) VerifyCode(ctx context.Context, email, code string) (*VerifyResult, error) {
	email = normalizeEmail(email)

	loginCode, err := s.codeRepo.LatestActive(ctx, email, s.now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCode
	}
	if err != nil {
		return nil, err
	}
	if loginCode.Attempts >= loginCodeMaxAttempts {
		return nil, ErrInvalidCode
	}

	expected := s.hashCode(email, strings.TrimSpace(code))
	if !hmac.Equal([]byte(loginCode.CodeHash), []byte(expected)) {
		if err := s.codeRepo.IncrementAttempts(ctx, loginCode.ID); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCode
	}

	consumed, err := s.codeRepo.MarkConsumed(ctx, loginCode.ID, s.now())
	if err != nil {
		return nil, err
	}
	if !consumed {
		return nil, ErrInvalidCode
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &VerifyResult{Email: email, NewAccount: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return &VerifyResult{Email: email, User: user.ToResponse()}, nil
}

// CompleteSignup creates the account for an email whose code was verified.
func (s *AuthService) CompleteSignup(ctx context.Context, email, fullName string) (*models.UserResponse, error) {
	email = normalizeEmail(email)
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return nil, ErrFullNameRequired
	}

	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	username, err := s.uniqueUsername(ctx, email)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: username,
		Email:    email,
		FullName: fullName,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}

// uniqueUsername derives a username from the email's local part, adding a
// random numeric suffix when the plain form is taken.
func (s *AuthService) uniqueUsername(ctx context.Context, email string) (string, error) {
	base := usernameUnsafeChars.ReplaceAllString(strings.SplitN(email, "@", 2)[0], "")
	if base == "" {
		base = "user"
	}
	if len(base) > usernameMaxLength-4 {
		base = base[:usernameMaxLength-4]
	}

	candidate := base
	for i := 0; i < usernameMaxTries; i++ {
		_, err := s.userRepo.GetByUsername(ctx, candidate)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		suffix, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			return "", err
		}
		candidate = fmt.Sprintf("%s%04d", base, suffix.Int64())
	}
	return "", ErrUsernameExists
}

func (s *AuthService) hashCode(email, code string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(email + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
