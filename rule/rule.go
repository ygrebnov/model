package rule

import (
	"fmt"
	"reflect"
)

var (
	ErrRuleTypeMismatch = fmt.Errorf("rule type mismatch")
	ErrInvalidRule      = fmt.Errorf("rule must have non-empty Name and non-nil Fn")
)

type Rule interface {
	GetName() string
	GetFieldTypeName() string
	GetValidationFn() func(v reflect.Value, params ...string) error
	IsOfType(t reflect.Type) bool
	IsAssignableTo(t reflect.Type) bool
}

// rule[FieldType] defines a named validation rule for a specific FieldType.
type rule struct {
	metadata  Metadata
	fieldType reflect.Type
	//fn          func(value FieldType, params ...string) error
	reflectedFn func(v reflect.Value, params ...string) error
}

func NewRule[FieldType any](name string, fn func(value FieldType, params ...string) error) (Rule, error) {
	if name == "" || fn == nil {
		return nil, ErrInvalidRule
	}

	// Capture the static type of FieldType even when FieldType is an interface.
	fieldType := reflect.TypeOf((*FieldType)(nil)).Elem()

	return rule{
		metadata:  Metadata{Name: name},
		fieldType: fieldType,
		//fn:        fn,
		reflectedFn: func(v reflect.Value, params ...string) error {
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

func (r rule) GetName() string {
	return r.metadata.Name
}

func (r rule) GetFieldTypeName() string {
	if r.fieldType == nil {
		return ""
	}
	return r.fieldType.String()
}

func (r rule) GetValidationFn() func(v reflect.Value, params ...string) error {
	return r.reflectedFn
}

func (r rule) IsOfType(t reflect.Type) bool {
	return r.fieldType == t
}

func (r rule) IsAssignableTo(t reflect.Type) bool {
	// TODO: if r.fieldType is nil?
	return t.AssignableTo(r.fieldType)
}

// Adapter holds a type-erased validation function along with the type it applies to.
type Adapter struct {
	fieldType reflect.Type
	Fn        func(v reflect.Value, params ...string) error
}
