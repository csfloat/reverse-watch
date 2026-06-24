package models

// SearchCount tracks how many times a Steam ID has been looked up via the
// public user-status endpoint. It lives in the public database so the public
// summary stats query can read it directly. The row is upserted on every
// lookup, so the number of rows equals the number of distinct Steam IDs ever
// searched and the sum of Count equals the total number of searches.
type SearchCount struct {
	SteamID SteamID `gorm:"primaryKey;not null;autoIncrement:false" json:"steam_id"`
	Count   uint64  `gorm:"not null;default:0" json:"count"`
	// LastSearchedAt is the timestamp of the most recent lookup in
	// milliseconds since the Unix epoch.
	LastSearchedAt uint64 `gorm:"autoUpdateTime:milli" json:"last_searched_at"`
}
