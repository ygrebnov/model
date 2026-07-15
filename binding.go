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

// Binding provides defaulting, value application, value export, and validation
// capabilities for type T.
//
// A Binding is safe for concurrent use after construction because its compiled
// schema, validation plan, and environment snapshot are immutable. The objects,
// sources, and sinks passed to its methods remain owned by the caller:
//
//   - concurrent calls using different objects are safe when the supplied
//     sources and sinks are themselves safe for concurrent use;
//   - concurrent operations on the same object require caller-side
//     synchronization;
//   - callers must not mutate an object concurrently with Validate or
//     WriteValues;
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
// The validation function may be called concurrently by different bindings and
// validation operations. It must therefore synchronize access to any mutable
// state it captures.
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

// NewBinding compiles the schema and validation plan for T.
//
// Built-in validation rules are available implicitly. Use WithRules to add
// custom rules and WithEnvPrefix to prefix environment variable names used by
// ApplyEnv and ValidateWithDefaults.
//
// The returned Binding is fully initialized and immutable. Environment values
// are snapshotted during this call.
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

// ApplyDefaults applies values declared through default and defaultElem tags to
// zero fields of obj.
//
// ApplyDefaults evaluates defaults on every call. It is idempotent but is not
// guarded to run only once. Concurrent mutation of obj requires caller-side
// synchronization.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.SetDefaultsStruct(elem)
}

// ApplyValues applies values supplied by source to obj using the compiled
// schema metadata.
//
// The caller must synchronize concurrent access to obj. The source must also be
// safe for concurrent use when shared between calls.
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

// ApplyEnv applies the environment values snapshotted when the Binding was
// constructed. Concurrent mutation of obj requires caller-side synchronization.
func (b *Binding[T]) ApplyEnv(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyEnvStruct(elem)
}

// WriteValues writes values from obj to sink using the compiled schema
// metadata.
//
// The caller must not mutate obj concurrently with this method. A shared sink
// must provide its own synchronization.
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

// Validate applies rules declared through validate and validateElem tags.
//
// It returns *validation.Error when one or more constraints fail. Context
// cancellation and operational errors are returned directly. The caller must
// not mutate obj concurrently with validation.
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

// ValidateWithDefaults applies defaults and snapshotted environment values
// before validating obj.
//
// This sequence is not transactional: when a later step fails, changes made by
// earlier steps remain in obj. The caller must provide exclusive access to obj
// for the complete operation.
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
