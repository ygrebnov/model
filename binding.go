package model

import (
	"context"
	"reflect"

	"github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/core"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/validation"
)

// Binding provides defaulting and validation capabilities for type T.
type Binding[T any] struct {
	// service is the underlying core service for type T.
	service service[T]
}

type service[T any] interface {
	SetDefaultsStruct(v reflect.Value) error
	ApplyValuesStruct(v reflect.Value, source field.ValueSource) error
	ApplyEnvStruct(v reflect.Value, source field.EnvSource) error
	ApplySnapshotEnvStruct(v reflect.Value) error
	WriteValuesStruct(v reflect.Value, sink field.ValueSink) error
	AddRule(r validation.Rule) error
	ValidateStruct(ctx context.Context, v reflect.Value, fieldPath string, ve *validation.Error) error
}

func newService[T any](
	rr validation.RulesRegistry,
	rm validation.RulesMapping,
	sc *schema.Controller[T],
	envPrefix string,
) service[T] {
	return core.NewService[T](rr, rm, sc, envPrefix)
}

// NewBinding constructs a Binding for the type parameter T. T must be a struct.
//
// Use WithEnvPrefix option to add an environment variable name prefix for ApplyEnv
// and ValidateWithDefaults.
//
// Builtin rules are applied implicitly.
//
// Use WithRules option to register custom validation rules at construction time.
// See [validation.Rule] and [validation.NewRule] for details on creating rules.
func NewBinding[T any](opts ...Option) (*Binding[T], error) {
	// Obtain the reflect.Type for T. The zero value of *T is never dereferenced.
	var zero *T
	t := reflect.TypeOf(zero).Elem()
	if t.Kind() != reflect.Struct {
		// Mirror New's constraint that only struct types are supported.
		return nil, errors.ErrTypeParamNotStruct
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	rulesRegistry := validation.NewRulesRegistry()
	rulesMapping := validation.NewRulesMapping()
	schemaController, err := schema.NewController[T]()
	if err != nil {
		return nil, err
	}

	s := newService(rulesRegistry, rulesMapping, schemaController, o.envPrefix)

	b := &Binding[T]{service: s}

	if len(o.rules) > 0 {
		if err := registerRules(s, o.rules...); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// ApplyDefaults applies default values to zero fields of obj according to its `default` / `defaultElem` tags.
// ApplyDefaults applies defaults each time it is called.
// It is idempotent, but not once-guarded.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.SetDefaultsStruct(elem)
}

// ApplyValues applies field values supplied by source to obj using compiled field metadata.
func (b *Binding[T]) ApplyValues(obj *T, source field.ValueSource) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyValuesStruct(elem, source)
}

// ApplyEnv applies environment-backed values from source to obj using compiled field metadata.
func (b *Binding[T]) ApplyEnv(obj *T, source field.EnvSource) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyEnvStruct(elem, source)
}

// Validate runs validation rules declared via `validate` / `validateElem` tags on obj
// with the provided context.
//
// If validation fails, a *validation.Error is returned; if the context is canceled, ctx.Err() is returned.
func (b *Binding[T]) Validate(ctx context.Context, obj *T) error {
	if ctx == nil {
		return errors.ErrNilContext
	}
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}
	ve := &validation.Error{}
	if err := b.service.ValidateStruct(ctx, elem, "", ve); err != nil {
		return err
	}
	if ve.Empty() {
		return nil
	}
	return ve
}

// ValidateWithDefaults first applies defaults and snapshotted environment-backed values to obj,
// then runs validation. This is a convenience for service-level flows that expect resolved inputs
// before validation.
func (b *Binding[T]) ValidateWithDefaults(ctx context.Context, obj *T) error {
	if err := b.ApplyDefaults(obj); err != nil {
		return err
	}
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}
	if err := b.service.ApplySnapshotEnvStruct(elem); err != nil {
		return err
	}
	return b.Validate(ctx, obj)
}

func registerRules[T any](s service[T], rules ...validation.Rule) error {
	for _, r := range rules {
		if err := s.AddRule(r); err != nil {
			return err
		}
	}
	return nil
}

func (b *Binding[T]) WriteValues(obj *T, sink field.ValueSink) error {
	elem, err := bindingTargetValue(obj)
	if err != nil {
		return err
	}

	return b.service.WriteValuesStruct(elem, sink)
}

func bindingTargetValue[T any](obj *T) (reflect.Value, error) {
	if obj == nil {
		return reflect.Value{}, errors.ErrNilObject
	}

	return reflect.ValueOf(obj).Elem(), nil
}
