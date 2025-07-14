package main

import (
	"fmt"
	"log"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
)

func main() {
	// Create a new Jupiter client
	client := juptokenclient.New()

	// Test 1: Get a specific token first (faster)
	fmt.Println("=== Testing with specific token (SOL) ===")
	solToken, err := client.GetTokenByAddress("So11111111111111111111111111111111111111112")
	if err != nil {
		log.Fatalf("Error getting SOL token: %v", err)
	}
	fmt.Printf("✓ Successfully retrieved token: %s (%s)\n", solToken.Name, solToken.Symbol)
	fmt.Printf("  Address: %s\n", solToken.Address)
	fmt.Printf("  Decimals: %d\n", solToken.Decimals)
	fmt.Printf("  Tags: %v\n", solToken.Tags)
	if solToken.DailyVolume != nil {
		fmt.Printf("  Daily Volume: %.2f\n", *solToken.DailyVolume)
	}

	// Test 2: Get verified tokens (smaller subset)
	fmt.Println("\n=== Testing with verified tokens ===")
	verifiedTokens, err := client.GetVerifiedTokens()
	if err != nil {
		log.Fatalf("Error getting verified tokens: %v", err)
	}
	fmt.Printf("✓ Successfully retrieved %d verified tokens\n", len(verifiedTokens))

	// Show first 3 verified tokens
	for i, token := range verifiedTokens {
		if i >= 3 {
			break
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, token.Name, token.Symbol)
	}

	// Test 3: Try to get all tokens (this might be slow)
	fmt.Println("\n=== Testing with all tokens (this may take a moment) ===")
	allTokens, err := client.GetAllTokens()
	if err != nil {
		log.Fatalf("Error getting all tokens: %v", err)
	}
	fmt.Printf("✓ Successfully retrieved %d total tokens\n", len(allTokens))

	// Test 4: Search functionality
	fmt.Println("\n=== Testing search functionality ===")
	usdTokens, err := client.SearchTokensBySymbol("USDC")
	if err != nil {
		log.Fatalf("Error searching tokens by symbol: %v", err)
	}
	fmt.Printf("✓ Found %d tokens with 'USDC' in symbol\n", len(usdTokens))

	fmt.Println("\n🎉 All tests passed! The Jupiter client is working correctly.")
}
