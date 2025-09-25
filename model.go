package model

import (
	"fmt"
	"reflect"
	"sync"
)

type Model[TObject any] struct {
	once               sync.Once
	applyDefaultsOnNew bool
	validateOnNew      bool
	obj                *TObject
	rulesRegistry      rulesRegistry
	rulesMapping       rulesMapping
}

type rulesRegistry interface {
	add(r validationRule)
	get(name string, v reflect.Value) (validationRule, error)
}

func newRulesRegistry() rulesRegistry {
	return newRegistry()
}

type rulesMapping interface {
	add(parent reflect.Type, fieldIndex int, tagName string, rules []ruleNameParams)
	get(parent reflect.Type, fieldIndex int, tagName string) ([]ruleNameParams, bool)
}

func newRulesMapping() rulesMapping {
	return newMapping()
}

// applyRule dispatches to the best-matching overload of rule `name` for the given field value.
// Selection strategy:
//  1. Prefer exact type match (v.Type() == fieldType).
//  2. Otherwise accept AssignableTo matches (interfaces, named types), preferring the first declared.
//  3. If no matches, return a descriptive error listing available overload types.
//  4. If multiple exact matches (shouldn't happen), return an ambiguity error.
func (m *Model[TObject]) applyRule(name string, v reflect.Value, params ...string) error {
	r, err := m.rulesRegistry.get(name, v)
	if err != nil {
		return err
	}

	return r.getValidationFn()(v, params...)
}

// rootStructValue validates that m.obj is a non-nil pointer to a struct and returns the struct value.
// The phase string is used in error messages (e.g., "Validate", "SetDefaults").
func (m *Model[TObject]) rootStructValue(phase string) (reflect.Value, error) {
	if m.obj == nil {
		return reflect.Value{}, fmt.Errorf("model: %s: nil object", phase)
	}
	rv := reflect.ValueOf(m.obj)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		// effectively unreachable due to generic type constraint, left for completeness
		return reflect.Value{}, fmt.Errorf("model: %s: object must be a non-nil pointer to struct; got %s", phase, rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("model: %s: object must point to a struct; got %s", phase, rv.Kind())
	}
	return rv, nil
}
