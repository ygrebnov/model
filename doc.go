// Package model provides struct defaulting and validation based on field tags.
//
// It exposes two main entry points:
//   - Binding[T], which provides a reusable engine for applying defaults and validation
//     to multiple instances of the same type,
//   - SetDefaults[T], Validate[T], and ValidateWithDefaults[T] convenience wrappers for one-time operations.
//
// Defaults are driven by `default` and `defaultElem` tags.
//
// Setting defaults walks the object and applies defaults according to `default` and `defaultElem` tags.
// Supported forms:
//   - `default:"<literal>"` sets the field if it is zero
//   - `default:"dive"` on a struct or pointer-to-struct recurses into its fields
//   - `default:"alloc"` allocates an empty map/slice when the field is nil
//   - `defaultElem:"dive"` recurses into slice/array elements or map values that are structs
//
// Notes:
//   - Literals are parsed by kind: string, bool, int/uint, float, time.Duration, rune, complex.
//   - For pointer scalar fields, nil pointers are allocated when a literal default is present.
//
// Environment-backed values can be applied explicitly via ApplyEnv using a source,
// or implicitly by ValidateWithDefaults using the binding's constructor-time env snapshot.
//
// Validation is driven by `validate` and `validateElem` tags plus built-in and custom rules.
// It supports rule parameters via the syntax "rule" or "rule(p1,p2)" and multiple rules separated by commas.
package model
