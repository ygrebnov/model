package model

import (
	"context"
	"reflect"

	"github.com/ygrebnov/model/errors"
	"github.com/ygrebnov/model/internal/core"
	"github.com/ygrebnov/model/validation"
)

// Binding provides defaulting and validation capabilities for type T.
type Binding[T any] struct {
	// service is the underlying core service for type T.
	service service
}

type service interface {
	SetDefaultsStruct(v reflect.Value) error
	AddRule(r validation.Rule) error
	ValidateStruct(ctx context.Context, v reflect.Value, fieldPath string, ve *validation.Error) error
}

func newService(typ reflect.Type, rr validation.RulesRegistry, rm validation.RulesMapping) (service, error) {
	return core.NewService(typ, rr, rm)
}

// NewBinding constructs a Binding for the type parameter T using the default
// rules registry and mapping configuration.
func NewBinding[T any]() (*Binding[T], error) {
	// Obtain the reflect.Type for T. The zero value of *T is never dereferenced.
	var zero *T
	t := reflect.TypeOf(zero).Elem()
	if t.Kind() != reflect.Struct {
		// Mirror New's constraint that only struct types are supported.
		return nil, errors.ErrNotStructPtr
	}

	rulesRegistry := validation.NewRulesRegistry()
	rulesMapping := validation.NewRulesMapping()

	s, err := newService(t, rulesRegistry, rulesMapping)
	if err != nil {
		return nil, err
	}
	return &Binding[T]{service: s}, nil
}

// ApplyDefaults applies default values to zero fields of obj according to
// its `default` / `defaultElem` tags. It is safe to call multiple times.
// ApplyDefaults applies defaults each time it is called.
// It is idempotent, but not once-guarded; callers control how often to invoke it.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	if obj == nil {
		return errors.ErrNilObject
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.ErrNotStructPtr
	}
	return b.service.SetDefaultsStruct(elem)
}

// Validate runs validation rules declared via `validate` / `validateElem`
// tags on obj with the provided context. If validation fails, a
// *Error is returned; if the context is canceled, ctx.Err() is
// returned.
func (b *Binding[T]) Validate(ctx context.Context, obj *T) error {
	if obj == nil {
		return errors.ErrNilObject
	}
	if ctx == nil {
		ctx = context.Background()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.ErrNotStructPtr
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

// ValidateWithDefaults first applies defaults to obj and then runs
// validation. This is a convenience for service-level flows that expect
// defaulted inputs before validation.
func (b *Binding[T]) ValidateWithDefaults(ctx context.Context, obj *T) error {
	if err := b.ApplyDefaults(obj); err != nil {
		return err
	}
	return b.Validate(ctx, obj)
}

// RegisterRules registers one or many named custom validation rules of the same field type
// into the registry.
//
// See the validation.Rule type and validation.NewRule function for details on creating rules.
func (b *Binding[T]) RegisterRules(rules ...validation.Rule) error {
	for _, r := range rules {
		if err := b.service.AddRule(r); err != nil {
			return err
		}
	}
	return nil
}
