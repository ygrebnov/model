package core

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
	"github.com/ygrebnov/model/pkg/types"
)

// SetDefaultsStruct walks the struct value and applies environment overrides and defaults.
//
// Environment variable names are built from `env` tags when present, otherwise from
// `json` tags, and finally from the Go field name. Nested names are joined with `_`.
// Environment values are snapshotted when the Service is created, override already-loaded values,
// and `default` tags still only fill zero values.
func (s *Service) SetDefaultsStruct(rv reflect.Value) error {
	return s.setDefaultsStruct(rv, envPrefixPath(s.envPrefix))
}

func snapshotEnv() map[string]string {
	env := os.Environ()
	snapshot := make(map[string]string, len(env))
	for _, entry := range env {
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			snapshot[entry[:idx]] = entry[idx+1:]
			continue
		}
		snapshot[entry] = ""
	}

	return snapshot
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

func (s *Service) setDefaultsStruct(rv reflect.Value, envPath []string) error {
	typ := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := typ.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		fv := rv.Field(i)
		fieldEnvPath, useEnv := appendFieldEnvPath(envPath, field)

		// Handle default tag first. Defaults only fill zero values.
		if dtag := field.Tag.Get(tagDefault); dtag != "" && dtag != "-" {
			if err := s.applyDefaultTag(fv, dtag, field.Name); err != nil {
				return err
			}
		}

		// Then apply environment values. Environment variables override both
		// provided object values and default tag values, even when the env value
		// itself is the type's zero value.
		if useEnv {
			if err := s.applyEnvValue(fv, fieldEnvPath, field.Name); err != nil {
				return err
			}
		}

		if useEnv {
			if err := s.applyNestedEnvValues(fv, fieldEnvPath); err != nil {
				return err
			}
		}

		// Element defaults for collections
		if etag := field.Tag.Get(tagDefaultElem); etag != "" && etag != "-" {
			if err := s.applyDefaultElemTag(fv, etag); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendFieldEnvPath(parent []string, field reflect.StructField) ([]string, bool) {
	part, ok := fieldEnvNamePart(field)
	if !ok {
		return parent, false
	}

	path := make([]string, 0, len(parent)+1)
	path = append(path, parent...)
	path = append(path, part)
	return path, true
}

func fieldEnvNamePart(field reflect.StructField) (string, bool) {
	if tag := field.Tag.Get("env"); tag != "" {
		if tag == "-" {
			return "", false
		}
		return strings.ToUpper(tag), true
	}

	if tag := field.Tag.Get("json"); tag != "" {
		name := tagName(tag)
		if name != "" && name != "-" {
			return strings.ToUpper(name), true
		}
	}

	return strings.ToUpper(field.Name), true
}

func tagName(tag string) string {
	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		return tag[:idx]
	}
	return tag
}

func joinEnvPath(path []string) string {
	return strings.Join(path, "_")
}

func (s *Service) lookupEnv(name string) (string, bool) {
	if s == nil || s.envValues == nil {
		return "", false
	}

	value, ok := s.envValues[name]
	return value, ok
}

func (s *Service) applyEnvValue(fv reflect.Value, envPath []string, fieldName string) error {
	if !canSetLiteralValue(fv) {
		return nil
	}

	name := joinEnvPath(envPath)
	value, ok := s.lookupEnv(name)
	if !ok {
		return nil
	}

	if err := setLiteralValue(fv, value, false); err != nil {
		return errorc.With(
			errors.ErrSetDefault,
			errorc.String(keys.FieldName, fieldName),
			errorc.String("env", name),
			errorc.Error(keys.Cause, err),
		)
	}

	return nil
}

func (s *Service) applyNestedEnvValues(fv reflect.Value, envPath []string) error {
	value := fv

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			if value.Type().Elem().Kind() != reflect.Struct {
				return nil
			}

			if !s.hasNestedEnvValue(value.Type().Elem(), envPath) {
				return nil
			}

			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		if isDurationType(value.Type()) {
			return nil
		}
		return s.setDefaultsStruct(value, envPath)

	case reflect.Map:
		return s.applyMapEnvValues(value, envPath)

	case reflect.Slice, reflect.Array:
		// Environment variable support for slices is intentionally skipped.
		return nil
	}

	return nil
}

func (s *Service) hasNestedEnvValue(typ reflect.Type, envPath []string) bool {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct || isDurationType(typ) {
		return false
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		fieldEnvPath, useEnv := appendFieldEnvPath(envPath, field)
		if !useEnv {
			continue
		}

		if isSupportedLiteralType(field.Type) {
			if _, ok := s.lookupEnv(joinEnvPath(fieldEnvPath)); ok {
				return true
			}
			continue
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct && s.hasNestedEnvValue(fieldType, fieldEnvPath) {
			return true
		}
	}

	return false
}

func isSupportedLiteralType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return isSupportedLiteralKind(typ)
}

func (s *Service) applyMapEnvValues(mapValue reflect.Value, envPath []string) error {
	if mapValue.IsNil() {
		return nil
	}

	for _, key := range mapValue.MapKeys() {
		mapElemValue := mapValue.MapIndex(key)
		keyPath := appendEnvPart(envPath, fmt.Sprint(key.Interface()))

		if canSetLiteralValue(mapElemValue) {
			name := joinEnvPath(keyPath)
			value, ok := s.lookupEnv(name)
			if !ok {
				continue
			}

			updated := reflect.New(mapElemValue.Type()).Elem()
			updated.Set(mapElemValue)
			if err := setLiteralValue(updated, value, false); err != nil {
				return errorc.With(
					errors.ErrSetDefault,
					errorc.String("env", name),
					errorc.Error(keys.Cause, err),
				)
			}
			mapValue.SetMapIndex(key, updated)
			continue
		}

		if mapElemValue.Kind() == reflect.Ptr {
			if !mapElemValue.IsNil() && mapElemValue.Elem().Kind() == reflect.Struct {
				if err := s.setDefaultsStruct(mapElemValue.Elem(), keyPath); err != nil {
					return err
				}
			}
			continue
		}

		if mapElemValue.Kind() == reflect.Struct {
			structValue := reflect.New(mapElemValue.Type()).Elem()
			structValue.Set(mapElemValue)
			if err := s.setDefaultsStruct(structValue, keyPath); err != nil {
				return err
			}
			mapValue.SetMapIndex(key, structValue)
		}
	}

	return nil
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
			return s.setDefaultsStruct(fv.Elem(), envPrefixPath(s.envPrefix))
		}
		return nil
	case reflect.Struct:
		return s.setDefaultsStruct(fv, envPrefixPath(s.envPrefix))
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
			if err := s.setDefaultsStruct(dv, envPrefixPath(s.envPrefix)); err != nil {
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
				if err := s.setDefaultsStruct(mapElemValue.Elem(), envPrefixPath(s.envPrefix)); err != nil {
					return err
				}
			}
			continue
		}

		// Value-typed struct map values: copy-modify-write-back
		if mapElemValue.Kind() == reflect.Struct {
			structValue := reflect.New(mapElemValue.Type()).Elem()
			structValue.Set(mapElemValue)
			if err := s.setDefaultsStruct(structValue, envPrefixPath(s.envPrefix)); err != nil {
				return err
			}
			mapValue.SetMapIndex(key, structValue)
		}
	}

	return nil
}

func isDurationType(typ reflect.Type) bool {
	return typ == reflect.TypeOf(types.Duration(0)) || typ == reflect.TypeOf(time.Duration(0))
}

// setLiteralDefault sets a literal default value into fv if it is zero.
// For pointer-to-scalar fields, it allocates and sets the pointed value.
func setLiteralDefault(fv reflect.Value, lit string) error {
	return setLiteralValue(fv, lit, true)
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
