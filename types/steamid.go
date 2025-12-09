package types

import "strconv"

type SteamID uint64

func (s SteamID) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s SteamID) IsValid() bool {
	if s < 76561197960265728 {
		return false
	}

	universe := s >> 56
	if universe > 5 {
		return false
	}

	instance := (s >> 32) & 0xFFFFF
	return instance <= 32
}
