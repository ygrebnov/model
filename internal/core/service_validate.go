package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/validation"
)

// AddRule registers a validation rule in the service's rules registry.
func (s *Service[T]) AddRule(r validation.Rule) error {
	return s.rulesRegistry.Add(r)
}

// ValidateStruct walks the compiled schema and applies rules declared through
// `validate` and `validateElem` tags.
//
// Nested structs and non-nil pointers to structs are traversed automatically.
// Collection element structs are traversed only when the collection declares
// `validateElem:"dive"`.
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
			return isValidateElemDive(ctx.Node.ValidateElemTag)
		},
		AllocPtrStruct: func(_ walkContext, _ reflect.Value) bool {
			return false
		},
	}

	return walkSchema(
		rv,
		s.schemaController.GetRoot(),
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

			if err := s.processValidateTag(
				ctx,
				walkCtx.Node,
				fieldPath,
				field,
				ve,
			); err != nil {
				return err
			}

			return s.processValidateElemTag(
				ctx,
				walkCtx.Node,
				fieldPath,
				field,
				ve,
			)
		},
	)
}

func (s *Service[T]) processValidateTag(
	ctx context.Context,
	node *schema.N,
	fieldPath string,
	fieldValue reflect.Value,
	ve *validation.Error,
) error {
	rawTag := node.ValidateTag
	if rawTag == "" {
		return nil
	}

	rules := validation.ParseTag(rawTag)
	for _, rule := range rules {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := s.applyRule(
			rule.Name,
			fieldValue,
			rule.Optional,
			rule.Params...,
		); err != nil {
			ve.Add(validation.FieldError{
				Path:   fieldPath,
				Rule:   rule.Name,
				Params: rule.Params,
				Err:    err,
			})
		}
	}

	return nil
}

func (s *Service[T]) processValidateElemTag(
	ctx context.Context,
	node *schema.N,
	fieldPath string,
	fieldValue reflect.Value,
	ve *validation.Error,
) error {
	rawTag := node.ValidateElemTag
	if rawTag == "" {
		return nil
	}

	rules := validation.ParseTag(rawTag)
	if len(rules) == 0 {
		return nil
	}

	if isValidateElemDiveRules(rules) {
		return validateDiveElements(
			ctx,
			fieldValue,
			fieldPath,
			ve,
		)
	}

	return s.validateElements(
		ctx,
		fieldValue,
		fieldPath,
		rules,
		ve,
	)
}

func isValidateElemDive(rawTag string) bool {
	return isValidateElemDiveRules(
		validation.ParseTag(rawTag),
	)
}

func isValidateElemDiveRules(
	rules []validation.RuleNameParams,
) bool {
	return len(rules) == 1 &&
		rules[0].Name == tagDive &&
		len(rules[0].Params) == 0
}

// validateDiveElements records errors for nil pointer elements and unsupported
// element kinds. Non-nil struct elements are validated by walkSchema itself.
func validateDiveElements(
	ctx context.Context,
	fieldValue reflect.Value,
	fieldPath string,
	ve *validation.Error,
) error {
	container := unwrapInterface(fieldValue)

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

			elementPath := fmt.Sprintf(
				"%s[%d]",
				fieldPath,
				i,
			)

			validateDiveElement(
				container.Index(i),
				elementPath,
				ve,
			)
		}

	case reflect.Map:
		for _, key := range container.MapKeys() {
			if err := ctx.Err(); err != nil {
				return err
			}

			elementPath := fmt.Sprintf(
				"%s[%v]",
				fieldPath,
				key.Interface(),
			)

			validateDiveElement(
				container.MapIndex(key),
				elementPath,
				ve,
			)
		}
	}

	return nil
}

func validateDiveElement(
	element reflect.Value,
	path string,
	ve *validation.Error,
) {
	element = unwrapInterface(element)

	if element.Kind() == reflect.Ptr {
		if element.IsNil() {
			ve.Add(validation.FieldError{
				Path: path,
				Rule: tagDive,
				Err: fmt.Errorf(
					"validateElem:\"dive\" requires " +
						"non-nil struct element",
				),
			})

			return
		}

		element = unwrapInterface(element.Elem())
	}

	if element.Kind() != reflect.Struct {
		ve.Add(validation.FieldError{
			Path: path,
			Rule: tagDive,
			Err: fmt.Errorf(
				"validateElem:\"dive\" requires "+
					"struct element, got %s",
				element.Kind(),
			),
		})
	}
}

func (s *Service[T]) validateElements(
	ctx context.Context,
	fieldValue reflect.Value,
	fieldPath string,
	rules []validation.RuleNameParams,
	ve *validation.Error,
) error {
	container := unwrapInterface(fieldValue)

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

			if err := s.validateSingleElement(
				ctx,
				container.Index(i),
				fmt.Sprintf(
					"%s[%d]",
					fieldPath,
					i,
				),
				rules,
				ve,
			); err != nil {
				return err
			}
		}

	case reflect.Map:
		for _, key := range container.MapKeys() {
			if err := ctx.Err(); err != nil {
				return err
			}

			if err := s.validateSingleElement(
				ctx,
				container.MapIndex(key),
				fmt.Sprintf(
					"%s[%v]",
					fieldPath,
					key.Interface(),
				),
				rules,
				ve,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service[T]) validateSingleElement(
	ctx context.Context,
	element reflect.Value,
	path string,
	rules []validation.RuleNameParams,
	ve *validation.Error,
) error {
	for _, rule := range rules {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := s.applyRule(
			rule.Name,
			element,
			rule.Optional,
			rule.Params...,
		); err != nil {
			ve.Add(validation.FieldError{
				Path:   path,
				Rule:   rule.Name,
				Params: rule.Params,
				Err:    err,
			})
		}
	}

	return nil
}

// applyRule fetches the named rule from the registry and applies it to v.
func (s *Service[T]) applyRule(
	name string,
	v reflect.Value,
	optional bool,
	params ...string,
) error {
	if optional && v.IsZero() {
		return nil
	}

	rule, err := s.rulesRegistry.Get(name, v)
	if err != nil {
		return err
	}

	return rule.GetValidationFn()(v, params...)
}
