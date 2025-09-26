package model

import (
	"encoding/json"
	"fmt"
)

// FieldError represents a single validation failure for a specific field and validationRule.
// It implements error and unwraps to the underlying cause so callers can use errors.Is/As.
type FieldError struct {
	Path   string   // dotted path to the field (e.g., Address.Street)
	Rule   string   // validationRule name that failed
	Params []string // parameters provided to the validationRule via validate tag
	Err    error    // underlying error from the validationRule
}

func (e FieldError) Error() string {
	if e.Rule != "" {
		return fmt.Sprintf("%s: %s (validationRule %s)", e.Path, e.Err, e.Rule)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Err)
}

func (e FieldError) Unwrap() error { return e.Err }

// MarshalJSON exports FieldError as an object with path, validationRule, and message fields.
func (e FieldError) MarshalJSON() ([]byte, error) {
	msg := ""
	if e.Err != nil {
		msg = e.Err.Error()
	}
	return json.Marshal(struct {
		Path    string   `json:"path"`
		Rule    string   `json:"validationRule"`
		Params  []string `json:"params,omitempty"`
		Message string   `json:"message"`
	}{
		Path:    e.Path,
		Rule:    e.Rule,
		Params:  e.Params,
		Message: msg,
	})
}
