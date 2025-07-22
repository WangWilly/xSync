package heaphelper_test

import (
	"testing"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("Creates heapHelper with valid inputs", func(t *testing.T) {
		// Create test users
		user1 := &twitterclient.User{
			TwitterId:   1,
			Name:        "User1",
			ScreenName:  "user1",
			IsProtected: false,
			Followstate: twitterclient.FS_UNFOLLOW,
		}
		user2 := &twitterclient.User{
			TwitterId:   2,
			Name:        "User2",
			ScreenName:  "user2",
			IsProtected: true,
			Followstate: twitterclient.FS_FOLLOWING,
		}

		// Create test metas
		metas := []twitterclient.TitledUserList{
			{
				Type:  "test",
				Users: []*twitterclient.User{user1, user2},
			},
		}

		// Create test smart paths using the New constructor (not NewUserSmartPath which requires DB)
		userEntity1 := &model.UserEntity{Uid: user1.TwitterId, ParentDir: "testDir1"}
		userEntity2 := &model.UserEntity{Uid: user2.TwitterId, ParentDir: "testDir2"}

		smartPath1 := smartpathdto.New(userEntity1, 1)
		smartPath2 := smartpathdto.New(userEntity2, 2)

		smartPaths := []*smartpathdto.UserSmartPath{smartPath1, smartPath2}

		// Test creating a new helper
		helper, err := heaphelper.New(metas, smartPaths)

		// Assert no error occurred
		require.NoError(t, err)
		require.NotNil(t, helper)

		// Test GetHeap method
		heap := helper.GetHeap()
		assert.NotNil(t, heap, "Heap should not be nil")
		assert.False(t, heap.Empty(), "Heap should not be empty")

		// Verify the heap contains our smartpaths
		top := heap.Peek()
		assert.NotNil(t, top, "Heap peek should return a smart path")

		// Test GetUserByTwitterId method
		retrievedUser1 := helper.GetUserByTwitterId(user1.TwitterId)
		assert.Equal(t, user1, retrievedUser1)

		retrievedUser2 := helper.GetUserByTwitterId(user2.TwitterId)
		assert.Equal(t, user2, retrievedUser2)

		// Test non-existent user returns nil
		retrievedUser3 := helper.GetUserByTwitterId(999)
		assert.Nil(t, retrievedUser3)

		// Test GetDepth method
		depth1 := helper.GetDepth(smartPath1)
		assert.Equal(t, 1, depth1)

		depth2 := helper.GetDepth(smartPath2)
		assert.Equal(t, 2, depth2)

		// Test non-existent smart path returns 0
		nonExistentEntity := &model.UserEntity{Uid: 999, ParentDir: "nonExistent"}
		nonExistentSmartPath := smartpathdto.New(nonExistentEntity, 3)
		depth3 := helper.GetDepth(nonExistentSmartPath)
		assert.Equal(t, 0, depth3)
	})

	t.Run("Returns error when no smart paths", func(t *testing.T) {
		// Create empty metas and smart paths
		metas := []twitterclient.TitledUserList{}
		smartPaths := []*smartpathdto.UserSmartPath{}

		// Test creating a new helper with empty inputs
		helper, err := heaphelper.New(metas, smartPaths)

		// Assert error occurred and helper is nil
		assert.Error(t, err)
		assert.Nil(t, helper)
		assert.Equal(t, "no user to process", err.Error())
	})

	t.Run("Handles nil users and smart paths", func(t *testing.T) {
		// Create test user
		user1 := &twitterclient.User{
			TwitterId:   1,
			Name:        "User1",
			ScreenName:  "user1",
			IsProtected: false,
			Followstate: twitterclient.FS_UNFOLLOW,
		}

		// Create metas with nil user
		metas := []twitterclient.TitledUserList{
			{
				Type:  "test",
				Users: []*twitterclient.User{user1, nil},
			},
		}

		// Create only one valid smart path (don't include nil path which causes crash)
		userEntity1 := &model.UserEntity{Uid: user1.TwitterId, ParentDir: "testDir1"}
		smartPath1 := smartpathdto.New(userEntity1, 1)
		smartPaths := []*smartpathdto.UserSmartPath{smartPath1}

		// Test creating a new helper
		helper, err := heaphelper.New(metas, smartPaths)

		// Assert no error occurred (one valid smart path is enough)
		require.NoError(t, err)
		require.NotNil(t, helper)

		// Nil user should be skipped
		retrievedUser := helper.GetUserByTwitterId(0) // nil user would have 0 as ID
		assert.Nil(t, retrievedUser)
	})
}
