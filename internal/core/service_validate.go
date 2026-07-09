package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/validation"
)

type validationState struct {
	activePointers map[validationVisit]struct{}
}

type validationVisit struct {
	typ  reflect.Type
	addr uintptr
}

func (vs *validationState) enterStruct(rv reflect.Value) (b bool, f func()) {
	if rv.Kind() != reflect.Struct || !rv.CanAddr() {
		return true, func() {}
	}

	visit := validationVisit{
		typ:  rv.Type(),
		addr: rv.Addr().Pointer(),
	}
	if _, exists := vs.activePointers[visit]; exists {
		return false, func() {}
	}

	vs.activePointers[visit] = struct{}{}
	return true, func() {
		delete(vs.activePointers, visit)
	}
}

// AddRule registers a validation rule in the service's rules registry.
func (s *Service[T]) AddRule(r validation.Rule) error {
	return s.rulesRegistry.Add(r)
}

// ValidateStruct walks a struct value and applies rules on each field according to its `validate` tag.
// Nested structs and pointers to structs are traversed recursively. Cycles in pointer graphs are
// skipped on the current traversal path so validation terminates even for self-referential data.
// The `path` argument tracks the dotted field path for clearer error messages.
func (s *Service[T]) ValidateStruct(
	ctx context.Context,
	rv reflect.Value,
	path string,
	ve *validation.Error,
) error {
	state := &validationState{
		activePointers: make(map[validationVisit]struct{}),
	}
	return s.validateStruct(ctx, rv, path, ve, state)
}

func (s *Service[T]) validateStruct(
	ctx context.Context,
	rv reflect.Value,
	path string,
	ve *validation.Error,
	state *validationState,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	entered, leave := state.enterStruct(rv)
	if !entered {
		return nil
	}
	defer leave()

	compiled, err := s.schemaFor(rv.Type())
	if err != nil {
		return err
	}

	for _, field := range compiled.Root.Children {
		if err := ctx.Err(); err != nil {
			return err
		}
		fv := rv.FieldByIndex(field.Index)

		fpath := field.Name
		if path != "" {
			fpath = path + "." + field.Name
		}

		// Recurse into pointers to structs
		if fv.Kind() == reflect.Ptr && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
			if err := s.validateStruct(ctx, fv.Elem(), fpath, ve, state); err != nil {
				return err
			}
		}

		// Recurse into embedded/inline structs
		if fv.Kind() == reflect.Struct {
			if err := s.validateStruct(ctx, fv, fpath, ve, state); err != nil {
				return err
			}
		}

		// Process `validate` tag
		if err := s.processValidateTag(ctx, field, fpath, fv, rv.Type(), ve); err != nil {
			return err
		}

		// Process `validateElem` tag for slices, arrays, and maps
		if err := s.processValidateElemTag(ctx, field, fpath, fv, rv.Type(), ve, state); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service[T]) processValidateTag(
	ctx context.Context,
	field *schema.Node,
	fieldPath string,
	fieldValue reflect.Value,
	structType reflect.Type,
	ve *validation.Error,
) error {
	rawTag := field.ValidateTag
	if rawTag == "" {
		return nil
	}

	fieldIndex := fieldRulesIndex(field)
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
		if err := s.applyRule(r.Name, fieldValue, r.Optional, r.Params...); err != nil {
			ve.Add(validation.FieldError{Path: fieldPath, Rule: r.Name, Params: r.Params, Err: err})
		}
	}

	return nil
}

func (s *Service[T]) processValidateElemTag(
	ctx context.Context,
	field *schema.Node,
	fieldPath string,
	fieldValue reflect.Value,
	structType reflect.Type,
	ve *validation.Error,
	state *validationState,
) error {
	elemRaw := field.ValidateElemTag
	if elemRaw == "" {
		return nil
	}

	fieldIndex := fieldRulesIndex(field)

	// Check cache for parsed rules
	elemRules, exists := s.rulesMapping.Get(structType, fieldIndex, tagValidateElem)
	if !exists {
		elemRules = validation.ParseTag(elemRaw)
		s.rulesMapping.Add(structType, fieldIndex, tagValidateElem, elemRules)
	}

	if err := s.validateElements(ctx, fieldValue, fieldPath, elemRules, ve, state); err != nil {
		return err
	}

	return nil
}

func (s *Service[T]) validateElements(
	ctx context.Context,
	fv reflect.Value,
	fpath string,
	rules []validation.RuleNameParams,
	ve *validation.Error,
	state *validationState,
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
			if err := s.validateSingleElement(ctx, elem, pathIdx, rules, isDiveOnly, ve, state); err != nil {
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
			if err := s.validateSingleElement(ctx, elem, pathKey, rules, isDiveOnly, ve, state); err != nil {
				return err
			}
		}
	}
	return nil
}

func fieldRulesIndex(field *schema.Node) int {
	if field == nil || len(field.Index) == 0 {
		return 0
	}

	return field.Index[0]
}

// validateSingleElement handles validation for a single item from a collection.
func (s *Service[T]) validateSingleElement(
	ctx context.Context,
	elem reflect.Value,
	path string,
	rules []validation.RuleNameParams,
	isDiveOnly bool,
	ve *validation.Error,
	state *validationState,
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
			return s.validateStruct(ctx, dv, path, ve, state)
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
		if err := s.applyRule(r.Name, elem, r.Optional, r.Params...); err != nil {
			ve.Add(validation.FieldError{Path: path, Rule: r.Name, Params: r.Params, Err: err})
		}
	}
	return nil
}

// applyRule fetches the named rule from the registry and applies it to the given reflect.Value v,
// passing any additional string parameters.
// If the rule is not found or fails, an error is returned.
func (s *Service[T]) applyRule(name string, v reflect.Value, optional bool, params ...string) error {
	isZero := v.IsZero()
	if optional && isZero {
		return nil
	}
	r, err := s.rulesRegistry.Get(name, v)
	if err != nil {
		return err
	}
	return r.GetValidationFn()(v, params...)
}
