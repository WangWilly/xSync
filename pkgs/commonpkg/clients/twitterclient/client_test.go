package twitterclient

import (
	"context"
	"net/url"
	"testing"
)

func TestClientCreation(t *testing.T) {
	// Test basic client creation
	client := New()
	if client == nil {
		t.Fatal("New() returned nil client")
	}

	if client.restyClient == nil {
		t.Fatal("Client missing resty client")
	}

	if client.rateLimiter == nil {
		t.Fatal("Client missing rate limiter")
	}
}

func TestClientStateManagement(t *testing.T) {
	client := New()

	// Test initial state
	if !client.IsAvailable() {
		t.Error("New client should be available")
	}

	if client.GetError() != nil {
		t.Error("New client should have no error")
	}

	// Test error setting
	testErr := &TestError{message: "test error"}
	client.SetError(testErr)

	if client.IsAvailable() {
		t.Error("Client with error should not be available")
	}

	if client.GetError() != testErr {
		t.Error("Error not set correctly")
	}

	// Test error clearing
	client.SetError(nil)

	if !client.IsAvailable() {
		t.Error("Client should be available after error cleared")
	}
}

func TestRateLimiter(t *testing.T) {
	rateLimiter := newRateLimiter(true)

	if rateLimiter == nil {
		t.Fatal("newRateLimiter returned nil")
	}

	if rateLimiter.nonBlocking != true {
		t.Error("Rate limiter nonBlocking not set correctly")
	}

	// Test wouldBlock with no limits
	if rateLimiter.wouldBlock("/test/path") {
		t.Error("Empty rate limiter should not block")
	}
}

func TestManagerCreation(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.GetClientCount() != 0 {
		t.Error("New manager should have 0 clients")
	}

	if manager.GetAvailableClientCount() != 0 {
		t.Error("New manager should have 0 available clients")
	}
}

func TestManagerClientOperations(t *testing.T) {
	manager := NewManager()
	client := New()

	// Test adding client
	manager.AddClient(client)

	if manager.GetClientCount() != 1 {
		t.Error("Manager should have 1 client after adding")
	}

	if manager.GetAvailableClientCount() != 1 {
		t.Error("Manager should have 1 available client")
	}

	// Test setting master client
	manager.SetMasterClient(client)

	if manager.GetMasterClient() != client {
		t.Error("Master client not set correctly")
	}

}

func TestClientSelection(t *testing.T) {
	manager := NewManager()
	client := New()
	manager.AddClient(client)

	ctx := context.Background()

	// Test client selection
	selected := manager.SelectClient(ctx, "/test/path")

	if selected != client {
		t.Error("Should select the available client")
	}

	// Test with unavailable client
	client.SetError(&TestError{message: "test error"})

	selected = manager.SelectClient(ctx, "/test/path")

	if selected != nil {
		t.Error("Should not select unavailable client")
	}
}

// Test helper types
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

// Benchmark tests
func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		client := New()
		_ = client
	}
}

func BenchmarkRateLimiterCheck(b *testing.B) {
	rateLimiter := newRateLimiter(true)
	ctx := context.Background()
	testURL, _ := url.Parse("https://api.twitter.com/test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := rateLimiter.check(ctx, testURL)
		if err != nil {
			b.Fatal(err)
		}
	}
}
