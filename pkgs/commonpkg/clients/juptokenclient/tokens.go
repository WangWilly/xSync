package juptokenclient

import (
	"context"
	"fmt"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

const (
	TOKENS_ENDPOINT          = "/tokens"
	TRADABLE_TOKENS_ENDPOINT = "/tokens_with_markets"
)

const (
	// All tokens would either be verified or unknown.
	TOKEN_TAG_VERIFIED = "verified"
	TOKEN_TAG_UNKNOWN  = "unknown"

	TOKEN_TAG_COMMUNITY = "community"        // Tokens that are verified by the Jupiter community. To get a community tag for your project, go to https://catdetlist.jup.ag
	TOKEN_TAG_STRICT    = "strict"           // Tokens that were validated previously in the strict-list repo. This repo will be deprecated, please use the community site to get a community tag going forward.
	TOKEN_TAG_LST       = "lst"              // Sanctum’s list from their repo which we automatically pull: https://github.com/igneous-labs/sanctum-lst-list/blob/master/sanctum-lst-list.toml
	TOKEN_TAG_BIRDEYE   = "birdeye-trending" // Top 100 trending tokens from birdeye: https://birdeye.so/find-gems?chain=solana
	TOKEN_TAG_CLONE     = "clone"            // Tokens that are clones of other tokens, e.g. meme coins
	TOKEN_TAG_PUMP      = "pump"             // Tokens that graduated from pump, from their API
)

////////////////////////////////////////////////////////////////////////////////

// GetVerifiedTokens retrieves only verified tokens
func (c *client) GetVerifiedTokens(ctx context.Context) ([]JupTokenDto, error) {
	return c.GetTokensByTags(ctx, []string{TOKEN_TAG_VERIFIED})
}

// GetTokensByTags retrieves tokens filtered by specific tags
func (c *client) GetTokensByTags(ctx context.Context, tags []string) ([]JupTokenDto, error) {
	if len(tags) == 0 {
		return c.GetAllTokens(ctx)
	}

	params := make(map[string]string)
	params["tags"] = strings.Join(tags, ",")
	return c.getTokens(ctx, TOKENS_ENDPOINT, params)
}

// GetAllTokens retrieves all listed tokens from Jupiter
func (c *client) GetAllTokens(ctx context.Context) ([]JupTokenDto, error) {
	return c.getTokens(ctx, TOKENS_ENDPOINT, nil)
}

// GetTradableTokens retrieves only tokens that have active markets
func (c *client) GetTradableTokens(ctx context.Context) ([]JupTokenDto, error) {
	return c.getTokens(ctx, TRADABLE_TOKENS_ENDPOINT, nil)
}

// getTokens is a helper method to retrieve tokens from any endpoint
func (c *client) getTokens(ctx context.Context, endpoint string, params map[string]string) ([]JupTokenDto, error) {
	var tokens []JupTokenDto

	req := c.restyClient.R().SetContext(ctx).SetResult(&tokens)

	// Add query parameters if provided
	for key, value := range params {
		req.SetQueryParam(key, value)
	}

	resp, err := req.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens from %s: %w", endpoint, err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API returned status %d for endpoint %s", resp.StatusCode(), endpoint)
	}

	return tokens, nil
}

////////////////////////////////////////////////////////////////////////////////

// SearchTokensByName searches for tokens by name (case-insensitive partial match)
func (c *client) SearchTokensByName(ctx context.Context, name string) ([]JupTokenDto, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	allTokens, err := c.GetAllTokens(ctx)
	if err != nil {
		return nil, err
	}

	var matchingTokens []JupTokenDto
	lowerName := strings.ToLower(name)

	for _, token := range allTokens {
		tokenName := strings.ToLower(token.Name)
		if strings.Contains(tokenName, lowerName) {
			matchingTokens = append(matchingTokens, token)
		}
	}

	return matchingTokens, nil
}

// SearchTokensBySymbol searches for tokens by symbol (case-insensitive)
func (c *client) SearchTokensBySymbol(ctx context.Context, symbol string) ([]JupTokenDto, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	allTokens, err := c.GetAllTokens(ctx)
	if err != nil {
		return nil, err
	}

	var matchingTokens []JupTokenDto
	lowerSymbol := strings.ToLower(symbol)

	for _, token := range allTokens {
		tokenSymbol := strings.ToLower(token.Symbol)
		if strings.Contains(tokenSymbol, lowerSymbol) {
			matchingTokens = append(matchingTokens, token)
		}
	}

	return matchingTokens, nil
}

////////////////////////////////////////////////////////////////////////////////

// GetTokenByAddress retrieves a specific token by its mint address
func (c *client) GetTokenByAddress(ctx context.Context, address string) (*JupTokenDto, error) {
	if address == "" {
		return nil, fmt.Errorf("token address cannot be empty")
	}

	var token JupTokenDto
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetResult(&token).
		Get(fmt.Sprintf("/token/%s", address))

	if err != nil {
		return nil, fmt.Errorf("failed to get token %s: %w", address, err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API returned status %d for token %s", resp.StatusCode(), address)
	}

	return &token, nil
}

////////////////////////////////////////////////////////////////////////////////

// GetTokenStats returns basic statistics about the token list
func (c *client) GetTokenStats(ctx context.Context) (*JupTokenStatsDto, error) {
	allTokens, err := c.GetAllTokens(ctx)
	if err != nil {
		return nil, err
	}

	stats := &JupTokenStatsDto{}
	stats.TotalTokensCount = len(allTokens)

	// Count by tags
	tagCounts := make(map[string]int)
	var totalVolume float64
	var tokensWithVolume int

	for _, token := range allTokens {
		for _, tag := range token.Tags {
			tagCounts[tag]++
		}

		if token.DailyVolume != nil {
			totalVolume += *token.DailyVolume
			tokensWithVolume++
		}
	}

	stats.TagCounts = tagCounts
	stats.TotalDailyVolume = totalVolume
	stats.TokensWithVolume = tokensWithVolume

	return stats, nil
}
