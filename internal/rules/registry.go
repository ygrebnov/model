package rules

import (
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/internal/errors"
)

// Registry is a registry of validation rules.
type Registry struct {
	mu    sync.RWMutex
	rules map[string][]Rule // rule name -> overloads by type
}

func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string][]Rule),
	}
}

func (r *Registry) Add(rule Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rule == nil {
		return nil
	}

	name := rule.getName()
	existing, exists := r.rules[name]
	if exists {
		// Prevent duplicate overloads for the same field type.
		for _, er := range existing {
			if er.isOfType(rule.getFieldType()) {
				return errorc.With(
					errors.ErrDuplicateOverloadRule,
					errorc.String(errors.ErrorFieldRuleName, name),
					errorc.String(errors.ErrorFieldFieldType, rule.getFieldTypeName()),
				)
			}
		}
	}

	r.rules[name] = append(r.rules[name], rule)
	return nil
}

// get returns the best-matching overload of rule `name` for the given field value.
// Selection strategy:
//  1. Prefer exact type match (v.Type() == fieldType).
//  2. Otherwise accept AssignableTo matches (interfaces, named types), preferring the first declared.
//  3. Otherwise, if no matches, fetch a built-in rule if available.
//  4. If no matches, return a descriptive error listing available overload types.
//  5. If multiple exact matches (shouldn't happen), return an ambiguity error.
func (r *Registry) Get(name string, v reflect.Value) (Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !v.IsValid() {
		return nil,
			errorc.With(errors.ErrInvalidValue, errorc.String(errors.ErrorFieldRuleName, name))
	}

	valueType := v.Type()
	rules, _ := r.rules[name]

	var (
		exacts  []Rule
		assigns []Rule
	)
	for _, rule := range rules {
		if rule.getFieldType() == nil || rule.getValidationFn() == nil {
			continue // defensive, should not happen due to checks in NewRule
		}
		if rule.isOfType(valueType) {
			exacts = append(exacts, rule)
			continue
		}
		if rule.isAssignableTo(valueType) {
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
			errorc.String(errors.ErrorFieldRuleName, name),
			errorc.String(errors.ErrorFieldValueType, valueType.String()),
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
			// No rules by the given name neither in Registry no from in built-ins.
			return nil,
				errorc.With(errors.ErrRuleNotFound, errorc.String(errors.ErrorFieldRuleName, name))
		}

		// Some rules exist by the given name, but none match the value type.
		// Construct helpful message of available overload types.
		return nil, errorc.With(
			errors.ErrRuleOverloadNotFound,
			errorc.String(errors.ErrorFieldRuleName, name),
			errorc.String(errors.ErrorFieldValueType, valueType.String()),
			errorc.String(errors.ErrorFieldAvailableTypes, strings.Join(getFieldTypesNames(rules), ", ")),
		)
	}
}

func getFieldTypesNames(rules []Rule) []string {
	var names []string
	for _, rule := range rules {
		fieldTypeName := rule.getFieldTypeName()
		if fieldTypeName != "" { // defensive, cannot be empty due to checks in NewRule
			names = append(names, fieldTypeName)
		}
	}
	slices.Sort(names)

	return names
}
