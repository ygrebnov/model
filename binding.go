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
// can be passed together to WithRules.
type Rule struct {
	compiled *rules.Rule
}

// NewRule creates a named custom validation rule for FieldType.
func NewRule[FieldType any](
	name string,
	fn func(FieldType, ...string) error,
) (Rule, error) {
	compiled, err := rules.NewRule(name, fn)
	if err != nil {
		return Rule{}, err
	}

	return Rule{compiled: compiled}, nil
}

// NewBinding compiles the schema and validation plan for T.
//
// Built-in validation rules are available implicitly. Use WithRules to add
// custom rules and WithEnvPrefix to prefix environment variable names used by
// ApplyEnv and ValidateWithDefaults.
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
// guarded to run only once.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.SetDefaultsStruct(elem)
}

// ApplyValues applies values supplied by source to obj using the compiled
// schema metadata.
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
// constructed.
func (b *Binding[T]) ApplyEnv(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyEnvStruct(elem)
}

// WriteValues writes values from obj to sink using the compiled schema
// metadata.
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
// cancellation and operational errors are returned directly.
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

// ValidateWithDefaults applies defaults and snapshotted environment values
// before validating obj.
func (b *Binding[T]) ValidateWithDefaults(
	ctx context.Context,
	obj *T,
) error {
	if ctx == nil {
		return errors.ErrNilContext
	}

	if err := b.ApplyDefaults(obj); err != nil {
		return err
	}

	if err := b.ApplyEnv(obj); err != nil {
		return err
	}

	return b.Validate(ctx, obj)
}

func bindingTargetValue[T any](
	obj *T,
) (reflect.Value, error) {
	if obj == nil {
		return reflect.Value{}, errors.ErrNilObject
	}

	return reflect.ValueOf(obj).Elem(), nil
}
