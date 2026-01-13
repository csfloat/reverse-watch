package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type RawJsonb struct {
	Raw json.RawMessage
}

func (j *RawJsonb) Value() (driver.Value, error) {
	if j == nil || j.Raw == nil {
		return nil, nil
	}
	return []byte(j.Raw), nil
}

func (j *RawJsonb) Scan(value interface{}) error {
	if value == nil {
		j.Raw = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion as []byte failed")
	}

	j.Raw = bytes
	return nil
}

func ToRawJsonb(value interface{}) (*RawJsonb, error) {
	if value == nil {
		return nil, fmt.Errorf("cannot convert nil to RawJsonb")
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &RawJsonb{
		Raw: bytes,
	}, nil
}
