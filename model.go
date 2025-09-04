package model

import (
	"fmt"
	"reflect"
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
		return fmt.Errorf(
			"model: rule %q has no overload for type %s (available: %s)",
			name,
			typ,
			strings.Join(names, ", "),
		)
	}
}

// Validate runs the registered validation rules against the model's bound object.
// It delegates to the internal validate method which performs the actual work.
func (m *Model[TObject]) Validate() error { return m.validate() }

// validate is the internal implementation that walks struct fields and applies rules
// declared in `validate:"..."` tags. It supports rule parameters via the syntax
// "rule" or "rule(p1,p2)" and multiple rules separated by commas.
func (m *Model[TObject]) validate() error {
	if m.obj == nil {
		return fmt.Errorf("model: Validate: nil object")
	}
	rv := reflect.ValueOf(m.obj)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("model: Validate: object must be a non-nil pointer to struct; got %s", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("model: Validate: object must point to a struct; got %s", rv.Kind())
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
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		fv := rv.Field(i)

		// Build field path for messages
		fpath := field.Name
		if path != "" {
			fpath = path + "." + field.Name
		}

		// Recurse into pointers to structs
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				// nothing to validate
			} else if fv.Elem().Kind() == reflect.Struct {
				m.validateStruct(fv.Elem(), fpath, ve)
			}
		}

		// Recurse into embedded/inline structs
		if fv.Kind() == reflect.Struct {
			m.validateStruct(fv, fpath, ve)
		}

		// Read validate tag (process if present, but do not skip validateElem when absent)
		raw := field.Tag.Get("validate")
		if raw != "" && raw != "-" {
			// Parse rules with optional params: "rule" or "rule(p1,p2)".
			// Multiple rules are comma-separated at the top level (commas inside parens are ignored).
			var tokens []string
			{
				depth := 0
				start := 0
				for i, r := range raw {
					switch r {
					case '(':
						depth++
					case ')':
						if depth > 0 {
							depth--
						}
					case ',':
						if depth == 0 {
							tokens = append(tokens, strings.TrimSpace(raw[start:i]))
							start = i + 1
						}
					}
				}
				// last token
				if start <= len(raw) {
					tokens = append(tokens, strings.TrimSpace(raw[start:]))
				}
			}

			for _, tok := range tokens {
				if tok == "" || tok == "-" {
					continue
				}
				name := tok
				var params []string
				if idx := strings.IndexRune(tok, '('); idx != -1 && strings.HasSuffix(tok, ")") {
					name = strings.TrimSpace(tok[:idx])
					inner := strings.TrimSpace(tok[idx+1 : len(tok)-1])
					if inner != "" {
						// split params by comma and trim spaces
						parts := strings.Split(inner, ",")
						params = make([]string, 0, len(parts))
						for _, p := range parts {
							p = strings.TrimSpace(p)
							if p != "" {
								params = append(params, p)
							}
						}
					}
				}
				if name == "" {
					continue
				}
				if err := m.applyRule(name, fv, params...); err != nil {
					ve.Add(FieldError{Path: fpath, Rule: name, Params: append([]string(nil), params...), Err: err})
				}
			}
		}

		// --- Element validation for slices/arrays/maps via `validateElem` ---
		// This tag applies the listed rules to each element of a slice/array, or to each value of a map.
		if elemRaw := field.Tag.Get("validateElem"); elemRaw != "" && elemRaw != "-" {
			// Resolve the container value: deref pointer if needed
			cont := fv
			if cont.Kind() == reflect.Ptr && !cont.IsNil() {
				cont = cont.Elem()
			}

			// Tokenize elemRaw the same way as for `validate`
			parseRules := func(spec string) []struct {
				name   string
				params []string
			} {
				var toks []string
				{
					depth := 0
					start := 0
					for i, r := range spec {
						switch r {
						case '(':
							depth++
						case ')':
							if depth > 0 {
								depth--
							}
						case ',':
							if depth == 0 {
								toks = append(toks, strings.TrimSpace(spec[start:i]))
								start = i + 1
							}
						}
					}
					if start <= len(spec) {
						toks = append(toks, strings.TrimSpace(spec[start:]))
					}
				}
				out := make([]struct {
					name   string
					params []string
				}, 0, len(toks))
				for _, tok := range toks {
					if tok == "" || tok == "-" {
						continue
					}
					nm := tok
					var ps []string
					if idx := strings.IndexRune(tok, '('); idx != -1 && strings.HasSuffix(tok, ")") {
						nm = strings.TrimSpace(tok[:idx])
						inner := strings.TrimSpace(tok[idx+1 : len(tok)-1])
						if inner != "" {
							parts := strings.Split(inner, ",")
							for _, p := range parts {
								p = strings.TrimSpace(p)
								if p != "" {
									ps = append(ps, p)
								}
							}
						}
					}
					if nm != "" {
						out = append(out, struct {
							name   string
							params []string
						}{nm, ps})
					}
				}
				return out
			}

			rules := parseRules(elemRaw)

			// Special case: validateElem:"dive" means recurse into element structs
			onlyDive := len(rules) == 1 && rules[0].name == "dive" && len(rules[0].params) == 0

			switch cont.Kind() {
			case reflect.Slice, reflect.Array:
				l := cont.Len()
				for i := 0; i < l; i++ {
					ev := cont.Index(i)
					pathIdx := fmt.Sprintf("%s[%d]", fpath, i)

					if onlyDive {
						// Deref pointer elements
						dv := ev
						if dv.Kind() == reflect.Ptr && !dv.IsNil() {
							dv = dv.Elem()
						}
						if dv.Kind() == reflect.Struct {
							m.validateStruct(dv, pathIdx, ve)
							continue
						}
						// Not a struct: record an error for misuse of dive
						ve.Add(FieldError{Path: pathIdx, Rule: "dive", Err: fmt.Errorf("validateElem:\"dive\" requires struct element, got %s", dv.Kind())})
						continue
					}

					// Apply each listed rule to the element value
					for _, r := range rules {
						if err := m.applyRule(r.name, ev, r.params...); err != nil {
							ve.Add(FieldError{Path: pathIdx, Rule: r.name, Params: append([]string(nil), r.params...), Err: err})
						}
					}
				}

			case reflect.Map:
				// Validate map VALUES (not keys) with validateElem
				for _, key := range cont.MapKeys() {
					val := cont.MapIndex(key)
					pathKey := fmt.Sprintf("%s[%v]", fpath, key.Interface())

					if onlyDive {
						dv := val
						if dv.Kind() == reflect.Ptr && !dv.IsNil() {
							dv = dv.Elem()
						}
						if dv.Kind() == reflect.Struct {
							m.validateStruct(dv, pathKey, ve)
							continue
						}
						ve.Add(FieldError{Path: pathKey, Rule: "dive", Err: fmt.Errorf("validateElem:\"dive\" requires struct element, got %s", dv.Kind())})
						continue
					}

					for _, r := range rules {
						if err := m.applyRule(r.name, val, r.params...); err != nil {
							ve.Add(FieldError{Path: pathKey, Rule: r.name, Params: append([]string(nil), r.params...), Err: err})
						}
					}
				}

			default:
				// Non-container kinds: ignore validateElem silently
			}
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
	if m.obj == nil {
		return fmt.Errorf("model: SetDefaults: nil object")
	}
	rv := reflect.ValueOf(m.obj)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("model: SetDefaults: object must be a non-nil pointer to struct; got %s", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("model: SetDefaults: object must point to a struct; got %s", rv.Kind())
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
			switch dtag {
			case "dive":
				// Recurse into structs / *struct
				if fv.Kind() == reflect.Ptr {
					if fv.IsNil() {
						// allocate if pointer to struct
						if fv.Type().Elem().Kind() == reflect.Struct {
							fv.Set(reflect.New(fv.Type().Elem()))
						} else {
							// not a struct pointer: ignore dive
							goto after_default
						}
					}
					if fv.Elem().Kind() == reflect.Struct {
						if err := m.setDefaultsStruct(fv.Elem()); err != nil {
							return err
						}
					}
				} else if fv.Kind() == reflect.Struct {
					if err := m.setDefaultsStruct(fv); err != nil {
						return err
					}
				} // else: ignore dive for non-structs
			case "alloc":
				// Allocate empty slice/map if nil
				if fv.Kind() == reflect.Slice && fv.IsNil() {
					fv.Set(reflect.MakeSlice(fv.Type(), 0, 0))
				} else if fv.Kind() == reflect.Map && fv.IsNil() {
					fv.Set(reflect.MakeMap(fv.Type()))
				}
			default:
				// Literal default: set only if zero
				if err := setLiteralDefault(fv, dtag); err != nil {
					return fmt.Errorf("default for %s: %w", field.Name, err)
				}
			}
		}
	after_default:
		// Element defaults for collections: recurse into struct elements when asked
		if etag := field.Tag.Get("defaultElem"); etag != "" && etag != "-" {
			if etag == "dive" {
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
						// Pointer-to-struct map values: we can mutate the pointed struct in-place.
						if val.Kind() == reflect.Ptr {
							if !val.IsNil() && val.Elem().Kind() == reflect.Struct {
								if err := m.setDefaultsStruct(val.Elem()); err != nil {
									return err
								}
							}
							continue
						}
						// Value-typed struct map values are not addressable; copy, mutate, and write back.
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
			}
		}
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
