package model

import (
	"context"
	"reflect"

	"github.com/ygrebnov/model/internal/core"
	"github.com/ygrebnov/model/internal/errors"
	"github.com/ygrebnov/model/internal/rules"
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
	ValidateStruct(ctx context.Context, v reflect.Value, fieldPath string, ve *ValidationError) error
}

func newTypeBinding(typ reflect.Type, rr rulesRegistry, rm rulesMapping) (typeBinding, error) {
	return core.NewTypeBinding(typ, rr, rm)
}

type rulesRegistry interface {
	Add(r rules.Rule) error
	Get(name string, v reflect.Value) (rules.Rule, error)
}

func newRulesRegistry() rulesRegistry {
	return core.NewRulesRegistry()
}

type rulesMapping interface{}

func newRulesMapping() rulesMapping {
	return core.NewRulesMapping()
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

	tb, err := newTypeBinding(typ, newRulesRegistry(), newRulesMapping())
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
// *ValidationError is returned; if the context is canceled, ctx.Err() is
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
	ve := &ValidationError{}
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
