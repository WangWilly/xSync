// Package serverdto contains Data Transfer Objects (DTOs) used by the xSync server.
//
// This package defines the data structures used for transferring data between
// the server handlers and clients (both web UI and API consumers).
//
// The DTOs include:
//   - UserStats: User statistics and activity information
//   - DashboardData: Complete dashboard view data
//   - TweetData: Tweet information for display
//   - TweetWithMedia: Tweet data with associated media files
//   - MediaResponse: Media file response data
//   - TweetsWithMediaResponse: Combined tweets and media response
//
// These structures are designed to be JSON-serializable and provide
// a clean separation between internal data models and external APIs.
package serverdto
