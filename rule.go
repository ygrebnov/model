package model

import (
	"github.com/ygrebnov/model/internal/rule"

	"github.com/ygrebnov/model/internal/rules"
)

// Rule defines a named validation function for a specific field type.
type Rule interface{}

// NewRule creates a new Rule with the given name and validation function.
// The validation function must accept a value of type TField and optional string parameters,
// returning an error if validation fails or nil if it passes.
// An error is returned if the name is empty or the function is nil.
func NewRule[TField any](name string, fn func(v TField, params ...string) error) (Rule, error) {
	return rules.NewRule(name, fn)
}
