package juptokenclient

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

type JupTokenStatsDto struct {
	TotalTokensCount int
	TotalDailyVolume float64
	TokensWithVolume int
	TagCounts        map[string]int
}
