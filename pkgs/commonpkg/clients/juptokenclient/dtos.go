package juptokenclient

import (
	"encoding/json"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
)

type JupTokenDto struct {
	Address           string            `json:"address"`
	Name              string            `json:"name"`
	Symbol            string            `json:"symbol"`
	Decimals          int               `json:"decimals"`
	LogoURI           string            `json:"logoURI"`
	Tags              []string          `json:"tags"`
	DailyVolume       *float64          `json:"daily_volume"`
	CreatedAt         string            `json:"created_at"`
	FreezeAuthority   *string           `json:"freeze_authority"`
	MintAuthority     *string           `json:"mint_authority"`
	PermanentDelegate *string           `json:"permanent_delegate"`
	MintedAt          *string           `json:"minted_at"`
	Extensions        map[string]string `json:"extensions"`
}

func BatchNewJupTokenDtoFromModel(tokens []model.Token) []JupTokenDto {
	var dtos []JupTokenDto
	for _, token := range tokens {
		dtos = append(dtos, *NewJupTokenDtoFromModel(&token))
	}
	return dtos
}

func NewJupTokenDtoFromModel(token *model.Token) *JupTokenDto {
	return &JupTokenDto{
		Address:  token.Address,
		Decimals: token.Decimals,
		Name:     token.Name,
		Symbol:   token.Symbol,
		LogoURI:  token.LogoURI,
		Tags:     parseTagsFromJSON(token.Tags),
	}
}

func parseTagsFromJSON(tagsJSON string) []string {
	if tagsJSON == "" || tagsJSON == "null" {
		return []string{}
	}

	var tags []string
	err := json.Unmarshal([]byte(tagsJSON), &tags)
	if err != nil {
		// If JSON parsing fails, return empty slice
		return []string{}
	}

	return tags
}

////////////////////////////////////////////////////////////////////////////////

type JupTokenStatsDto struct {
	TotalTokensCount int
	TotalDailyVolume float64
	TokensWithVolume int
	TagCounts        map[string]int
}
