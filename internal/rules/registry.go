package rules

import (
	"reflect"
	"slices"
	"strings"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// Registry is a registry of validation rules.
type Registry struct {
	rules map[string][]*Rule // rule Name -> overloads by type
}

// NewRegistry creates an empty validation rules registry.
func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string][]*Rule),
	}
}

func (r *Registry) Add(rule *Rule) error {
	if rule == nil {
		return nil
	}

	name := rule.GetName()
	existing, exists := r.rules[name]
	if exists {
		// Prevent duplicate overloads for the same field type.
		for _, er := range existing {
			if er.isOfType(rule.getFieldType()) {
				return errorc.With(
					errors.ErrDuplicateOverloadRule,
					errorc.String(keys.RuleName, name),
					errorc.String(keys.FieldType, rule.GetFieldTypeName()),
				)
			}
		}
	}

	r.rules[name] = append(r.rules[name], rule)

	return nil
}

// GetByValue returns the best-matching overload of rule `Name` for the given field value.
// Selection strategy:
//  1. Prefer exact type match (v.Type() == fieldType).
//  2. Otherwise, accept AssignableTo matches (interfaces, named types), preferring the first declared.
//  3. Otherwise, if no matches, fetch a built-in rule if available.
//  4. If no matches, return a descriptive error listing available overload types.
//  5. If multiple exact matches (shouldn't happen), return an ambiguity error.
func (r *Registry) GetByValue(name string, v reflect.Value) (*Rule, error) {
	if !v.IsValid() {
		return nil,
			errorc.With(errors.ErrInvalidValue, errorc.String(keys.RuleName, name))
	}

	return r.GetByType(name, v.Type())
}

// GetByType returns the best-matching overload of rule `Name` for the given field value type.
func (r *Registry) GetByType(name string, valueType reflect.Type) (*Rule, error) {
	return r.getByType(name, valueType)
}

func (r *Registry) getByType(name string, valueType reflect.Type) (*Rule, error) {
	rules := r.rules[name]

	var (
		exacts  []*Rule
		assigns []*Rule
	)
	for _, rule := range rules {
		if rule.getFieldType() == nil || rule.GetValidationFn() == nil {
			continue // defensive, should not happen due to checks in NewRule
		}
		if rule.isOfType(valueType) {
			exacts = append(exacts, rule)
			continue
		}
		if rule.IsAssignableTo(valueType) {
			assigns = append(assigns, rule)
		}
	}

	switch {
	case len(exacts) == 1:
		return exacts[0], nil
	case len(exacts) > 1:
		// defensive: should not happen due to add() checks
		return nil, errorc.With(
			errors.ErrAmbiguousRule,
			errorc.String(keys.RuleName, name),
			errorc.String(keys.ValueType, valueType.String()),
		)
	case len(assigns) >= 1:
		return assigns[0], nil
	default:
		// No matches; check for built-in rule as fallback.
		builtinRule, hasBuiltin := lookupBuiltin(name, valueType)
		if hasBuiltin {
			return builtinRule, nil
		}

		if len(rules) == 0 {
			// No rules by the given Name neither in rulesRegistry no from in built-ins.
			return nil,
				errorc.With(errors.ErrRuleNotFound, errorc.String(keys.RuleName, name))
		}

		// Some rules exist by the given Name, but none match the value type.
		// Construct helpful message of available overload types.
		return nil, errorc.With(
			errors.ErrRuleOverloadNotFound,
			errorc.String(keys.RuleName, name),
			errorc.String(keys.ValueType, valueType.String()),
			errorc.String(keys.FieldAvailableTypes, strings.Join(getFieldTypesNames(rules), ", ")),
		)
	}
}

func getFieldTypesNames(rules []*Rule) []string {
	var names []string
	for _, rule := range rules {
		fieldTypeName := rule.GetFieldTypeName()
		if fieldTypeName != "" { // defensive, cannot be empty due to checks in NewRule
			names = append(names, fieldTypeName)
		}
	}
	slices.Sort(names)

	return names
}
