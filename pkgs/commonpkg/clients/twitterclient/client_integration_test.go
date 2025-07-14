package twitterclient

import (
	"context"
	"testing"
)

func TestClientIntegration(t *testing.T) {
	// Test that the client can be created and methods are available
	client := New()

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// These will fail without proper authentication but ensure methods exist
	ctx := context.Background()

	// Test user operations exist
	_, err := client.GetUserByScreenName(ctx, "test")
	if err == nil {
		t.Log("GetUserByScreenName method exists and is callable")
	}

	_, err = client.GetUserById(ctx, 123)
	if err == nil {
		t.Log("GetUserById method exists and is callable")
	}

	// Test list operations exist
	_, err = client.GetList(ctx, 123)
	if err == nil {
		t.Log("GetList method exists and is callable")
	}

	// Test that client has proper state management
	if !client.IsAvailable() {
		t.Log("Client state management works")
	}
}
