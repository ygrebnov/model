package model

import (
	"fmt"
	"reflect"
)

var (
	ErrRuleTypeMismatch = fmt.Errorf("rule type mismatch")
	ErrInvalidRule      = fmt.Errorf("rule must have non-empty name and non-nil Fn")
)

// rule defines a named validation function for a specific field type.
type rule struct {
	name      string
	fieldType reflect.Type
	fn        func(v reflect.Value, params ...string) error
}

func newRule[FieldType any](name string, fn func(value FieldType, params ...string) error) (*rule, error) {
	if name == "" || fn == nil {
		return nil, ErrInvalidRule
	}

	// Capture the static type of FieldType even when FieldType is an interface.
	fieldType := reflect.TypeOf((*FieldType)(nil)).Elem()

	return &rule{
		name:      name,
		fieldType: fieldType,
		//fn:        fn,
		fn: func(v reflect.Value, params ...string) error {
			// Ensure the reflect.Value `v` is compatible with FieldType.
			if v.Type() != fieldType {
				// Accept assignable values (including types implementing an interface FieldType)
				if !v.Type().AssignableTo(fieldType) {
					// As a fallback for interface FieldType, use Implements for clarity.
					if !(fieldType.Kind() == reflect.Interface && v.Type().Implements(fieldType)) {
						return fmt.Errorf(
							"%w: cannot use %s value with rule for type %s",
							ErrRuleTypeMismatch,
							v.Type(),
							fieldType,
						)
					}
				}
			}
			val := v.Interface().(FieldType)
			return fn(val, params...)
		},
	}, nil
}

func (r rule) getName() string {
	return r.name
}

func (r rule) getFieldTypeName() string {
	if r.fieldType == nil {
		return ""
	}
	return r.fieldType.String()
}

func (r rule) getValidationFn() func(v reflect.Value, params ...string) error {
	return r.fn
}

func (r rule) isOfType(t reflect.Type) bool {
	return r.fieldType == t
}

func (r rule) isAssignableTo(t reflect.Type) bool {
	// TODO: if r.fieldType is nil?
	return t.AssignableTo(r.fieldType)
}
