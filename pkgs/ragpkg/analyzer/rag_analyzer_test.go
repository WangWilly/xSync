package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPotentialTokens(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "Basic token symbols",
			content:  "Just bought some $BTC and $ETH for my portfolio",
			expected: []string{"BTC", "ETH"},
		},
		{
			name:     "Mixed case and hashtags",
			content:  "Looking at $btc trends and #Ethereum developments",
			expected: []string{"BTC", "ETHEREUM"},
		},
		{
			name:     "No tokens",
			content:  "This is just a regular tweet about weather",
			expected: []string{},
		},
		{
			name:     "Complex tweet",
			content:  "$BTC hitting new highs while $LINK and #DeFi tokens surge",
			expected: []string{"BTC", "LINK", "DEFI"},
		},
		{
			name:     "Duplicate tokens",
			content:  "$BTC is great, I love $BTC and think $btc will moon",
			expected: []string{"BTC"},
		},
		{
			name:     "Edge cases",
			content:  "$A $1 $ABC123 #a #A1 #valid_token",
			expected: []string{"ABC123", "VALID_TOKEN"},
		},
	}

	// Create a simple analyzer instance to test the extraction method
	analyzer := &RAGAnalyzer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractPotentialTokens(tt.content)
			assert.ElementsMatch(t, tt.expected, result, "Expected tokens %v, got %v", tt.expected, result)
		})
	}
}

func TestTokenSymbolRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"$BTC", []string{"BTC"}},
		{"$ETH $LINK", []string{"ETH", "LINK"}},
		{"$btc", []string{"BTC"}}, // Should convert to uppercase
		{"$1", []string{}},        // Should not match single character
		{"$AB", []string{}},       // Should not match two characters
		{"$ABC", []string{"ABC"}}, // Should match three or more characters
		{"Buy $BTC now!", []string{"BTC"}},
		{"$$BTC", []string{"BTC"}}, // Should handle double $
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			analyzer := &RAGAnalyzer{}
			result := analyzer.extractPotentialTokens(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestHashtagRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"#Bitcoin", []string{"BITCOIN"}},
		{"#DeFi #Ethereum", []string{"DEFI", "ETHEREUM"}},
		{"#defi", []string{"DEFI"}}, // Should convert to uppercase
		{"#A", []string{}},          // Should not match single character
		{"#AB", []string{}},         // Should not match two characters
		{"#ABC", []string{"ABC"}},   // Should match three or more characters
		{"Love #Bitcoin community!", []string{"BITCOIN"}},
		{"##Bitcoin", []string{"BITCOIN"}}, // Should handle double #
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			analyzer := &RAGAnalyzer{}
			result := analyzer.extractPotentialTokens(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
