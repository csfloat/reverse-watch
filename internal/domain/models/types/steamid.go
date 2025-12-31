package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type SteamID uint64

func (s *SteamID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SteamID) UnmarshalJSON(b []byte) error {
	var idStr string
	if err := json.Unmarshal(b, &idStr); err != nil {
		return err
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	*s = SteamID(id)
	return nil
}

func (s *SteamID) String() string {
	return strconv.FormatUint(uint64(*s), 10)
}

func (s *SteamID) IsValid() bool {
	if s == nil {
		return false
	}

	if *s < 76561197960265728 {
		return false
	}

	universe := *s >> 56
	if universe > 5 {
		return false
	}

	instance := (*s >> 32) & 0xFFFFF
	return instance <= 32
}

func ToSteamID(str string) (*SteamID, error) {
	n, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return nil, err
	}

	id := SteamID(n)
	if !id.IsValid() {
		return nil, fmt.Errorf("invalid steam id: %s", str)
	}
	return &id, nil
}
