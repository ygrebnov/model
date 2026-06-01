// Package model provides struct defaulting and validation based on field tags.
//
// It exposes two main entry points:
//   - Model[T], which binds a single struct instance to defaulting and validation.
//   - Binding[T], which provides a reusable engine for applying defaults and validation
//     to multiple instances of the same type.
//
// Defaults are driven by `default` and `defaultElem` tags. Validation is driven by
// `validate` and `validateElem` tags plus custom or built-in rules.
package model
