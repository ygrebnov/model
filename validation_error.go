package model

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

// ValidationError accumulates multiple FieldError entries.
// It implements error and unwraps to errors.Join of underlying causes so
// errors.Is/As continue to work for callers.
type ValidationError struct {
	mu     sync.Mutex
	issues []FieldError
}

// Add appends a FieldError.
func (ve *ValidationError) Add(fe FieldError) {
	if ve == nil {
		return
	}
	ve.mu.Lock()
	ve.issues = append(ve.issues, fe)
	ve.mu.Unlock()
}

// Addf is a convenience to add from parts.
func (ve *ValidationError) Addf(path, rule string, err error) {
	ve.Add(FieldError{Path: path, Rule: rule, Err: err})
}

// Len returns the number of accumulated issues.
func (ve *ValidationError) Len() int {
	if ve == nil {
		return 0
	}
	ve.mu.Lock()
	n := len(ve.issues)
	ve.mu.Unlock()
	return n
}

// Empty reports whether there are no issues.
func (ve *ValidationError) Empty() bool { return ve.Len() == 0 }

// Error returns a human-readable, multi-line description of all issues.
func (ve *ValidationError) Error() string {
	if ve == nil {
		return ""
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	switch len(ve.issues) {
	case 0:
		return ""
	case 1:
		return ve.issues[0].Error()
	default:
		var b strings.Builder
		b.WriteString("validation failed (\n")
		for i, fe := range ve.issues {
			b.WriteString("  ")
			b.WriteString(fe.Error())
			if i < len(ve.issues)-1 {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n)")
		return b.String()
	}
}

// Unwrap joins underlying causes so errors.Is/As keep working on the combined error.
func (ve *ValidationError) Unwrap() error {
	if ve == nil {
		return nil
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	errs := make([]error, 0, len(ve.issues))
	for _, fe := range ve.issues {
		if fe.Err != nil {
			errs = append(errs, fe.Err)
		}
	}
	return errors.Join(errs...)
}

// ForField returns all issues for a given dotted field path.
func (ve *ValidationError) ForField(path string) []FieldError {
	if ve == nil {
		return nil
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	var out []FieldError
	for _, fe := range ve.issues {
		if fe.Path == path {
			out = append(out, fe)
		}
	}
	return out
}

// ByField groups issues by dotted field path.
func (ve *ValidationError) ByField() map[string][]FieldError {
	m := make(map[string][]FieldError)
	if ve == nil {
		return m
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	for _, fe := range ve.issues {
		m[fe.Path] = append(m[fe.Path], fe)
	}
	return m
}

// Fields returns the list of field paths that have issues (unique, order preserved by first occurrence).
func (ve *ValidationError) Fields() []string {
	if ve == nil {
		return nil
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for _, fe := range ve.issues {
		if _, ok := seen[fe.Path]; !ok {
			seen[fe.Path] = struct{}{}
			out = append(out, fe.Path)
		}
	}
	return out
}

// MarshalJSON exports ValidationError as a map of field path -> list of error messages.
// Example:
//
//	{
//	  "Name": ["must not be empty"],
//	  "Age":  ["must be > 0", "must not be zero"]
//	}
func (ve *ValidationError) MarshalJSON() ([]byte, error) {
	if ve == nil {
		return []byte("null"), nil
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	by := make(map[string][]string, len(ve.issues))
	for _, fe := range ve.issues {
		msg := ""
		if fe.Err != nil {
			msg = fe.Err.Error()
		}
		by[fe.Path] = append(by[fe.Path], msg)
	}
	return json.Marshal(by)
}
