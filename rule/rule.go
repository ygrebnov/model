package rule

import (
	"fmt"
	"reflect"
)

var ErrRuleTypeMismatch = fmt.Errorf("model: rule type mismatch")

type Rule interface {
	GetName() string
	IsOfType(t reflect.Type) bool
	GetAdapter() Adapter
}

// rule[FieldType] defines a named validation rule for a specific FieldType.
type rule[FieldType any] struct {
	metadata    Metadata
	fieldType   reflect.Type
	fn          func(value FieldType, params ...string) error
	reflectedFn func(v reflect.Value, params ...string) error
}

func NewRule[FieldType any](name string, fn func(value FieldType, params ...string) error) (Rule, error) {
	if name == "" || fn == nil {
		return nil, fmt.Errorf("rule must have non-empty Name and non-nil Fn")
	}

	// Capture the static type of FieldType even when FieldType is an interface.
	fieldType := reflect.TypeOf((*FieldType)(nil)).Elem()

	return rule[FieldType]{
		metadata:  Metadata{Name: name},
		fieldType: fieldType,
		fn:        fn,
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

func (r rule[FieldType]) GetName() string {
	return r.metadata.Name
}

func (r rule[FieldType]) IsOfType(t reflect.Type) bool {
	return r.fieldType == t
}

func (r rule[FieldType]) GetFn() func(value FieldType, params ...string) error {
	return r.fn
}

// Adapter holds a type-erased validation function along with the type it applies to.
type Adapter struct {
	fieldType reflect.Type
	fn        func(v reflect.Value, params ...string) error
}

// GetAdapter returns a type-erased adapter for the rule.
func (r rule[FieldType]) GetAdapter() Adapter {
	return Adapter{
		fieldType: r.fieldType,
		fn:        r.reflectedFn,
	}
}
