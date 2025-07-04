package repositories

import (
	"store-service/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{})
	assert.NoError(t, err)

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err := repo.Create(user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)

	// Verify user was created
	var found models.User
	err = db.First(&found, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.Name, found.Name)
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Create user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := repo.Create(user)
	assert.NoError(t, err)

	// Get user by ID
	found, err := repo.GetByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.Name, found.Name)

	// Test non-existent user
	_, err = repo.GetByID(999)
	assert.Error(t, err)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Create user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := repo.Create(user)
	assert.NoError(t, err)

	// Get user by email
	found, err := repo.GetByEmail(user.Email)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.Name, found.Name)

	// Test non-existent email
	_, err = repo.GetByEmail("nonexistent@example.com")
	assert.Error(t, err)
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Create user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := repo.Create(user)
	assert.NoError(t, err)

	// Get user by username
	found, err := repo.GetByUsername(user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.Name, found.Name)

	// Test non-existent username
	_, err = repo.GetByUsername("nonexistent")
	assert.Error(t, err)
}

func TestUserRepository_Update(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Create user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := repo.Create(user)
	assert.NoError(t, err)

	// Update user
	user.Name = "Updated Name"
	user.Email = "updated@example.com"
	err = repo.Update(user)
	assert.NoError(t, err)

	// Verify update
	found, err := repo.GetByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "updated@example.com", found.Email)
	assert.Equal(t, "testuser", found.Username) // Should remain unchanged
}

func TestUserRepository_UpsertFromJWT_CreateNew(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Test creating new user from JWT
	user, err := repo.UpsertFromJWT(1, "newuser", "new@example.com", "New User")
	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New User", user.Name)

	// Verify user was created in database
	var found models.User
	err = db.First(&found, 1).Error
	assert.NoError(t, err)
	assert.Equal(t, "newuser", found.Username)
	assert.Equal(t, "new@example.com", found.Email)
	assert.Equal(t, "New User", found.Name)
}

func TestUserRepository_UpsertFromJWT_UpdateExisting(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Create initial user
	initialUser := &models.User{
		ID:       1,
		Username: "olduser",
		Email:    "old@example.com",
		Name:     "Old User",
	}
	err := repo.Create(initialUser)
	assert.NoError(t, err)

	// Update user via JWT upsert
	user, err := repo.UpsertFromJWT(1, "updateduser", "updated@example.com", "Updated User")
	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "updateduser", user.Username)
	assert.Equal(t, "updated@example.com", user.Email)
	assert.Equal(t, "Updated User", user.Name)

	// Verify user was updated in database
	var found models.User
	err = db.First(&found, 1).Error
	assert.NoError(t, err)
	assert.Equal(t, "updateduser", found.Username)
	assert.Equal(t, "updated@example.com", found.Email)
	assert.Equal(t, "Updated User", found.Name)
}

func TestUserRepository_UpsertFromJWT_EmptyFields(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Test with empty username (should still work due to NOT NULL constraint being database-level)
	user, err := repo.UpsertFromJWT(1, "", "test@example.com", "Test User")
	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
}

func TestUserRepository_UniqueConstraints(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Create first user
	user1 := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := repo.Create(user1)
	assert.NoError(t, err)

	// Try to create second user with same username
	user2 := &models.User{
		ID:       2,
		Username: "testuser", // Same username
		Email:    "test2@example.com",
		Name:     "Test User 2",
	}
	err = repo.Create(user2)
	assert.Error(t, err) // Should fail due to unique constraint

	// Try to create second user with same email
	user3 := &models.User{
		ID:       3,
		Username: "testuser3",
		Email:    "test@example.com", // Same email
		Name:     "Test User 3",
	}
	err = repo.Create(user3)
	assert.Error(t, err) // Should fail due to unique constraint
}