package twitterclient

import (
	"context"
	"fmt"
)

// DoFollowByUserId sends a follow request to the specified user
func (c *Client) DoFollowByUserId(ctx context.Context, userId uint64) error {
	url := API_HOST + API_FRIENDSHIPS_CREATE

	_, err := c.restyClient.R().SetFormData(map[string]string{
		"user_id": fmt.Sprintf("%d", userId),
	}).SetContext(ctx).Post(url)

	return err
}
