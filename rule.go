package model

import (
	"fmt"
	"reflect"
)

// RuleFn is the user-facing signature for a validation rule on fields of type TField.
// Params come from the `validate` tag, e.g. validate:"min(3)" -> params ["3"].
// Return nil if valid, or an error describing the validation failure.
type RuleFn[TField any] func(value TField, params ...string) error

// Rule associates a name with a RuleFn for registration in the model's validator registry.
// The Name is referenced by the `validate:"..."` tag on struct fields.
type Rule[TField any] struct {
	Name string
	Fn   RuleFn[TField]
}

// ruleFn is the internal signature for field-level validation rule.
// value: the reflect.Value of the field being validated
type ruleFn func(value reflect.Value, params ...string) error

// Internal carrier holding both the adapter and the accepted type (for overload resolution).
type typedAdapter struct {
	fieldType reflect.Type // the TField type
	fn        ruleFn       // calls user's Rule[TField]
}

func wrapRule[TField any](r RuleFn[TField]) typedAdapter {
	expectedType := reflect.TypeOf((*TField)(nil)).Elem() // works for interfaces too
	return typedAdapter{
		fieldType: expectedType,
		fn: func(v reflect.Value, params ...string) error {
			if !v.IsValid() || !v.Type().AssignableTo(expectedType) {
				panic(fmt.Sprintf(
					"model: rule type mismatch: field type %s not assignable to rule type %s",
					v.Type(), expectedType,
				))
			}
			val := v.Interface().(TField)
			return r(val, params...)
		},
	}
}
