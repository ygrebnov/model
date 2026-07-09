package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
	"github.com/ygrebnov/model/pkg/types"
)

// SetDefaultsStruct walks the struct value and applies default tags using the
// compiled schema controller.
//
// Precedence:
//   - existing non-zero field value wins;
//   - `default` is applied only when the field is zero;
//   - slice and array element defaults require `defaultElem:"dive"`;
//   - map value defaults are applied to existing entries;
//   - nil slices/maps are not allocated unless `default:"alloc"` is used;
//   - nil pointer-to-struct values are allocated only when nested defaults or
//     `default:"dive"` require them.
func (s *Service[T]) SetDefaultsStruct(root reflect.Value) error {
	for _, child := range s.schemaController.GetRoot().Children {
		if err := applyDefaultNode(root, child); err != nil {
			return err
		}
	}
	return nil
}

// applyDefaultNode applies defaults for one compiled node and then recurses into
// its children when the node represents a nested struct or collection of structs.
func applyDefaultNode(base reflect.Value, node *schema.N) error {
	field, ok := fieldByIndex(base, node.I)
	if !ok {
		return nil
	}

	if err := applyFieldDefault(field, node); err != nil {
		return fmt.Errorf("apply default for %s: %w", node.GetName("."), err)
	}

	if len(node.Children) == 0 {
		return nil
	}

	return applyDefaultChildren(base, field, node)
}

// applyDefaultChildren applies child defaults below a struct, pointer-to-struct,
// slice/array of structs, or map of structs.
func applyDefaultChildren(base reflect.Value, field reflect.Value, node *schema.N) error {
	field = unwrapInterface(field)

	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			if field.Type().Elem().Kind() != reflect.Struct {
				return nil
			}

			if (node.DefaultTag == tagDive || hasDefaults(node.Children)) && field.CanSet() {
				field.Set(reflect.New(field.Type().Elem()))
			} else {
				return nil
			}
		}

		return applyDefaultChildren(base, field.Elem(), node)

	case reflect.Struct:
		if isDurationType(field.Type()) {
			return nil
		}

		// Important: children of ordinary nested structs keep root-based indexes,
		// so they must be resolved against base, not against field.
		for _, child := range node.Children {
			if err := applyDefaultNode(base, child); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice, reflect.Array:
		if node.DefaultElemTag != tagDive {
			return nil
		}

		for i := 0; i < field.Len(); i++ {
			elem := unwrapInterface(field.Index(i))

			if elem.Kind() == reflect.Ptr {
				if elem.IsNil() {
					continue
				}
				elem = elem.Elem()
			}

			if elem.Kind() != reflect.Struct || isDurationType(elem.Type()) {
				continue
			}

			// Collection child indexes are relative to the concrete element.
			for _, child := range node.Children {
				if err := applyDefaultNode(elem, child); err != nil {
					return err
				}
			}
		}
		return nil

	case reflect.Map:
		if field.IsNil() {
			return nil
		}

		for _, key := range field.MapKeys() {
			value := unwrapInterface(field.MapIndex(key))
			if !value.IsValid() {
				continue
			}

			// Map values are not directly settable. Work on a copy, then write it back.
			updated := reflect.New(value.Type()).Elem()
			updated.Set(value)

			target := updated
			if target.Kind() == reflect.Ptr {
				if target.IsNil() {
					continue
				}
				target = target.Elem()
			}

			if target.Kind() != reflect.Struct || isDurationType(target.Type()) {
				continue
			}

			// Collection child indexes are relative to the concrete map value.
			for _, child := range node.Children {
				if err := applyDefaultNode(target, child); err != nil {
					return err
				}
			}

			field.SetMapIndex(key, updated)
		}
		return nil

	default:
		return nil
	}
}

// applyFieldDefault applies the node's DefaultTag to field.
func applyFieldDefault(field reflect.Value, node *schema.N) error {
	if node.DefaultTag == "" {
		return nil
	}

	field = unwrapInterface(field)
	if !field.IsValid() || !field.CanSet() {
		return nil
	}

	return applyDefaultTag(field, node.DefaultTag)
}

// applyDefaultTag applies one `default` tag to a field value.
//
// Supported values are:
//   - `dive`, which allocates nil pointer-to-struct fields before child defaults
//     are applied by applyDefaultChildren;
//   - `alloc`, which allocates nil slices and maps;
//   - literal scalar values, which are delegated to setLiteralValue and are set
//     only when the target value is zero.
func applyDefaultTag(field reflect.Value, tag string) error {
	switch tag {
	case tagDive:
		if field.Kind() == reflect.Ptr &&
			field.IsNil() &&
			field.Type().Elem().Kind() == reflect.Struct &&
			field.CanSet() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return nil

	case tagAlloc:
		switch field.Kind() {
		case reflect.Slice:
			if field.IsNil() {
				field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			}
		case reflect.Map:
			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}
		}
		return nil

	default:
		return setLiteralValue(field, tag, true)
	}
}

// hasDefaults reports whether this subtree contains any default tags.
func hasDefaults(nodes []*schema.N) bool {
	for _, node := range nodes {
		if node.DefaultTag != "" || node.DefaultElemTag != "" {
			return true
		}
		if hasDefaults(node.Children) {
			return true
		}
	}
	return false
}

// fieldByIndex resolves a field from base using a compiled index path.
//
// For ordinary nested structs, node.I is root-based.
// For collection element fields, node.I is relative to the concrete element.
func fieldByIndex(base reflect.Value, index []int) (reflect.Value, bool) {
	v := base

	for _, i := range index {
		v = unwrapInterface(v)

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if !v.CanSet() || v.Type().Elem().Kind() != reflect.Struct {
					return reflect.Value{}, false
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}

		if i < 0 || i >= v.NumField() {
			return reflect.Value{}, false
		}

		v = v.Field(i)
	}

	return v, true
}

// unwrapInterface unwraps interface values when they contain a concrete value.
func unwrapInterface(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func isDurationType(typ reflect.Type) bool {
	return typ == reflect.TypeOf(types.Duration(0)) || typ == reflect.TypeOf(time.Duration(0))
}

// setLiteralValue sets a literal value into fv.
// If onlyZero is true, the value is set only when the target is zero.
// For pointer-to-scalar fields, it allocates and sets the pointed value.
//
//nolint:gocyclo,funlen // cyclomatic complexity is acceptable here
func setLiteralValue(fv reflect.Value, lit string, onlyZero bool) error {
	target := fv
	// Allocate for pointer-to-scalar when nil
	if target.Kind() == reflect.Ptr {
		// If nil and element is not struct/map/slice, allocate
		if target.IsNil() {
			ek := target.Type().Elem().Kind()
			switch ek {
			case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
				// Do not auto-allocate complex types on literal values
			default:
				target.Set(reflect.New(target.Type().Elem()))
			}
		}
		if !target.IsNil() {
			target = target.Elem()
		}
	}

	// Only set if zero when requested by defaults.
	if !target.CanSet() || onlyZero && !target.IsZero() {
		return nil
	}

	// Handle special case: time.Duration typed fields
	if isDurationType(target.Type()) {
		d, err := time.ParseDuration(lit)
		if err != nil {
			return errorc.With(
				errors.ErrCannotParseDuration,
				errorc.String(keys.Value, lit),
				errorc.Error(keys.Cause, err),
			)
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
		iv, err := parseInt64(lit, target.Kind())
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
	case reflect.Complex64, reflect.Complex128:
		cv, err := parseComplex128(lit, target.Type().Bits())
		if err != nil {
			return err
		}
		target.SetComplex(cv)
	default:
		return errorc.With(
			errors.ErrDefaultLiteralUnsupportedKind,
			errorc.String(keys.DefaultLiteralKind, target.Kind().String()),
		)
	}
	return nil
}

func parseInt64(lit string, kind reflect.Kind) (int64, error) {
	lit = strings.TrimSpace(lit)

	if r, ok, err := parseQuotedRuneLiteral(lit); ok || err != nil {
		if err != nil {
			return 0, err
		}
		return int64(r), nil
	}

	iv, err := strconv.ParseInt(lit, 0, 64)
	if err == nil {
		return iv, nil
	}

	// rune is an alias for int32, so reflection exposes rune fields as
	// reflect.Int32. Support env values like Ж for rune fields.
	if kind == reflect.Int32 {
		if r, ok := parseUnquotedRuneLiteral(lit); ok {
			return int64(r), nil
		}
	}

	return 0, errorc.With(
		errors.ErrCannotParseInt,
		errorc.String(keys.Value, lit),
		errorc.Error(keys.Cause, err),
	)
}

func parseQuotedRuneLiteral(lit string) (r rune, b bool, e error) {
	if len(lit) < 2 || lit[0] != '\'' || lit[len(lit)-1] != '\'' {
		return 0, false, nil
	}

	unquoted, err := strconv.Unquote(lit)
	if err != nil {
		return 0, true, errorc.With(
			errors.ErrCannotParseRuneLiteral,
			errorc.String(keys.Value, lit),
			errorc.Error(keys.Cause, err),
		)
	}

	rs := []rune(unquoted)
	if len(rs) != 1 {
		return 0, true, errorc.With(
			errors.ErrCannotParseRuneLiteral,
			errorc.String(keys.Value, lit),
			errorc.String(keys.Cause, fmt.Sprintf("expected one rune, got %d", len(rs))),
		)
	}

	return rs[0], true, nil
}

func parseUnquotedRuneLiteral(lit string) (rune, bool) {
	rs := []rune(lit)
	if len(rs) != 1 {
		return 0, false
	}
	return rs[0], true
}

func parseUint64(s string) (uint64, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 0, 64)
	if err != nil {
		return 0, errorc.With(errors.ErrCannotParseUint, errorc.String(keys.Value, s), errorc.Error(keys.Cause, err))
	}
	return v, nil
}

func parseFloat64(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, errorc.With(errors.ErrCannotParseFloat, errorc.String(keys.Value, s), errorc.Error(keys.Cause, err))
	}
	return v, nil
}

func parseComplex128(s string, bitSize int) (complex128, error) {
	v, err := strconv.ParseComplex(strings.TrimSpace(s), bitSize)
	if err != nil {
		return 0, errorc.With(errors.ErrCannotParseComplex, errorc.String(keys.Value, s), errorc.Error(keys.Cause, err))
	}
	return v, nil
}
