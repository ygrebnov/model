package validation

import (
	"reflect"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/errors"
)

type Rule interface {
	GetName() string
	GetValidationFn() func(v reflect.Value, params ...string) error

	getFieldTypeName() string
	getFieldType() reflect.Type
	isOfType(t reflect.Type) bool
	isAssignableTo(t reflect.Type) bool
}

// rule defines a named validation function for a specific field type.
type rule struct {
	name      string
	fieldType reflect.Type
	fn        func(v reflect.Value, params ...string) error
}

func NewRule[FieldType any](name string, fn func(value FieldType, params ...string) error) (Rule, error) {
	if name == "" || fn == nil {
		return nil, errors.ErrInvalidRule
	}

	// Capture the static type of FieldType even when FieldType is an interface.
	fieldType := reflect.TypeOf((*FieldType)(nil)).Elem()

	return &rule{
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
							errorc.String(errors.ErrorFieldValueType, v.Type().String()),
							errorc.String(errors.ErrorFieldFieldType, fieldType.String()),
						)
					}
				}
			}
			val := v.Interface().(FieldType)
			return fn(val, params...)
		},
	}, nil
}

func (r *rule) GetName() string {
	return r.name
}

func (r *rule) getFieldTypeName() string {
	if r.fieldType == nil {
		return "" // defensive, cannot happen due to constructor check
	}
	return r.fieldType.String()
}

func (r *rule) getFieldType() reflect.Type {
	return r.fieldType
}

func (r *rule) GetValidationFn() func(v reflect.Value, params ...string) error {
	return r.fn
}

func (r *rule) isOfType(t reflect.Type) bool {
	return r.fieldType == t
}

func (r *rule) isAssignableTo(t reflect.Type) bool {
	if r.fieldType == nil {
		return false // defensive, cannot happen due to constructor check
	}
	return t.AssignableTo(r.fieldType)
}
