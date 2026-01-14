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

	j.Raw = append([]byte(nil), bytes...)
	return nil
}

func (j *RawJsonb) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Raw)
}

func (j *RawJsonb) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.Raw)
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
