package model

import (
	"fmt"
	"reflect"
)

func New[TObject any](obj *TObject, opts ...Option[TObject]) (*Model[TObject], error) {
	// Validate: obj must be a non-nil pointer to a struct
	if obj == nil {
		panic("model: obj is nil; want pointer to struct")
	}
	elem := reflect.TypeOf(obj).Elem()
	if elem.Kind() != reflect.Struct {
		panic(fmt.Errorf("model: obj must be a pointer to struct; got pointer to %s", elem.Kind()))
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

// WithRule registers a named validation rule into the model's validator registry.
func WithRule[TObject any, TField any](rule Rule[TField]) Option[TObject] {
	return func(m *Model[TObject]) {
		if rule.Name == "" || rule.Fn == nil {
			panic("model: WithRule: rule must have non-empty Name and non-nil Fn")
		}
		if m.validators == nil {
			m.validators = make(map[string][]typedAdapter)
		}
		ad := wrapRule(rule.Fn)
		m.validators[rule.Name] = append(m.validators[rule.Name], ad)
	}
}

// WithRules registers multiple named rules of the same field type at once.
func WithRules[TObject any, TField any](rules []Rule[TField]) Option[TObject] {
	return func(m *Model[TObject]) {
		for _, r := range rules {
			WithRule[TObject, TField](r)(m)
		}
	}
}

// ensureRule adds a rule if an exact overload for the same rule name and field type is not already present.
func ensureRule[TObject any, TField any](m *Model[TObject], r Rule[TField]) {
	if r.Name == "" || r.Fn == nil {
		return
	}
	if m.validators == nil {
		m.validators = make(map[string][]typedAdapter)
	}
	typ := reflect.TypeOf((*TField)(nil)).Elem()
	for _, ad := range m.validators[r.Name] {
		if ad.fieldType != nil && ad.fieldType == typ {
			return // exact overload already exists; keep user's version
		}
	}
	ad := wrapRule(r.Fn)
	m.validators[r.Name] = append(m.validators[r.Name], ad)
}

// WithDefaults enables applying defaults during New(). If not specified, defaults are NOT applied automatically.
func WithDefaults[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) { m.applyDefaultsOnNew = true }
}

// WithValidation enables running Validate() during New(). If validation fails, New() returns an error.
// Additionally, builtin rules are implicitly registered (if not already provided) to improve UX.
func WithValidation[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) {
		m.validateOnNew = true
		// Implicitly register builtin rules, but do not override existing exact overloads.
		for _, r := range BuiltinStringRules() {
			ensureRule[TObject, string](m, r)
		}
		for _, r := range BuiltinIntRules() {
			ensureRule[TObject, int](m, r)
		}
		for _, r := range BuiltinInt64Rules() {
			ensureRule[TObject, int64](m, r)
		}
		for _, r := range BuiltinFloat64Rules() {
			ensureRule[TObject, float64](m, r)
		}
	}
}
