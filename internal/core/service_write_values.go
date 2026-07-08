package core

import (
	"reflect"

	"github.com/ygrebnov/errorc"

	fieldPkg "github.com/ygrebnov/model/field"

	"github.com/ygrebnov/model/internal/schema"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// WriteValuesStruct walks the compiled schema for rv and writes reachable field
// values to sink. Nested pointer-to-struct fields are traversed only when
// non-nil.
func (s *Service) WriteValuesStruct(rv reflect.Value, sink fieldPkg.ValueSink) error {
	if sink == nil {
		return errorc.With(
			modelerrors.ErrInvalidValue,
			errorc.String(keys.Phase, "write_values"),
			errorc.String(keys.ValueType, "<nil>"),
		)
	}

	compiled, err := s.schemaFor(rv.Type())
	if err != nil {
		return err
	}

	for _, node := range compiled.Root.Children {
		if err := s.writeNodeValues(rv, node, sink); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) writeNodeValues(parent reflect.Value, node *schema.Node, sink fieldPkg.ValueSink) error {
	fieldValue := parent.FieldByIndex(node.Index)
	field := publicField(node)

	if err := sink.Set(field, fieldValue.Interface()); err != nil {
		return errorc.With(
			err,
			errorc.String(keys.Phase, "write_values"),
			errorc.String(keys.FieldName, field.Path),
		)
	}

	if len(node.Children) == 0 {
		return nil
	}

	nested := fieldValue
	if nested.Kind() == reflect.Ptr {
		if nested.IsNil() || nested.Type().Elem().Kind() != reflect.Struct {
			return nil
		}
		nested = nested.Elem()
	}

	if nested.Kind() != reflect.Struct || isDurationType(nested.Type()) {
		return nil
	}

	for _, child := range node.Children {
		if err := s.writeNodeValues(nested, child, sink); err != nil {
			return err
		}
	}

	return nil
}
