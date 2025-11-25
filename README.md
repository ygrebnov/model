[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/model)](https://pkg.go.dev/github.com/ygrebnov/model)
[![Build Status](https://github.com/ygrebnov/model/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/model/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/model)](https://goreportcard.com/report/github.com/ygrebnov/model)

# model — defaults & validation for Go structs

`model` is a tiny helper that binds a **Model** (and optionally a reusable **Binding**) to your structs. It can:

- **Set defaults** from struct tags like `default:"…"` and `defaultElem:"…"`.
- **Validate** fields using named rules from `validate:"…"` and `validateElem:"…"`.
- Accumulate all issues into a single **ValidationError** (no fail-fast).
- Recurse through nested structs, pointers, slices/arrays, and map values.

It’s designed to be **small, explicit, and type-safe** (uses generics). You register rules (via `NewRule`) and `model` handles traversal, dispatch, and error reporting. Built‑in rules are always available implicitly (you don’t have to register them unless you want to override their behavior). For reusable validation across many values of the same type, you can use `Binding[T]` as a shared engine for defaults and validation.

## Table of Contents
- [Install](#install)
- [Why use this?](#why-use-this)
- [Quick start](#quick-start)
- [Binding[T] – reusable defaults and validation](#bindingt--reusable-defaults-and-validation)
- [Constructor: `New`](#constructor-new)
- [Why no MustNew?](#why-no-mustnew)
- [Functional options](#functional-options)
- [Model methods](#model-methods)
- [Struct tags (how it works)](#struct-tags-how-it-works)
- [Built-in rules](#built-in-rules)
- [Structured errors: errorc, sentinels, and ErrorField* keys](#structured-errors-errorc-sentinels-and-errorfield-keys)
- [Overriding a builtin rule](#overriding-a-builtin-rule)
- [Custom rules (with parameters)](#custom-rules-with-parameters)
- [Error types](#error-types)
- [Performance & benchmarks](#performance--benchmarks)
  - [Performance tuning tips](#performance-tuning-tips)
- [Behavior notes](#behavior-notes)
- [Integration example: validation failure with sorted available types](#integration-example-validation-failure-with-sorted-available-types)
- [Missing rule vs missing overload](#missing-rule-vs-missing-overload)
- [Minimal example](#minimal-example)
- [Examples](#examples)
- [License](#license)

---

## Install

```bash
go get github.com/ygrebnov/model
```

---

## Why use this?

- **Simple API**: one constructor and two main methods on `Model[T]`: `SetDefaults()` and `Validate(ctx)`. For reusable engines, use `Binding[T]` to apply the same defaults/validation to many instances.
- **Predictable behavior**: defaults fill *only zero values*; validation gathers *all* issues.
- **Extensible**: register your own rules; supports interface-based rules (e.g., rules for `fmt.Stringer`).
- **Structured errors**: built-in rules and many internal errors use sentinel values plus structured key/value metadata (via `errorc`), making it easier to inspect and transform validation failures.

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
}

func main() {
    u := User{Aliases: []string{"", "ok"}} // index 0 will fail validation

    m, err := model.New(&u,
        model.WithDefaults[User](),                       // apply defaults during construction
        model.WithValidation[User](context.Background()), // run validation during construction (cancellable)
    )
    if err != nil {
        var ve *model.ValidationError
        if errors.As(err, &ve) {
            b, _ := json.MarshalIndent(ve, "", "  ")
            fmt.Println(string(b))
        } else {
            fmt.Println("error:", err)
        }
        return
    }

    fmt.Printf("User after defaults: %+v\n", u)

    // You can also call them later:
    _ = m.SetDefaults()                  // guarded by sync.Once — no double work
    _ = m.Validate(context.Background()) // returns *ValidationError on failure
}
```

---

## Binding[T] – reusable defaults and validation

`Model[T]` binds a single struct instance; sometimes you want a reusable engine for a type that you can apply to many instances. That is what `Binding[T]` is for.

```go
import (
    "context"

    "github.com/ygrebnov/model"
)

type Payload struct {
    ID      string `validate:"uuid"`
    Email   string `validate:"email"`
    Retries int    `validate:"min(0),max(5)"`
}

func validatePayload(ctx context.Context, p *Payload) error {
    // Construct once per process (or cache it) and reuse.
    b, err := model.NewBinding[Payload]()
    if err != nil {
        return err
    }
    // If you have custom rules:
    //   customRule, _ := model.NewRule[Payload](...)
    //   _ = b.RegisterRules(customRule)

    // Apply defaults (from `default` tags) and then validate.
    if err := b.ValidateWithDefaults(ctx, p); err != nil {
        return err
    }
    return nil
}
```

Key points:

- `NewBinding[T]` builds a reusable binding for the type `T` using a fresh `RulesRegistry` and `RulesMapping`.
- `Binding[T].ApplyDefaults(*T)` applies `default` / `defaultElem` tags to a concrete instance.
- `Binding[T].Validate(ctx, *T)` validates a concrete instance using `validate` / `validateElem` tags.
- `Binding[T].ValidateWithDefaults(ctx, *T)` combines both in a single call.
- `Binding[T].RegisterRules(...)` lets you register custom validation rules for that binding’s type; these participate alongside the built-ins.

A typical service pattern is to construct a `Binding[T]` once at startup and reuse it in handlers:

```go
var payloadBinding *model.Binding[Payload]

func init() {
    var err error
    payloadBinding, err = model.NewBinding[Payload]()
    if err != nil {
        panic(err) // or return a startup error from main
    }
}

func handleRequest(ctx context.Context, p *Payload) error {
    if err := payloadBinding.ValidateWithDefaults(ctx, p); err != nil {
        return err
    }
    // p is now defaulted and validated
    return nil
}
```

---

## Constructor: `New`

```go
ctx := context.Background()
m, err := model.New(&user,
    model.WithDefaults[User](),
    model.WithValidation[User](ctx),  // run validation during New() with cancellation support
)
if err != nil {
    var ve *model.ValidationError
    switch {
    case errors.Is(err, model.ErrNilObject):
        // handle nil object
    case errors.Is(err, model.ErrNotStructPtr):
        // handle pointer to non-struct
    case errors.As(err, &ve):
        // handle field validation failures
    default:
        // defaults parsing or other errors
    }
}
```

To validate later explicitly, call `m.Validate(ctx)` with a context appropriate for the request.

---

## Why no MustNew?

`MustNew` (a variant that panics on configuration errors) is intentionally omitted:

- Panics hinder graceful startup error reporting in services / CLIs.
- All failure modes (`nil` object, non-struct pointer, duplicate rule overload, validation failures when requested) are ordinary and recoverable.
- Returning `error` keeps initialization explicit and test-friendly (you can assert exact sentinel errors or unwrap `*ValidationError`).
- If you truly want a panic wrapper, you can write a 2‑line helper in your own code:
  ```go
  func MustNew[T any](o *T, opts ...model.Option[T]) *model.Model[T] {
      m, err := model.New(o, opts...); if err != nil { panic(err) }; return m
  }
  ```

If enough users request it, a helper can be added later—keeping the core API minimal for now.

---

## Functional options

All options run in the order provided. If an option returns an error (e.g., attempting to register a duplicate overload for the same type & name), `New` stops and returns that error.

### `WithDefaults[T]()` — apply defaults during construction

```go
m, err := model.New(&u, model.WithDefaults[User]())
```

- Runs once per `Model` (guarded by `sync.Once`).
- Writes only zero values.

### `WithValidation[T](ctx context.Context)` — run validation during construction

```go
ctx := context.Background()
m, err := model.New(&u,
    model.WithValidation[User](ctx),
)
```

- Gathers **all** field errors; returns a `*ValidationError` on failure.
- Built-ins are always considered first for matching types.
- Cancellation/deadlines follow the provided context.
- To override a built-in rule, register a custom rule *before* `WithValidation`:

```go
minCustom, _ := model.NewRule[string]("min", func(s string, _ ...string) error {
    if strings.TrimSpace(s) == "" { return fmt.Errorf("must not be blank or whitespace") }
    return nil
})

m, err := model.New(&u,
    model.WithRules[User](minCustom), // override
    model.WithValidation[User](ctx),
)
```

### `WithRules[T](rules ...Rule)` — register one or many rules

Create rules with `NewRule` and pass them. You can supply multiple different rule names and/or multiple overloads (different field types) in a single call. Duplicate exact overloads (same rule name & identical field type) are rejected.

```go
maxLen, _ := model.NewRule[string]("maxLen", func(s string, params ...string) error {
    if len(params) < 1 { return fmt.Errorf("maxLen requires 1 param") }
    n, _ := strconv.Atoi(params[0])
    if len(s) > n { return fmt.Errorf("must be <= %d chars", n) }
    return nil
})

positive64, _ := model.NewRule[int64]("positive", func(v int64, _ ...string) error {
    if v <= 0 { return fmt.Errorf("must be > 0") }
    return nil
})

m, _ := model.New(&u,
    model.WithRules[User](maxLen, positive64), // different names & types allowed
)
```

Duplicate exact overloads (same rule name & identical field type) are **rejected at registration time** with `ErrDuplicateOverloadRule`. This prevents later runtime ambiguity.

---

## Model methods

### `SetDefaults() error`

Apply `default:"…"` / `defaultElem:"…"` recursively. Safe to call multiple times (subsequent calls no-op).

### `Validate(ctx context.Context) error`

Walk fields and apply rules from `validate:"…"` / `validateElem:"…"` tags. Returns `*ValidationError` on failure.

- Returns `ctx.Err()` immediately if the context is canceled or its deadline is exceeded.

---

## Struct tags (how it works)

### Defaults: `default:"…"` and `defaultElem:"…"`

- **Literals**: string, bool, ints/uints, floats, `time.Duration`.
- **`dive`**: recurse into struct or `*struct` (allocating a new struct for nil pointers).
- **`alloc`**: allocate empty slice/map if nil.
- **`defaultElem:"dive"`**: recurse into struct elements (slice/array) or map values.

Pointer-to-scalar fields (e.g., `*int`, `*bool`) are auto-allocated for literal defaults when nil. Pointer-to-complex types (struct/map/slice) are **not** auto-allocated for literals.

### Validation: `validate:"…"` and `validateElem:"…"`

- Comma-separated top-level rules.
- Parameters in parentheses: `rule(p1,p2)`.
- Empty tokens skipped (`,email,` → `email`).
- `validateElem:"dive"` recurses into struct elements; non-struct or nil pointer elements produce a misuse error under rule name `dive`.

---

## Built-in rules

Built-in rules are always implicitly available (you do **not** need to register or import anything for them):

- String:
  - `min(N)` – length must be **>= N** (N ≥ 1). If N < 1, the rule is a no-op.
  - `max(N)` – length must be **<= N** (N ≥ 0). If N < 0, the rule is a no-op.
  - `oneof(v1,v2,...)` – value must be exactly one of the listed strings.
  - `email` – lightweight email check (single `@`, non-empty local/domain, domain contains `.`, no whitespace).
  - `uuid` – canonical UUID string (`8-4-4-4-12` hex format with hyphens).
- Int / Int64 / Float64:
  - `min(V)` – value must be **>= V**.
  - `max(V)` – value must be **<= V**.
  - `nonzero` – value must not be zero.
  - `oneof(v1,v2,...)` – value must be equal to one of the listed values.

Overriding: if you register a custom rule with the same name and exact type **before** validation runs, your rule is chosen (duplicate exact registrations for the same name & type are rejected). Interface-based rules still participate via assignable matching when no exact rule exists.

> The library lazy-loads built-ins on first use, so unused numeric/string sets impose no startup cost.

---

### Numeric min / max examples

```go
type Limits struct {
    Port    int     `validate:"min(1),max(65535)"`
    Retries int64   `validate:"min(0),max(10)"`
    Ratio   float64 `validate:"min(0.0),max(1.0)"`
}
```

- `Port` must be between 1 and 65535.
- `Retries` must be between 0 and 10.
- `Ratio` must be between 0.0 and 1.0 inclusive.

### String max and uuid examples

```go
type Account struct {
    ID    string `validate:"uuid"`
    Name  string `validate:"min(3),max(100)"`
    Email string `validate:"email"`
}
```

- `ID` must be a canonical UUID string (e.g. `123e4567-e89b-12d3-a456-426614174000`).
- `Name` must be between 3 and 100 characters.
- `Email` must satisfy a simple email heuristic (single `@`, etc.).

---

## Structured errors: errorc, sentinels, and ErrorField* keys

Under the hood, `model` uses [`github.com/ygrebnov/errorc`](https://github.com/ygrebnov/errorc) to build structured errors. The `errors` package defines sentinel errors and strongly-typed keys:

- Sentinels (examples):
  - `errors.ErrNilObject`
  - `errors.ErrNotStructPtr`
  - `errors.ErrRuleMissingParameter`
  - `errors.ErrRuleInvalidParameter`
  - `errors.ErrRuleConstraintViolated`
- Keys (examples):
  - `errors.ErrorFieldRuleName` (e.g. `model.rule.name`)
  - `errors.ErrorFieldRuleParamName` (e.g. `model.rule.param_name`)
  - `errors.ErrorFieldRuleParamValue` (e.g. `model.rule.param_value`)
  - `errors.ErrorFieldFieldName` (e.g. `model.field.name`)
  - `errors.ErrorFieldCause` (the underlying cause error)

Builtin rules attach metadata when they fail. For example, the string `min` rule:

```go
return errorc.With(
    errors.ErrRuleConstraintViolated,
    errorc.String(errors.ErrorFieldRuleName, "min"),
    errorc.String(errors.ErrorFieldRuleParamName, "length"),
    errorc.String(errors.ErrorFieldRuleParamValue, raw),
)
```

From your code, you can inspect these errors using `errors.Is` and by reading the message (which includes the structured key/value pairs), or by using `errors.As` into `*validation.Error` for field-level failures.

Example:

```go
m, err := model.New(&u, model.WithValidation[User](ctx))
if err != nil {
    var ve *model.ValidationError
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

If you need to work directly with the structured error metadata (e.g., to localize messages), you can call into `errorc` from your own code, or build small helpers around the keys exposed by `github.com/ygrebnov/model/errors`.

---

## Overriding a builtin rule

You can override a builtin rule by registering a custom rule with the same name and exact field type before validation runs. For example, to replace the builtin string `min` rule with a whitespace-aware version:

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "strings"

    "github.com/ygrebnov/model"
)

type Comment struct {
    Text string `validate:"min(3)"`
}

func main() {
    trimMin, err := model.NewRule[string]("min", func(s string, params ...string) error {
        // Treat leading/trailing whitespace as insignificant.
        s = strings.TrimSpace(s)
        if len(params) == 0 {
            return fmt.Errorf("min requires a length parameter")
        }
        // For brevity, we skip full structured errorc usage here;
        // in production, use sentinel errors + errorc.With, similar to builtin rules.
        n := len(params[0]) // pretend this is parsed
        if len(s) < n {
            return fmt.Errorf("must be at least %d characters after trimming", n)
        }
        return nil
    })
    if err != nil {
        panic(err)
    }

    c := Comment{Text: "  x "}

    m, err := model.New(&c,
        model.WithRules[Comment](trimMin),           // override builtin string min
        model.WithValidation[Comment](context.Background()),
    )
    if err != nil {
        var ve *model.ValidationError
        if errors.As(err, &ve) {
            fmt.Println("validation error:", ve)
        } else {
            fmt.Println("error:", err)
        }
        return
    }

    _ = m
    fmt.Println("comment is valid")
}
```

In this example, tag `validate:"min(3)"` for `Comment.Text` uses the overridden rule because it shares the same name and exact field type (`string`) as the builtin.
