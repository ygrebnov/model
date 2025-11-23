package model

import (
	"context"
	"reflect"

	"github.com/ygrebnov/model/errors"
	"github.com/ygrebnov/model/internal/core"
	"github.com/ygrebnov/model/validation"
)

// Binding is a reusable, precompiled view for a specific struct type T.
// It reuses the existing tag parsing, defaulting, and validation logic of
// Model without requiring a Model instance per object.
type Binding[T any] struct {
	// tb holds the type-level metadata for T.
	tb typeBinding
}

type typeBinding interface {
	SetDefaultsStruct(v reflect.Value) error
	AddRule(r validation.Rule) error
	ValidateStruct(ctx context.Context, v reflect.Value, fieldPath string, ve *validation.Error) error
}

func newTypeBinding(typ reflect.Type, rr validation.Registry, rm validation.Mapping) (typeBinding, error) {
	return core.NewTypeBinding(typ, rr, rm)
}

// NewBinding constructs a Binding for the type parameter T using the default
// rules registry and mapping configuration.
func NewBinding[T any]() (*Binding[T], error) {
	// Obtain the reflect.Type for T. The zero value of *T is never dereferenced.
	var zero *T
	typ := reflect.TypeOf(zero).Elem()
	if typ.Kind() != reflect.Struct {
		// Mirror New's constraint that only struct types are supported.
		return nil, errors.ErrNotStructPtr
	}

	rulesRegistry := validation.NewRegistry()
	rulesMapping := validation.NewMapping()

	tb, err := newTypeBinding(typ, rulesRegistry, rulesMapping)
	if err != nil {
		return nil, err
	}
	return &Binding[T]{tb: tb}, nil
}

// ApplyDefaults applies default values to zero fields of obj according to
// its `default` / `defaultElem` tags. It is safe to call multiple times.
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
	return b.tb.SetDefaultsStruct(elem)
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
	if err := b.tb.ValidateStruct(ctx, elem, "", ve); err != nil {
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
