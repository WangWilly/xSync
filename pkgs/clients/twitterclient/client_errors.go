package twitterclient

import "fmt"

// Error definitions
var (
	ErrWouldBlock = fmt.Errorf("EWOULDBLOCK")
)

// isKnownError checks if an error is a known Twitter API error
func isKnownError(err error) bool {
	// This would need to be implemented based on the specific error types
	// from the twitter package
	return false
}
