package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Cursor struct {
	ID         Snowflake
	ReversedAt uint64
}

func (c *Cursor) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Encode())
}

func (c *Cursor) UnmarshalJSON(data []byte) error {
	cursor, err := ToCursor(string(data))
	if err != nil {
		return err
	}
	*c = *cursor
	return nil
}

func (c *Cursor) Encode() string {
	cursor := strconv.FormatUint(c.ReversedAt, 10) + ":" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(cursor))
}

func ToCursor(str string) (*Cursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	cursorStr := string(bytes)
	parts := strings.Split(cursorStr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor")
	}

	reversedAt, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}

	id, err := ToSnowflake(parts[1])
	if err != nil {
		return nil, err
	}

	return &Cursor{
		ID:         id,
		ReversedAt: reversedAt,
	}, nil
}
