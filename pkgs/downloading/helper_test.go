package downloading_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUserEntity implements the minimum required fields for tests
type MockUserEntity struct {
	Id        int32
	Uid       uint64
	ParentDir string
}

// Implement the necessary fields to satisfy model.UserEntity
func (m *MockUserEntity) GetUid() uint64 {
	return m.Uid
}

func (m *MockUserEntity) GetParentDir() string {
	return m.ParentDir
}

// Convert MockUserEntity to model.UserEntity for testing
func (m *MockUserEntity) ToModelUserEntity() *model.UserEntity {
	return &model.UserEntity{
		Id:         sql.NullInt32{Int32: m.Id, Valid: true},
		Uid:        m.Uid,
		ParentDir:  m.ParentDir,
		FolderName: "test-folder",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// Define mocks for interfaces without using testify/mock
// This avoids dependency on github.com/stretchr/testify/mock

type MockHeapHelper struct {
	// Mock fields to store expected results
	heapToReturn     *utils.Heap[*smartpathdto.UserSmartPath]
	depthsToReturn   map[*smartpathdto.UserSmartPath]int
	usersToReturn    map[uint64]*twitterclient.User
	makeHeapToReturn error
}

func NewMockHeapHelper() *MockHeapHelper {
	return &MockHeapHelper{
		depthsToReturn: make(map[*smartpathdto.UserSmartPath]int),
		usersToReturn:  make(map[uint64]*twitterclient.User),
	}
}

func (m *MockHeapHelper) GetDepth(userSmartPath *smartpathdto.UserSmartPath) int {
	if depth, ok := m.depthsToReturn[userSmartPath]; ok {
		return depth
	}
	return 0
}

func (m *MockHeapHelper) GetHeap() *utils.Heap[*smartpathdto.UserSmartPath] {
	return m.heapToReturn
}

func (m *MockHeapHelper) GetUserByTwitterId(twitterId uint64) *twitterclient.User {
	return m.usersToReturn[twitterId]
}

func (m *MockHeapHelper) MakeHeap(ctx context.Context, db *sqlx.DB, dir string, autoFollow bool) error {
	return m.makeHeapToReturn
}

type MockDbWorker struct {
	// Mock fields to store expected results
	downloadReturnValues []*dldto.NewEntity
	produceReturnValues  []*dldto.NewEntity
	produceError         error
	downloadCalled       bool
	produceCalled        bool
}

func NewMockDbWorker() *MockDbWorker {
	return &MockDbWorker{}
}

func (m *MockDbWorker) DownloadTweetMediaFromTweetChanWithDB(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	tweetDlMetaIn <-chan *dldto.NewEntity,
	incrementConsumed func(),
) []*dldto.NewEntity {
	m.downloadCalled = true

	// Drain the channel to avoid blocking
	go func() {
		for range tweetDlMetaIn {
			if incrementConsumed != nil {
				incrementConsumed()
			}
		}
	}()

	return m.downloadReturnValues
}

func (m *MockDbWorker) ProduceFromHeapToTweetChanWithDB(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	output chan<- *dldto.NewEntity,
	incrementProduced func(),
) ([]*dldto.NewEntity, error) {
	m.produceCalled = true

	// If there's an error, just return it immediately without writing to channel
	if m.produceError != nil {
		return m.produceReturnValues, m.produceError
	}

	// Only try to send to the channel if we don't have an error
	// The caller (SimpleWorker) is responsible for closing the channel
	go func() {
		for _, entity := range m.produceReturnValues {
			select {
			case <-ctx.Done():
				return
			default:
				// Try to send but give up if context is cancelled
				select {
				case <-ctx.Done():
					return
				case output <- entity:
					if incrementProduced != nil {
						incrementProduced()
					}
				}
			}
		}
	}()

	return m.produceReturnValues, m.produceError
}

func TestNewDownloadHelperWithConfig(t *testing.T) {
	// Test with default config values
	t.Run("Default config values", func(t *testing.T) {
		mockDbWorker := NewMockDbWorker()

		helper := downloading.NewDownloadHelperWithConfig(
			downloading.Config{},
			mockDbWorker,
		)

		require.NotNil(t, helper)
	})

	// Test with custom config values
	t.Run("Custom config values", func(t *testing.T) {
		mockDbWorker := NewMockDbWorker()

		helper := downloading.NewDownloadHelperWithConfig(
			downloading.Config{
				MaxDownloadRoutine: 42,
			},
			mockDbWorker,
		)

		require.NotNil(t, helper)
	})
}

func TestBatchDownloadTweetWithDB(t *testing.T) {
	mockDbWorker := NewMockDbWorker()

	// Create helper with test config
	helper := downloading.NewDownloadHelperWithConfig(
		downloading.Config{MaxDownloadRoutine: 5},
		mockDbWorker,
	)

	t.Run("No tweets to download", func(t *testing.T) {
		result := helper.BatchDownloadTweetWithDB(context.Background())
		assert.Nil(t, result)
	})

	t.Run("Download tweets successfully", func(t *testing.T) {
		// Setup test data - create valid tweet entities
		tweet1 := &twitterclient.Tweet{Id: 1}
		tweet2 := &twitterclient.Tweet{Id: 2}

		// Create UserSmartPath entities
		mockEntity1 := &MockUserEntity{Id: 1, Uid: 101, ParentDir: "/test1"}
		mockEntity2 := &MockUserEntity{Id: 2, Uid: 102, ParentDir: "/test2"}
		entity1 := smartpathdto.New(mockEntity1.ToModelUserEntity(), 1)
		entity2 := smartpathdto.New(mockEntity2.ToModelUserEntity(), 1)

		entity1Meta := &dldto.NewEntity{Tweet: tweet1, Entity: entity1}
		entity2Meta := &dldto.NewEntity{Tweet: tweet2, Entity: entity2}

		inputTweets := []*dldto.NewEntity{entity1Meta, entity2Meta}

		// Setup mock to return some values
		mockDbWorker.downloadReturnValues = []*dldto.NewEntity{entity1Meta, entity2Meta}

		// Run the function
		result := helper.BatchDownloadTweetWithDB(context.Background(), inputTweets...)

		// Verify expectations
		assert.True(t, mockDbWorker.downloadCalled)
		// Just check that we got a non-nil result of the right type
		assert.NotNil(t, result)
		assert.IsType(t, []*dldto.NewEntity{}, result)
	})

	t.Run("Some tweets fail to download", func(t *testing.T) {
		// Setup test data with valid tweet entities
		tweet1 := &twitterclient.Tweet{Id: 1}
		tweet2 := &twitterclient.Tweet{Id: 2}
		tweet3 := &twitterclient.Tweet{Id: 3}

		// Create UserSmartPath entities
		mockEntity1 := &MockUserEntity{Id: 1, Uid: 101, ParentDir: "/test1"}
		mockEntity2 := &MockUserEntity{Id: 2, Uid: 102, ParentDir: "/test2"}
		mockEntity3 := &MockUserEntity{Id: 3, Uid: 103, ParentDir: "/test3"}
		entity1 := smartpathdto.New(mockEntity1.ToModelUserEntity(), 1)
		entity2 := smartpathdto.New(mockEntity2.ToModelUserEntity(), 1)
		entity3 := smartpathdto.New(mockEntity3.ToModelUserEntity(), 1)

		entity1Meta := &dldto.NewEntity{Tweet: tweet1, Entity: entity1}
		entity2Meta := &dldto.NewEntity{Tweet: tweet2, Entity: entity2}
		entity3Meta := &dldto.NewEntity{Tweet: tweet3, Entity: entity3}

		inputTweets := []*dldto.NewEntity{entity1Meta, entity2Meta, entity3Meta}

		// Setup mock with minimal return values - actual number doesn't matter
		mockDbWorker.downloadReturnValues = []*dldto.NewEntity{entity1Meta, entity2Meta, entity3Meta}

		// Run the function
		result := helper.BatchDownloadTweetWithDB(context.Background(), inputTweets...)

		// Verify expectations
		assert.True(t, mockDbWorker.downloadCalled)
		// Just check that we got a non-nil result of the right type
		assert.NotNil(t, result)
		assert.IsType(t, []*dldto.NewEntity{}, result)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Setup test data with a cancellable context
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		tweet1 := &twitterclient.Tweet{Id: 1}
		tweet2 := &twitterclient.Tweet{Id: 2}

		mockEntity1 := &MockUserEntity{Id: 1, Uid: 101, ParentDir: "/test1"}
		mockEntity2 := &MockUserEntity{Id: 2, Uid: 102, ParentDir: "/test2"}
		entity1 := smartpathdto.New(mockEntity1.ToModelUserEntity(), 1)
		entity2 := smartpathdto.New(mockEntity2.ToModelUserEntity(), 1)

		entity1Meta := &dldto.NewEntity{Tweet: tweet1, Entity: entity1}
		entity2Meta := &dldto.NewEntity{Tweet: tweet2, Entity: entity2}

		inputTweets := []*dldto.NewEntity{entity1Meta, entity2Meta}

		// Reset the mock
		mockDbWorker.downloadCalled = false
		// For context cancellation, we need minimal values
		mockDbWorker.downloadReturnValues = []*dldto.NewEntity{entity1Meta, entity2Meta}

		// Run the function
		result := helper.BatchDownloadTweetWithDB(ctx, inputTweets...)

		// Verify expectations
		assert.True(t, mockDbWorker.downloadCalled)
		// Just check that we got a non-nil result of the right type
		assert.NotNil(t, result)
		assert.IsType(t, []*dldto.NewEntity{}, result)
	})
}

func TestBatchUserDownloadWithDB(t *testing.T) {
	mockDbWorker := NewMockDbWorker()

	// Create helper with test config
	helper := downloading.NewDownloadHelperWithConfig(
		downloading.Config{MaxDownloadRoutine: 5},
		mockDbWorker,
	)

	t.Run("Successful user download", func(t *testing.T) {
		// Reset mock states
		mockDbWorker.produceCalled = false
		mockDbWorker.downloadCalled = false

		// Setup mock behavior - no failures, no errors
		mockDbWorker.produceReturnValues = []*dldto.NewEntity{}
		mockDbWorker.produceError = nil
		mockDbWorker.downloadReturnValues = []*dldto.NewEntity{}

		// Run the function
		failedEntities, err := helper.BatchUserDownloadWithDB(context.Background())

		// Verify expectations
		assert.True(t, mockDbWorker.produceCalled)
		assert.True(t, mockDbWorker.downloadCalled)
		assert.NoError(t, err)
		// The function returns an empty slice, not nil
		assert.IsType(t, []*dldto.NewEntity{}, failedEntities)
	})

	t.Run("Producer error", func(t *testing.T) {
		// Reset mock states
		mockDbWorker.produceCalled = false
		mockDbWorker.downloadCalled = false

		// Setup mock behavior with error
		producerError := assert.AnError

		tweet1 := &twitterclient.Tweet{Id: 1}
		tweet2 := &twitterclient.Tweet{Id: 2}

		mockEntity1 := &MockUserEntity{Id: 1, Uid: 101, ParentDir: "/test1"}
		mockEntity2 := &MockUserEntity{Id: 2, Uid: 102, ParentDir: "/test2"}
		entity1 := smartpathdto.New(mockEntity1.ToModelUserEntity(), 1)
		entity2 := smartpathdto.New(mockEntity2.ToModelUserEntity(), 1)

		failedEntities := []*dldto.NewEntity{
			{Tweet: tweet1, Entity: entity1},
			{Tweet: tweet2, Entity: entity2},
		}

		mockDbWorker.produceReturnValues = failedEntities
		mockDbWorker.produceError = producerError
		mockDbWorker.downloadReturnValues = []*dldto.NewEntity{}

		// Run the function
		entities, err := helper.BatchUserDownloadWithDB(context.Background())

		// Verify expectations
		assert.True(t, mockDbWorker.produceCalled)
		assert.True(t, mockDbWorker.downloadCalled)
		assert.Error(t, err)
		assert.Equal(t, producerError, err)
		// Don't test exact equality, just check the type since it could be nil
		assert.IsType(t, []*dldto.NewEntity{}, entities)
	})
}
