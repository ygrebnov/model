[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/model)](https://pkg.go.dev/github.com/ygrebnov/model)
[![Build Status](https://github.com/ygrebnov/model/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/model/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/model)](https://goreportcard.com/report/github.com/ygrebnov/model)

# model — defaults & validation for Go structs

`model` creates reusable **Bindings** for your structs. It can:

- **Set defaults** from struct tags like `default:"…"` and `defaultElem:"…"` as well as environment variables.
- **Validate** fields using named rules from `validate:"…"` and `validateElem:"…"`.
- Accumulate all issues into a single `*validation.Error` (no fail-fast).
- Recurse through nested structs, pointers, slices/arrays, and map values, including cycle-safe validation of pointer graphs.

Library is designed to be **small, explicit, and type-safe** (uses generics). You register rules (via `NewRule`) and `model` handles traversal, dispatch, and error reporting. Built‑in rules are always available implicitly (you don’t have to register them unless you want to override their behavior). `Binding[T]` is a shared engine you can use for defaults and validation across many values of the same type. Library provides a set of convenient wrappers for one-off use cases, too. `DynamicBinding` is available for cases where you need to validate arbitrary types at runtime.

## Features

- **Simple API**: a constructor and three main methods on `Binding[T]` and `DynamicBinding`: `SetDefaults()`, `Validate(ctx)`, and `ValidateWithDefaults(ctx)`. Convenience wrappers correspond to main methods for one-off use cases.
- **Predictable behavior**: defaults fill *only zero values*; validation gathers *all* issues.
- **Extensible**: register your own rules; supports interface-based rules (e.g., rules for `fmt.Stringer`).
- **Structured errors**: built-in rules and many internal errors use sentinel values plus structured key/value metadata (via `errorc`), making it easier to inspect and transform validation failures.

---

## Install

```bash
go get github.com/ygrebnov/model
```

---

## Quick start

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/ygrebnov/model"
    "github.com/ygrebnov/model/validation"
)

type Address struct {
    City    string `default:"Paris"  validate:"min(3),max(50)"`
    Country string `default:"France" validate:"min(3),max(50)"`
}

type User struct {
    Name     string        `default:"Anonymous" validate:"min(3),max(50)"`
    Age      int           `default:"18"        validate:"min(1),nonzero"`
    Timeout  time.Duration `default:"1s"`
    Home     Address       `default:"dive"`           // recurse into nested struct
    Aliases  []string      `validateElem:"min(3)"`   // validate each element
    Profiles map[string]Address `default:"alloc" defaultElem:"dive"`
	Email string `validate:"email,omitempty"` // skip validation if empty
}

func main() {
	customRule, _ := validation.NewRule("email", func(s string, _ ...string) error {
	    if s != "user@company.com" {
	        return fmt.Errorf("must be 'user@company.com'")
	    }
	    return nil
	})
	b, err := model.NewBinding[User]( // reusable engine for User
	    model.WithRules(customRule),
	)
	if err != nil {
	    fmt.Println("binding error:", err)
	    return
    }

    u1 := User{Aliases: []string{"", "ok"}} // will fail validation
		
    err = b.ValidateWithDefaults( // apply defaults and validate in one call
		context.Background(), // context-aware validation with cancellation support
		&u1,
	)
    if err != nil {
        var ve *validation.Error
        if errors.As(err, &ve) {
            b, _ := json.MarshalIndent(ve, "", "  ")
            fmt.Println(string(b))
        } else {
            fmt.Println("error:", err)
        }
        return
    }

    fmt.Printf("User1 after defaults: %+v\n", u1)

    // You can also set defaults and validate separately:
    u2 := User{Aliases: []string{"pass"}} // will pass validation
    _ = b.ApplyDefaults(&u2)
    _ = b.Validate(context.Background(), &u2) // returns *validation.Error on failure

    // Custom rules are passed during binding construction via WithRules:
    u3 := User{
		Aliases: []string{"pass"}, 
		Email: "user@company.com", // will pass validation with custom rule
	}
	_ = b.ValidateWithDefaults(context.Background(), &u3)
	
	// If you do not need reusable binding, you can use wrappers:
    u4 := User{Aliases: []string{"pass"}} // will pass validation
	_ = model.SetDefaults(&u4) // apply defaults
    _ = model.Validate(context.Background(), &u4) // returns *validation.Error on failure
	_ = model.ValidateWithDefaults(context.Background(), &u4) // apply defaults and validate in one call
}
```

---

## Setting defaults

Library supports setting defaults for exported fields via struct tags and environment variables.

### `default:"…"` struct tag

Supported forms:
  - `default:"<literal>"` sets the field if it is zero
  - `default:"dive"` on a struct or pointer-to-struct recurses into its fields
  - `default:"alloc"` allocates an empty map/slice when the field is nil

Supported literal types:
- string
- bool
- int, int8, int16, int32, int64
- uint, uint8, uint16, uint32, uint64
- float32, float64
- complex64, complex128
- time.Duration, types.Duration (from `github.com/ygrebnov/model/pkg/types`)

Defaults are applied only to **zero values**. Pointer-to-scalar fields (e.g., `*int`, `*bool`) are auto-allocated for literal defaults when nil. Pointer-to-complex types (struct/map/slice) are **not** auto-allocated for literals. Use `dive` to recurse into struct or `*struct` elements (allocating a new struct for nil pointers). Use `alloc` to allocate an empty slice/map if nil.

---

### `defaultElem:"…"` struct tag

Applies a default to elements of a slice/array or values of a map. Supported literal types are the same as for `default:"…"`. `defaultElem:"dive"` recurses into slice/array elements or map values that are structs.

---

### environment variables

Top-level struct field can be populated from an environment variable corresponding to the field uppercase name. For example, `Name string` field value can be set via `NAME` environment variable. 

Nested struct fields can be populated via `PARENT_CHILD` style environment variable names. For example, `Parent struct { Child string }` field value can be set via `PARENT_CHILD` environment variable. 

You can also use `env:"ENV_VAR_NAME"` struct tag to specify a custom environment variable name for a field.

Library also supports environment variables names prefixes, which can be set via `WithEnvPrefix` option.

```go
type S struct {
    Name string `env:"CUSTOM_NAME"`
    Age  int
}

_ = os.Setenv("MYAPP_CUSTOM_NAME", "Alice")

var s S
_ = model.SetDefaults(&s, model.WithEnvPrefix("MYAPP"))

fmt.Printf("S after defaults: %+v\n", s) 
// Output: S after defaults: {Name:Alice Age:0}
```

Values from environment variables are applied **after** literal defaults, so they can override them.
Reusable `Binding[T]` and `DynamicBinding` instances snapshot environment variables when they are
constructed, so later process env changes are not observed by that binding. Convenience wrappers
such as `SetDefaults` and `ValidateWithDefaults` create a fresh binding per call and therefore read
the current environment each time.

---

## Validation

Validation is driven by `validate` and `validateElem` tags plus built-in and custom rules. Validation gathers **all** field errors and returns a `*validation.Error` on failure. Cancellation/deadlines follow the provided context. Nested structs and non-nil pointers to structs are traversed recursively, and pointer cycles are skipped on the current traversal path so self-referential object graphs terminate safely. Shared subgraphs can still be validated through each reachable field path.

### `validate:"…"` and `validateElem:"…"` struct tags

It supports rule parameters via the syntax "rule" or "rule(p1,p2)" and multiple rules separated by commas. Empty tokens are skipped (`,email,` → `email`). `validateElem:"dive"` recurses into struct elements, non-struct or nil pointer elements produce a misuse error under rule name `dive`. Shared nodes can still be validated through multiple field paths; only the active recursion path is cycle-guarded.

If you want to skip validation for zero values, you can use `omitempty` in the `validate` tag. For example:

```go
type User struct {
    Name string `validate:"omitempty,min(3),max(50)"`
}
```

---

### Rules

Create rules with `NewRule` and pass them. You can supply multiple different rule names and/or multiple overloads (different field types) in a single call. Duplicate exact overloads (same rule name & identical field type) are rejected.

```go
maxLen, _ := validation.NewRule[string]("maxLen", func(s string, params ...string) error {
    if len(params) < 1 { return fmt.Errorf("maxLen requires 1 param") }
    n, _ := strconv.Atoi(params[0])
    if len(s) > n { return fmt.Errorf("must be <= %d chars", n) }
    return nil
})

positive64, _ := validation.NewRule[int64]("positive", func(v int64, _ ...string) error {
    if v <= 0 { return fmt.Errorf("must be > 0") }
    return nil
})

b, _ := model.NewBinding[User](
    model.WithRules(maxLen, positive64), // different names & types allowed
)
```

To override a built-in rule, register a custom rule *before* running validation:

```go
type S struct {
    Name string `validate:"min(1)"`
}

minCustom, _ := validation.NewRule[string]("min", func(s string, _ ...string) error {
    if strings.TrimSpace(s) == "" { return fmt.Errorf("must not be blank or whitespace") }
    return nil
})

b, _ := model.NewBinding[S](
    model.WithRules(minCustom), // override builtin string min
)

s := S{Name: "  "}

_ = b.Validate(context.Background(), &s) // returns *validation.Error with rule "min" failure
```

Duplicate exact registrations for the same name & type are rejected. Interface-based rules still participate via assignable matching when no exact rule exists.

---

### Built-in rules

Built-in rules are always implicitly available (you do **not** need to register or import anything for them):

- String:
  - `min(N)` – length must be **>= N** (N ≥ 1). If N < 1, the rule is a no-op.
  - `max(N)` – length must be **<= N** (N ≥ 0). If N < 0, the rule is a no-op.
  - `oneof(v1,v2,...)` – value must be exactly one of the listed strings.
  - `email` – lightweight email check (single `@`, non-empty local/domain, domain contains `.`, no whitespace).
  - `uuid` – canonical UUID string (`8-4-4-4-12` hex format with hyphens).
  - `semver` – semantic version string (e.g., `1.2.3`, `1.2.3-alpha`, `1.2.3-beta+exp.sha.5114f85`).
- Numeric (`int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `uintptr`, `float32`, `float64`):
  - `min(V)` – value must be **>= V**.
  - `max(V)` – value must be **<= V**.
  - `nonzero` – value must not be zero.
  - `oneof(v1,v2,...)` – value must be equal to one of the listed values.

> The library lazy-loads built-ins on first use, so unused numeric/string sets impose no startup cost.

---

### Structured errors: errorc, sentinels, and structured keys

Under the hood, `model` uses [`github.com/ygrebnov/errorc`](https://github.com/ygrebnov/errorc) to build structured errors. Sentinel errors live in `github.com/ygrebnov/model/errors`, and strongly-typed keys live in `github.com/ygrebnov/model/keys`:

- Sentinels (examples):
  - `modelerrors.ErrNilObject`
  - `modelerrors.ErrNotStructPtr`
  - `modelerrors.ErrRuleMissingParameter`
  - `modelerrors.ErrRuleInvalidParameter`
  - `modelerrors.ErrRuleConstraintViolated`
- Keys (examples):
  - `modelkeys.RuleName` (e.g. `model.rule.name`)
  - `modelkeys.RuleParamName` (e.g. `model.rule.param_name`)
  - `modelkeys.RuleParamValue` (e.g. `model.rule.param_value`)
  - `modelkeys.FieldName` (e.g. `model.field.name`)
  - `modelkeys.Cause` (the underlying cause error)

Builtin rules attach metadata when they fail. For example, the string `min` rule:

```go
return errorc.With(
    modelerrors.ErrRuleConstraintViolated,
    errorc.String(modelkeys.RuleName, "min"),
    errorc.String(modelkeys.RuleParamName, "length"),
    errorc.String(modelkeys.RuleParamValue, raw),
)
```

Validation formatting keeps the top-level `*validation.Error` concise for built-in and internal structured errors (for example: `- Field "Name": rule "min": constraint violated (length=3)`), while the underlying `FieldError.Err` values still preserve the raw structured metadata for inspection and mapping. Errors returned by custom rules are left as-is so their original messages are not rewritten heuristically.

From your code, you can inspect these errors using `errors.Is`, inspect the wrapped `FieldError.Err`, or use `errors.As` into `*validation.Error` for field-level failures.

Example:

```go
m, err := model.New(&u, model.WithValidation[User](ctx))
if err != nil {
    var ve *validation.Error
    if errors.As(err, &ve) {
        // Per-field errors
        for path, fes := range ve.ByField() {
            for _, fe := range fes {
                fmt.Printf("field=%s rule=%s err=%v\n", path, fe.Rule, fe.Err)
            }
        }
    }
}
```

If you need to work directly with the structured error metadata (for example, to localize messages), inspect `FieldError.Err` and use the keys exposed by `github.com/ygrebnov/model/keys` together with the sentinels from `github.com/ygrebnov/model/errors`.

## Contributing

Contributions are welcome!
Please open an [issue](https://github.com/ygrebnov/model/issues) or submit a [pull request](https://github.com/ygrebnov/model/pulls).

## License

Distributed under the BSD 3-Clause License. See the [LICENSE](LICENSE) file for details.