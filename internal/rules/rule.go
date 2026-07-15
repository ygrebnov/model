package rules

import (
	"reflect"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

/*
// Rule represents a named validation rule bound to a specific field type.
type Rule interface {
	// GetName returns the rule name used in struct tags and registration.
	GetName() string
	// GetValidationFn returns the reflect-based validation function for the rule.
	GetValidationFn() func(v reflect.Value, params ...string) error

	getFieldTypeName() string
	getFieldType() reflect.Type
	isOfType(t reflect.Type) bool
	isAssignableTo(t reflect.Type) bool
}
*/

// Rule defines a named validation function for a specific field type.
type Rule struct {
	name      string
	fieldType reflect.Type
	fn        func(v reflect.Value, params ...string) error
}

// NewRule creates a typed validation rule with the given name and validation function.
func NewRule[FieldType any](name string, fn func(value FieldType, params ...string) error) (*Rule, error) {
	if name == "" || fn == nil {
		return nil, errors.ErrInvalidRule
	}

	// Capture the static type of FieldType even when FieldType is an interface.
	fieldType := reflect.TypeOf((*FieldType)(nil)).Elem()

	return &Rule{
		name:      name,
		fieldType: fieldType,
		// fn:        fn,
		fn: func(v reflect.Value, params ...string) error {
			// Ensure the reflect.Value `v` is compatible with FieldType.
			if v.Type() != fieldType {
				// Accept assignable values (including types implementing an interface FieldType)
				if !v.Type().AssignableTo(fieldType) {
					// As a fallback for interface FieldType, use Implements for clarity.
					if !(fieldType.Kind() == reflect.Interface && v.Type().Implements(fieldType)) {
						return errorc.With(
							errors.ErrRuleTypeMismatch,
							errorc.String(keys.ValueType, v.Type().String()),
							errorc.String(keys.FieldType, fieldType.String()),
						)
					}
				}
			}
			val := v.Interface().(FieldType)
			return fn(val, params...)
		},
	}, nil
}

func (r *Rule) GetName() string {
	return r.name
}

func (r *Rule) GetFieldTypeName() string {
	if r.fieldType == nil {
		return "" // defensive, cannot happen due to constructor check
	}
	return r.fieldType.String()
}

func (r *Rule) getFieldType() reflect.Type {
	return r.fieldType
}

func (r *Rule) GetValidationFn() func(v reflect.Value, params ...string) error {
	return r.fn
}

func (r *Rule) isOfType(t reflect.Type) bool {
	return r.fieldType == t
}

func (r *Rule) IsAssignableTo(t reflect.Type) bool {
	if r.fieldType == nil {
		return false // defensive, cannot happen due to constructor check
	}
	return t.AssignableTo(r.fieldType)
}
