package model

import (
	"context"
	"reflect"

	"github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/core"
	"github.com/ygrebnov/model/internal/rules"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/validation"
)

// Binding provides defaulting, external value application, value export, and
// validation for struct type T.
//
// NewBinding compiles the schema and validation plan once. A Binding is safe
// for concurrent use after construction because that metadata and its
// environment snapshot are immutable. The objects, sources, and sinks passed
// to its methods remain owned by the caller:
//
//   - concurrent calls using different objects are safe when the supplied
//     sources and sinks are themselves safe for concurrent use;
//   - concurrent operations on the same object require caller-side
//     synchronization;
//   - callers must not mutate an object concurrently with ApplyDefaults,
//     ApplyValues, ApplyEnv, Validate, ValidateWithDefaults, or WriteValues;
//   - ValueSink implementations that retain maps, slices, pointers, or other
//     reference values receive references to the object's current values rather
//     than deep copies.
type Binding[T any] struct {
	service service[T]
}

type service[T any] interface {
	SetDefaultsStruct(v reflect.Value) error
	ApplyValuesStruct(v reflect.Value, source field.ValueSource) error
	ApplyEnvStruct(v reflect.Value) error
	WriteValuesStruct(v reflect.Value, sink field.ValueSink) error
	ValidateStruct(
		ctx context.Context,
		v reflect.Value,
		fieldPath string,
		ve *validation.Error,
	) error
}

// Rule is an opaque custom validation rule created by NewRule.
//
// Rule is intentionally non-generic so rules concerning different field types
// can be passed together to WithRules. Rule values are immutable after
// construction and may be reused when constructing multiple bindings.
type Rule struct {
	compiled *rules.Rule
}

// NewRule creates a named custom validation rule for FieldType.
//
// The name must be non-empty and fn must be non-nil. Custom rules with the
// same name and field type replace the corresponding built-in rule; duplicate
// custom overloads are rejected when NewBinding is called.
//
// The validation function may be called concurrently by different bindings or
// validation operations. It must therefore synchronize access to mutable state
// it captures.
func NewRule[FieldType any](
	name string,
	fn func(FieldType, ...string) error,
) (Rule, error) {
	compiled, err := rules.NewRule(name, fn)
	if err != nil {
		return Rule{}, err
	}

	return Rule{
		compiled: compiled,
	}, nil
}

// NewBinding compiles a reusable Binding for struct type T.
//
// T must be a struct, not a pointer. Built-in validation rules are available
// implicitly. Use WithRules to add custom rules and WithEnvPrefix to prefix
// environment variable names used by ApplyEnv and ValidateWithDefaults.
//
// The returned Binding is fully initialized and immutable. It snapshots
// environment values during this call, so later process-environment changes do
// not affect ApplyEnv or ValidateWithDefaults.
func NewBinding[T any](opts ...Option) (*Binding[T], error) {
	o := &options{}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	registry, err := newRulesRegistry(o.rules)
	if err != nil {
		return nil, err
	}

	compiledSchema, err := schema.New[T]()
	if err != nil {
		return nil, err
	}

	s, err := core.NewService[T](
		registry,
		compiledSchema,
		o.envPrefix,
	)
	if err != nil {
		return nil, err
	}

	return &Binding[T]{
		service: s,
	}, nil
}

func newRulesRegistry(
	customRules []Rule,
) (*rules.Registry, error) {
	registry := rules.NewRegistry()

	for _, customRule := range customRules {
		if customRule.compiled == nil {
			return nil, errors.ErrInvalidRule
		}

		if err := registry.Add(customRule.compiled); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// ApplyDefaults applies default and defaultElem tags to zero fields of obj.
//
// Literal defaults do not overwrite non-zero values. default:"alloc"
// initializes nil slices and maps, and default:"dive" allocates a nil
// pointer-to-struct before traversing it, except at a recursive schema
// boundary. ApplyDefaults evaluates defaults on every call and is idempotent.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.SetDefaultsStruct(elem)
}

// ApplyValues applies values supplied by source to obj using compiled schema
// paths.
//
// ValueSource.Get is called with exact exported field paths such as "Name" and
// "Server.Host". Collection fields use a "[]" suffix, such as "Items[]"; the
// source receives descendant collection paths, but ApplyValues only assigns
// direct collection values rather than individual collection elements.
//
// A missing source value leaves its field unchanged. A found nil value resets
// the field to its zero value. Other values must be assignable to, convertible
// to, or assignable or convertible to the element type of a pointer field.
// Scalar values for pointer fields allocate the pointer. Nil
// pointer-to-struct fields are allocated only when a descendant value is
// supplied. Lookup and assignment errors are returned; either can leave earlier
// assignments in place.
func (b *Binding[T]) ApplyValues(
	obj *T,
	source field.ValueSource,
) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyValuesStruct(elem, source)
}

// ApplyEnv applies environment values snapshotted when the Binding was
// constructed.
//
// An explicit env tag takes precedence. Without one, a JSON tag name is used
// when present; otherwise the exported field name is used. env:"-" and
// json:"-" disable environment traversal for that field and its descendants.
// Path segments are joined with underscores and optionally prefixed by
// WithEnvPrefix. Values present in the snapshot replace existing values. Nil
// pointer-to-struct fields are allocated only when the snapshot contains a
// descendant value.
func (b *Binding[T]) ApplyEnv(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyEnvStruct(elem)
}

// WriteValues writes reachable values from obj to sink using compiled schema
// paths.
//
// Paths use exact exported field names, such as "Name" and "Server.Host".
// A collection of structs uses a "[]" suffix, such as "Items[]", and each
// reachable element's fields are written using paths such as "Items[].Name".
// Scalar collections are written once using their field path, such as "Tags".
//
// Nil pointer-to-struct fields and nil pointer collection elements are written
// but not traversed. Maps, slices, pointers, and other reference values are
// passed to sink as the original values rather than deep copies. Sink errors
// stop traversal and are returned; values already passed to sink remain there.
func (b *Binding[T]) WriteValues(
	obj *T,
	sink field.ValueSink,
) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.WriteValuesStruct(elem, sink)
}

// Validate applies validate and validateElem rules declared on obj.
//
// It returns *validation.Error when one or more constraints fail. Field paths
// use exact exported field names. Context cancellation and operational errors
// are returned directly.
func (b *Binding[T]) Validate(
	ctx context.Context,
	obj *T,
) error {
	if ctx == nil {
		return errors.ErrNilContext
	}

	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.validateValue(ctx, elem)
}

// ValidateWithDefaults applies defaults, snapshotted environment values, and
// validation to obj, in that order.
//
// This sequence is not transactional: when a later step fails, changes made by
// earlier steps remain in obj.
func (b *Binding[T]) ValidateWithDefaults(
	ctx context.Context,
	obj *T,
) error {
	if ctx == nil {
		return errors.ErrNilContext
	}

	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	if err := b.service.SetDefaultsStruct(elem); err != nil {
		return err
	}

	if err := b.service.ApplyEnvStruct(elem); err != nil {
		return err
	}

	return b.validateValue(ctx, elem)
}

func (b *Binding[T]) validateValue(
	ctx context.Context,
	elem reflect.Value,
) error {
	validationErr := &validation.Error{}

	if err := b.service.ValidateStruct(
		ctx,
		elem,
		"",
		validationErr,
	); err != nil {
		return err
	}

	if validationErr.Empty() {
		return nil
	}

	return validationErr
}

func bindingTargetValue[T any](
	obj *T,
) (reflect.Value, error) {
	if obj == nil {
		return reflect.Value{}, errors.ErrNilObject
	}

	return reflect.ValueOf(obj).Elem(), nil
}
