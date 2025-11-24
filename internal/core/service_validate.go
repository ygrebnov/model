package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ygrebnov/model/validation"
)

func (s *Service) AddRule(r validation.Rule) error {
	return s.rulesRegistry.Add(r)
}

// ValidateStruct walks a struct value and applies rules on each field according to its `validate` tag.
// Nested structs and pointers to structs are traversed recursively. The `path` argument tracks the
// dotted field path for clearer error messages.
func (s *Service) ValidateStruct(
	ctx context.Context,
	rv reflect.Value,
	path string,
	ve *validation.Error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	typ := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		field := typ.Field(i)
		if field.PkgPath != "" { // Skip unexported fields
			continue
		}
		fv := rv.Field(i)

		fpath := field.Name
		if path != "" {
			fpath = path + "." + field.Name
		}

		// Recurse into pointers to structs
		if fv.Kind() == reflect.Ptr && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
			if err := s.ValidateStruct(ctx, fv.Elem(), fpath, ve); err != nil {
				return err
			}
		}

		// Recurse into embedded/inline structs
		if fv.Kind() == reflect.Struct {
			if err := s.ValidateStruct(ctx, fv, fpath, ve); err != nil {
				return err
			}
		}

		// Process `validate` tag
		if err := s.processValidateTag(ctx, &field, fpath, fv, typ, i, ve); err != nil {
			return err
		}

		// Process `validateElem` tag for slices, arrays, and maps
		if err := s.processValidateElemTag(ctx, &field, fpath, fv, typ, i, ve); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) processValidateTag(
	ctx context.Context,
	field *reflect.StructField,
	fieldPath string,
	fieldValue reflect.Value,
	structType reflect.Type,
	fieldIndex int,
	ve *validation.Error,
) error {
	rawTag := field.Tag.Get(tagValidate)
	if rawTag == "" || rawTag == "-" {
		return nil
	}
	// Check cache for parsed rules
	rules, exists := s.rulesMapping.Get(structType, fieldIndex, tagValidate)
	if !exists {
		rules = validation.ParseTag(rawTag)
		s.rulesMapping.Add(structType, fieldIndex, tagValidate, rules)
	}

	for _, r := range rules {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.applyRule(r.Name, fieldValue, r.Params...); err != nil {
			ve.Add(validation.FieldError{Path: fieldPath, Rule: r.Name, Params: r.Params, Err: err})
		}
	}

	return nil
}

func (s *Service) processValidateElemTag(
	ctx context.Context,
	field *reflect.StructField,
	fieldPath string,
	fieldValue reflect.Value,
	structType reflect.Type,
	fieldIndex int,
	ve *validation.Error,
) error {
	elemRaw := field.Tag.Get(tagValidateElem)
	if elemRaw == "" || elemRaw == "-" {
		return nil
	}

	// Check cache for parsed rules
	elemRules, exists := s.rulesMapping.Get(structType, fieldIndex, tagValidateElem)
	if !exists {
		elemRules = validation.ParseTag(elemRaw)
		s.rulesMapping.Add(structType, fieldIndex, tagValidateElem, elemRules)
	}

	if err := s.validateElements(ctx, fieldValue, fieldPath, elemRules, ve); err != nil {
		return err
	}

	return nil
}

// validateElements applies validation rules to elements of a slice, array, or map
// using pre-parsed rules (e.g., retrieved from the cache).
func (s *Service) validateElements(
	ctx context.Context,
	fv reflect.Value,
	fpath string,
	rules []validation.RuleNameParams,
	ve *validation.Error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cont := fv
	if cont.Kind() == reflect.Ptr && !cont.IsNil() {
		cont = cont.Elem()
	}
	if len(rules) == 0 {
		return nil
	}
	// Special case: validateElem:"dive" means recurse into element structs
	isDiveOnly := len(rules) == 1 && rules[0].Name == tagDive && len(rules[0].Params) == 0

	switch cont.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < cont.Len(); i++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			elem := cont.Index(i)
			pathIdx := fmt.Sprintf("%s[%d]", fpath, i)
			if err := s.validateSingleElement(ctx, elem, pathIdx, rules, isDiveOnly, ve); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range cont.MapKeys() {
			if err := ctx.Err(); err != nil {
				return err
			}
			elem := cont.MapIndex(key)
			pathKey := fmt.Sprintf("%s[%v]", fpath, key.Interface())
			if err := s.validateSingleElement(ctx, elem, pathKey, rules, isDiveOnly, ve); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateSingleElement handles validation for a single item from a collection.
func (s *Service) validateSingleElement(
	ctx context.Context,
	elem reflect.Value,
	path string,
	rules []validation.RuleNameParams,
	isDiveOnly bool,
	ve *validation.Error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if isDiveOnly {
		dv := elem
		if dv.Kind() == reflect.Ptr && !dv.IsNil() {
			dv = dv.Elem()
		}
		if dv.Kind() == reflect.Struct {
			return s.ValidateStruct(ctx, dv, path, ve)
		}
		ve.Add(
			validation.FieldError{
				Path: path,
				Rule: tagDive,
				Err:  fmt.Errorf("validateElem:\"dive\" requires struct element, got %s", dv.Kind()),
			},
		)
		return nil
	}

	for _, r := range rules {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.applyRule(r.Name, elem, r.Params...); err != nil {
			ve.Add(validation.FieldError{Path: path, Rule: r.Name, Params: r.Params, Err: err})
		}
	}
	return nil
}

// applyRule fetches the named rule from the registry and applies it to the given reflect.Value v,
// passing any additional string parameters.
// If the rule is not found or fails, an error is returned.
func (s *Service) applyRule(name string, v reflect.Value, params ...string) error {
	r, err := s.rulesRegistry.Get(name, v)
	if err != nil {
		return err
	}
	return r.GetValidationFn()(v, params...)
}
