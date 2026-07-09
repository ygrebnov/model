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

/*
// SetDefaultsStruct walks the struct value and applies default tags.
//
// The `default` tag fills zero values only. The `defaultElem` tag applies
// element defaults to supported collections.
func (s *Service) SetDefaultsStruct(rv reflect.Value) error {
	return s.setDefaultsStruct(rv)
}

func envPrefixPath(prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil
	}

	prefix = strings.Trim(prefix, "_")
	if prefix == "" {
		return nil
	}

	parts := strings.Split(prefix, "_")
	path := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		path = append(path, strings.ToUpper(part))
	}

	if len(path) == 0 {
		return nil
	}
	return path
}

func (s *Service) setDefaultsStruct(rv reflect.Value) error {
	compiled, err := s.schemaFor(rv.Type())
	if err != nil {
		return err
	}

	for _, field := range compiled.Root.Children {
		fv := rv.FieldByIndex(field.Index)

		// Handle default tag first. Defaults only fill zero values.
		if field.DefaultTag != "" {
			if err := s.applyDefaultTag(fv, field.DefaultTag, field.Name); err != nil {
				return err
			}
		}

		if err := s.applyNestedDefaultValues(fv); err != nil {
			return err
		}

		// Element defaults for collections
		if field.DefaultElemTag != "" {
			if err := s.applyDefaultElemTag(fv, field.DefaultElemTag); err != nil {
				return err
			}
		}
	}

	return nil
}

func joinEnvPath(path []string) string {
	return strings.Join(path, "_")
}

func isSupportedLiteralType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return isSupportedLiteralKind(typ)
}

func appendEnvPart(parent []string, part string) []string {
	path := make([]string, 0, len(parent)+1)
	path = append(path, parent...)
	path = append(path, strings.ToUpper(part))
	return path
}

func canSetLiteralValue(value reflect.Value) bool {
	if !value.IsValid() {
		return false
	}

	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return isSupportedLiteralKind(value.Type().Elem())
		}
		value = value.Elem()
	}

	return isSupportedLiteralKind(value.Type())
}

func isSupportedLiteralKind(typ reflect.Type) bool {
	if isDurationType(typ) {
		return true
	}

	switch typ.Kind() {
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

// applyDefaultTag applies the `default` tag semantics to a single field value.
// Supported values: "dive", "alloc", or a literal (delegated to setLiteralDefault).
func (s *Service) applyDefaultTag(fv reflect.Value, tag, fieldName string) error {
	switch tag {
	case tagDive:
		return s.diveDefaultsIntoValue(fv)
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
				errorc.String(keys.FieldName, fieldName),
				errorc.Error(keys.Cause, err),
			)
		}
		return nil
	}
}

func (s *Service) applyNestedDefaultValues(fv reflect.Value) error {
	value := fv
	if value.Kind() == reflect.Ptr {
		if value.IsNil() || value.Type().Elem().Kind() != reflect.Struct {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		if isDurationType(value.Type()) {
			return nil
		}
		return s.setDefaultsStruct(value)
	case reflect.Map:
		return s.setMapElementsDefaultValues(value)
	default:
		return nil
	}
}

// diveDefaultsIntoValue recurses into a struct or *struct field to apply nested defaults.
// For nil *struct, it allocates the struct before diving. Non-structs are ignored.
func (s *Service) diveDefaultsIntoValue(fv reflect.Value) error {
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
			return s.setDefaultsStruct(fv.Elem())
		}
		return nil
	case reflect.Struct:
		return s.setDefaultsStruct(fv)
	default:
		return nil
	}
}

// applyDefaultElemTag applies defaults to elements/values of collections based on `defaultElem`.
// Currently supports: defaultElem:"dive".
func (s *Service) applyDefaultElemTag(fv reflect.Value, tag string) error {
	if tag != tagDive {
		return nil
	}
	cont := fv
	if cont.Kind() == reflect.Ptr && !cont.IsNil() {
		cont = cont.Elem()
	}
	switch cont.Kind() {
	case reflect.Slice, reflect.Array:
		if err := s.setSliceArrayElementsDefaultValues(cont); err != nil {
			return err
		}
	case reflect.Map:
		if err := s.setMapElementsDefaultValues(cont); err != nil {
			return err
		}
	default:
		// ignore for non-collections
	}
	return nil
}

func (s *Service) setSliceArrayElementsDefaultValues(value reflect.Value) error {
	l := value.Len()
	for j := 0; j < l; j++ {
		ev := value.Index(j)
		dv := ev

		if dv.Kind() == reflect.Ptr && !dv.IsNil() {
			dv = dv.Elem()
		}

		if dv.Kind() == reflect.Struct {
			if err := s.setDefaultsStruct(dv); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) setMapElementsDefaultValues(mapValue reflect.Value) error {
	for _, key := range mapValue.MapKeys() {
		mapElemValue := mapValue.MapIndex(key)

		// Pointer-to-struct map values: mutate in place
		if mapElemValue.Kind() == reflect.Ptr {
			if !mapElemValue.IsNil() && mapElemValue.Elem().Kind() == reflect.Struct {
				if err := s.setDefaultsStruct(mapElemValue.Elem()); err != nil {
					return err
				}
			}
			continue
		}

		// Value-typed struct map values: copy-modify-write-back
		if mapElemValue.Kind() == reflect.Struct {
			structValue := reflect.New(mapElemValue.Type()).Elem()
			structValue.Set(mapElemValue)
			if err := s.setDefaultsStruct(structValue); err != nil {
				return err
			}
			mapValue.SetMapIndex(key, structValue)
		}
	}

	return nil
}

// setLiteralDefault sets a literal default value into fv if it is zero.
// For pointer-to-scalar fields, it allocates and sets the pointed value.
func setLiteralDefault(fv reflect.Value, lit string) error {
	return setLiteralValue(fv, lit, true)
}
*/

// ApplyDefaultsUsingSchema applies `default` tags using a compiled schema controller.

//

// Precedence:

//   - existing non-zero field value wins;

//   - `default` is applied only when the field is zero;

//   - collection element defaults are applied to existing elements only;

//   - nil slices/maps are not allocated here;

//   - nil pointer-to-struct values are allocated only when nested defaults need to be applied.

func (s *Service[T]) SetDefaultsStruct(root reflect.Value) error {
	/*
		if obj == nil {
			return fmt.Errorf("defaults target is nil")
		}
		if c == nil || c.Tree == nil {
			return fmt.Errorf("schema controller is nil")
		}
		root := reflect.ValueOf(obj)
		for root.Kind() == reflect.Ptr {
			if root.IsNil() {
				return fmt.Errorf("defaults target is nil")
			}
			root = root.Elem()
		}
		if root.Kind() != reflect.Struct {
			return fmt.Errorf("defaults target must point to struct, got %s", root.Kind())
		}

	*/
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
	return applyDefaultChildren(field, node)
}

// applyDefaultChildren applies child defaults below a struct, pointer-to-struct,
// slice/array of structs, or map of structs.
func applyDefaultChildren(field reflect.Value, node *schema.N) error {
	field = unwrapInterface(field)
	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			if field.Type().Elem().Kind() != reflect.Struct {
				return nil
			}
			// Allocate pointer-to-struct only when the schema has child defaults.
			// This lets nested defaults materialize optional nested config blocks.
			if hasDefaults(node.Children) && field.CanSet() {
				field.Set(reflect.New(field.Type().Elem()))
			} else {
				return nil
			}
		}
		return applyDefaultChildren(field.Elem(), node)
	case reflect.Struct:
		for _, child := range node.Children {
			if err := applyDefaultNode(field, child); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		for i := 0; i < field.Len(); i++ {
			elem := unwrapInterface(field.Index(i))
			if elem.Kind() == reflect.Ptr {
				if elem.IsNil() {
					continue
				}
				elem = elem.Elem()
			}
			if elem.Kind() != reflect.Struct {
				continue
			}
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
			if target.Kind() != reflect.Struct {
				continue
			}
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

// applyFieldDefault applies the node's DefaultTag to field when field is zero.
func applyFieldDefault(field reflect.Value, node *schema.N) error {
	if node.DefaultTag == "" {
		return nil
	}
	field = unwrapInterface(field)
	if !field.IsValid() || !field.CanSet() {
		return nil
	}
	if !field.IsZero() {
		return nil
	}
	return setLiteralValue(field, node.DefaultTag, true)
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

/*
// setLiteralValue sets a scalar field from a string literal.

//

// This is intentionally small. You can replace it with your existing

// setLiteralValue implementation from model.

func setLiteralValue(v reflect.Value, lit string) error {

	if !v.CanSet() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setLiteralValue(v.Elem(), lit)
	}
	if v.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(lit)
		if err != nil {
			return err
		}
		v.SetInt(int64(d))
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(lit)
		return nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(lit)
		if err != nil {
			return err
		}
		v.SetBool(parsed)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(lit, 0, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(parsed)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(lit, 0, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(parsed)
		return nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(lit, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(parsed)
		return nil
	case reflect.Complex64, reflect.Complex128:
		parsed, err := strconv.ParseComplex(lit, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetComplex(parsed)
		return nil
	default:
		return fmt.Errorf("unsupported default target type %s", v.Type())
	}

}
*/

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
