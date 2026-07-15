package core

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ygrebnov/model/validation"
)

// ValidateStruct walks the compiled schema and executes the validation plan
// prepared when the Service was constructed.
//
// Nested structs and non-nil pointers to structs are traversed automatically.
// Collection element structs are traversed only when validateElem contains the
// dive directive.
func (s *Service[T]) ValidateStruct(
	ctx context.Context,
	rv reflect.Value,
	path string,
	ve *validation.Error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	policy := walkPolicy{
		DiveCollection: func(ctx walkContext, _ reflect.Value) bool {
			return s.validation.nodes[ctx.Node].dive
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
		func(walkCtx walkContext, field reflect.Value) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			fieldPath := walkCtx.Path
			if path != "" {
				fieldPath = joinRuntimePath(path, fieldPath)
			}

			compiled := s.validation.nodes[walkCtx.Node]

			if err := applyCompiledRules(
				ctx,
				field,
				fieldPath,
				compiled.field,
				ve,
			); err != nil {
				return err
			}

			if compiled.dive {
				return validateDiveElements(
					ctx,
					field,
					fieldPath,
					ve,
				)
			}

			return validateCollectionElements(
				ctx,
				field,
				fieldPath,
				compiled.elem,
				ve,
			)
		},
	)
}

func applyCompiledRules(
	ctx context.Context,
	value reflect.Value,
	path string,
	compiled []compiledRule,
	ve *validation.Error,
) error {
	for _, rule := range compiled {
		if err := ctx.Err(); err != nil {
			return err
		}

		if rule.optional && value.IsZero() {
			continue
		}

		if err := rule.rule.GetValidationFn()(
			value,
			rule.params...,
		); err != nil {
			ve.Add(validation.FieldError{
				Path:   path,
				Rule:   rule.rule.GetName(),
				Params: rule.params,
				Err:    err,
			})
		}
	}

	return nil
}

func validateCollectionElements(
	ctx context.Context,
	fieldValue reflect.Value,
	fieldPath string,
	compiled []compiledRule,
	ve *validation.Error,
) error {
	if len(compiled) == 0 {
		return nil
	}

	container := unwrapInterface(fieldValue)
	if !container.IsValid() {
		return nil
	}

	if container.Kind() == reflect.Ptr {
		if container.IsNil() {
			return nil
		}

		container = unwrapInterface(container.Elem())
	}

	switch container.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < container.Len(); i++ {
			if err := applyCompiledRules(
				ctx,
				container.Index(i),
				collectionElementRuntimePath(fieldPath, i),
				compiled,
				ve,
			); err != nil {
				return err
			}
		}

	case reflect.Map:
		iterator := container.MapRange()
		for iterator.Next() {
			if err := applyCompiledRules(
				ctx,
				iterator.Value(),
				collectionElementRuntimePath(
					fieldPath,
					iterator.Key().Interface(),
				),
				compiled,
				ve,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateDiveElements records errors for nil pointer elements and unsupported
// element kinds. Non-nil struct elements are traversed and validated by
// walkSchema itself.
func validateDiveElements(
	ctx context.Context,
	fieldValue reflect.Value,
	fieldPath string,
	ve *validation.Error,
) error {
	container := unwrapInterface(fieldValue)
	if !container.IsValid() {
		return nil
	}

	if container.Kind() == reflect.Ptr {
		if container.IsNil() {
			return nil
		}

		container = unwrapInterface(container.Elem())
	}

	switch container.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < container.Len(); i++ {
			if err := ctx.Err(); err != nil {
				return err
			}

			validateDiveElement(
				container.Index(i),
				collectionElementRuntimePath(fieldPath, i),
				ve,
			)
		}

	case reflect.Map:
		iterator := container.MapRange()
		for iterator.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}

			validateDiveElement(
				iterator.Value(),
				collectionElementRuntimePath(
					fieldPath,
					iterator.Key().Interface(),
				),
				ve,
			)
		}
	}

	return nil
}

// collectionElementRuntimePath replaces the schema collection marker []
// with a concrete runtime slice/array index or map key.
func collectionElementRuntimePath(
	path string,
	key any,
) string {
	if strings.HasSuffix(path, "[]") {
		return fmt.Sprintf(
			"%s[%v]",
			strings.TrimSuffix(path, "[]"),
			key,
		)
	}

	return fmt.Sprintf("%s[%v]", path, key)
}

func validateDiveElement(
	element reflect.Value,
	path string,
	ve *validation.Error,
) {
	element = unwrapInterface(element)
	if !element.IsValid() {
		ve.Add(validation.FieldError{
			Path: path,
			Rule: tagDive,
			Err: fmt.Errorf(
				"validateElem:\"dive\" requires a valid struct element",
			),
		})
		return
	}

	if element.Kind() == reflect.Ptr {
		if element.IsNil() {
			ve.Add(validation.FieldError{
				Path: path,
				Rule: tagDive,
				Err: fmt.Errorf(
					"validateElem:\"dive\" requires non-nil struct element",
				),
			})
			return
		}

		element = unwrapInterface(element.Elem())
	}

	if !element.IsValid() || element.Kind() != reflect.Struct {
		kind := reflect.Invalid
		if element.IsValid() {
			kind = element.Kind()
		}

		ve.Add(validation.FieldError{
			Path: path,
			Rule: tagDive,
			Err: fmt.Errorf(
				"validateElem:\"dive\" requires struct element, got %s",
				kind,
			),
		})
	}
}
