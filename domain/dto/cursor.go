package dto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	cpb "reverse-watch/domain/dto/cursorpb"
	"reverse-watch/domain/models"

	"google.golang.org/protobuf/proto"
)

type Cursor struct {
	ID models.Snowflake
}

func (c *Cursor) MarshalJSON() ([]byte, error) {
	cursor, err := c.Marshal()
	if err != nil {
		return nil, err
	}
	return json.Marshal(cursor)
}

func (c *Cursor) UnmarshalJSON(data []byte) error {
	var cursorStr string
	if err := json.Unmarshal(data, &cursorStr); err != nil {
		return err
	}

	cursor, err := UnmarshalCursor(cursorStr)
	if err != nil {
		return err
	}
	*c = *cursor
	return nil
}

func (c *Cursor) Marshal() (*string, error) {
	cursorpb := &cpb.Cursor{
		Id: uint64(c.ID),
	}

	data, err := proto.Marshal(cursorpb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cursor: %s", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	return &encoded, nil
}

func UnmarshalCursor(cursorStr string) (*Cursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %s", err)
	}

	var cursorpb cpb.Cursor
	if err := proto.Unmarshal(bytes, &cursorpb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor: %s", err)
	}

	return &Cursor{
		ID: models.Snowflake(cursorpb.Id),
	}, nil
}
