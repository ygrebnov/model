package core

import (
	"reflect"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// ApplyEnvStruct walks the compiled schema and applies environment-backed values
// from the snapshot captured when the Service was created.
func (s *Service[T]) ApplyEnvStruct(root reflect.Value) error {
	if s.envSource == nil {
		return errors.ErrNilEnvSource
	}

	policy := walkPolicy{
		DiveCollection: func(ctx walkContext, _ reflect.Value) bool {
			return ctx.EnvEnabled
		},
		AllocPtrStruct: func(ctx walkContext, _ reflect.Value) bool {
			return ctx.EnvEnabled &&
				s.hasNestedEnvValue(ctx.Node, ctx.EnvPath)
		},
	}

	return walkSchema(
		root,
		s.schema.GetRoot(),
		envPrefixPath(s.envPrefix),
		policy,
		s.applyEnvWalkValue,
	)
}

// applyEnvWalkValue applies one environment value to the concrete field visited
// by walkSchema.
func (s *Service[T]) applyEnvWalkValue(
	ctx walkContext,
	field reflect.Value,
) error {
	if !ctx.EnvEnabled {
		return nil
	}

	field = unwrapInterface(field)
	s.materializeEnvSliceElements(ctx, field)

	if !canSetLiteralValue(field) {
		return nil
	}

	envName := joinEnvPath(ctx.EnvPath)
	value, ok := s.envSource.Lookup(envName)
	if !ok {
		return nil
	}

	if err := setLiteralValue(field, value, false); err != nil {
		return errorc.With(
			errors.ErrSetDefault,
			errorc.String(keys.FieldName, ctx.Path),
			errorc.String("env", envName),
			errorc.Error(keys.Cause, err),
		)
	}

	return nil
}

// hasNestedEnvValue reports whether any descendant node has a value in the
// environment snapshot below envPath.
func (s *Service[T]) hasNestedEnvValue(
	node *schema.Node,
	envPath []string,
) bool {
	for _, child := range node.Children {
		childEnvPath, childEnvEnabled := applyWalkNodeEnvPath(
			envPath,
			true,
			child,
		)
		if !childEnvEnabled {
			continue
		}

		if isStructSliceNode(child) &&
			len(s.envCollectionIndexes(childEnvPath)) > 0 {
			return true
		}

		if isSupportedLiteralType(child.Type) {
			if _, ok := s.envSource.Lookup(
				joinEnvPath(childEnvPath),
			); ok {
				return true
			}

			continue
		}

		if len(child.Children) > 0 &&
			s.hasNestedEnvValue(child, childEnvPath) {
			return true
		}
	}

	return false
}

func (s *Service[T]) materializeEnvSliceElements(
	ctx walkContext,
	field reflect.Value,
) {
	if !isStructSliceNode(ctx.Node) ||
		!field.IsValid() ||
		field.Kind() != reflect.Slice ||
		!field.CanSet() {
		return
	}

	indexes := s.envCollectionIndexes(ctx.EnvPath)
	if len(indexes) == 0 {
		return
	}

	lastIndex := indexes[len(indexes)-1]
	if field.Len() <= lastIndex {
		expanded := reflect.MakeSlice(field.Type(), lastIndex+1, lastIndex+1)
		reflect.Copy(expanded, field)
		field.Set(expanded)
	}

	elementType := field.Type().Elem()
	if elementType.Kind() != reflect.Ptr ||
		elementType.Elem().Kind() != reflect.Struct {
		return
	}

	for _, index := range indexes {
		element := field.Index(index)
		if element.IsNil() {
			element.Set(reflect.New(elementType.Elem()))
		}
	}
}

func (s *Service[T]) envCollectionIndexes(envPath []string) []int {
	source, ok := s.envSource.(envSnapshotSource)
	if !ok {
		return nil
	}

	return source.CollectionIndexes(joinEnvPath(envPath))
}

func isStructSliceNode(node *schema.Node) bool {
	if node == nil {
		return false
	}

	typ := node.Type
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Slice {
		return false
	}

	return nestedStructOrPointerType(typ.Elem())
}

func nestedStructOrPointerType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ.Kind() == reflect.Struct && !isDurationType(typ)
}
