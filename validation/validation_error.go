package validation

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"

	errorspkg "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// Error accumulates multiple FieldError entries.
// It implements error and unwraps to errors.Join of underlying causes so
// errors.Is/As continue to work for callers.
type Error struct {
	mu     sync.Mutex
	issues []FieldError
}

// Add appends a FieldError.
func (ve *Error) Add(fe FieldError) {
	if ve == nil {
		return
	}
	ve.mu.Lock()
	defer ve.mu.Unlock()
	ve.issues = append(ve.issues, fe)
}

// Addf is a convenience to add from parts.
func (ve *Error) Addf(path, rule string, err error) {
	ve.Add(FieldError{Path: path, Rule: rule, Err: err})
}

// Len returns the number of accumulated issues.
func (ve *Error) Len() int {
	if ve == nil {
		return 0
	}

	return len(ve.snapshotIssues())
}

// Empty reports whether there are no issues.
func (ve *Error) Empty() bool { return ve.Len() == 0 }

// Error returns a human-readable, multi-line description of all issues.
func (ve *Error) Error() string {
	if ve == nil {
		return ""
	}

	issues := ve.snapshotIssues()
	if len(issues) == 0 {
		return ""
	}

	var b strings.Builder
	for i, fe := range issues {
		b.WriteString(fe.Error())
		if i < len(issues)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Unwrap joins underlying causes so errors.Is/As keep working on the combined error.
func (ve *Error) Unwrap() error {
	if ve == nil {
		return nil
	}

	issues := ve.snapshotIssues()
	errs := make([]error, 0, len(issues))
	for _, fe := range issues {
		if fe.Err != nil {
			errs = append(errs, fe.Err)
		}
	}
	return errors.Join(errs...)
}

// ForField returns all issues for a given dotted field path.
func (ve *Error) ForField(path string) []FieldError {
	if ve == nil {
		return nil
	}

	issues := ve.snapshotIssues()
	var out []FieldError
	for _, fe := range issues {
		if fe.Path == path {
			out = append(out, fe)
		}
	}
	return out
}

// ByField groups issues by dotted field path.
func (ve *Error) ByField() map[string][]FieldError {
	m := make(map[string][]FieldError)
	if ve == nil {
		return m
	}

	for _, fe := range ve.snapshotIssues() {
		m[fe.Path] = append(m[fe.Path], fe)
	}
	return m
}

// Fields returns the list of field paths that have issues (unique, order preserved by first occurrence).
func (ve *Error) Fields() []string {
	if ve == nil {
		return nil
	}

	issues := ve.snapshotIssues()
	seen := make(map[string]struct{})
	var out []string
	for _, fe := range issues {
		if _, ok := seen[fe.Path]; !ok {
			seen[fe.Path] = struct{}{}
			out = append(out, fe.Path)
		}
	}
	return out
}

// MarshalJSON exports Error as a map of field path -> list of error messages.
// Example:
//
//	{
//	  "Name": ["must not be empty"],
//	  "Age":  ["must be > 0", "must not be zero"]
//	}
func (ve *Error) MarshalJSON() ([]byte, error) {
	if ve == nil {
		return []byte("null"), nil
	}

	issues := ve.snapshotIssues()
	by := make(map[string][]string, len(issues))
	for _, fe := range issues {
		by[fe.Path] = append(by[fe.Path], errorMessage(fe.Err))
	}
	return json.Marshal(by)
}

func (ve *Error) snapshotIssues() []FieldError {
	if ve == nil {
		return nil
	}

	ve.mu.Lock()
	defer ve.mu.Unlock()

	return slices.Clone(ve.issues)
}

// FieldError represents a single validation failure for a specific field and rule.
// It implements error and unwraps to the underlying cause so callers can use errors.Is/As.
type FieldError struct {
	Path   string   // dotted path to the field (e.g., Address.Street)
	Rule   string   // rule Name that failed
	Params []string // parameters provided to the rule via validate tag
	Err    error    // underlying error from the rule
}

// Error returns a compact human-readable representation of the field error.
func (e FieldError) Error() string {
	var b strings.Builder
	b.WriteString("- Field \"")
	b.WriteString(e.Path)
	b.WriteString("\"")
	if e.Rule != "" {
		b.WriteString(": rule \"")
		b.WriteString(e.Rule)
		b.WriteString("\"")
	}
	if msg := formatFieldErrorMessage(e.Err); msg != "" {
		b.WriteString(": ")
		b.WriteString(msg)
	}
	return b.String()
}

func formatFieldErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	base := errorspkg.GetBase(err)
	if base == nil || !isCompactModelFieldError(base) {
		return err.Error()
	}

	detail := compactErrorDetail(err)
	if detail == "" {
		return err.Error()
	}

	return appendFieldErrorDetail(base.Error(), detail)
}

func appendFieldErrorDetail(base, detail string) string {
	if detail == "" {
		return base
	}

	return base + " (" + detail + ")"
}

func compactErrorDetail(err error) string {
	if err == nil {
		return ""
	}

	msg := errorMessage(err)
	if paramName, ok := extractErrorField(msg, string(keys.RuleParamName)); ok {
		if paramValue, ok := extractErrorField(msg, string(keys.RuleParamValue)); ok {
			return paramName + "=" + paramValue
		}
		return paramName
	}

	valueType, hasValueType := extractErrorField(msg, string(keys.ValueType))
	availableTypes, hasAvailableTypes := extractErrorField(msg, string(keys.FieldAvailableTypes))
	fieldType, hasFieldType := extractErrorField(msg, string(keys.FieldType))

	parts := make([]string, 0, 3)
	if hasValueType {
		parts = append(parts, string(keys.ValueType)+"="+valueType)
	}
	if hasFieldType {
		parts = append(parts, string(keys.FieldType)+"="+fieldType)
	}
	if hasAvailableTypes {
		parts = append(parts, string(keys.FieldAvailableTypes)+"="+availableTypes)
	}

	return strings.Join(parts, ", ")
}

func isCompactModelFieldError(base error) bool {
	switch base {
	case errorspkg.ErrRuleConstraintViolated,
		errorspkg.ErrRuleInvalidParameter,
		errorspkg.ErrRuleMissingParameter,
		errorspkg.ErrRuleTypeMismatch,
		errorspkg.ErrRuleOverloadNotFound,
		errorspkg.ErrRuleNotFound,
		errorspkg.ErrAmbiguousRule,
		errorspkg.ErrInvalidValue:
		return true
	default:
		return false
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

func extractErrorField(msg, key string) (string, bool) {
	prefix := key + ": "
	start := strings.Index(msg, prefix)
	if start < 0 {
		return "", false
	}

	start += len(prefix)
	end := len(msg)
	if next := strings.Index(msg[start:], ", "); next >= 0 {
		end = start + next
	}

	return msg[start:end], true
}

// Unwrap returns the underlying rule error.
func (e FieldError) Unwrap() error { return e.Err }

// MarshalJSON exports FieldError as an object with path, rule, and message fields.
func (e FieldError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Path    string   `json:"path"`
		Rule    string   `json:"rule"`
		Params  []string `json:"Params,omitempty"`
		Message string   `json:"message"`
	}{
		Path:    e.Path,
		Rule:    e.Rule,
		Params:  e.Params,
		Message: errorMessage(e.Err),
	})
}
