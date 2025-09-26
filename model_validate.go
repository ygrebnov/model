package model

import (
	"fmt"
	"reflect"
)

// Validate runs the registered validation rules against the model's bound object.
// It delegates to the internal validate method which performs the actual work.
func (m *Model[TObject]) Validate() error { return m.validate() }

// validate is the internal implementation that walks struct fields and applies rules
// declared in `validate:"..."` tags. It supports validationRule parameters via the syntax
// "validationRule" or "validationRule(p1,p2)" and multiple rules separated by commas.
func (m *Model[TObject]) validate() error {
	rv, err := m.rootStructValue("Validate")
	if err != nil {
		return err
	}
	ve := &ValidationError{}
	m.validateStruct(rv, "", ve)
	if ve.Empty() {
		return nil
	}
	return ve
}

// applyRule fetches the named rule from the registry and applies it to the given reflect.Value v,
// passing any additional string parameters.
// If the rule is not found or fails, an error is returned.
func (m *Model[TObject]) applyRule(name string, v reflect.Value, params ...string) error {
	r, err := m.rulesRegistry.get(name, v)
	if err != nil {
		return err
	}

	return r.getValidationFn()(v, params...)
}

// validateStruct walks a struct value and applies rules on each field according to its `validate` tag.
// Nested structs and pointers to structs are traversed recursively. The `path` argument tracks the
// dotted field path for clearer error messages.
func (m *Model[TObject]) validateStruct(rv reflect.Value, path string, ve *ValidationError) {
	typ := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
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
			m.validateStruct(fv.Elem(), fpath, ve)
		}

		// Recurse into embedded/inline structs
		if fv.Kind() == reflect.Struct {
			m.validateStruct(fv, fpath, ve)
		}

		// Process `validate` tag
		if rawTag := field.Tag.Get(tagValidate); rawTag != "" && rawTag != "-" {
			rules, exists := m.rulesMapping.get(typ, i, tagValidate)
			if !exists {
				rules = parseTag(rawTag)
				m.rulesMapping.add(typ, i, tagValidate, rules)
			}

			for _, r := range rules {
				if err := m.applyRule(r.name, fv, r.params...); err != nil {
					ve.Add(FieldError{Path: fpath, Rule: r.name, Params: r.params, Err: err})
				}
			}
		}

		// Process `validateElem` tag for slices, arrays, and maps
		if elemRaw := field.Tag.Get(tagValidateElem); elemRaw != "" && elemRaw != "-" {
			elemRules, exists := m.rulesMapping.get(typ, i, tagValidateElem)
			if !exists {
				elemRules = parseTag(elemRaw)
				m.rulesMapping.add(typ, i, tagValidateElem, elemRules)
			}

			m.validateElementsWithRules(fv, fpath, elemRules, ve)
		}
	}
}

// validateElements applies validation rules to elements of a slice, array, or map.
func (m *Model[TObject]) validateElements(fv reflect.Value, fpath, elemRaw string, ve *ValidationError) {
	cont := fv
	if cont.Kind() == reflect.Ptr && !cont.IsNil() {
		cont = cont.Elem()
	}

	rules := parseTag(elemRaw)
	if len(rules) == 0 {
		return
	}

	// Special case: validateElem:"dive" means recurse into element structs
	isDiveOnly := len(rules) == 1 && rules[0].name == tagDive && len(rules[0].params) == 0

	switch cont.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < cont.Len(); i++ {
			elem := cont.Index(i)
			pathIdx := fmt.Sprintf("%s[%d]", fpath, i)
			m.validateSingleElement(elem, pathIdx, rules, isDiveOnly, ve)
		}
	case reflect.Map:
		for _, key := range cont.MapKeys() {
			elem := cont.MapIndex(key)
			pathKey := fmt.Sprintf("%s[%v]", fpath, key.Interface())
			m.validateSingleElement(elem, pathKey, rules, isDiveOnly, ve)
		}
	}
}

// validateElementsWithRules applies validation rules to elements of a slice, array, or map
// using pre-parsed rules (e.g., retrieved from the cache).
func (m *Model[TObject]) validateElementsWithRules(fv reflect.Value, fpath string, rules []ruleNameParams, ve *ValidationError) {
	cont := fv
	if cont.Kind() == reflect.Ptr && !cont.IsNil() {
		cont = cont.Elem()
	}
	if len(rules) == 0 {
		return
	}
	// Special case: validateElem:"dive" means recurse into element structs
	isDiveOnly := len(rules) == 1 && rules[0].name == tagDive && len(rules[0].params) == 0

	switch cont.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < cont.Len(); i++ {
			elem := cont.Index(i)
			pathIdx := fmt.Sprintf("%s[%d]", fpath, i)
			m.validateSingleElement(elem, pathIdx, rules, isDiveOnly, ve)
		}
	case reflect.Map:
		for _, key := range cont.MapKeys() {
			elem := cont.MapIndex(key)
			pathKey := fmt.Sprintf("%s[%v]", fpath, key.Interface())
			m.validateSingleElement(elem, pathKey, rules, isDiveOnly, ve)
		}
	}
}

// validateSingleElement handles validation for a single item from a collection.
func (m *Model[TObject]) validateSingleElement(elem reflect.Value, path string, rules []ruleNameParams, isDiveOnly bool, ve *ValidationError) {
	if isDiveOnly {
		dv := elem
		if dv.Kind() == reflect.Ptr && !dv.IsNil() {
			dv = dv.Elem()
		}
		if dv.Kind() == reflect.Struct {
			m.validateStruct(dv, path, ve)
		} else {
			ve.Add(FieldError{Path: path, Rule: tagDive, Err: fmt.Errorf("validateElem:\"dive\" requires struct element, got %s", dv.Kind())})
		}
		return
	}

	for _, r := range rules {
		if err := m.applyRule(r.name, elem, r.params...); err != nil {
			ve.Add(FieldError{Path: path, Rule: r.name, Params: r.params, Err: err})
		}
	}
}
