// Package types contains reusable value types supported by model bindings.
package types

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that marshals to and unmarshals from its textual
// representation in JSON and YAML. Numeric values are accepted as nanoseconds.
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

// UnmarshalYAML decodes a duration string or an integer nanosecond count.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!str":
		parsed, err := time.ParseDuration(value.Value)
		if err != nil {
			return err
		}

		*d = Duration(parsed)

		return nil

	case "!!int":
		var nanoseconds int64
		if err := value.Decode(&nanoseconds); err != nil {
			return err
		}

		*d = Duration(time.Duration(nanoseconds))

		return nil

	default:
		return fmt.Errorf(
			"duration must be a YAML string or integer nanosecond count, got %s",
			value.Tag,
		)
	}
}

// MarshalYAML encodes d as a duration string.
func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

// Duration returns d as a time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
