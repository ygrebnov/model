package model

import (
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/ygrebnov/errorc"
)

// Rule defines a named validation function for a specific field type.
type Rule interface {
	getName() string
	getFieldTypeName() string
	getFieldType() reflect.Type
	getValidationFn() func(v reflect.Value, params ...string) error
	isOfType(t reflect.Type) bool
	isAssignableTo(t reflect.Type) bool
}

// NewRule creates a new Rule with the given name and validation function.
// The validation function must accept a value of type TField and optional string parameters,
// returning an error if validation fails or nil if it passes.
// An error is returned if the name is empty or the function is nil.
func NewRule[TField any](name string, fn func(v TField, params ...string) error) (Rule, error) {
	return newRule(name, fn)
}

// registry is a registry of validation rules.
type registry struct {
	mu    sync.RWMutex
	rules map[string][]Rule // rule name -> overloads by type
}

func newRegistry() *registry {
	return &registry{
		rules: make(map[string][]Rule),
	}
}

func (r *registry) add(rule Rule) error {
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
					ErrDuplicateOverloadRule,
					errorc.Field("rule_name", name),
					errorc.Field("field_type", rule.getFieldTypeName()),
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
func (r *registry) get(name string, v reflect.Value) (Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !v.IsValid() {
		return nil, errorc.With(ErrInvalidValue, errorc.Field("rule_name", name))
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
			ErrAmbiguousRule,
			errorc.Field("rule_name", name),
			errorc.Field("value_type", valueType.String()),
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
			// No rules by the given name neither in registry no from in built-ins.
			return nil, errorc.With(ErrRuleNotFound, errorc.Field("rule_name", name))
		}

		// Some rules exist by the given name, but none match the value type.
		// Construct helpful message of available overload types.
		return nil, errorc.With(
			ErrRuleOverloadNotFound,
			errorc.Field("rule_name", name),
			errorc.Field("value_type", valueType.String()),
			errorc.Field("available_types", strings.Join(getFieldTypesNames(rules), ", ")),
		)
	}
}

func getFieldTypesNames(rules []Rule) []string {
	var names []string
	for _, rule := range rules {
		filedTypeName := rule.getFieldTypeName()
		if filedTypeName != "" { // defensive, cannot be empty due to checks in NewRule
			names = append(names, filedTypeName)
		}
	}
	slices.Sort(names)

	return names
}
