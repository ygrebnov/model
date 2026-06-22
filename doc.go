// Package model provides struct defaulting and validation based on field tags.
//
// It exposes two main entry points:
//   - Binding[T], which provides a reusable engine for applying defaults and validation
//     to multiple instances of the same type,
//   - SetDefaults[T], Validate[T], and ValidateWithDefaults[T] convenience wrappers for one-time operations.
//
// Defaults are driven by `default`, `defaultElem` tags and environment variables.
//
// Setting defaults walks the object and applies defaults according to `default`, `defaultElem` tags and
// environment variables.
// Supported forms:
//   - `default:"<literal>"` sets the field if it is zero
//   - `default:"dive"` on a struct or pointer-to-struct recurses into its fields
//   - `default:"alloc"` allocates an empty map/slice when the field is nil
//   - `defaultElem:"dive"` recurses into slice/array elements or map values that are structs
//
// Notes:
//   - Literals are parsed by kind: string, bool, int/uint, float, time.Duration, rune, complex.
//   - For pointer scalar fields, nil pointers are allocated when a literal default is present.
//   - Reusable bindings snapshot environment variables when they are constructed.
//
// Validation is driven by `validate` and `validateElem` tags plus built-in and custom rules.
// It supports rule parameters via the syntax "rule" or "rule(p1,p2)" and multiple rules separated by commas.
//
// Package also supports invariants of Binding and convenience wrappers which resolve type at runtime:
//   - DynamicBinding,
//   - SetDefaultsAny, ValidateAny, ValidateWithDefaultsAny.
//
// Although it may be necessary to use these dynamic wrappers in some cases, the preferred approach
// is to use the type-safe Binding[T] or the generic wrappers for compile-time safety.
package model
