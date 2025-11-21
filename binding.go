package model

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ygrebnov/errorc"
)

type typeBinding struct {
	// typ is the underlying struct type this binding was built for.
	typ           reflect.Type
	rulesRegistry rulesRegistry
	rulesMapping  rulesMapping
}

// buildTypeBinding creates a typeBinding for the given struct type using the
// provided registry and mapping instances. For now it wires the provided
// registry/mapping into the binding; any future per-type precomputation can be
// added here.
func buildTypeBinding(typ reflect.Type, reg rulesRegistry, mapping rulesMapping) (*typeBinding, error) {
	return &typeBinding{
		typ:           typ,
		rulesRegistry: reg,
		rulesMapping:  mapping,
	}, nil
}

// setDefaultsStruct walks the struct value and applies defaults according to
// `default` and `defaultElem` tags. This is the type-level equivalent of the
// previous Model.setDefaultsStruct.
func (tb *typeBinding) setDefaultsStruct(rv reflect.Value) error {
	typ := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := typ.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		fv := rv.Field(i)

		// Handle default tag
		if dtag := field.Tag.Get(tagDefault); dtag != "" && dtag != "-" {
			if err := tb.applyDefaultTag(fv, dtag, field.Name); err != nil {
				return err
			}
		}
		// Element defaults for collections
		if etag := field.Tag.Get(tagDefaultElem); etag != "" && etag != "-" {
			if err := tb.applyDefaultElemTag(fv, etag); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyDefaultTag applies the `default` tag semantics to a single field value.
// Supported values: "dive", "alloc", or a literal (delegated to setLiteralDefault).
func (tb *typeBinding) applyDefaultTag(fv reflect.Value, tag, fieldName string) error {
	switch tag {
	case tagDive:
		return tb.diveDefaultsIntoValue(fv)
	case tagAlloc:
		// Allocate empty slice/map if nil
		if fv.Kind() == reflect.Slice && fv.IsNil() {
			fv.Set(reflect.MakeSlice(fv.Type(), 0, 0))
		} else if fv.Kind() == reflect.Map && fv.IsNil() {
			fv.Set(reflect.MakeMap(fv.Type()))
		}
		return nil
	default:
		if err := setLiteralDefault(fv, tag); err != nil {
			return errorc.With(
				ErrSetDefault,
				errorc.String(ErrorFieldFieldName, fieldName),
				errorc.Error(ErrorFieldCause, err),
			)
		}
		return nil
	}
}

// diveDefaultsIntoValue recurses into a struct or *struct field to apply nested defaults.
// For nil *struct, it allocates the struct before diving. Non-structs are ignored.
func (tb *typeBinding) diveDefaultsIntoValue(fv reflect.Value) error {
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
			return tb.setDefaultsStruct(fv.Elem())
		}
		return nil
	case reflect.Struct:
		return tb.setDefaultsStruct(fv)
	default:
		return nil
	}
}

// applyDefaultElemTag applies defaults to elements/values of collections based on `defaultElem`.
// Currently supports: defaultElem:"dive".
func (tb *typeBinding) applyDefaultElemTag(fv reflect.Value, tag string) error {
	if tag != tagDive {
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
				if err := tb.setDefaultsStruct(dv); err != nil {
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
					if err := tb.setDefaultsStruct(val.Elem()); err != nil {
						return err
					}
				}
				continue
			}
			// Value-typed struct map values: copy-modify-write-back
			if val.Kind() == reflect.Struct {
				copyVal := reflect.New(val.Type()).Elem()
				copyVal.Set(val)
				if err := tb.setDefaultsStruct(copyVal); err != nil {
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

// applyRule fetches the named rule from the registry and applies it to the given reflect.Value v,
// passing any additional string parameters.
// If the rule is not found or fails, an error is returned.
func (tb *typeBinding) applyRule(name string, v reflect.Value, params ...string) error {
	r, err := tb.rulesRegistry.get(name, v)
	if err != nil {
		return err
	}
	return r.getValidationFn()(v, params...)
}

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

// Binding is a reusable, precompiled view for a specific struct type T.
// It reuses the existing tag parsing, defaulting, and validation logic of
// Model without requiring a Model instance per object.
type Binding[T any] struct {
	// tb holds the type-level metadata for T.
	tb *typeBinding
}

// NewBinding constructs a Binding for the type parameter T using the default
// rules registry and mapping configuration.
func NewBinding[T any]() (*Binding[T], error) {
	// Obtain the reflect.Type for T. The zero value of *T is never dereferenced.
	var zero *T
	typ := reflect.TypeOf(zero).Elem()
	if typ.Kind() != reflect.Struct {
		// Mirror New's constraint that only struct types are supported.
		return nil, ErrNotStructPtr
	}

	tb, err := buildTypeBinding(typ, newRulesRegistry(), newRulesMapping())
	if err != nil {
		return nil, err
	}
	return &Binding[T]{tb: tb}, nil
}

// ApplyDefaults applies default values to zero fields of obj according to
// its `default` / `defaultElem` tags. It is safe to call multiple times.
func (b *Binding[T]) ApplyDefaults(obj *T) error {
	if obj == nil {
		return ErrNilObject
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrNotStructPtr
	}
	return b.tb.setDefaultsStruct(elem)
}

// Validate runs validation rules declared via `validate` / `validateElem`
// tags on obj with the provided context. If validation fails, a
// *ValidationError is returned; if the context is canceled, ctx.Err() is
// returned.
func (b *Binding[T]) Validate(ctx context.Context, obj *T) error {
	if obj == nil {
		return ErrNilObject
	}
	if ctx == nil {
		ctx = context.Background()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrNotStructPtr
	}
	ve := &ValidationError{}
	if err := b.tb.validateStruct(ctx, elem, "", ve); err != nil {
		return err
	}
	if ve.Empty() {
		return nil
	}
	return ve
}

// ValidateWithDefaults first applies defaults to obj and then runs
// validation. This is a convenience for service-level flows that expect
// defaulted inputs before validation.
func (b *Binding[T]) ValidateWithDefaults(ctx context.Context, obj *T) error {
	if err := b.ApplyDefaults(obj); err != nil {
		return err
	}
	return b.Validate(ctx, obj)
}
