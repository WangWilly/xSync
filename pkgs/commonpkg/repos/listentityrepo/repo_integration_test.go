package listentityrepo

import (
	"context"
	"database/sql"
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

func cleanupTestData(t *testing.T) {
	_, err := db.Exec("TRUNCATE TABLE lst_entities CASCADE")
	require.NoError(t, err)
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("create list entity successfully", func(t *testing.T) {
		defer cleanupTestData(t)

		// Arrange
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}

		// Act
		err := repo.Create(ctx, db, entity)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, entity.Id.Int32)
		assert.Equal(t, int64(123456789), entity.LstId)
		assert.Equal(t, "Test List Entity", entity.Name)
		assert.Contains(t, entity.ParentDir, "/tmp/test") // May be absolute path
		assert.Equal(t, "test_folder", entity.FolderName)
		assert.False(t, entity.StorageSaved)
		assert.False(t, entity.CreatedAt.IsZero())
		assert.False(t, entity.UpdatedAt.IsZero())
	})

	t.Run("create list entity with unique constraint violation", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create first entity
		entity1 := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity 1",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder1",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity1)
		require.NoError(t, err)

		// Arrange - same lst_id and parent_dir
		entity2 := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity 2",
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

	t.Run("upsert new list entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Arrange
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}

		// Act
		err := repo.Upsert(ctx, db, entity)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, entity.Id.Int32)
		assert.Equal(t, "Test List Entity", entity.Name)
		assert.False(t, entity.StorageSaved)
	})

	t.Run("upsert existing list entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create initial entity
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Original Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "original_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)
		originalID := entity.Id.Int32

		// Arrange for upsert
		updatedEntity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Updated Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "updated_folder",
			StorageSaved: true,
		}

		// Act
		err = repo.Upsert(ctx, db, updatedEntity)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, originalID, updatedEntity.Id.Int32)
		assert.Equal(t, "Updated Entity", updatedEntity.Name)
		assert.Equal(t, "updated_folder", updatedEntity.FolderName)
		assert.True(t, updatedEntity.StorageSaved)
	})
}

func TestRepoIntegration_GetById(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("get existing list entity by id", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create test entity
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		result, err := repo.GetById(ctx, db, int(entity.Id.Int32))

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, entity.Id.Int32, result.Id.Int32)
		assert.Equal(t, "Test List Entity", result.Name)
		assert.Equal(t, int64(123456789), result.LstId)
	})

	t.Run("get non-existent list entity by id returns nil", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		result, err := repo.GetById(ctx, db, 999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_Get(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("get existing list entity by lst_id and parent_dir", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create test entity
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		result, err := repo.Get(ctx, db, 123456789, "/tmp/test")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Test List Entity", result.Name)
		assert.Equal(t, int64(123456789), result.LstId)
		assert.Contains(t, result.ParentDir, "/tmp/test")
	})

	t.Run("get non-existent list entity returns nil", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		result, err := repo.Get(ctx, db, 999999999, "/nonexistent/path")

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_Update(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update list entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create test entity
		entity := &model.ListEntity{
			LstId:        123456789,
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

		// Act
		err = repo.Update(ctx, db, entity)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Updated Entity", entity.Name)
		assert.Equal(t, "updated_folder", entity.FolderName)
		assert.True(t, entity.StorageSaved)

		// Verify in database
		result, err := repo.GetById(ctx, db, int(entity.Id.Int32))
		require.NoError(t, err)
		assert.Equal(t, "Updated Entity", result.Name)
		assert.Equal(t, "updated_folder", result.FolderName)
		assert.True(t, result.StorageSaved)
	})

	t.Run("update non-existent list entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Arrange
		entity := &model.ListEntity{
			Id:           sql.NullInt32{Int32: 999, Valid: true},
			LstId:        123456789,
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

func TestRepoIntegration_UpdateStorageSavedByTwitterId(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("update storage saved by twitter id", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create test entity
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		err = repo.UpdateStorageSavedByTwitterId(ctx, db, 123456789, true)

		// Assert
		require.NoError(t, err)

		// Verify update
		result, err := repo.GetById(ctx, db, int(entity.Id.Int32))
		require.NoError(t, err)
		assert.True(t, result.StorageSaved)
	})
}

func TestRepoIntegration_Delete(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("delete existing list entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Create test entity
		entity := &model.ListEntity{
			LstId:        123456789,
			Name:         "Test List Entity",
			ParentDir:    "/tmp/test",
			FolderName:   "test_folder",
			StorageSaved: false,
		}
		err := repo.Create(ctx, db, entity)
		require.NoError(t, err)

		// Act
		err = repo.Delete(ctx, db, int(entity.Id.Int32))

		// Assert
		require.NoError(t, err)

		// Verify deletion
		result, err := repo.GetById(ctx, db, int(entity.Id.Int32))
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("delete non-existent list entity", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act
		err := repo.Delete(ctx, db, 999)

		// Assert
		require.NoError(t, err) // Should not error even if record doesn't exist
	})
}
