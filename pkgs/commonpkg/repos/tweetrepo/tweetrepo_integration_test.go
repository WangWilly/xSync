package tweetrepo

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
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

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tweets (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			tweet_id BIGINT NOT NULL,
			content TEXT,
			tweet_time TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		
		CREATE TABLE IF NOT EXISTS medias (
			id SERIAL PRIMARY KEY,
			tweet_id BIGINT,
			location TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Fatalf("Could not create tables: %s", err)
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

	t.Run("create tweet", func(t *testing.T) {
		// Arrange
		tweet := &model.Tweet{
			UserId:    12345,
			TweetId:   67890,
			Content:   "Test tweet content",
			TweetTime: time.Now(),
		}

		// Act
		err := repo.Create(ctx, db, tweet)

		// Assert
		require.NoError(t, err)
		assert.NotZero(t, tweet.Id)
	})
}

func TestRepoIntegration_GetById(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a tweet first
	tweet := &model.Tweet{
		UserId:    12345,
		TweetId:   67890,
		Content:   "Test tweet content for GetById",
		TweetTime: time.Now(),
	}
	err := repo.Create(ctx, db, tweet)
	require.NoError(t, err)
	require.NotZero(t, tweet.Id)

	t.Run("get existing tweet", func(t *testing.T) {
		// Act
		result, err := repo.GetById(ctx, db, tweet.Id)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tweet.Id, result.Id)
		assert.Equal(t, tweet.Content, result.Content)
	})

	t.Run("get non-existent tweet", func(t *testing.T) {
		// Act
		result, err := repo.GetById(ctx, db, 9999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_Update(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a tweet first
	tweet := &model.Tweet{
		UserId:    12345,
		TweetId:   67890,
		Content:   "Original content",
		TweetTime: time.Now(),
	}
	err := repo.Create(ctx, db, tweet)
	require.NoError(t, err)
	require.NotZero(t, tweet.Id)

	t.Run("update tweet", func(t *testing.T) {
		// Arrange
		tweet.Content = "Updated content"
		originalUpdatedAt := tweet.UpdatedAt

		// Act
		err := repo.Update(ctx, db, tweet)

		// Assert
		require.NoError(t, err)

		// Verify the update by fetching it again
		updated, err := repo.GetById(ctx, db, tweet.Id)
		require.NoError(t, err)
		assert.Equal(t, "Updated content", updated.Content)
		assert.NotEqual(t, originalUpdatedAt, updated.UpdatedAt)
	})
}

func TestRepoIntegration_GetByUserId(t *testing.T) {
	ctx := context.Background()

	repo := New()
	userId := uint64(54321)

	// Create multiple tweets for the same user
	for i := 0; i < 3; i++ {
		tweet := &model.Tweet{
			UserId:    userId,
			TweetId:   uint64(100000 + i),
			Content:   fmt.Sprintf("User tweet %d", i),
			TweetTime: time.Now().Add(time.Duration(-i) * time.Hour), // Decreasing time
		}
		err := repo.Create(ctx, db, tweet)
		require.NoError(t, err)
	}

	t.Run("get user tweets", func(t *testing.T) {
		// Act
		tweets, err := repo.GetByUserId(ctx, db, userId)

		// Assert
		require.NoError(t, err)
		assert.Len(t, tweets, 3)
		// Verify they're in descending order by tweet_time
		assert.True(t, tweets[0].TweetTime.After(tweets[1].TweetTime))
		assert.True(t, tweets[1].TweetTime.After(tweets[2].TweetTime))
	})
}

func TestRepoIntegration_Delete(t *testing.T) {
	ctx := context.Background()

	repo := New()

	// Create a tweet first
	tweet := &model.Tweet{
		UserId:    12345,
		TweetId:   67890,
		Content:   "Content to be deleted",
		TweetTime: time.Now(),
	}
	err := repo.Create(ctx, db, tweet)
	require.NoError(t, err)
	require.NotZero(t, tweet.Id)

	t.Run("delete tweet", func(t *testing.T) {
		// Act
		err := repo.Delete(ctx, db, tweet.Id)

		// Assert
		require.NoError(t, err)

		// Verify deletion
		result, err := repo.GetById(ctx, db, tweet.Id)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_GetByTweetId(t *testing.T) {
	ctx := context.Background()

	repo := New()
	tweetId := uint64(555555)

	// Create a tweet with specific tweet_id
	tweet := &model.Tweet{
		UserId:    12345,
		TweetId:   tweetId,
		Content:   "Tweet with specific tweet_id",
		TweetTime: time.Now(),
	}
	err := repo.Create(ctx, db, tweet)
	require.NoError(t, err)

	t.Run("get by tweet_id", func(t *testing.T) {
		// Act
		result, err := repo.GetByTweetId(ctx, db, tweetId)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tweetId, result.TweetId)
	})

	t.Run("get by non-existent tweet_id", func(t *testing.T) {
		// Act
		result, err := repo.GetByTweetId(ctx, db, 999999)

		// Assert
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepoIntegration_GetWithMedia(t *testing.T) {
	ctx := context.Background()

	repo := New()
	userId := uint64(777777)

	// Create a tweet
	tweet := &model.Tweet{
		UserId:    userId,
		TweetId:   uint64(888888),
		Content:   "Tweet with media",
		TweetTime: time.Now(),
	}
	err := repo.Create(ctx, db, tweet)
	require.NoError(t, err)

	// Add media for this tweet
	_, err = db.Exec(`INSERT INTO medias(tweet_id, location) VALUES($1, $2)`, tweet.Id, "path/to/media.jpg")
	require.NoError(t, err)

	t.Run("get tweets with media", func(t *testing.T) {
		// Act
		results, err := repo.GetWithMedia(ctx, db, userId)

		// Assert
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, tweet.Id, results[0]["id"])
		assert.Equal(t, "path/to/media.jpg", results[0]["media_location"])
	})
}
