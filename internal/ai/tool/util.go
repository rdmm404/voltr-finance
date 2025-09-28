package tool

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/invopop/jsonschema"
)


type DateTime struct {
	time.Time
}

var parseLayouts = []string{
	time.RFC3339Nano,           // "2006-01-02T15:04:05.999999999Z07:00"
	time.RFC3339,               // "2006-01-02T15:04:05Z07:00"
	"2006-01-02 15:04:05 MST",  // "2025-09-27 22:00:00 EDT"
	"2006-01-02 15:04:05 -0700",// "2025-09-27 22:00:00 -0400"
	"2006-01-02 15:04:05",      // naive local/server time
	"2006-01-02",               // date only
}

func (d *DateTime) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		d.Time = time.Time{}
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	var lastErr error
	for _, layout := range parseLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			d.Time = parsed
			return  nil
		} else {
			lastErr = err
		}
	}

	return lastErr
}

func (DateTime) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
	}
}