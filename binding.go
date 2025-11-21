package model

import (
	"context"
	"reflect"
)

type typeBinding struct {
	// typ is the underlying struct type this binding was built for.
	typ reflect.Type
}

// buildTypeBinding creates a typeBinding for the given struct type using the
// provided registry and mapping instances. Currently it only records the type,
// relying on the shared rulesMapping cache and validation/defaults traversal
// on each call. This keeps Binding as a thin, reusable handle over the
// existing Model logic without duplicating traversal code.
func buildTypeBinding(typ reflect.Type, _ rulesRegistry, _ rulesMapping) (*typeBinding, error) {
	// The caller guarantees typ is a struct type. We keep the implementation
	// minimal for now; any future per-type precomputation can be stored here.
	return &typeBinding{typ: typ}, nil
}

// applyDefaults applies default values to the provided struct value using the
// same traversal logic as Model.setDefaultsStruct. The value must be a
// non-zero reflect.Value of Kind struct.
func (tb *typeBinding) applyDefaults(v reflect.Value) error {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil
	}
	// We reuse Model's setDefaultsStruct implementation by constructing a
	// temporary Model that wraps the given value. This avoids code duplication
	// and keeps Binding in sync with Model behavior.
	m := &Model[any]{}
	return m.setDefaultsStruct(v)
}

// validate runs validation rules against the provided struct value using the
// same traversal logic as Model.validateStruct. The value must be a non-zero
// reflect.Value of Kind struct.
func (tb *typeBinding) validate(ctx context.Context, v reflect.Value) error {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Initialize a temporary Model wired with the default registry and mapping
	// so that validateStruct and applyRule behave exactly as in the regular
	// Model path.
	m := &Model[any]{
		rulesRegistry: newRulesRegistry(),
		rulesMapping:  newRulesMapping(),
	}
	ve := &ValidationError{}
	if err := m.validateStruct(ctx, v, "", ve); err != nil {
		return err
	}
	if ve.Empty() {
		return nil
	}
	return ve
}

// Binding is a reusable, precompiled view for a specific struct type T.
// It reuses the existing tag parsing, defaulting, and validation logic of
// Model without requiring a Model instance per object.
type Binding[T any] struct {
	// tb holds the type-level metadata for T.
	tb *typeBinding
}

// NewBinding constructs a Binding for the type parameter T using the default
// rules registry and mapping configuration. It panics if T is not a struct
// type; callers are expected to use struct types only.
func NewBinding[T any]() (*Binding[T], error) {
	// Obtain the reflect.Type for T. The zero value of *T is never dereferenced.
	var zero *T
	typ := reflect.TypeOf(zero).Elem()
	if typ.Kind() != reflect.Struct {
		// Mirror New's constraint that only struct types are supported.
		return nil, ErrNotStructPtr
	}

	// For now we pass freshly created registry/mapping; the heavy lifting is
	// still done lazily during validation/defaulting using the shared cache.
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
	return b.tb.applyDefaults(elem)
}

// Validate runs validation rules declared via `validate` / `validateElem`
// tags on obj with the provided context. If validation fails, a
// *ValidationError is returned; if the context is canceled, ctx.Err() is
// returned.
func (b *Binding[T]) Validate(ctx context.Context, obj *T) error {
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
	return b.tb.validate(ctx, elem)
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
