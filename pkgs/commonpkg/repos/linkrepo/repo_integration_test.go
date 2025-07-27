package linkrepo

import (
	"context"
	"log"
	"testing"

	"github.com/WangWilly/xSync/migration/automigrate"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=testdb",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		db, err = sqlx.Open("postgres", "postgres://postgres:postgres@localhost:"+resource.GetPort("5432/tcp")+"/testdb?sslmode=disable")
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

	// Exit with the status code from the tests
	log.Printf("Tests finished with exit code %d", code)
}

func setupTestData(t *testing.T) (uint64, int32) {
	// Create a test user
	userID := uint64(123456789)
	_, err := db.Exec(`
		INSERT INTO users (id, screen_name, name, protected, friends_count)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, "testuser", "Test User", false, 100)
	require.NoError(t, err)

	// Create a test list entity
	var listEntityID int32
	err = db.QueryRow(`
		INSERT INTO lst_entities (lst_id, name, parent_dir, folder_name, storage_saved)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, 987654321, "Test List Entity", "/test/path", "test_folder", false).Scan(&listEntityID)
	require.NoError(t, err)

	return userID, listEntityID
}

func cleanupTestData(t *testing.T) {
	_, err := db.Exec("TRUNCATE TABLE user_links CASCADE")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE TABLE lst_entities CASCADE")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE TABLE users CASCADE")
	require.NoError(t, err)
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("create user link successfully", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Arrange
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Test Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}

		// Act
		err := repo.Create(ctx, db, link)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, link.Id.Int32)
		assert.Equal(t, userID, link.UserTwitterId)
		assert.Equal(t, "Test Link", link.Name)
		assert.Equal(t, listEntityID, link.ListEntityIdBelongTo)
		assert.False(t, link.StorageSaved)
		assert.False(t, link.CreatedAt.IsZero())
		assert.False(t, link.UpdatedAt.IsZero())
	})

	t.Run("create user link with foreign key violation", func(t *testing.T) {
		defer cleanupTestData(t)

		// Arrange
		link := &model.UserLink{
			UserTwitterId:        999999999, // Non-existent user
			Name:                 "Test Link",
			ListEntityIdBelongTo: 999, // Non-existent list entity
			StorageSaved:         false,
		}

		// Act
		err := repo.Create(ctx, db, link)

		// Assert
		assert.Error(t, err)
	})
}

func TestRepoIntegration_Upsert(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("upsert new user link", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Arrange
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Test Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}

		// Act
		err := repo.Upsert(ctx, db, link)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, link.Id.Int32)
		assert.Equal(t, "Test Link", link.Name)
		assert.False(t, link.StorageSaved)
	})

	t.Run("upsert existing user link", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Create initial link
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Original Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}
		err := repo.Create(ctx, db, link)
		require.NoError(t, err)
		originalID := link.Id.Int32

		// Arrange for upsert
		updatedLink := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Updated Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         true,
		}

		// Act
		err = repo.Upsert(ctx, db, updatedLink)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, originalID, updatedLink.Id.Int32)
		assert.Equal(t, "Updated Link", updatedLink.Name)
		assert.True(t, updatedLink.StorageSaved)
	})
}

func TestRepoIntegration_Get(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("get existing user link", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Create test link
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Test Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}
		err := repo.Create(ctx, db, link)
		require.NoError(t, err)

		// Act
		result, err := repo.Get(ctx, db, userID, listEntityID)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Test Link", result.Name)
		assert.Equal(t, userID, result.UserTwitterId)
		assert.Equal(t, listEntityID, result.ListEntityIdBelongTo)
	})

	t.Run("get non-existent user link returns nil", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		result, err := repo.Get(ctx, db, 999999999, 999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_ListAll(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("list all user links for user", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Create test links
		link1 := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Link 1",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}
		err := repo.Create(ctx, db, link1)
		require.NoError(t, err)

		// Act
		links, err := repo.ListAll(ctx, db, userID)

		// Assert
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "Link 1", links[0].Name)
		assert.Equal(t, userID, links[0].UserTwitterId)
	})

	t.Run("list all returns empty for non-existent user", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		links, err := repo.ListAll(ctx, db, 999999999)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, links)
	})
}

func TestRepoIntegration_Update(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update user link name", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Create test link
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Original Name",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}
		err := repo.Create(ctx, db, link)
		require.NoError(t, err)

		// Act
		err = repo.Update(ctx, db, link.Id.Int32, "Updated Name")

		// Assert
		require.NoError(t, err)

		// Verify update
		result, err := repo.Get(ctx, db, userID, listEntityID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", result.Name)
	})
}

func TestRepoIntegration_UpdateStorageSaved(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update storage saved status", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Create test link
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Test Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}
		err := repo.Create(ctx, db, link)
		require.NoError(t, err)

		// Act
		err = repo.UpdateStorageSaved(ctx, db, link.Id.Int32, true)

		// Assert
		require.NoError(t, err)

		// Verify update
		result, err := repo.Get(ctx, db, userID, listEntityID)
		require.NoError(t, err)
		assert.True(t, result.StorageSaved)
	})
}

func TestRepoIntegration_Delete(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("delete existing user link", func(t *testing.T) {
		defer cleanupTestData(t)
		userID, listEntityID := setupTestData(t)

		// Create test link
		link := &model.UserLink{
			UserTwitterId:        userID,
			Name:                 "Test Link",
			ListEntityIdBelongTo: listEntityID,
			StorageSaved:         false,
		}
		err := repo.Create(ctx, db, link)
		require.NoError(t, err)

		// Act
		err = repo.Delete(ctx, db, link.Id.Int32)

		// Assert
		require.NoError(t, err)

		// Verify deletion
		result, err := repo.Get(ctx, db, userID, listEntityID)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("delete non-existent user link", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		err := repo.Delete(ctx, db, 999)

		// Assert
		require.NoError(t, err) // Should not error even if record doesn't exist
	})
}
