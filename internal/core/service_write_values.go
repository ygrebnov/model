package core

import (
	"reflect"

	"github.com/ygrebnov/errorc"
	fieldPkg "github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// WriteValuesStruct walks the compiled schema and writes all reachable field
// values to sink.
//
// Existing nested structs and collection elements are traversed. Nil
// pointer-to-struct fields and nil pointer collection elements are written as
// fields but are not allocated or traversed.
func (s *Service[T]) WriteValuesStruct(
	rv reflect.Value,
	sink fieldPkg.ValueSink,
) error {
	if sink == nil {
		return errorc.With(
			modelerrors.ErrInvalidValue,
			errorc.String(keys.Phase, "write_values"),
			errorc.String(keys.ValueType, "<nil>"),
		)
	}

	policy := walkPolicy{
		DiveCollection: func(ctx walkContext, _ reflect.Value) bool {
			return len(ctx.Node.Children) > 0 ||
				ctx.Node.Reference != nil
		},
		AllocPtrStruct: func(_ walkContext, _ reflect.Value) bool {
			return false
		},
	}

	return walkSchema(
		rv,
		s.schema.GetRoot(),
		nil,
		policy,
		func(ctx walkContext, field reflect.Value) error {
			field = unwrapInterface(field)
			if !field.IsValid() || !field.CanInterface() {
				return nil
			}

			name := ctx.Node.GetName(".")
			if err := sink.Set(name, field.Interface()); err != nil {
				return errorc.With(
					err,
					errorc.String(keys.Phase, "write_values"),
					errorc.String(keys.FieldName, ctx.Path),
				)
			}

			return nil
		},
	)
}
