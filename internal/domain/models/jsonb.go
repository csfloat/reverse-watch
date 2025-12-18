package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Jsonb map[string]interface{}

func (j Jsonb) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *Jsonb) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion as []byte failed")
	}
	return json.Unmarshal(b, j)
}
