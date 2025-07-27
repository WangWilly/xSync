package previousnamerepo

import (
	"context"
	"log"
	"testing"

	"github.com/WangWilly/xSync/migration/automigrate"
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
	_, err := db.Exec("TRUNCATE TABLE user_previous_names CASCADE")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE TABLE users CASCADE")
	require.NoError(t, err)
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()
	repo := New()

	t.Run("create user previous name successfully", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Act
		err := repo.Create(ctx, db, userID, "Test User", "testuser")

		// Assert
		require.NoError(t, err)

		// Verify record was created
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM user_previous_names WHERE uid = $1", userID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify values
		var storedScreenName, storedName string
		err = db.QueryRow("SELECT screen_name, name FROM user_previous_names WHERE uid = $1", userID).Scan(&storedScreenName, &storedName)
		require.NoError(t, err)
		assert.Equal(t, "testuser", storedScreenName)
		assert.Equal(t, "Test User", storedName)
	})

	t.Run("create multiple previous names for same user", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Act - Create multiple previous names
		err := repo.Create(ctx, db, userID, "First Name", "first_screen")
		require.NoError(t, err)

		err = repo.Create(ctx, db, userID, "Second Name", "second_screen")
		require.NoError(t, err)

		err = repo.Create(ctx, db, userID, "Third Name", "third_screen")
		require.NoError(t, err)

		// Assert
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM user_previous_names WHERE uid = $1", userID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		// Verify all names exist
		type PreviousName struct {
			ScreenName string `db:"screen_name"`
			Name       string `db:"name"`
		}
		var names []PreviousName
		err = db.Select(&names, "SELECT screen_name, name FROM user_previous_names WHERE uid = $1 ORDER BY id", userID)
		require.NoError(t, err)

		assert.Len(t, names, 3)
		assert.Equal(t, "first_screen", names[0].ScreenName)
		assert.Equal(t, "First Name", names[0].Name)
		assert.Equal(t, "second_screen", names[1].ScreenName)
		assert.Equal(t, "Second Name", names[1].Name)
		assert.Equal(t, "third_screen", names[2].ScreenName)
		assert.Equal(t, "Third Name", names[2].Name)
	})

	t.Run("create with foreign key violation", func(t *testing.T) {
		defer cleanupTestData(t)

		// Act - Try to create previous name for non-existent user
		err := repo.Create(ctx, db, 999999999, "Test Name", "testscreen")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "foreign key constraint")
	})

	t.Run("create with empty name", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Act
		err := repo.Create(ctx, db, userID, "", "")

		// Assert
		require.NoError(t, err) // Empty strings should be allowed

		// Verify record was created
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM user_previous_names WHERE uid = $1", userID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("create with special characters", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Act
		err := repo.Create(ctx, db, userID, "Test User with 特殊字符 & symbols!", "test_user_123")

		// Assert
		require.NoError(t, err)

		// Verify special characters are preserved
		var storedName string
		err = db.QueryRow("SELECT name FROM user_previous_names WHERE uid = $1", userID).Scan(&storedName)
		require.NoError(t, err)
		assert.Equal(t, "Test User with 特殊字符 & symbols!", storedName)
	})

	t.Run("create with long strings", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Create very long strings
		longName := string(make([]byte, 1000))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}
		longScreenName := string(make([]byte, 500))
		for i := range longScreenName {
			longScreenName = longScreenName[:i] + "b" + longScreenName[i+1:]
		}

		// Act
		err := repo.Create(ctx, db, userID, longName, longScreenName)

		// Assert
		require.NoError(t, err)

		// Verify long strings are stored correctly
		var storedName, storedScreenName string
		err = db.QueryRow("SELECT name, screen_name FROM user_previous_names WHERE uid = $1", userID).Scan(&storedName, &storedScreenName)
		require.NoError(t, err)
		assert.Equal(t, longName, storedName)
		assert.Equal(t, longScreenName, storedScreenName)
	})

	t.Run("create with todo context", func(t *testing.T) {
		defer cleanupTestData(t)
		userID := setupTestUser(t)

		// Act
		err := repo.Create(context.TODO(), db, userID, "Test Name", "testscreen")

		// Assert
		require.NoError(t, err) // Should work with context.TODO()
	})
}
