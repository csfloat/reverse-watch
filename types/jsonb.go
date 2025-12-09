package types

import (
	"database/sql/driver"
	"encoding/json"

	"reverse-watch/errors"
)

type Jsonb map[string]interface{}

func (j Jsonb) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *Jsonb) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New(errors.InternalServerError, "type assertion as []byte failed")
	}
	return json.Unmarshal(b, j)
}
