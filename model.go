package model

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Model[TObject any] struct {
	mu                 sync.RWMutex
	once               sync.Once
	obj                *TObject
	validators         map[string][]typedAdapter // per-model registry: rule name -> overloads by type
	applyDefaultsOnNew bool
	validateOnNew      bool
}

// registerRuleAdapter registers/overwrites a rule overload at runtime (internal use).
func (m *Model[TObject]) registerRuleAdapter(name string, ad typedAdapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if name == "" || ad.fn == nil || ad.fieldType == nil {
		return
	}
	if m.validators == nil {
		m.validators = make(map[string][]typedAdapter)
	}
	m.validators[name] = append(m.validators[name], ad)
}

// getRuleAdapters retrieves all overloads for a rule name.
func (m *Model[TObject]) getRuleAdapters(name string) []typedAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validators[name]
}

// applyRule dispatches to the best-matching overload of rule `name` for the given field value.
// Selection strategy:
//  1. Prefer exact type match (v.Type() == fieldType).
//  2. Otherwise accept AssignableTo matches (interfaces, named types), preferring the first declared.
//  3. If no matches, return a descriptive error listing available overload types.
//  4. If multiple exact matches (shouldn't happen), return an ambiguity error.
func (m *Model[TObject]) applyRule(name string, v reflect.Value, params ...string) error {
	adapters := m.getRuleAdapters(name)
	if len(adapters) == 0 {
		return fmt.Errorf("model: rule %q is not registered", name)
	}
	if !v.IsValid() {
		return fmt.Errorf("model: invalid value for rule %q", name)
	}

	typ := v.Type()
	var (
		exacts  []typedAdapter
		assigns []typedAdapter
	)
	for _, ad := range adapters {
		if ad.fieldType == nil || ad.fn == nil {
			continue
		}
		if typ == ad.fieldType {
			exacts = append(exacts, ad)
			continue
		}
		if typ.AssignableTo(ad.fieldType) {
			assigns = append(assigns, ad)
		}
	}

	switch {
	case len(exacts) == 1:
		return exacts[0].fn(v, params...)
	case len(exacts) > 1:
		return fmt.Errorf(
			"model: rule %q is ambiguous for type %s; %d exact overloads registered",
			name,
			typ,
			len(exacts),
		)
	case len(assigns) >= 1:
		return assigns[0].fn(v, params...)
	default:
		// Construct helpful message of available overload types.
		var names []string
		for _, ad := range adapters {
			if ad.fieldType != nil {
				names = append(names, ad.fieldType.String())
			}
		}
		sort.Strings(names)
		return fmt.Errorf(
			"model: rule %q has no overload for type %s (available: %s)",
			name,
			typ,
			strings.Join(names, ", "),
		)
	}
}

// rootStructValue validates that m.obj is a non-nil pointer to a struct and returns the struct value.
// The phase string is used in error messages (e.g., "Validate", "SetDefaults").
func (m *Model[TObject]) rootStructValue(phase string) (reflect.Value, error) {
	if m.obj == nil {
		return reflect.Value{}, fmt.Errorf("model: %s: nil object", phase)
	}
	rv := reflect.ValueOf(m.obj)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, fmt.Errorf("model: %s: object must be a non-nil pointer to struct; got %s", phase, rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("model: %s: object must point to a struct; got %s", phase, rv.Kind())
	}
	return rv, nil
}

// Validate runs the registered validation rules against the model's bound object.
// It delegates to the internal validate method which performs the actual work.
func (m *Model[TObject]) Validate() error { return m.validate() }

// validate is the internal implementation that walks struct fields and applies rules
// declared in `validate:"..."` tags. It supports rule parameters via the syntax
// "rule" or "rule(p1,p2)" and multiple rules separated by commas.
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
		if rawTag := field.Tag.Get("validate"); rawTag != "" && rawTag != "-" {
			rules := parseRules(rawTag)
			for _, rule := range rules {
				if err := m.applyRule(rule.name, fv, rule.params...); err != nil {
					ve.Add(FieldError{Path: fpath, Rule: rule.name, Params: rule.params, Err: err})
				}
			}
		}

		// Process `validateElem` tag for slices, arrays, and maps
		if elemRaw := field.Tag.Get("validateElem"); elemRaw != "" && elemRaw != "-" {
			m.validateElements(fv, fpath, elemRaw, ve)
		}
	}
}

// validateElements applies validation rules to elements of a slice, array, or map.
func (m *Model[TObject]) validateElements(fv reflect.Value, fpath, elemRaw string, ve *ValidationError) {
	cont := fv
	if cont.Kind() == reflect.Ptr && !cont.IsNil() {
		cont = cont.Elem()
	}

	rules := parseRules(elemRaw)
	if len(rules) == 0 {
		return
	}

	// Special case: validateElem:"dive" means recurse into element structs
	isDiveOnly := len(rules) == 1 && rules[0].name == "dive" && len(rules[0].params) == 0

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
func (m *Model[TObject]) validateSingleElement(elem reflect.Value, path string, rules []parsedRule, isDiveOnly bool, ve *ValidationError) {
	if isDiveOnly {
		dv := elem
		if dv.Kind() == reflect.Ptr && !dv.IsNil() {
			dv = dv.Elem()
		}
		if dv.Kind() == reflect.Struct {
			m.validateStruct(dv, path, ve)
		} else {
			ve.Add(FieldError{Path: path, Rule: "dive", Err: fmt.Errorf("validateElem:\"dive\" requires struct element, got %s", dv.Kind())})
		}
		return
	}

	for _, r := range rules {
		if err := m.applyRule(r.name, elem, r.params...); err != nil {
			ve.Add(FieldError{Path: path, Rule: r.name, Params: r.params, Err: err})
		}
	}
}

// SetDefaults applies default values based on `default:"..."` tags to the model's object.
// It is safe to call multiple times; only zero-valued fields are set.
func (m *Model[TObject]) SetDefaults() error {
	var err error
	m.once.Do(func() { err = m.applyDefaults() })
	return err
}

// applyDefaults walks the object and applies defaults according to `default` and `defaultElem` tags.
// Supported forms:
//   - `default:"<literal>"` sets the field if it is zero
//   - `default:"dive"` on a struct or pointer-to-struct recurses into its fields
//   - `default:"alloc"` allocates an empty map/slice when the field is nil
//   - `defaultElem:"dive"` recurses into slice/array elements or map values that are structs
//
// Notes:
//   - Literals are parsed by kind: string, bool, ints/uints, floats, time.Duration.
//   - For pointer scalar fields, nil pointers are allocated when a literal default is present.
func (m *Model[TObject]) applyDefaults() error {
	rv, err := m.rootStructValue("SetDefaults")
	if err != nil {
		return err
	}
	return m.setDefaultsStruct(rv)
}

func (m *Model[TObject]) setDefaultsStruct(rv reflect.Value) error {
	typ := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := typ.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		fv := rv.Field(i)

		// Handle default tag
		if dtag := field.Tag.Get("default"); dtag != "" && dtag != "-" {
			if err := m.applyDefaultTag(fv, dtag, field.Name); err != nil {
				return err
			}
		}
		// Element defaults for collections
		if etag := field.Tag.Get("defaultElem"); etag != "" && etag != "-" {
			if err := m.applyDefaultElemTag(fv, etag); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyDefaultTag applies the `default` tag semantics to a single field value.
// Supported values: "dive", "alloc", or a literal (delegated to setLiteralDefault).
func (m *Model[TObject]) applyDefaultTag(fv reflect.Value, tag, fieldName string) error {
	switch tag {
	case "dive":
		return m.diveDefaultsIntoValue(fv)
	case "alloc":
		// Allocate empty slice/map if nil
		if fv.Kind() == reflect.Slice && fv.IsNil() {
			fv.Set(reflect.MakeSlice(fv.Type(), 0, 0))
		} else if fv.Kind() == reflect.Map && fv.IsNil() {
			fv.Set(reflect.MakeMap(fv.Type()))
		}
		return nil
	default:
		if err := setLiteralDefault(fv, tag); err != nil {
			return fmt.Errorf("default for %s: %w", fieldName, err)
		}
		return nil
	}
}

// diveDefaultsIntoValue recurses into a struct or *struct field to apply nested defaults.
// For nil *struct, it allocates the struct before diving. Non-structs are ignored.
func (m *Model[TObject]) diveDefaultsIntoValue(fv reflect.Value) error {
	switch fv.Kind() {
	case reflect.Ptr:
		if fv.IsNil() {
			if fv.Type().Elem().Kind() == reflect.Struct {
				fv.Set(reflect.New(fv.Type().Elem()))
			} else {
				return nil // ignore dive for non-struct pointers
			}
		}
		if fv.Elem().Kind() == reflect.Struct {
			return m.setDefaultsStruct(fv.Elem())
		}
		return nil
	case reflect.Struct:
		return m.setDefaultsStruct(fv)
	default:
		return nil
	}
}

// applyDefaultElemTag applies defaults to elements/values of collections based on `defaultElem`.
// Currently supports: defaultElem:"dive".
func (m *Model[TObject]) applyDefaultElemTag(fv reflect.Value, tag string) error {
	if tag != "dive" {
		return nil
	}
	cont := fv
	if cont.Kind() == reflect.Ptr && !cont.IsNil() {
		cont = cont.Elem()
	}
	switch cont.Kind() {
	case reflect.Slice, reflect.Array:
		l := cont.Len()
		for j := 0; j < l; j++ {
			ev := cont.Index(j)
			dv := ev
			if dv.Kind() == reflect.Ptr && !dv.IsNil() {
				dv = dv.Elem()
			}
			if dv.Kind() == reflect.Struct {
				if err := m.setDefaultsStruct(dv); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		for _, key := range cont.MapKeys() {
			val := cont.MapIndex(key)
			// Pointer-to-struct map values: mutate in place
			if val.Kind() == reflect.Ptr {
				if !val.IsNil() && val.Elem().Kind() == reflect.Struct {
					if err := m.setDefaultsStruct(val.Elem()); err != nil {
						return err
					}
				}
				continue
			}
			// Value-typed struct map values: copy-modify-write-back
			if val.Kind() == reflect.Struct {
				copyVal := reflect.New(val.Type()).Elem()
				copyVal.Set(val)
				if err := m.setDefaultsStruct(copyVal); err != nil {
					return err
				}
				cont.SetMapIndex(key, copyVal)
			}
		}
	default:
		// ignore for non-collections
	}
	return nil
}

// setLiteralDefault sets a literal default value into fv if it is zero.
// For pointer-to-scalar fields, it allocates and sets the pointed value.
func setLiteralDefault(fv reflect.Value, lit string) error {
	target := fv
	// Allocate for pointer-to-scalar when nil
	if target.Kind() == reflect.Ptr {
		// If nil and element is not struct/map/slice, allocate
		if target.IsNil() {
			ek := target.Type().Elem().Kind()
			switch ek {
			case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
				// Do not auto-allocate complex types on literal defaults
			default:
				target.Set(reflect.New(target.Type().Elem()))
			}
		}
		if !target.IsNil() {
			target = target.Elem()
		}
	}

	// Only set if zero
	if !target.CanSet() || !target.IsZero() {
		return nil
	}

	// Handle special case: time.Duration typed fields
	durType := reflect.TypeOf(time.Duration(0))
	if target.Type() == durType {
		d, err := time.ParseDuration(lit)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}
		target.SetInt(int64(d))
		return nil
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(lit)
	case reflect.Bool:
		switch strings.ToLower(lit) {
		case "1", "true", "t", "yes", "y", "on":
			target.SetBool(true)
		case "0", "false", "f", "no", "n", "off":
			target.SetBool(false)
		default:
			return fmt.Errorf("parse bool: %q", lit)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		iv, err := parseInt64(lit)
		if err != nil {
			return err
		}
		// Convert handles named types
		target.SetInt(iv)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		uv, err := parseUint64(lit)
		if err != nil {
			return err
		}
		target.SetUint(uv)
	case reflect.Float32, reflect.Float64:
		fv, err := parseFloat64(lit)
		if err != nil {
			return err
		}
		target.SetFloat(fv)
	default:
		return fmt.Errorf("unsupported kind %s for default literal", target.Kind())
	}
	return nil
}

func parseInt64(s string) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse int: %w", err)
	}
	return v, nil
}

func parseUint64(s string) (uint64, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse uint: %w", err)
	}
	return v, nil
}

func parseFloat64(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("parse float: %w", err)
	}
	return v, nil
}
