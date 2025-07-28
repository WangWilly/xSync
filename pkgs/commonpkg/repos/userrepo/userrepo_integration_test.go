package userrepo

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/WangWilly/xSync/migration/automigrate"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	// Setup
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Start a PostgreSQL container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=testdb",
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that the container is removed when stopped
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// Exponential backoff-retry until the database is ready
	if err := pool.Retry(func() error {
		var err error
		db, err = sqlx.Connect("postgres", fmt.Sprintf("postgres://postgres:postgres@localhost:%s/testdb?sslmode=disable", resource.GetPort("5432/tcp")))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Set up database schema using auto migration
	err = automigrate.AutoMigrateUp(automigrate.AutoMigrateConfig{
		SqlxDB: db,
	})
	if err != nil {
		log.Fatalf("Could not run auto migration: %s", err)
	}

	// Run tests
	code := m.Run()

	// Clean up
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()

	repo := New()

	t.Run("create user", func(t *testing.T) {
		// Arrange
		user := &model.User{
			Id:           12345,
			ScreenName:   "testuser",
			Name:         "Test User",
			IsProtected:  false,
			FriendsCount: 42,
		}

		// Act
		err := repo.Create(ctx, db, user)

		// Assert
		require.NoError(t, err)

		// Verify creation
		created, err := repo.GetById(ctx, db, user.Id)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, user.Id, created.Id)
		assert.Equal(t, user.ScreenName, created.ScreenName)
		assert.Equal(t, user.Name, created.Name)
		assert.Equal(t, user.IsProtected, created.IsProtected)
		assert.Equal(t, user.FriendsCount, created.FriendsCount)
		assert.NotZero(t, created.CreatedAt)
		assert.NotZero(t, created.UpdatedAt)
	})
}

func TestRepoIntegration_Upsert(t *testing.T) {
	ctx := context.Background()

	repo := New()

	t.Run("upsert new user", func(t *testing.T) {
		// Arrange
		user := &model.User{
			Id:           23456,
			ScreenName:   "newuser",
			Name:         "New User",
			IsProtected:  true,
			FriendsCount: 100,
		}

		// Act
		err := repo.Upsert(ctx, db, user)

		// Assert
		require.NoError(t, err)

		// Verify insertion
		created, err := repo.GetById(ctx, db, user.Id)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, user.ScreenName, created.ScreenName)
		assert.Equal(t, user.Name, created.Name)
		assert.Equal(t, user.IsProtected, created.IsProtected)
		assert.Equal(t, user.FriendsCount, created.FriendsCount)
	})

	t.Run("upsert existing user", func(t *testing.T) {
		// Arrange - Get the existing user
		userId := uint64(23456)
		existingUser, err := repo.GetById(ctx, db, userId)
		require.NoError(t, err)
		require.NotNil(t, existingUser)

		// Store initial updated_at for comparison
		initialUpdatedAt := existingUser.UpdatedAt

		// Wait a bit to ensure timestamp changes
		time.Sleep(10 * time.Millisecond)

		// Update user data
		existingUser.ScreenName = "updateduser"
		existingUser.Name = "Updated User"
		existingUser.IsProtected = false
		existingUser.FriendsCount = 200

		// Act
		err = repo.Upsert(ctx, db, existingUser)

		// Assert
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetById(ctx, db, userId)
		require.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, "updateduser", updated.ScreenName)
		assert.Equal(t, "Updated User", updated.Name)
		assert.Equal(t, false, updated.IsProtected)
		assert.Equal(t, 200, updated.FriendsCount)
		assert.True(t, updated.UpdatedAt.After(initialUpdatedAt))
	})
}

func TestRepoIntegration_GetById(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a user first
	user := &model.User{
		Id:           34567,
		ScreenName:   "getbyiduser",
		Name:         "GetById User",
		IsProtected:  false,
		FriendsCount: 50,
	}
	err := repo.Create(ctx, db, user)
	require.NoError(t, err)

	t.Run("get existing user", func(t *testing.T) {
		// Act
		result, err := repo.GetById(ctx, db, user.Id)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, user.Id, result.Id)
		assert.Equal(t, user.ScreenName, result.ScreenName)
		assert.Equal(t, user.Name, result.Name)
	})

	t.Run("get non-existent user", func(t *testing.T) {
		// Act
		result, err := repo.GetById(ctx, db, 99999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_Update(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a user first
	user := &model.User{
		Id:           45678,
		ScreenName:   "updateuser",
		Name:         "Update User",
		IsProtected:  false,
		FriendsCount: 75,
	}
	err := repo.Create(ctx, db, user)
	require.NoError(t, err)

	// Get the user to have the created timestamps
	createdUser, err := repo.GetById(ctx, db, user.Id)
	require.NoError(t, err)
	initialUpdatedAt := createdUser.UpdatedAt

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	t.Run("update user", func(t *testing.T) {
		// Arrange
		createdUser.ScreenName = "updatedscreenname"
		createdUser.Name = "Updated Name"
		createdUser.IsProtected = true
		createdUser.FriendsCount = 150

		// Act
		err := repo.Update(ctx, db, createdUser)

		// Assert
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetById(ctx, db, user.Id)
		require.NoError(t, err)
		assert.Equal(t, "updatedscreenname", updated.ScreenName)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, true, updated.IsProtected)
		assert.Equal(t, 150, updated.FriendsCount)
		assert.True(t, updated.UpdatedAt.After(initialUpdatedAt))
	})
}

func TestRepoIntegration_Delete(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a user first
	user := &model.User{
		Id:           56789,
		ScreenName:   "deleteuser",
		Name:         "Delete User",
		IsProtected:  false,
		FriendsCount: 30,
	}
	err := repo.Create(ctx, db, user)
	require.NoError(t, err)

	t.Run("delete user", func(t *testing.T) {
		// Act
		err := repo.Delete(ctx, db, user.Id)

		// Assert
		require.NoError(t, err)

		// Verify deletion
		deleted, err := repo.GetById(ctx, db, user.Id)
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

func TestRepoIntegration_CreatePreviousName(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a user first
	user := &model.User{
		Id:           67890,
		ScreenName:   "prevnameuser",
		Name:         "Previous Name User",
		IsProtected:  false,
		FriendsCount: 60,
	}
	err := repo.Create(ctx, db, user)
	require.NoError(t, err)
}
