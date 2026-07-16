// Package model applies defaults and external values to Go structs, exports
// struct values, and validates fields declared with tags.
//
// NewBinding compiles reusable metadata for a struct type. A Binding can then
// apply defaults, apply typed values from a ValueSource, apply environment
// values snapshotted during construction, write values to a ValueSink, and
// validate one or more instances of that type. SetDefaults, Validate, and
// ValidateWithDefaults construct a Binding for one-time operations.
//
// Schema paths use exact exported Go field names. Nested fields use dots, such
// as "Server.Host"; collection schema paths use "[]" such as "Items[].Name";
// concrete validation paths include an index or map key, such as
// "Items[0].Name". This preserves distinct names such as "URL" and "Url".
//
// Defaults are declared with default and defaultElem tags. A literal default is
// applied only to a zero value; default:"dive" traverses a nested struct and
// allocates a nil pointer-to-struct when needed; default:"alloc" initializes a
// nil slice or map. Pointer-to-scalar fields are allocated for literal
// defaults.
//
// ApplyValues reads typed values from a ValueSource by schema path. Found
// values replace existing fields, nil resets a field to its zero value, and
// assignable or convertible values are accepted. It allocates a nil
// pointer-to-struct only when a descendant value is supplied. WriteValues
// reports reachable field values to a ValueSink using the same schema paths.
//
// ApplyEnv reads the operating-system environment once during NewBinding.
// ValidateWithDefaults applies defaults, then that snapshot, then validation.
//
// Validation is declared with validate and validateElem tags. Built-in rules
// are available automatically; custom rules are registered with NewRule and
// WithRules. Rules support parameters with "rule(p1,p2)" syntax, and a
// validation failure is returned as a *validation.Error containing all field
// errors.
package model
