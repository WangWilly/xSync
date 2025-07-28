package mediarepo

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

	// Run the tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// Exit with the status code from the tests
	log.Printf("Tests finished with exit code %d", code)
}

func clearData() {
	// Clear medias first due to foreign key constraints
	db.Exec("DELETE FROM medias")
	db.Exec("DELETE FROM tweets")
	db.Exec("DELETE FROM users")
}

func setupTestData() {
	// Insert test users
	_, err := db.Exec(`
		INSERT INTO users (id, screen_name, name, protected, friends_count) 
		VALUES (12345, 'test_user', 'Test User', false, 100)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		log.Printf("Warning: Failed to setup test users: %v", err)
	}

	// Insert test tweets - using tweet_id values that match what the test expects
	_, err = db.Exec(`
		INSERT INTO tweets (id, user_id, tweet_id, content, tweet_time) 
		VALUES 
			(67890, 12345, 67890, 'Test tweet 1', NOW()),
			(67891, 12345, 67891, 'Test tweet 2', NOW())
		ON CONFLICT (tweet_id) DO NOTHING
	`)
	if err != nil {
		log.Printf("Warning: Failed to setup test tweets: %v", err)
	}
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()

	// Clear data and setup test data before tests
	clearData()
	setupTestData()

	t.Run("create media", func(t *testing.T) {
		// Create a new media
		media := &model.Media{
			UserId:   12345,
			TweetId:  67890,
			Location: "/path/to/media.jpg",
		}

		// Verify ID is initially 0
		assert.Equal(t, int64(0), media.Id)

		// Create the media using our repository
		err := repo.Create(ctx, db, media)
		require.NoError(t, err)

		// Verify ID was updated
		assert.Greater(t, media.Id, int64(0))

		// Verify timestamps were set
		assert.False(t, media.CreatedAt.IsZero())
		assert.False(t, media.UpdatedAt.IsZero())

		// Verify the media was created in the database
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM medias WHERE id = $1", media.Id)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Create another media
		media2 := &model.Media{
			UserId:   12345,
			TweetId:  67891,
			Location: "/path/to/another/media.jpg",
		}

		// Create the media in the database
		err = repo.Create(ctx, db, media2)
		require.NoError(t, err)

		// Verify ID was updated sequentially
		assert.Equal(t, int64(2), media2.Id)
	})
}

func TestRepoIntegration_Update(t *testing.T) {
	ctx := context.Background()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()

	// Clear data and setup test data before tests
	clearData()
	setupTestData()

	t.Run("update with timestamp update", func(t *testing.T) {
		// Create a new media
		media := &model.Media{
			UserId:   12345,
			TweetId:  67890,
			Location: "/path/to/media.jpg",
		}

		// Create the media in the database
		err := repo.Create(ctx, db, media)
		require.NoError(t, err)

		// Get the original media to compare timestamps
		var originalMedia model.Media
		err = db.Get(&originalMedia, "SELECT * FROM medias WHERE id = $1", media.Id)
		require.NoError(t, err)

		// Force updated_at to be older
		_, err = db.Exec("UPDATE medias SET updated_at = updated_at - interval '1 minute' WHERE id = $1", media.Id)
		require.NoError(t, err)

		// Get the media again with the forced older timestamp
		err = db.Get(&originalMedia, "SELECT * FROM medias WHERE id = $1", media.Id)
		require.NoError(t, err)

		// Update the media
		media.Location = "/path/to/updated/media.jpg"
		err = repo.Update(ctx, db, media)
		require.NoError(t, err)

		// Get the updated media
		var updatedMedia model.Media
		err = db.Get(&updatedMedia, "SELECT * FROM medias WHERE id = $1", media.Id)
		require.NoError(t, err)

		// Verify location was updated
		assert.Equal(t, "/path/to/updated/media.jpg", updatedMedia.Location)

		// Verify updated_at was changed
		assert.Greater(t, updatedMedia.UpdatedAt.Unix(), originalMedia.UpdatedAt.Unix())

		// Verify created_at did not change
		assert.Equal(t, originalMedia.CreatedAt.Unix(), updatedMedia.CreatedAt.Unix())
	})
}
