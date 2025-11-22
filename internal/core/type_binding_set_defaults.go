package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/internal/errors"
)

// SetDefaultsStruct walks the struct value and applies defaults according to
// `default` and `defaultElem` tags. This is the type-level equivalent of the
// previous Model.SetDefaultsStruct.
func (tb *TypeBinding) SetDefaultsStruct(rv reflect.Value) error {
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
func (tb *TypeBinding) applyDefaultTag(fv reflect.Value, tag, fieldName string) error {
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
				errors.ErrSetDefault,
				errorc.String(errors.ErrorFieldFieldName, fieldName),
				errorc.Error(errors.ErrorFieldCause, err),
			)
		}
		return nil
	}
}

// diveDefaultsIntoValue recurses into a struct or *struct field to apply nested defaults.
// For nil *struct, it allocates the struct before diving. Non-structs are ignored.
func (tb *TypeBinding) diveDefaultsIntoValue(fv reflect.Value) error {
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
			return tb.SetDefaultsStruct(fv.Elem())
		}
		return nil
	case reflect.Struct:
		return tb.SetDefaultsStruct(fv)
	default:
		return nil
	}
}

// applyDefaultElemTag applies defaults to elements/values of collections based on `defaultElem`.
// Currently supports: defaultElem:"dive".
func (tb *TypeBinding) applyDefaultElemTag(fv reflect.Value, tag string) error {
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
				if err := tb.SetDefaultsStruct(dv); err != nil {
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
					if err := tb.SetDefaultsStruct(val.Elem()); err != nil {
						return err
					}
				}
				continue
			}
			// Value-typed struct map values: copy-modify-write-back
			if val.Kind() == reflect.Struct {
				copyVal := reflect.New(val.Type()).Elem()
				copyVal.Set(val)
				if err := tb.SetDefaultsStruct(copyVal); err != nil {
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
			errors.ErrDefaultLiteralUnsupportedKind,
			errorc.String(errors.ErrorFieldDefaultLiteralKind, target.Kind().String()),
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
