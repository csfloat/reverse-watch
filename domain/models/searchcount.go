package models

type SearchCount struct {
	SteamID        SteamID `gorm:"primaryKey;not null;autoIncrement:false" json:"steam_id"`
	Count          uint64  `gorm:"not null;default:0" json:"count"`
	LastSearchedAt uint64  `gorm:"autoUpdateTime:milli" json:"last_searched_at"`
}
