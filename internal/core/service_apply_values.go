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
// when descendant field values are present.
func (s *Service) ApplyValuesStruct(rv reflect.Value, source fieldPkg.ValueSource) error {
	if source == nil {
		return errorc.With(
			modelerrors.ErrInvalidValue,
			errorc.String(keys.Phase, "apply_values"),
			errorc.String(keys.ValueType, "<nil>"),
		)
	}

	compiled, err := s.schemaFor(rv.Type())
	if err != nil {
		return err
	}

	values := make(map[string]any)
	for _, node := range compiled.Fields() {
		field := publicField(node)
		value, ok, err := source.Get(field)
		if err != nil {
			return errorc.With(
				err,
				errorc.String(keys.Phase, "apply_values"),
				errorc.String(keys.FieldName, field.Path),
			)
		}
		if ok {
			values[node.Path] = value
		}
	}

	for _, node := range compiled.Root.Children {
		if err := s.applyNodeValues(rv, node, values); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) applyNodeValues(parent reflect.Value, node *schema.Node, values map[string]any) error {
	fieldValue := parent.FieldByIndex(node.Index)

	if value, ok := values[node.Path]; ok {
		if err := setAssignedValue(fieldValue, value); err != nil {
			return errorc.With(
				err,
				errorc.String(keys.Phase, "apply_values"),
				errorc.String(keys.FieldName, node.Path),
			)
		}
	}

	if len(node.Children) == 0 || !hasDescendantValue(node, values) {
		return nil
	}

	nested := fieldValue
	if nested.Kind() == reflect.Ptr {
		if nested.IsNil() {
			if nested.Type().Elem().Kind() != reflect.Struct {
				return nil
			}
			nested.Set(reflect.New(nested.Type().Elem()))
		}
		nested = nested.Elem()
	}

	if nested.Kind() != reflect.Struct || isDurationType(nested.Type()) {
		return nil
	}

	for _, child := range node.Children {
		if err := s.applyNodeValues(nested, child, values); err != nil {
			return err
		}
	}

	return nil
}

func hasDescendantValue(node *schema.Node, values map[string]any) bool {
	for _, child := range node.Children {
		if _, ok := values[child.Path]; ok {
			return true
		}
		if hasDescendantValue(child, values) {
			return true
		}
	}

	return false
}

func publicField(node *schema.Node) fieldPkg.Field {
	envPath := append([]string(nil), node.EnvPath...)

	return fieldPkg.Field{
		Path:            node.Path,
		Name:            node.Name,
		Type:            node.Type,
		JSONName:        node.JSONName,
		EnvPath:         envPath,
		DefaultTag:      node.DefaultTag,
		DefaultElemTag:  node.DefaultElemTag,
		ValidateTag:     node.ValidateTag,
		ValidateElemTag: node.ValidateElemTag,
	}
}

func setAssignedValue(target reflect.Value, value any) error {
	if !target.CanSet() {
		return modelerrors.ErrInvalidValue
	}
	if value == nil {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	return assignValue(target, reflect.ValueOf(value))
}

func assignValue(target, source reflect.Value) error {
	if source.Type().AssignableTo(target.Type()) {
		target.Set(source)
		return nil
	}
	if source.Type().ConvertibleTo(target.Type()) {
		target.Set(source.Convert(target.Type()))
		return nil
	}

	if target.Kind() == reflect.Ptr {
		elemType := target.Type().Elem()
		if source.Type().AssignableTo(elemType) || source.Type().ConvertibleTo(elemType) {
			if target.IsNil() {
				target.Set(reflect.New(elemType))
			}
			return assignValue(target.Elem(), source)
		}
	}

	if source.Kind() == reflect.Ptr && !source.IsNil() {
		return assignValue(target, source.Elem())
	}

	return errorc.With(
		modelerrors.ErrTypeMismatch,
		errorc.String(keys.FieldType, target.Type().String()),
		errorc.String(keys.ValueType, source.Type().String()),
	)
}
