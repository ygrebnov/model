package model

import (
	"context"
	"reflect"
)

// Binding is a reusable, precompiled view for a specific struct type T.
// It reuses the existing tag parsing, defaulting, and validation logic of
// Model without requiring a Model instance per object.
type Binding[T any] struct {
	// tb holds the type-level metadata for T.
	tb *typeBinding
}

// NewBinding constructs a Binding for the type parameter T using the default
// rules registry and mapping configuration.
func NewBinding[T any]() (*Binding[T], error) {
	// Obtain the reflect.Type for T. The zero value of *T is never dereferenced.
	var zero *T
	typ := reflect.TypeOf(zero).Elem()
	if typ.Kind() != reflect.Struct {
		// Mirror New's constraint that only struct types are supported.
		return nil, ErrNotStructPtr
	}

	tb, err := buildTypeBinding(typ, newRulesRegistry(), newRulesMapping())
	if err != nil {
		return nil, err
	}
	return &Binding[T]{tb: tb}, nil
}

// ApplyDefaults applies default values to zero fields of obj according to
// its `default` / `defaultElem` tags. It is safe to call multiple times.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	if obj == nil {
		return ErrNilObject
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrNotStructPtr
	}
	return b.tb.setDefaultsStruct(elem)
}

// Validate runs validation rules declared via `validate` / `validateElem`
// tags on obj with the provided context. If validation fails, a
// *ValidationError is returned; if the context is canceled, ctx.Err() is
// returned.
func (b *Binding[T]) Validate(ctx context.Context, obj *T) error {
	if obj == nil {
		return ErrNilObject
	}
	if ctx == nil {
		ctx = context.Background()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrNotStructPtr
	}
	ve := &ValidationError{}
	if err := b.tb.validateStruct(ctx, elem, "", ve); err != nil {
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
