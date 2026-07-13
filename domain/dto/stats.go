package dto

import "time"

type SummaryStats struct {
	SteamIDsSearched  uint64 `json:"steam_ids_searched"`
	TotalSearches     uint64 `json:"total_searches"`
	TradersFlagged    uint64 `json:"traders_flagged"`
	TradersFlagged24h uint64 `json:"traders_flagged_24h"`
}

type DailyCount struct {
	Date  time.Time `json:"date"`
	Count uint64    `json:"count"`
}
