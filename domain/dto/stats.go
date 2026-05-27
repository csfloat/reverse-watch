package dto

type SummaryStats struct {
	TradersIndexed    uint64 `json:"traders_indexed"`
	TradersFlagged    uint64 `json:"traders_flagged"`
	TradersFlagged24h uint64 `json:"traders_flagged_24h"`
}

type DailyCount struct {
	Date  string `json:"date"`
	Count uint64 `json:"count"`
}
