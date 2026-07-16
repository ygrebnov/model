package model

import (
	"context"
)

// SetDefaults creates a Binding for TObject and applies default values to obj.
//
// It is equivalent to creating a Binding with opts and calling ApplyDefaults.
func SetDefaults[TObject any](obj *TObject, opts ...Option) error {
	b, err := NewBinding[TObject](opts...)
	if err != nil {
		return err
	}

	return b.ApplyDefaults(obj)
}

// Validate creates a Binding for TObject and validates obj.
//
// It is equivalent to creating a Binding with opts and calling Validate.
// Built-in rules are applied implicitly; use WithRules to register custom
// rules.
func Validate[TObject any](ctx context.Context, obj *TObject, opts ...Option) error {
	b, err := NewBinding[TObject](opts...)
	if err != nil {
		return err
	}

	return b.Validate(ctx, obj)
}

// ValidateWithDefaults creates a Binding for TObject, then applies defaults,
// applies its snapshotted environment values, and validates obj.
//
// It is equivalent to creating a Binding with opts and calling
// Binding.ValidateWithDefaults. Built-in rules are applied implicitly; use
// WithRules to register custom rules.
func ValidateWithDefaults[TObject any](ctx context.Context, obj *TObject, opts ...Option) error {
	b, err := NewBinding[TObject](opts...)
	if err != nil {
		return err
	}

	return b.ValidateWithDefaults(ctx, obj)
}
