package types

import (
	"encoding/json"
	"time"
)

type Duration time.Duration

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

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
