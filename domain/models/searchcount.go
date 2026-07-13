package models

type SearchCount struct {
	SteamID SteamID `gorm:"primaryKey;not null;autoIncrement:false" json:"steam_id"`
	Count   uint64  `gorm:"not null;default:0" json:"count"`
	// LastSearchedAt is the timestamp of the most recent lookup in
	// milliseconds since the Unix epoch.
	LastSearchedAt uint64 `gorm:"autoUpdateTime:milli" json:"last_searched_at"`
}
