package model

import (
	"context"
	"reflect"
	"sync"

	"github.com/ygrebnov/errorc"
)

type rulesRegistry interface {
	add(r Rule) error
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
	binding            *typeBinding
	ctx                context.Context // used only for validation during New when WithValidation(ctx) is provided
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
		return nil, errorc.With(ErrNotStructPtr, errorc.String(ErrorFieldObjectType, elem.Kind().String()))
	}

	m := &Model[TObject]{obj: obj, ctx: context.Background()}
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
		if err := m.validate(m.ctx); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Option configures a Model at construction time.
type Option[TObject any] func(*Model[TObject]) error

// WithDefaults enables applying defaults during New(). If not specified, defaults are NOT applied automatically.
func WithDefaults[TObject any]() Option[TObject] {
	return func(m *Model[TObject]) error { m.applyDefaultsOnNew = true; return nil }
}

// WithValidation enables running Validate(ctx) during New(). If validation fails, New() returns an error.
// If not specified, validation is NOT run automatically.
// If no custom rules are registered, built-in rules are used for any `validate` tags present.
// The provided context controls cancellation/deadlines for this New-time validation.
func WithValidation[TObject any](ctx context.Context) Option[TObject] {
	return func(m *Model[TObject]) error {
		m.validateOnNew = true
		if ctx == nil {
			m.ctx = context.Background()
		} else {
			m.ctx = ctx
		}
		return nil
	}
}

// WithRules registers one or many named custom validation rules into the model's validator.
// Registered validation rules can be executed in New() if WithValidation is also specified,
// or later by calling Validate(ctx).
//
// All rules must be of the same field type (e.g., string, int).
//
// See the Rule type and NewRule function for details on creating rules.
func WithRules[TObject any](rules ...Rule) Option[TObject] {
	return func(m *Model[TObject]) error {
		return m.RegisterRules(rules...)
	}
}

// rootStructValue validates that m.obj is a non-nil pointer to a struct and returns the struct value.
// The phase string is used in error messages (e.g., "Validate", "SetDefaults").
func (m *Model[TObject]) rootStructValue(phase string) (reflect.Value, error) {
	if m.obj == nil {
		// defensive, cannot happen due to New() checks
		return reflect.Value{}, errorc.With(ErrNilObject, errorc.String(ErrorFieldPhase, phase))
	}
	rv := reflect.ValueOf(m.obj)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		// defensive: unreachable under normal generic use
		return reflect.Value{},
			errorc.With(
				ErrNotStructPtr,
				errorc.String(ErrorFieldObjectType, rv.Kind().String()),
				errorc.String(ErrorFieldPhase, phase),
			)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return reflect.Value{},
			errorc.With(
				ErrNotStructPtr,
				errorc.String(ErrorFieldObjectType, rv.Kind().String()),
				errorc.String(ErrorFieldPhase, phase),
			)
	}
	return rv, nil
}

// SetDefaults applies default values based on `default:"..."` tags to the model's object.
// It is safe to call multiple times; only zero-valued fields are set.
func (m *Model[TObject]) SetDefaults() error {
	var err error
	m.once.Do(func() { err = m.applyDefaults() })
	return err
}

// applyDefaults walks the object and applies defaults according to `default` and `defaultElem` tags.
// Supported forms:
//   - `default:"<literal>"` sets the field if it is zero
//   - `default:"dive"` on a struct or pointer-to-struct recurses into its fields
//   - `default:"alloc"` allocates an empty map/slice when the field is nil
//   - `defaultElem:"dive"` recurses into slice/array elements or map values that are structs
//
// Notes:
//   - Literals are parsed by kind: string, bool, ints/uints, floats, time.Duration.
//   - For pointer scalar fields, nil pointers are allocated when a literal default is present.
func (m *Model[TObject]) applyDefaults() error {
	if rv, err := m.rootStructValue("SetDefaults"); err != nil {
		return err
	} else {
		if err := m.ensureBinding(); err != nil {
			return err
		}
		return m.binding.setDefaultsStruct(rv)
	}
}

// ensureBinding initializes the model's typeBinding, rulesRegistry, and rulesMapping lazily.
func (m *Model[TObject]) ensureBinding() error {
	if m.binding != nil {
		return nil
	}
	// Derive the concrete struct type from the bound object.
	rv, err := m.rootStructValue("initBinding")
	if err != nil {
		return err
	}
	typ := rv.Type()
	reg := newRulesRegistry()
	mapping := newRulesMapping()
	tb, err := buildTypeBinding(typ, reg, mapping)
	if err != nil {
		return err
	}
	m.binding = tb
	return nil
}

// RegisterRules registers one or many named custom validation rules of the same field type
// into the model's validator rulesRegistry.
//
// See the Rule type and NewRule function for details on creating rules.
func (m *Model[TObject]) RegisterRules(rules ...Rule) error {
	if err := m.ensureBinding(); err != nil {
		return err
	}
	for _, rule := range rules {
		if err := m.binding.rulesRegistry.add(rule); err != nil {
			return err
		}
	}
	return nil
}

// Validate runs the registered validation rules against the model's bound object with the provided context.
// If the context is canceled or its deadline exceeded, validation stops early and ctx.Err() is returned.
func (m *Model[TObject]) Validate(ctx context.Context) error {
	if err := m.ensureBinding(); err != nil {
		return err
	}
	return m.validate(ctx)
}

// validate is the internal implementation that walks struct fields and applies rules
// declared in `validate:"..."` tags. It supports rule parameters via the syntax
// "rule" or "rule(p1,p2)" and multiple rules separated by commas.
func (m *Model[TObject]) validate(ctx context.Context) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := m.ensureBinding(); err != nil {
		return err
	}

	var rv reflect.Value
	if rv, err = m.rootStructValue("Validate"); err != nil {
		return err
	}
	ve := &ValidationError{}
	// Delegate traversal to typeBinding to keep logic centralized.
	if err := m.binding.validateStruct(ctx, rv, "", ve); err != nil {
		return err
	}
	if ve.Empty() {
		return nil
	}
	return ve
}
