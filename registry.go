package model

import (
	"fmt"
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
		return nil, fmt.Errorf("model: invalid value for rule %q", name)
	}

	valueType := v.Type()

	rules, ok := r.rules[name]
	builtinRule, hasBuiltin := builtInRules[key{name, valueType}] // TODO: move inside switch

	if (!ok || len(rules) == 0) && !hasBuiltin {
		return nil, errorc.With(ErrRuleNotFound, errorc.Field("rule_name", name))
	}

	var (
		exacts  []Rule
		assigns []Rule
	)
	for _, rr := range rules {
		//if ad.fieldType == nil || ad.fn == nil {
		//	continue
		//}
		// TODO: can valueType be nil?
		if rr.isOfType(valueType) {
			exacts = append(exacts, rr)
			continue
		}
		if rr.isAssignableTo(valueType) {
			assigns = append(assigns, rr)
		}
	}

	switch {
	case len(exacts) == 1:
		return exacts[0], nil
	case len(exacts) > 1:
		// TODO: check it is still possible
		return nil, fmt.Errorf(
			"model: rule %q is ambiguous for type %s; %d exact overloads registered",
			name,
			valueType,
			len(exacts),
		)
	case len(assigns) >= 1:
		return assigns[0], nil
	case hasBuiltin:
		return builtinRule, nil
	default:
		// Construct helpful message of available overload types.
		return nil, errorc.With(
			ErrRuleNotFound,
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
		if filedTypeName != "" {
			// TODO: is it possible?
			names = append(names, filedTypeName)
		}
	}
	slices.Sort(names)

	return names
}
