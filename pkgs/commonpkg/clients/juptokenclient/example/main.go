package main

import (
	"fmt"
	"log"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
)

func main() {
	// Create a new Jupiter client
	client := juptokenclient.New()

	// Example 1: Get all tokens
	fmt.Println("=== Getting all tokens ===")
	allTokens, err := client.GetAllTokens()
	if err != nil {
		log.Fatalf("Error getting all tokens: %v", err)
	}
	fmt.Printf("Total tokens: %d\n", len(allTokens))

	// Show first 5 tokens
	for i, token := range allTokens {
		if i >= 5 {
			break
		}
		fmt.Printf("Token %d: %s (%s) - %s\n", i+1, token.Name, token.Symbol, token.Address)
	}

	// Example 2: Get verified tokens only
	fmt.Println("\n=== Getting verified tokens ===")
	verifiedTokens, err := client.GetVerifiedTokens()
	if err != nil {
		log.Fatalf("Error getting verified tokens: %v", err)
	}
	fmt.Printf("Verified tokens: %d\n", len(verifiedTokens))

	// Example 3: Get tokens by specific tags
	fmt.Println("\n=== Getting tokens by tags (verified, community) ===")
	taggedTokens, err := client.GetTokensByTags([]string{"verified", "community"})
	if err != nil {
		log.Fatalf("Error getting tokens by tags: %v", err)
	}
	fmt.Printf("Tagged tokens: %d\n", len(taggedTokens))

	// Example 4: Get a specific token by address (SOL)
	fmt.Println("\n=== Getting specific token (SOL) ===")
	solToken, err := client.GetTokenByAddress("So11111111111111111111111111111111111111112")
	if err != nil {
		log.Fatalf("Error getting SOL token: %v", err)
	}
	fmt.Printf("Token: %s (%s)\n", solToken.Name, solToken.Symbol)
	fmt.Printf("Address: %s\n", solToken.Address)
	fmt.Printf("Decimals: %d\n", solToken.Decimals)
	fmt.Printf("Tags: %v\n", solToken.Tags)
	if solToken.DailyVolume != nil {
		fmt.Printf("Daily Volume: %.2f\n", *solToken.DailyVolume)
	}

	// Example 5: Search tokens by symbol
	fmt.Println("\n=== Searching tokens by symbol (USD) ===")
	usdTokens, err := client.SearchTokensBySymbol("USD")
	if err != nil {
		log.Fatalf("Error searching tokens by symbol: %v", err)
	}
	fmt.Printf("Found %d tokens with 'USD' in symbol:\n", len(usdTokens))
	for i, token := range usdTokens {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s (%s)\n", token.Name, token.Symbol)
	}

	// Example 6: Search tokens by name
	fmt.Println("\n=== Searching tokens by name (Bitcoin) ===")
	bitcoinTokens, err := client.SearchTokensByName("Bitcoin")
	if err != nil {
		log.Fatalf("Error searching tokens by name: %v", err)
	}
	fmt.Printf("Found %d tokens with 'Bitcoin' in name:\n", len(bitcoinTokens))
	for _, token := range bitcoinTokens {
		fmt.Printf("  %s (%s)\n", token.Name, token.Symbol)
	}

	// Example 7: Get token statistics
	fmt.Println("\n=== Token Statistics ===")
	stats, err := client.GetTokenStats()
	if err != nil {
		log.Fatalf("Error getting token stats: %v", err)
	}
	fmt.Printf("Total tokens: %v\n", stats.TotalTokensCount)
	fmt.Printf("Total daily volume: %.2f\n", stats.TotalDailyVolume)
	fmt.Printf("Tokens with volume: %v\n", stats.TokensWithVolume)

	// Show tag counts
	fmt.Println("Tag counts:")
	for tag, count := range stats.TagCounts {
		fmt.Printf("  %s: %d\n", tag, count)
	}

	// Example 8: Get tradable tokens
	fmt.Println("\n=== Getting tradable tokens ===")
	tradableTokens, err := client.GetTradableTokens()
	if err != nil {
		log.Fatalf("Error getting tradable tokens: %v", err)
	}
	fmt.Printf("Tradable tokens: %d\n", len(tradableTokens))
}
