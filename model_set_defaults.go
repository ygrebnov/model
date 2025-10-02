package model

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"
)

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
	if rv, err := m.rootStructValue("SetDefaults"); err != nil {
		return err
	} else {
		return m.setDefaultsStruct(rv)
	}
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
		if dtag := field.Tag.Get(tagDefault); dtag != "" && dtag != "-" {
			if err := m.applyDefaultTag(fv, dtag, field.Name); err != nil {
				return err
			}
		}
		// Element defaults for collections
		if etag := field.Tag.Get(tagDefaultElem); etag != "" && etag != "-" {
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
	case tagDive:
		return m.diveDefaultsIntoValue(fv)
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

var durationType = reflect.TypeOf(time.Duration(0))

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
	if target.Type() == durationType {
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
		return errorc.With(
			ErrDefaultLiteralUnsupportedKind,
			errorc.String(ErrorFieldDefaultLiteralKind, target.Kind().String()),
		)
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
