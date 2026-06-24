package dto

type SummaryStats struct {
	// SteamIDsSearched is the number of distinct Steam IDs ever looked up via
	// the public user-status endpoint (COUNT(*) of the search_counts table).
	SteamIDsSearched uint64 `json:"steam_ids_searched"`
	// TotalSearches is the total number of lookups across all Steam IDs
	// (SUM(count) of the search_counts table).
	TotalSearches     uint64 `json:"total_searches"`
	TradersFlagged    uint64 `json:"traders_flagged"`
	TradersFlagged24h uint64 `json:"traders_flagged_24h"`
}

type DailyCount struct {
	Date  string `json:"date"`
	Count uint64 `json:"count"`
}
