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

	m := &Model[TObject]{obj: obj, rulesMapping: newRulesMapping(), rulesRegistry: newRulesRegistry()} // TODO: do not initialize cache if no validation?
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

// WithRule registers a named validation rule into the model's validator rulesRegistry.
func WithRule[TObject any, TField any](name string, fn func(value TField, params ...string) error) Option[TObject] {
	return func(m *Model[TObject]) error {
		r, err := newValidationRule(name, fn)
		if err != nil {
			return fmt.Errorf("model: WithRule: cannot add rule: %w", err)
		}

		m.rulesRegistry.Add(r)

		//if rule.name == "" || rule.Fn == nil {
		//	return fmt.Errorf("model: WithRule: rule must have non-empty name and non-nil Fn")
		//}
		//if m.validators == nil {
		//	m.validators = make(map[string][]ruleAdapter)
		//}
		//ad := newRuleAdapter(rule.Fn)
		//m.validators[rule.name] = append(m.validators[rule.name], ad)
		return nil
	}
}

// WithRules registers multiple named rules of the same field type at once.
func WithRules[TObject any, TField any](rules []rule.Rule) Option[TObject] {
	return func(m *Model[TObject]) error {
		for _, r := range rules {
			m.rulesRegistry.Add(r)
		}
		return nil
	}
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
		// No need to implicitly register builtin rules, rulesRegistry will return them if no exact overload is found.
		// Implicitly register builtin rules, but do not override existing exact overloads.
		//for _, r := range BuiltinStringRules() {
		//	if err := ensureRule[TObject, string](m, r); err != nil {
		//		return err
		//	}
		//}
		//for _, r := range BuiltinIntRules() {
		//	if err := ensureRule[TObject, int](m, r); err != nil {
		//		return err
		//	}
		//}
		//for _, r := range BuiltinInt64Rules() {
		//	if err := ensureRule[TObject, int64](m, r); err != nil {
		//		return err
		//	}
		//}
		//for _, r := range BuiltinFloat64Rules() {
		//	if err := ensureRule[TObject, float64](m, r); err != nil {
		//		return err
		//	}
		//}
		return nil
	}
}
