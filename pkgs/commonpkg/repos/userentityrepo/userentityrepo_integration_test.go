package userentityrepo

import (
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

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

func setupTestUser(t *testing.T) uint64 {
	// Create a test user
	userID := uint64(123456789)
	_, err := db.Exec(`
		INSERT INTO users (id, screen_name, name, protected, friends_count)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, "testuser", "Test User", false, 100)
	require.NoError(t, err)

	return userID
}

func cleanupTestData(t *testing.T) {
	_, err := db.Exec("TRUNCATE TABLE user_entities CASCADE")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE TABLE users CASCADE")
	require.NoError(t, err)
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("create user entity successfully", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Arrange
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}

		// Act
		err := repo.Create(ctx, db, entity)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, entity.Id)
		assert.Equal(t, userID, entity.Uid)
		assert.Equal(t, "Test User Entity", entity.Name)
		assert.Contains(t, entity.ParentDir, "/tmp/test") // May be absolute path
		assert.Equal(t, "test_folder", entity.FolderName)
		assert.False(t, entity.StorageSaved)
		assert.False(t, entity.CreatedAt.IsZero())
		assert.False(t, entity.UpdatedAt.IsZero())
	})

	t.Run("create user entity with foreign key violation", func(t *testing.T) {
		defer cleanupTestData(t)

		// Arrange
		entity := &model.UserEntity{
			Uid:          999999999, // Non-existent user
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}

		// Act
		err := repo.Create(ctx, db, entity)

		// Assert
		assert.Error(t, err)
	})

	t.Run("create user entity with unique constraint violation", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create first entity
		entity1 := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity 1",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder1",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity1)
		require.NoError(t, err)

		// Arrange - same user_id and parent_dir
		entity2 := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity 2",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder2",
			StorageSaved: false,
		}

		// Act
		err = repo.Create(ctx, db, entity2)

		// Assert
		assert.Error(t, err)
	})
}

func TestRepoIntegration_Upsert(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("upsert new user entity", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Arrange
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}

		// Act
		err := repo.Upsert(ctx, db, entity)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, entity.Id)
		assert.Equal(t, "Test User Entity", entity.Name)
		assert.False(t, entity.StorageSaved)
	})

	t.Run("upsert existing user entity", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create initial entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Original Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "original_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)
		originalID := entity.Id

		// Arrange for upsert
		updatedEntity := &model.UserEntity{
			Uid:          userID,
			Name:         "Updated Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "updated_folder",
			StorageSaved: true,
		}

		// Act
		err = repo.Upsert(ctx, db, updatedEntity)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, originalID, updatedEntity.Id)
		assert.Equal(t, "Updated Entity", updatedEntity.Name)
		assert.Equal(t, "updated_folder", updatedEntity.FolderName)
		assert.True(t, updatedEntity.StorageSaved)
	})
}

func TestRepoIntegration_Get(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("get existing user entity", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		result, err := repo.Get(ctx, db, userID, "/tmp/test")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Test User Entity", result.Name)
		assert.Equal(t, userID, result.Uid)
		assert.Contains(t, result.ParentDir, "/tmp/test")
	})

	t.Run("get non-existent user entity returns nil", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		result, err := repo.Get(ctx, db, 999999999, "/nonexistent/path")

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_GetById(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("get existing user entity by id", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		result, err := repo.GetById(ctx, db, int(entity.Id))

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, entity.Id, result.Id)
		assert.Equal(t, "Test User Entity", result.Name)
		assert.Equal(t, userID, result.Uid)
	})

	t.Run("get non-existent user entity by id returns nil", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		result, err := repo.GetById(ctx, db, 999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_GetByTwitterId(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("get existing user entity by twitter id", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		result, err := repo.GetByTwitterId(ctx, db, userID)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Test User Entity", result.Name)
		assert.Equal(t, userID, result.Uid)
	})

	t.Run("get non-existent user entity by twitter id returns nil", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		result, err := repo.GetByTwitterId(ctx, db, 999999999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_Update(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update user entity", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Original Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "original_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Update entity fields
		entity.Name = "Updated Entity"
		entity.FolderName = "updated_folder"
		entity.StorageSaved = true
		entity.MediaCount = sql.NullInt32{Int32: 42, Valid: true}
		entity.LatestReleaseTime = sql.NullTime{Time: time.Now(), Valid: true}

		// Act
		err = repo.Update(ctx, db, entity)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Updated Entity", entity.Name)
		assert.Equal(t, "updated_folder", entity.FolderName)
		assert.True(t, entity.StorageSaved)
		assert.Equal(t, int32(42), entity.MediaCount.Int32)

		// Verify in database
		result, err := repo.GetById(ctx, db, int(entity.Id))
		require.NoError(t, err)
		assert.Equal(t, "Updated Entity", result.Name)
		assert.Equal(t, "updated_folder", result.FolderName)
		assert.True(t, result.StorageSaved)
		assert.Equal(t, int32(42), result.MediaCount.Int32)
		assert.True(t, result.LatestReleaseTime.Valid)
	})

	t.Run("update non-existent user entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Arrange
		entity := &model.UserEntity{
			Id:           999,
			Uid:          123456789,
			Name:         "Non-existent Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}

		// Act
		err := repo.Update(ctx, db, entity)

		// Assert
		assert.Error(t, err)
	})
}

func TestRepoIntegration_UpdateTweetStat(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update tweet statistics", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		releaseTime := time.Now().UTC()
		err = repo.UpdateTweetStat(ctx, db, int(entity.Id), releaseTime, 100)

		// Assert
		require.NoError(t, err)

		// Verify update
		result, err := repo.GetById(ctx, db, int(entity.Id))
		require.NoError(t, err)
		assert.Equal(t, int32(100), result.MediaCount.Int32)
		assert.True(t, result.LatestReleaseTime.Valid)
		assert.WithinDuration(t, releaseTime, result.LatestReleaseTime.Time, time.Second)
	})
}

func TestRepoIntegration_UpdateMediaCount(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update media count", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		err = repo.UpdateMediaCount(ctx, db, int(entity.Id), 250)

		// Assert
		require.NoError(t, err)

		// Verify update
		result, err := repo.GetById(ctx, db, int(entity.Id))
		require.NoError(t, err)
		assert.Equal(t, int32(250), result.MediaCount.Int32)
	})
}

func TestRepoIntegration_UpdateStorageSavedByTwitterId(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update storage saved by twitter id", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		err = repo.UpdateStorageSavedByTwitterId(ctx, db, userID, true)

		// Assert
		require.NoError(t, err)

		// Verify update
		result, err := repo.GetByTwitterId(ctx, db, userID)
		require.NoError(t, err)
		assert.True(t, result.StorageSaved)
	})
}

func TestRepoIntegration_Delete(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("delete existing user entity", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create test entity
		entity := &model.UserEntity{
			Uid:          userID,
			Name:         "Test User Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		err = repo.Delete(ctx, db, uint32(entity.Id))

		// Assert
		require.NoError(t, err)

		// Verify deletion
		result, err := repo.GetById(ctx, db, int(entity.Id))
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("delete non-existent user entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		err := repo.Delete(ctx, db, 999)

		// Assert
		require.NoError(t, err) // Should not error even if record doesn't exist
	})
}
