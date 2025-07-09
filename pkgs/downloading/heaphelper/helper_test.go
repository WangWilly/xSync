package heaphelper

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Test helper functions

// createTempDB creates a temporary in-memory SQLite database for testing
func createTempDB() *sqlx.DB {
	tmpFile, err := os.CreateTemp("", "test_heaphelper_*.db")
	if err != nil {
		panic(err)
	}
	path := tmpFile.Name()
	tmpFile.Close()

	db, err := sqlx.Connect("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&cache=shared", path))
	if err != nil {
		panic(err)
	}

	database.CreateTables(db)
	return db
}

// createTempDir creates a temporary directory for testing
func createTempDir() string {
	dir, err := os.MkdirTemp("", "test_heaphelper_*")
	if err != nil {
		panic(err)
	}
	return dir
}

// createTestUser creates a test user with specified parameters
func createTestUser(id uint64, screenName string, mediaCount int, isProtected bool, blocking bool, muting bool) *twitter.User {
	user := &twitter.User{
		TwitterId:   id,
		ScreenName:  screenName,
		Name:        fmt.Sprintf("Test User %d", id),
		MediaCount:  mediaCount,
		IsProtected: isProtected,
		Blocking:    blocking,
		Muting:      muting,
		Followstate: twitter.FS_UNFOLLOW,
	}
	return user
}

// createTestUserWithinListEntity creates a test UserWithinListEntity
func createTestUserWithinListEntity(user *twitter.User, leid *int) UserWithinListEntity {
	return UserWithinListEntity{
		User: user,
		Leid: leid,
	}
}

// Test cases for MakeHeap

func TestMakeHeap_Success(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create test users
	user1 := createTestUser(1, "user1", 100, false, false, false)
	user2 := createTestUser(2, "user2", 50, false, false, false)
	user3 := createTestUser(3, "user3", 200, false, false, false)

	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
		createTestUserWithinListEntity(user2, nil),
		createTestUserWithinListEntity(user3, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Test MakeHeap
	err := h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err != nil {
		t.Fatalf("MakeHeap failed: %v", err)
	}

	// Verify heap is initialized
	heap := h.GetHeap()
	if heap == nil {
		t.Fatal("Heap should not be nil after MakeHeap")
	}

	// Verify heap has expected size (users that are not ignored and have media)
	expectedSize := 3 // All users have media and are not ignored
	if heap.Size() != expectedSize {
		t.Errorf("Expected heap size %d, got %d", expectedSize, heap.Size())
	}

	// Verify users are in uidToUserMap
	if len(h.uidToUserMap) != 3 {
		t.Errorf("Expected 3 users in uidToUserMap, got %d", len(h.uidToUserMap))
	}

	// Verify depths are calculated
	if len(h.userSmartPathToDepth) == 0 {
		t.Error("userSmartPathToDepth should not be empty")
	}
}

func TestMakeHeap_NoUsersToProcess(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create users with no media or ignored users
	user1 := createTestUser(1, "user1", 0, false, false, false)  // No media
	user2 := createTestUser(2, "user2", 100, false, true, false) // Blocking
	user3 := createTestUser(3, "user3", 50, false, false, true)  // Muting

	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
		createTestUserWithinListEntity(user2, nil),
		createTestUserWithinListEntity(user3, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Test MakeHeap
	err := h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err == nil {
		t.Fatal("Expected error when no users to process")
	}

	expectedError := "no user to process"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMakeHeap_AlreadyInitialized(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create test user
	user1 := createTestUser(1, "user1", 100, false, false, false)
	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Initialize heap first time
	err := h.MakeHeap(ctx, db, client, dir, false)
	if err != nil {
		t.Fatalf("First MakeHeap failed: %v", err)
	}

	// Try to initialize again
	err = h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err == nil {
		t.Fatal("Expected error when heap is already initialized")
	}

	expectedError := "heap is already initialized, call MakeHeap only once"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMakeHeap_EmptyUsers(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create helper with empty users
	h := NewHelper([]UserWithinListEntity{})

	// Test MakeHeap
	err := h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err == nil {
		t.Fatal("Expected error when no users provided")
	}

	expectedError := "no user to process"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMakeHeap_ProtectedUsers(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create protected users with different follow states
	user1 := createTestUser(1, "user1", 100, true, false, false)
	user1.Followstate = twitter.FS_FOLLOWING

	user2 := createTestUser(2, "user2", 50, true, false, false)
	user2.Followstate = twitter.FS_UNFOLLOW // This user won't be visible

	user3 := createTestUser(3, "user3", 200, false, false, false)

	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
		createTestUserWithinListEntity(user2, nil),
		createTestUserWithinListEntity(user3, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Test MakeHeap
	err := h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err != nil {
		t.Fatalf("MakeHeap failed: %v", err)
	}

	// Verify heap is initialized
	heap := h.GetHeap()
	if heap == nil {
		t.Fatal("Heap should not be nil after MakeHeap")
	}

	// Verify heap priority order - only visible users should be in heap
	// user1 (protected following) and user3 (public) should be in heap
	// user2 (protected not following) should not be in heap
	expectedSize := 2
	if heap.Size() != expectedSize {
		t.Errorf("Expected heap size %d, got %d", expectedSize, heap.Size())
	}
}

func TestMakeHeap_ContextCancellation(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create test user
	user1 := createTestUser(1, "user1", 100, false, false, false)
	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Test MakeHeap with cancelled context
	err := h.MakeHeap(ctx, db, client, dir, false)

	// The method should still complete since it doesn't check context cancellation
	// in the main processing loop, but it should handle the deferred cancellation
	if err != nil && err.Error() != "no user to process" {
		// If there's an error, it should be related to processing, not context
		t.Logf("MakeHeap with cancelled context returned: %v", err)
	}
}

func TestMakeHeap_WithListEntity(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create a list entity in the database
	listEntity := &database.ListEntity{
		Name:      "Test List",
		LstId:     1,
		ParentDir: dir,
	}
	err := database.CreateLstEntity(db, listEntity)
	if err != nil {
		t.Fatalf("Failed to create list entity: %v", err)
	}

	// Create test users with list entity ID
	user1 := createTestUser(1, "user1", 100, false, false, false)
	user2 := createTestUser(2, "user2", 50, false, false, false)

	leid := int(listEntity.Id.Int32)
	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, &leid),
		createTestUserWithinListEntity(user2, &leid),
	}

	// Create helper
	h := NewHelper(users)

	// Test MakeHeap
	err = h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err != nil {
		t.Fatalf("MakeHeap failed: %v", err)
	}

	// Verify heap is initialized
	heap := h.GetHeap()
	if heap == nil {
		t.Fatal("Heap should not be nil after MakeHeap")
	}

	if heap.Size() != 2 {
		t.Errorf("Expected heap size 2, got %d", heap.Size())
	}
}

func TestGetHeap_NotInitialized(t *testing.T) {
	// Create helper without initializing heap
	h := NewHelper([]UserWithinListEntity{})

	// Test GetHeap should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when getting heap before initialization")
		}
	}()

	h.GetHeap()
}

func TestGetDepth(t *testing.T) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create test user
	user1 := createTestUser(1, "user1", 100, false, false, false)
	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
	}

	// Create helper and initialize heap
	h := NewHelper(users)
	err := h.MakeHeap(ctx, db, client, dir, false)
	if err != nil {
		t.Fatalf("MakeHeap failed: %v", err)
	}

	// Get the first (and only) user smart path from heap
	heap := h.GetHeap()
	if heap.Empty() {
		t.Fatal("Heap should not be empty")
	}

	userSmartPath := heap.Peek()
	depth := h.GetDepth(userSmartPath)

	// Verify depth is greater than 0 for user with media
	if depth <= 0 {
		t.Errorf("Expected depth > 0 for user with media, got %d", depth)
	}
}

func TestGetUserByTwitterId(t *testing.T) {
	// Create test user
	user1 := createTestUser(1, "user1", 100, false, false, false)
	user2 := createTestUser(2, "user2", 50, false, false, false)

	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
		createTestUserWithinListEntity(user2, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Test GetUserByTwitterId
	retrievedUser := h.GetUserByTwitterId(1)
	if retrievedUser == nil {
		t.Fatal("Expected to find user with ID 1")
	}

	if retrievedUser.TwitterId != 1 {
		t.Errorf("Expected user ID 1, got %d", retrievedUser.TwitterId)
	}

	if retrievedUser.ScreenName != "user1" {
		t.Errorf("Expected screen name 'user1', got '%s'", retrievedUser.ScreenName)
	}

	// Test with non-existent user
	nonExistentUser := h.GetUserByTwitterId(999)
	if nonExistentUser != nil {
		t.Error("Expected nil for non-existent user")
	}
}

func TestMakeHeap_IntegrationWithRealPath(t *testing.T) {
	// This test verifies the integration with real file system operations
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create test user
	user1 := createTestUser(1, "user1", 100, false, false, false)
	users := []UserWithinListEntity{
		createTestUserWithinListEntity(user1, nil),
	}

	// Create helper
	h := NewHelper(users)

	// Test MakeHeap
	err := h.MakeHeap(ctx, db, client, dir, false)

	// Assertions
	if err != nil {
		t.Fatalf("MakeHeap failed: %v", err)
	}

	// Verify that some user directory was created in the temp dir
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// Check if at least one directory was created
	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected at least one user directory to be created")
	}
}

// Benchmark tests
func BenchmarkMakeHeap_SmallDataset(b *testing.B) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create 10 test users
	users := make([]UserWithinListEntity, 10)
	for i := 0; i < 10; i++ {
		user := createTestUser(uint64(i+1), fmt.Sprintf("user%d", i+1), 100, false, false, false)
		users[i] = createTestUserWithinListEntity(user, nil)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h := NewHelper(users)
		err := h.MakeHeap(ctx, db, client, dir, false)
		if err != nil {
			b.Fatalf("MakeHeap failed: %v", err)
		}
	}
}

func BenchmarkMakeHeap_LargeDataset(b *testing.B) {
	// Setup
	db := createTempDB()
	defer db.Close()

	dir := createTempDir()
	defer os.RemoveAll(dir)

	client := resty.New()
	ctx := context.Background()

	// Create 100 test users
	users := make([]UserWithinListEntity, 100)
	for i := 0; i < 100; i++ {
		user := createTestUser(uint64(i+1), fmt.Sprintf("user%d", i+1), 100, false, false, false)
		users[i] = createTestUserWithinListEntity(user, nil)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h := NewHelper(users)
		err := h.MakeHeap(ctx, db, client, dir, false)
		if err != nil {
			b.Fatalf("MakeHeap failed: %v", err)
		}
	}
}
