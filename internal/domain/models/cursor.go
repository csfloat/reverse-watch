package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	cpb "reverse-watch/internal/domain/models/cursorpb"

	"google.golang.org/protobuf/proto"
)

type Cursor struct {
	ID         Snowflake
	ReversedAt uint64
}

func (c *Cursor) MarshalJSON() ([]byte, error) {
	cursor, err := c.Encode()
	if err != nil {
		return nil, err
	}
	return json.Marshal(cursor)
}

func (c *Cursor) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	cursor, err := DecodeCursor(encoded)
	if err != nil {
		return err
	}
	*c = *cursor
	return nil
}

func (c *Cursor) Encode() (*string, error) {
	cursorpb := &cpb.Cursor{
		Snowflake:  uint64(c.ID),
		ReversedAt: c.ReversedAt,
	}

	data, err := proto.Marshal(cursorpb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cursor: %s", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	return &encoded, nil
}

func DecodeCursor(encoded string) (*Cursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %s", err)
	}

	var cursorpb cpb.Cursor
	if err := proto.Unmarshal(bytes, &cursorpb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor: %s", err)
	}

	return &Cursor{
		ID:         Snowflake(cursorpb.Snowflake),
		ReversedAt: cursorpb.ReversedAt,
	}, nil
}
