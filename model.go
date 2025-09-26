package model

import (
	"fmt"
	"reflect"
	"sync"
)

type rulesRegistry interface {
	add(r Rule)
	get(name string, v reflect.Value) (Rule, error)
}

type rulesMapping interface {
	add(parent reflect.Type, fieldIndex int, tagName string, rules []ruleNameParams)
	get(parent reflect.Type, fieldIndex int, tagName string) ([]ruleNameParams, bool)
}

func newRulesRegistry() rulesRegistry {
	return newRegistry()
}

func newRulesMapping() rulesMapping {
	return newMapping()
}

type Model[TObject any] struct {
	once               sync.Once
	applyDefaultsOnNew bool
	validateOnNew      bool
	obj                *TObject
	rulesRegistry      rulesRegistry
	rulesMapping       rulesMapping
}

// New constructs a new Model for the given object pointer, applying any provided options.
// The object must be a non-nil pointer to a struct; otherwise, an error is returned.
// Options can enable setting default values and validation behavior during construction.
func New[TObject any](obj *TObject, opts ...Option[TObject]) (*Model[TObject], error) {
	// Validate: obj must be a non-nil pointer to a struct
	if obj == nil {
		return nil, ErrNilObject
	}
	elem := reflect.TypeOf(obj).Elem()
	if elem.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w; got pointer to %s", ErrNotStructPtr, elem.Kind())
	}

	m := &Model[TObject]{obj: obj}
	for _, opt := range opts {
		opt(m)
	}
	// Optionally apply defaults once per model instance
	if m.applyDefaultsOnNew {
		if errOnce := func() error {
			var err error
			m.once.Do(func() { err = m.applyDefaults() })
			return err
		}(); errOnce != nil {
			return nil, errOnce
		}
	}

	// Optionally run validation; return error on failure
	if m.validateOnNew {
		if err := m.validate(); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Option configures a Model at construction time.
type Option[TObject any] func(*Model[TObject])

// WithDefaults enables applying defaults during New(). If not specified, defaults are NOT applied automatically.
func WithDefaults[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) { m.applyDefaultsOnNew = true }
}

func initRules[TObject any](m *Model[TObject]) {
	if m.rulesMapping == nil {
		m.rulesMapping = newRulesMapping()
	}
	if m.rulesRegistry == nil {
		m.rulesRegistry = newRulesRegistry()
	}
}

// WithValidation enables running Validate() during New(). If validation fails, New() returns an error.
// If not specified, validation is NOT run automatically.
// If no custom rules are registered, built-in rules are used for any `validate` tags present.
func WithValidation[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) {
		m.validateOnNew = true
		initRules[TObject](m)
	}
}

// WithRules registers one or many named custom validation rules of the same field type
// into the model's validator rulesRegistry.
//
// WithRules does not imply WithValidation. If you want validation to run during New(),
// you must also specify WithValidation.
//
// Example:
//
//	m, err := model.New(&myObj,
//	    model.WithRules[MyType, string](myRule1, myRule2),
//	    model.WithValidation[MyType](),
//	)
//
// See NewRule for creating rules.
func WithRules[TObject any](rules ...Rule) Option[TObject] {
	return func(m *Model[TObject]) {
		initRules[TObject](m)
		for _, rule := range rules {
			m.rulesRegistry.add(rule)
		}
	}
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
