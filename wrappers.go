package model

import (
	"context"
)

// SetDefaults creates a new Binding for type TObject and applies default values
// to the provided object.
//
// Use WithEnvPrefix option to add environment variables names prefix.
func SetDefaults[TObject any](obj *TObject, opts ...Option) error {
	b, err := NewBinding[TObject](opts...)
	if err != nil {
		return err
	}

	return b.ApplyDefaults(obj)
}

// Validate creates a new Binding for type TObject and validates the provided object
// according to the registered rules.
//
// Builtin rules are applied implicitly.
//
// Use WithRules option to register custom validation rules.
// See [github.com/ygrebnov/model/validation.Rule] and [github.com/ygrebnov/model/validation.NewRule]
// for details on creating rules.
func Validate[TObject any](ctx context.Context, obj *TObject, opts ...Option) error {
	b, err := NewBinding[TObject](opts...)
	if err != nil {
		return err
	}

	return b.Validate(ctx, obj)
}

// ValidateWithDefaults creates a new Binding for type TObject, applies default values to the provided object,
// and then validates the object according to the registered rules.
//
// Use WithEnvPrefix option to add environment variables names prefix.
//
// Builtin rules are applied implicitly.
//
// Use WithRules option to register custom validation rules.
// See [github.com/ygrebnov/model/validation.Rule] and [github.com/ygrebnov/model/validation.NewRule]
// for details on creating rules.
func ValidateWithDefaults[TObject any](ctx context.Context, obj *TObject, opts ...Option) error {
	b, err := NewBinding[TObject](opts...)
	if err != nil {
		return err
	}

	return b.ValidateWithDefaults(ctx, obj)
}

// SetDefaultsAny creates a new DynamicBinding and applies default values to the provided object.
//
// Use WithEnvPrefix option to add environment variables names prefix.
func SetDefaultsAny(obj any, opts ...Option) error {
	b, err := NewDynamicBinding(obj, opts...)
	if err != nil {
		return err
	}

	return b.ApplyDefaults(obj)
}

// ValidateAny creates a new DynamicBinding and validates the provided object
// according to the registered rules.
//
// Builtin rules are applied implicitly.
//
// Use WithRules option to register custom validation rules.
// See [github.com/ygrebnov/model/validation.Rule] and [github.com/ygrebnov/model/validation.NewRule]
// for details on creating rules.
func ValidateAny(ctx context.Context, obj any, opts ...Option) error {
	b, err := NewDynamicBinding(obj, opts...)
	if err != nil {
		return err
	}

	return b.Validate(ctx, obj)
}

// ValidateWithDefaultsAny creates a new DynamicBinding, applies default values to the provided object,
// and then validates the object according to the registered rules.
//
// Use WithEnvPrefix option to add environment variables names prefix.
//
// Builtin rules are applied implicitly.
//
// Use WithRules option to register custom validation rules.
// See [github.com/ygrebnov/model/validation.Rule] and [github.com/ygrebnov/model/validation.NewRule]
// for details on creating rules.
func ValidateWithDefaultsAny(ctx context.Context, obj any, opts ...Option) error {
	b, err := NewDynamicBinding(obj, opts...)
	if err != nil {
		return err
	}

	return b.ValidateWithDefaults(ctx, obj)
}
