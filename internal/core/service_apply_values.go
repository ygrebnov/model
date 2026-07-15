package core

import (
	"reflect"

	"github.com/ygrebnov/errorc"

	fieldPkg "github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/schema"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// ApplyValuesStruct walks the compiled schema for rv and applies values supplied
// by source to matching fields. Nested pointer-to-struct fields are allocated
// only when a value is present for one of their descendants.
func (s *Service[T]) ApplyValuesStruct(
	rv reflect.Value,
	source fieldPkg.ValueSource,
) error {
	if source == nil {
		return errorc.With(
			modelerrors.ErrInvalidValue,
			errorc.String(keys.Phase, "apply_values"),
			errorc.String(keys.ValueType, "<nil>"),
		)
	}

	obj, ok := valuePointer[T](rv)
	if !ok {
		return errorc.With(
			modelerrors.ErrInvalidValue,
			errorc.String(keys.Phase, "apply_values"),
			errorc.String(keys.ValueType, rv.Type().String()),
		)
	}

	values := make(map[*schema.Node]any)

	// Use a temporary value to visit the complete schema, including fields
	// below nil pointer-to-struct nodes, without modifying the caller's value.
	probe := reflect.New(rv.Type()).Elem()
	probePolicy := walkPolicy{
		DiveCollection: func(walkContext, reflect.Value) bool {
			return false
		},
		AllocPtrStruct: func(walkContext, reflect.Value) bool {
			return true
		},
	}

	if err := walkSchema(
		probe,
		s.schemaController.GetRoot(),
		envPrefixPath(s.envPrefix),
		probePolicy,
		func(ctx walkContext, _ reflect.Value) error {
			value, ok, err := source.Get(ctx.Node.GetName("."))
			if err != nil {
				return errorc.With(
					err,
					errorc.String(keys.Phase, "apply_values"),
					errorc.String(keys.FieldName, ctx.Path),
				)
			}

			if ok {
				values[ctx.Node] = value
			}

			return nil
		},
	); err != nil {
		return err
	}

	policy := walkPolicy{
		DiveCollection: func(walkContext, reflect.Value) bool {
			return false
		},
		AllocPtrStruct: func(ctx walkContext, _ reflect.Value) bool {
			return hasProvidedDescendant(ctx.Node, values)
		},
	}

	return walkSchema(
		rv,
		s.schemaController.GetRoot(),
		envPrefixPath(s.envPrefix),
		policy,
		func(ctx walkContext, _ reflect.Value) error {
			value, ok := values[ctx.Node]
			if !ok {
				return nil
			}

			name := ctx.Node.GetName(".")
			if !s.schemaController.SetFieldValue(obj, name, value) {
				return errorc.With(
					modelerrors.ErrTypeMismatch,
					errorc.String(keys.Phase, "apply_values"),
					errorc.String(keys.FieldName, ctx.Path),
					errorc.String(keys.FieldType, ctx.Node.Type.String()),
					errorc.String(keys.ValueType, valueTypeName(value)),
				)
			}

			return nil
		},
	)
}

func hasProvidedDescendant(
	node *schema.Node,
	values map[*schema.Node]any,
) bool {
	if node == nil {
		return false
	}

	for _, child := range node.Children {
		if _, ok := values[child]; ok {
			return true
		}

		if hasProvidedDescendant(child, values) {
			return true
		}
	}

	return false
}

func valuePointer[T any](rv reflect.Value) (*T, bool) {
	rv = unwrapInterface(rv)
	if !rv.IsValid() {
		return nil, false
	}

	if rv.CanAddr() {
		if obj, ok := rv.Addr().Interface().(*T); ok {
			return obj, true
		}
	}

	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		if obj, ok := rv.Interface().(*T); ok {
			return obj, true
		}
	}

	return nil, false
}

func valueTypeName(value any) string {
	if value == nil {
		return "<nil>"
	}

	return reflect.TypeOf(value).String()
}
