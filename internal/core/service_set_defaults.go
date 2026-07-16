package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
	"github.com/ygrebnov/model/pkg/types"
)

// SetDefaultsStruct walks the compiled schema and applies `default` and
// `defaultElem` tags to root.
//
// Precedence:
//   - an existing non-zero scalar value wins;
//   - `default` is applied only when the field is zero;
//   - slice and array element defaults require `defaultElem:"dive"`;
//   - defaults are applied to existing map values;
//   - nil slices and maps are allocated only by `default:"alloc"`;
//   - nil pointer-to-struct fields are allocated only when `default:"dive"`.
func (s *Service[T]) SetDefaultsStruct(root reflect.Value) error {
	policy := walkPolicy{
		DiveCollection: func(ctx walkContext, field reflect.Value) bool {
			field = unwrapInterface(field)

			switch field.Kind() {
			case reflect.Slice, reflect.Array:
				return ctx.Node.DefaultElemTag == tagDive

			case reflect.Map:
				return true

			default:
				return false
			}
		},
		AllocPtrStruct: func(ctx walkContext, _ reflect.Value) bool {
			return ctx.Node.DefaultTag == tagDive
		},
	}

	return walkSchema(
		root,
		s.schema.GetRoot(),
		nil,
		policy,
		applyDefaultWalkValue,
	)
}

// applyDefaultWalkValue applies the current schema node's `default` tag to the
// concrete field visited by walkSchema.
func applyDefaultWalkValue(
	ctx walkContext,
	field reflect.Value,
) error {
	if ctx.Node.DefaultTag == "" {
		return nil
	}

	field = unwrapInterface(field)
	if !field.IsValid() || !field.CanSet() {
		return nil
	}

	if err := applyDefaultTag(field, ctx.Node.DefaultTag); err != nil {
		return errorc.With(
			errors.ErrSetDefault,
			errorc.String(keys.TagDefault, ctx.Node.DefaultTag),
			errorc.String(keys.FieldPath, ctx.Path),
			errorc.Error(keys.Cause, err),
		)
	}

	return nil
}

// applyDefaultTag applies one `default` tag to a field value.
//
// Supported values are:
//   - `dive`, which allocates nil pointer-to-struct fields before child defaults
//     are applied by walkSchema;
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

// fieldByIndex resolves a field from base using a compiled index path.
//
// For ordinary nested structs, node.I is root-based.
// For collection element fields, node.I is relative to the concrete element.
func fieldByIndex(
	base reflect.Value,
	index []int,
) (reflect.Value, bool) {
	v := base

	for _, i := range index {
		v = unwrapInterface(v)

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if !v.CanSet() ||
					v.Type().Elem().Kind() != reflect.Struct {
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
	for v.IsValid() &&
		v.Kind() == reflect.Interface &&
		!v.IsNil() {
		v = v.Elem()
	}

	return v
}

func isDurationType(typ reflect.Type) bool {
	return typ == reflect.TypeOf(types.Duration(0)) ||
		typ == reflect.TypeOf(time.Duration(0))
}

// setLiteralValue sets a literal value into fv.
// If onlyZero is true, the value is set only when the target is zero.
// For pointer-to-scalar fields, it allocates and sets the pointed value.
//
//nolint:gocyclo,funlen // cyclomatic complexity is acceptable here
func setLiteralValue(
	fv reflect.Value,
	lit string,
	onlyZero bool,
) error {
	target := fv

	// Allocate for pointer-to-scalar when nil.
	if target.Kind() == reflect.Ptr {
		// If nil and element is not struct/map/slice, allocate.
		if target.IsNil() {
			ek := target.Type().Elem().Kind()

			switch ek {
			case reflect.Struct,
				reflect.Map,
				reflect.Slice,
				reflect.Array:
				// Do not auto-allocate complex types on literal values.

			default:
				target.Set(
					reflect.New(target.Type().Elem()),
				)
			}
		}

		if !target.IsNil() {
			target = target.Elem()
		}
	}

	// Only set if zero when requested by defaults.
	if !target.CanSet() ||
		onlyZero && !target.IsZero() {
		return nil
	}

	// Handle special case: time.Duration typed fields.
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

	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		iv, err := parseInt64(lit, target.Kind())
		if err != nil {
			return err
		}

		target.SetInt(iv)

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		uv, err := parseUint64(lit)
		if err != nil {
			return err
		}

		target.SetUint(uv)

	case reflect.Float32, reflect.Float64:
		parsed, err := parseFloat64(lit)
		if err != nil {
			return err
		}

		target.SetFloat(parsed)

	case reflect.Complex64, reflect.Complex128:
		parsed, err := parseComplex128(
			lit,
			target.Type().Bits(),
		)
		if err != nil {
			return err
		}

		target.SetComplex(parsed)

	default:
		return errorc.With(
			errors.ErrDefaultLiteralUnsupportedKind,
			errorc.String(
				keys.DefaultLiteralKind,
				target.Kind().String(),
			),
		)
	}

	return nil
}

func parseInt64(
	lit string,
	kind reflect.Kind,
) (int64, error) {
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

func parseQuotedRuneLiteral(
	lit string,
) (r rune, b bool, e error) {
	if len(lit) < 2 ||
		lit[0] != '\'' ||
		lit[len(lit)-1] != '\'' {
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
			errorc.String(
				keys.Cause,
				fmt.Sprintf(
					"expected one rune, got %d",
					len(rs),
				),
			),
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
	v, err := strconv.ParseUint(
		strings.TrimSpace(s),
		0,
		64,
	)
	if err != nil {
		return 0, errorc.With(
			errors.ErrCannotParseUint,
			errorc.String(keys.Value, s),
			errorc.Error(keys.Cause, err),
		)
	}

	return v, nil
}

func parseFloat64(s string) (float64, error) {
	v, err := strconv.ParseFloat(
		strings.TrimSpace(s),
		64,
	)
	if err != nil {
		return 0, errorc.With(
			errors.ErrCannotParseFloat,
			errorc.String(keys.Value, s),
			errorc.Error(keys.Cause, err),
		)
	}

	return v, nil
}

func parseComplex128(
	s string,
	bitSize int,
) (complex128, error) {
	v, err := strconv.ParseComplex(
		strings.TrimSpace(s),
		bitSize,
	)
	if err != nil {
		return 0, errorc.With(
			errors.ErrCannotParseComplex,
			errorc.String(keys.Value, s),
			errorc.Error(keys.Cause, err),
		)
	}

	return v, nil
}
