package core

import (
	"fmt"
	"reflect"

	"github.com/ygrebnov/errorc"

	fieldPkg "github.com/ygrebnov/model/field"

	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// ApplyEnvStruct walks the compiled schema for rv and applies environment-backed
// values from source. Nested pointer-to-struct fields are allocated when
// descendant environment values are present.
func (s *Service) ApplyEnvStruct(rv reflect.Value, source fieldPkg.EnvSource) error {
	if source == nil {
		return errorc.With(
			errors.ErrInvalidValue,
			errorc.String(keys.Phase, "apply_env"),
			errorc.String(keys.ValueType, "<nil>"),
		)
	}

	return s.applyEnvStruct(rv, source, envPrefixPath(s.envPrefix))
}

// ApplySnapshotEnvStruct applies environment-backed values from the env snapshot
// captured when the Service was created.
func (s *Service) ApplySnapshotEnvStruct(rv reflect.Value) error {
	return s.ApplyEnvStruct(rv, s.envSource)
}

func (s *Service) applyEnvStruct(rv reflect.Value, source fieldPkg.EnvSource, envPath []string) error {
	compiled, err := s.schemaFor(rv.Type())
	if err != nil {
		return err
	}

	for _, node := range compiled.Root.Children {
		if err := s.applyNodeEnv(rv, node, envPath, source); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) applyNodeEnv(parent reflect.Value, node *schema.Node, envPath []string, source fieldPkg.EnvSource) error {
	if !node.EnvEnabled {
		return nil
	}

	fieldValue := parent.FieldByIndex(node.Index)
	fieldEnvPath := appendEnvPart(envPath, node.EnvName)

	if err := s.applyEnvValue(fieldValue, fieldEnvPath, node.Name, source); err != nil {
		return err
	}

	return s.applyNestedEnvValues(fieldValue, node, fieldEnvPath, source)
}

func (s *Service) applyEnvValue(fv reflect.Value, envPath []string, fieldName string, source fieldPkg.EnvSource) error {
	if !canSetLiteralValue(fv) {
		return nil
	}

	name := joinEnvPath(envPath)
	value, ok := source.Lookup(name)
	if !ok {
		return nil
	}

	if err := setLiteralValue(fv, value, false); err != nil {
		return errorc.With(
			errors.ErrSetDefault,
			errorc.String(keys.FieldName, fieldName),
			errorc.String("env", name),
			errorc.Error(keys.Cause, err),
		)
	}

	return nil
}

func (s *Service) applyNestedEnvValues(fv reflect.Value, node *schema.Node, envPath []string, source fieldPkg.EnvSource) error {
	value := fv

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			if value.Type().Elem().Kind() != reflect.Struct {
				return nil
			}
			if !s.hasNestedEnvValue(node, envPath, source) {
				return nil
			}
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		if isDurationType(value.Type()) {
			return nil
		}

		for _, child := range node.Children {
			if err := s.applyNodeEnv(value, child, envPath, source); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map:
		return s.applyMapEnvValues(value, envPath, source)

	case reflect.Slice, reflect.Array:
		// Environment variable support for slices is intentionally skipped.
		return nil
	}

	return nil
}

func (s *Service) hasNestedEnvValue(node *schema.Node, envPath []string, source fieldPkg.EnvSource) bool {
	for _, child := range node.Children {
		if !child.EnvEnabled {
			continue
		}

		childEnvPath := appendEnvPart(envPath, child.EnvName)
		if isSupportedLiteralType(child.Type) {
			if _, ok := source.Lookup(joinEnvPath(childEnvPath)); ok {
				return true
			}
			continue
		}

		if len(child.Children) > 0 && s.hasNestedEnvValue(child, childEnvPath, source) {
			return true
		}
	}

	return false
}

func (s *Service) applyMapEnvValues(mapValue reflect.Value, envPath []string, source fieldPkg.EnvSource) error {
	if mapValue.IsNil() {
		return nil
	}

	for _, key := range mapValue.MapKeys() {
		mapElemValue := mapValue.MapIndex(key)
		keyPath := appendEnvPart(envPath, fmt.Sprint(key.Interface()))

		if canSetLiteralValue(mapElemValue) {
			name := joinEnvPath(keyPath)
			value, ok := source.Lookup(name)
			if !ok {
				continue
			}

			updated := reflect.New(mapElemValue.Type()).Elem()
			updated.Set(mapElemValue)
			if err := setLiteralValue(updated, value, false); err != nil {
				return errorc.With(
					errors.ErrSetDefault,
					errorc.String("env", name),
					errorc.Error(keys.Cause, err),
				)
			}
			mapValue.SetMapIndex(key, updated)
			continue
		}

		if mapElemValue.Kind() == reflect.Ptr {
			if !mapElemValue.IsNil() && mapElemValue.Elem().Kind() == reflect.Struct {
				if err := s.applyEnvStruct(mapElemValue.Elem(), source, keyPath); err != nil {
					return err
				}
			}
			continue
		}

		if mapElemValue.Kind() == reflect.Struct {
			structValue := reflect.New(mapElemValue.Type()).Elem()
			structValue.Set(mapElemValue)
			if err := s.applyEnvStruct(structValue, source, keyPath); err != nil {
				return err
			}
			mapValue.SetMapIndex(key, structValue)
		}
	}

	return nil
}
