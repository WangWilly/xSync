package listrepo

import (
	"context"
	"log"
	"testing"

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

	// Create the lists table
	setupSchema()

	// Run the tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// Exit with the status code from the tests
	log.Printf("Tests finished with exit code %d", code)
}

func setupSchema() {
	// Create lists table
	schema := `
	CREATE TABLE IF NOT EXISTS lsts (
		id BIGINT PRIMARY KEY,
		name TEXT NOT NULL,
		owner_uid BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatalf("Could not create schema: %s", err)
	}
}

func clearData() {
	db.Exec("TRUNCATE TABLE lsts")
}

func TestRepoIntegration_Create(t *testing.T) {
	ctx := context.Background()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()

	// Clear data before tests
	clearData()

	t.Run("create list", func(t *testing.T) {
		// Create a new list
		list := &model.List{
			Id:      12345,
			Name:    "Test List",
			OwnerId: 67890,
		}

		// Create the list
		err := repo.Create(ctx, db, list)
		require.NoError(t, err)

		// Verify list was created in the database
		var dbList model.List
		err = db.Get(&dbList, "SELECT * FROM lsts WHERE id = $1", list.Id)
		require.NoError(t, err)

		// Verify fields match
		assert.Equal(t, list.Id, dbList.Id)
		assert.Equal(t, list.Name, dbList.Name)
		assert.Equal(t, list.OwnerId, dbList.OwnerId)

		// Verify timestamps were set
		assert.False(t, dbList.CreatedAt.IsZero())
		assert.False(t, dbList.UpdatedAt.IsZero())
	})
}

func TestRepoIntegration_Upsert(t *testing.T) {
	ctx := context.Background()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()

	// Clear data before tests
	clearData()

	t.Run("upsert new list", func(t *testing.T) {
		// Create a new list
		list := &model.List{
			Id:      67890,
			Name:    "New List",
			OwnerId: 12345,
		}

		// Upsert the list (should insert)
		err := repo.Upsert(ctx, db, list)
		require.NoError(t, err)

		// Verify list was created in the database
		var dbList model.List
		err = db.Get(&dbList, "SELECT * FROM lsts WHERE id = $1", list.Id)
		require.NoError(t, err)

		// Verify fields match
		assert.Equal(t, list.Id, dbList.Id)
		assert.Equal(t, list.Name, dbList.Name)
		assert.Equal(t, list.OwnerId, dbList.OwnerId)

		// Verify timestamps were set
		assert.False(t, dbList.CreatedAt.IsZero())
		assert.False(t, dbList.UpdatedAt.IsZero())
	})

	t.Run("upsert existing list", func(t *testing.T) {
		// Get the existing list
		var originalList model.List
		err := db.Get(&originalList, "SELECT * FROM lsts WHERE id = 67890")
		require.NoError(t, err)

		// Force updated_at to be older
		_, err = db.Exec("UPDATE lsts SET updated_at = updated_at - interval '1 minute' WHERE id = 67890")
		require.NoError(t, err)

		// Get the updated original list with older timestamp
		err = db.Get(&originalList, "SELECT * FROM lsts WHERE id = 67890")
		require.NoError(t, err)

		// Update list data
		list := &model.List{
			Id:      67890,
			Name:    "Updated List",
			OwnerId: 12345,
		}

		// Upsert the list (should update)
		err = repo.Upsert(ctx, db, list)
		require.NoError(t, err)

		// Verify list was updated in the database
		var updatedList model.List
		err = db.Get(&updatedList, "SELECT * FROM lsts WHERE id = $1", list.Id)
		require.NoError(t, err)

		// Verify fields were updated
		assert.Equal(t, list.Id, updatedList.Id)
		assert.Equal(t, list.Name, updatedList.Name)
		assert.Equal(t, list.OwnerId, updatedList.OwnerId)

		// Verify created_at didn't change
		assert.Equal(t, originalList.CreatedAt.Unix(), updatedList.CreatedAt.Unix())

		// Verify updated_at changed
		assert.Greater(t, updatedList.UpdatedAt.Unix(), originalList.UpdatedAt.Unix())
	})
}

func TestRepoIntegration_GetById(t *testing.T) {
	ctx := context.Background()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()

	// Clear data before tests
	clearData()

	t.Run("get existing list", func(t *testing.T) {
		// Insert a list
		list := &model.List{
			Id:      12345,
			Name:    "Test List",
			OwnerId: 67890,
		}

		_, err := db.NamedExec(
			`INSERT INTO lsts(id, name, owner_uid) 
			 VALUES(:id, :name, :owner_uid)`,
			list,
		)
		require.NoError(t, err)

		// Get the list
		dbList, err := repo.GetById(ctx, db, list.Id)
		require.NoError(t, err)
		require.NotNil(t, dbList)

		// Verify fields match
		assert.Equal(t, list.Id, dbList.Id)
		assert.Equal(t, list.Name, dbList.Name)
		assert.Equal(t, list.OwnerId, dbList.OwnerId)
	})

	t.Run("get non-existent list", func(t *testing.T) {
		// Get a list that doesn't exist
		dbList, err := repo.GetById(ctx, db, 99999)
		require.NoError(t, err)
		assert.Nil(t, dbList)
	})
}
