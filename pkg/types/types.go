// Package types contains reusable value types supported by model bindings.
package types

import (
	"encoding/json"
	"time"
)

// Duration is a time.Duration that marshals to and unmarshals from its textual
// representation in JSON. Numeric JSON values are accepted as nanoseconds.
type Duration time.Duration

// UnmarshalJSON decodes a duration string or a numeric nanosecond count.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = Duration(parsed)
		return nil
	}

	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}

	*d = Duration(time.Duration(n))
	return nil
}

// MarshalJSON encodes d as a duration string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// Duration returns d as a time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
