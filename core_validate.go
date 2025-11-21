package model

import (
	"context"
	"fmt"
	"reflect"
)

// validateStruct walks a struct value and applies rules on each field according to its `validate` tag.
// Nested structs and pointers to structs are traversed recursively. The `path` argument tracks the
// dotted field path for clearer error messages.
func (tb *typeBinding) validateStruct(ctx context.Context, rv reflect.Value, path string, ve *ValidationError) error {
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
			if err := tb.validateStruct(ctx, fv.Elem(), fpath, ve); err != nil {
				return err
			}
		}

		// Recurse into embedded/inline structs
		if fv.Kind() == reflect.Struct {
			if err := tb.validateStruct(ctx, fv, fpath, ve); err != nil {
				return err
			}
		}

		// Process `validate` tag
		if rawTag := field.Tag.Get(tagValidate); rawTag != "" && rawTag != "-" {
			rules, exists := tb.rulesMapping.get(typ, i, tagValidate)
			if !exists {
				rules = parseTag(rawTag)
				tb.rulesMapping.add(typ, i, tagValidate, rules)
			}

			for _, r := range rules {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := tb.applyRule(r.name, fv, r.params...); err != nil {
					ve.Add(FieldError{Path: fpath, Rule: r.name, Params: r.params, Err: err})
				}
			}
		}

		// Process `validateElem` tag for slices, arrays, and maps
		if elemRaw := field.Tag.Get(tagValidateElem); elemRaw != "" && elemRaw != "-" {
			elemRules, exists := tb.rulesMapping.get(typ, i, tagValidateElem)
			if !exists {
				elemRules = parseTag(elemRaw)
				tb.rulesMapping.add(typ, i, tagValidateElem, elemRules)
			}

			if err := tb.validateElements(ctx, fv, fpath, elemRules, ve); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateElements applies validation rules to elements of a slice, array, or map
// using pre-parsed rules (e.g., retrieved from the cache).
func (tb *typeBinding) validateElements(ctx context.Context, fv reflect.Value, fpath string, rules []ruleNameParams, ve *ValidationError) error {
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
	isDiveOnly := len(rules) == 1 && rules[0].name == tagDive && len(rules[0].params) == 0

	switch cont.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < cont.Len(); i++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			elem := cont.Index(i)
			pathIdx := fmt.Sprintf("%s[%d]", fpath, i)
			if err := tb.validateSingleElement(ctx, elem, pathIdx, rules, isDiveOnly, ve); err != nil {
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
			if err := tb.validateSingleElement(ctx, elem, pathKey, rules, isDiveOnly, ve); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateSingleElement handles validation for a single item from a collection.
func (tb *typeBinding) validateSingleElement(ctx context.Context, elem reflect.Value, path string, rules []ruleNameParams, isDiveOnly bool, ve *ValidationError) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if isDiveOnly {
		dv := elem
		if dv.Kind() == reflect.Ptr && !dv.IsNil() {
			dv = dv.Elem()
		}
		if dv.Kind() == reflect.Struct {
			return tb.validateStruct(ctx, dv, path, ve)
		}
		ve.Add(FieldError{Path: path, Rule: tagDive, Err: fmt.Errorf("validateElem:\"dive\" requires struct element, got %s", dv.Kind())})
		return nil
	}

	for _, r := range rules {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := tb.applyRule(r.name, elem, r.params...); err != nil {
			ve.Add(FieldError{Path: path, Rule: r.name, Params: r.params, Err: err})
		}
	}
	return nil
}
