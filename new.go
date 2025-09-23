package model

import (
	"fmt"
	"reflect"
)

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
		if err := opt(m); err != nil {
			return nil, err
		}
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
type Option[TObject any] func(*Model[TObject]) error

// WithRule registers a named validation rule into the model's validator registry.
func WithRule[TObject any, TField any](rule Rule[TField]) Option[TObject] {
	return func(m *Model[TObject]) error {
		if rule.Name == "" || rule.Fn == nil {
			return fmt.Errorf("model: WithRule: rule must have non-empty Name and non-nil Fn")
		}
		if m.validators == nil {
			m.validators = make(map[string][]typedAdapter)
		}
		ad := wrapRule(rule.Fn)
		m.validators[rule.Name] = append(m.validators[rule.Name], ad)
		return nil
	}
}

// WithRules registers multiple named rules of the same field type at once.
func WithRules[TObject any, TField any](rules []Rule[TField]) Option[TObject] {
	return func(m *Model[TObject]) error {
		for _, r := range rules {
			if err := WithRule[TObject, TField](r)(m); err != nil {
				return err
			}
		}
		return nil
	}
}

// ensureRule adds a rule if an exact overload for the same rule name and field type is not already present.
// It returns an error for invalid rule definitions (empty name or nil function).
func ensureRule[TObject any, TField any](m *Model[TObject], r Rule[TField]) error {
	if r.Name == "" || r.Fn == nil {
		return fmt.Errorf("model: ensureRule: rule must have non-empty Name and non-nil Fn")
	}
	if m.validators == nil {
		m.validators = make(map[string][]typedAdapter)
	}
	typ := reflect.TypeOf((*TField)(nil)).Elem()
	for _, ad := range m.validators[r.Name] {
		if ad.fieldType != nil && ad.fieldType == typ {
			return nil // exact overload already exists; keep user's version
		}
	}
	ad := wrapRule(r.Fn)
	m.validators[r.Name] = append(m.validators[r.Name], ad)
	return nil
}

// WithDefaults enables applying defaults during New(). If not specified, defaults are NOT applied automatically.
func WithDefaults[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) error { m.applyDefaultsOnNew = true; return nil }
}

// WithValidation enables running Validate() during New(). If validation fails, New() returns an error.
// Additionally, builtin rules are implicitly registered (if not already provided) to improve UX.
func WithValidation[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) error {
		m.validateOnNew = true
		// Implicitly register builtin rules, but do not override existing exact overloads.
		for _, r := range BuiltinStringRules() {
			if err := ensureRule[TObject, string](m, r); err != nil {
				return err
			}
		}
		for _, r := range BuiltinIntRules() {
			if err := ensureRule[TObject, int](m, r); err != nil {
				return err
			}
		}
		for _, r := range BuiltinInt64Rules() {
			if err := ensureRule[TObject, int64](m, r); err != nil {
				return err
			}
		}
		for _, r := range BuiltinFloat64Rules() {
			if err := ensureRule[TObject, float64](m, r); err != nil {
				return err
			}
		}
		return nil
	}
}
