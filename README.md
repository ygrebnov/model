[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/model)](https://pkg.go.dev/github.com/ygrebnov/model)
[![Build Status](https://github.com/ygrebnov/model/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/model/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/model)](https://goreportcard.com/report/github.com/ygrebnov/model)

# model — defaults & validation for Go structs

`model` is a tiny helper that binds a **Model** to your struct. It can:

- **Set defaults** from struct tags like `default:"…"` and `defaultElem:"…"`.
- **Validate** fields using named rules from `validate:"…"` and `validateElem:"…"`.
- Accumulate all issues into a single **ValidationError** (no fail-fast).
- Recurse through nested structs, pointers, slices/arrays, and map values.

It’s designed to be **small, explicit, and type-safe** (uses generics). You register rules (via `NewRule`) and `model` handles traversal, dispatch, and error reporting. Built‑in rules are always available implicitly (you don’t have to register them unless you want to override their behavior).

## Table of Contents
- [Install](#install)
- [Why use this?](#why-use-this)
- [Quick start](#quick-start)
- [Constructor: `New`](#constructor-new)
- [Why no MustNew?](#why-no-mustnew)
- [Functional options](#functional-options)
- [Model methods](#model-methods)
- [Struct tags (how it works)](#struct-tags-how-it-works)
- [Built-in rules](#built-in-rules)
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

- **Simple API**: one constructor and two main methods: `SetDefaults()` and `Validate(ctx)`.
- **Predictable behavior**: defaults fill *only zero values*; validation gathers *all* issues.
- **Extensible**: register your own rules; supports interface-based rules (e.g., rules for `fmt.Stringer`).

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
    City    string `default:"Paris"  validate:"nonempty"`
    Country string `default:"France" validate:"nonempty"`
}

type User struct {
    Name     string        `default:"Anonymous" validate:"nonempty"`
    Age      int           `default:"18"        validate:"positive,nonzero"`
    Timeout  time.Duration `default:"1s"`
    Home     Address       `default:"dive"`          // recurse into nested struct
    Aliases  []string      `validateElem:"nonempty"` // validate each element
    Profiles map[string]Address `default:"alloc" defaultElem:"dive"`
}

func main() {
    u := User{Aliases: []string{"", "ok"}} // index 0 will fail validation

    // Built-in rules (nonempty / numeric checks) are available implicitly.
    m, err := model.New(&u,
        model.WithDefaults[User](),                 // apply defaults during construction
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
    _ = m.SetDefaults()                          // guarded by sync.Once — no double work
    _ = m.Validate(context.Background())         // returns *ValidationError on failure
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
nonemptyCustom, _ := model.NewRule[string]("nonempty", func(s string, _ ...string) error {
    if strings.TrimSpace(s) == "" { return fmt.Errorf("must not be blank or whitespace") }
    return nil
})

m, err := model.New(&u,
    model.WithRules[User](nonemptyCustom), // override
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
- Empty tokens skipped (`,nonempty,` → `nonempty`).
- `validateElem:"dive"` recurses into struct elements; non-struct or nil pointer elements produce a misuse error under rule name `dive`.

---

## Built-in rules

Built-in rules are always implicitly available (you do **not** need to register or import anything for them):

- String: `nonempty`, `oneof(...)`
- Int / Int64 / Float64: `positive`, `nonzero`, `oneof(...)`

Overriding: if you register a custom rule with the same name and exact type **before** validation runs, your rule is chosen (duplicate exact registrations for the same name & type are rejected). Interface-based rules still participate via assignable matching when no exact rule exists.

> The library lazy-loads built-ins on first use, so unused numeric/string sets impose no startup cost.

---

## Custom rules (with parameters)

```go
minLen, _ := model.NewRule[string]("minLen", func(s string, params ...string) error {
    if len(params) < 1 { return fmt.Errorf("minLen requires 1 param") }
    n, err := strconv.Atoi(params[0])
    if err != nil { return fmt.Errorf("minLen: bad param: %w", err) }
    if len(s) < n { return fmt.Errorf("must be at least %d chars", n) }
    return nil
})

type Payload struct { Body string `validate:"minLen(3)"` }

p := Payload{Body: "xy"}
m, _ := model.New(&p, model.WithRules[Payload](minLen))
if err := m.Validate(context.Background()); err != nil {
    fmt.Println(err) // Body: must be at least 3 chars (rule minLen)
}
```

Interface rules:

```go
type stringer interface{ String() string }
stringerRule, _ := model.NewRule[stringer]("stringerBad", func(s stringer, _ ...string) error {
    if s.String() == "" { return fmt.Errorf("empty") }
    return fmt.Errorf("bad stringer: %s", s.String()) // demo
})

m, _ := model.New(&obj, model.WithRules[YourType](stringerRule))
```

---

## Error types

### FieldError

```go
fe := model.FieldError{Path: "User.Name", Rule: "nonempty", Err: fmt.Errorf("must not be empty")}
fmt.Println(fe.Error()) // "User.Name: must not be empty (rule nonempty)"
```

### ValidationError

Aggregates many `FieldError`s and provides helpers: `Len()`, `Fields()`, `ByField()`, `ForField(path)`, `Unwrap()`.

Errors for rules that cannot be dispatched include a deterministic, alphabetically sorted list of available overload types (see below).

### Deterministic ordering of available types

When a rule name has overloads but none match the field's type, the error includes a sorted list of available types for stable, testable output.

Example fragment: `available_types: int, string`

---

## Performance & benchmarks

A few micro-benchmarks (run on Go 1.22, macOS, Apple M-series; numbers are illustrative) show the cost profile:

| Benchmark | ns/op | B/op | allocs/op | Notes |
|-----------|-------|------|-----------|-------|
| BuiltinColdStart | ~3050 | ~1888 | 58 | First validation triggers lazy built-in rule initialization. |
| BuiltinWarm | ~720 | ~240 | 11 | Subsequent validations (same model, built-ins cached). |
| ValidateNoBuiltins | ~248 | ~120 | 5 | Custom rule only (no built-ins on the value path). |

Key points:
- The one-time lazy init cost (~3µs) is amortized quickly; warm validations drop below 1µs for this simple struct.
- Memory allocations drop significantly after warm-up (primarily slice/map allocations for rule lookups and error structures when no failures occur).
- You can reduce allocs further by reusing the same Model instance; `Validate(ctx)` does not mutate cached rule parsing state.

Run benchmarks locally:

```bash
go test -bench . -benchmem -run ^$
```

For real workloads, overall cost will scale with number of exported fields traversed and rules applied; rule functions themselves typically dominate if they perform parsing or heavy logic.

### Performance tuning tips

Practical ways to minimize overhead in hot paths:

1. Reuse Model instances: construct a single `Model[T]` per long‑lived object (or pool of objects) and call `Validate(ctx)` repeatedly. Tag parsing and rule registry lookups are cached; re-validation avoids re-parsing tags.
2. Separate construction from the hot loop: do `model.New(&obj, WithValidation[T](context.Background()))` outside tight request loops so the one‑time lazy built‑in initialization cost is amortized.
3. Prefer exact type rules over interface rules when possible: exact matches are resolved faster than walking assignable interface candidates.
4. Avoid unnecessary rule duplication: register each custom rule once up front. Re-registering (or creating rules every call) allocates and defeats caching.
5. Aggregate small validations: if you have many tiny structs, consider grouping related fields into a single struct to reduce reflective traversal overhead.
6. Keep rule bodies lean: do parameter parsing (e.g., strconv) once if values are static, or precompute maps/sets for frequent membership checks.
7. Zero‑alloc fast path: if you expect mostly valid data, write rule errors succinctly; fewer allocations happen when no `FieldError`s are produced.
8. Avoid per‑call defaults unless needed: `SetDefaults()` is guarded by `sync.Once`—calling it again is cheap, but skip it entirely in hot loops if defaults are already applied.
9. Profile before micro‑optimizing: use `go test -bench` or `pprof` to confirm hotspots (often custom rule logic, not the framework).

Minimal pattern for reuse inside a service handler:

```go
var userModel *model.Model[User]

func init() {
    u := User{} // template instance if you only validate (or allocate per request below)
    // Register overrides & built-ins once.
    userModel, _ = model.New(&u, model.WithValidation[User](context.Background()))
}

func handle(ctx context.Context, u *User) error {
    if err := userModel.Validate(ctx); err != nil { // reuses cached tag parsing
        return err
    }
    return nil
}
```

---

## Behavior notes

- `SetDefaults()` is idempotent per `Model` (via `sync.Once`).
- Creating a new `Model` on the same object re-applies defaults safely (only zero values set).
- `default:"dive"` allocates nil `*struct` pointers before recursing.
- Duplicate exact rule registrations are blocked early (no runtime ambiguity errors).
- Built-ins are always available even if you never call `WithRules`.
- Concurrency: After construction, calling `Validate(ctx)` and `SetDefaults()` on the same *Model* instance from multiple goroutines concurrently is safe for reads of cached metadata, but typical usage mutates the underlying struct. For concurrent validation of distinct objects, create one Model per object or guard shared object mutation. Register all custom rules before exposing the Model to multiple goroutines (rule registration is not concurrency-safe once validation begins).
- Cancellation: `Validate(ctx)` checks context at top-level and between field iterations, and returns `ctx.Err()` early when canceled or timed out. If you want cancellation when running validation during construction, pass `WithValidation(ctx)` to `New(...)`.

---

## Integration example: validation failure with sorted available types

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "github.com/ygrebnov/model"
)

type Ex struct { X float64 `validate:"r"` }

func main() {
    ex := Ex{X: 3.14}

    // Register only string and int overloads (float64 missing)
    rStr, _ := model.NewRule[string]("r", func(_ string, _ ...string) error { return nil })
    rInt, _ := model.NewRule[int]("r", func(_ int, _ ...string) error { return nil })

    m, err := model.New(&ex,
        model.WithRules[Ex](rStr, rInt),
        model.WithValidation[Ex](context.Background()),
    )
    if err != nil {
        var ve *model.ValidationError
        if errors.As(err, &ve) {
            fmt.Println(ve.Error())
            for field, fes := range ve.ByField() {
                for _, fe := range fes {
                    fmt.Printf("%s: %s\n", field, fe.Error())
                }
            }
        } else {
            fmt.Println("error:", err)
        }
        _ = m
        return
    }
}
```

Possible line (simplified):

```
X: model: rule overload not found, rule_name: r, value_type: float64, available_types: int, string (rule r)
```

---

## Missing rule vs missing overload

Two distinct error cases help diagnose configuration issues:

1. ErrRuleNotFound – no rule with that name exists (and no built-in with that name).
2. ErrRuleOverloadNotFound – at least one overload with that rule name exists, but none matches the field's type.

### 1. Missing rule name entirely
```go
package main
import (
  "context"
  "fmt"
  "errors"
  "github.com/ygrebnov/model"
)

type A struct { X int `validate:"unknownRule"` }

func main() {
  a := A{}
  m, _ := model.New(&a) // no rules registered
  if err := m.Validate(context.Background()); err != nil {
    var ve *model.ValidationError
    if errors.As(err, &ve) {
      fmt.Println("-- ErrRuleNotFound example --")
      fmt.Println(ve.Error())
    }
  }
}
```
Possible fragment (order of fields stable but other context may precede it):
```
X: model: rule not found, rule_name: unknownRule (rule unknownRule)
```

### 2. Rule name exists, but type overload missing
```go
package main
import (
  "context"
  "fmt"
  "errors"
  "github.com/ygrebnov/model"
)

type B struct { F float64 `validate:"r"` }

func main() {
  b := B{F: 1.23}
  rInt, _ := model.NewRule[int]("r", func(_ int, _ ...string) error { return nil })
  rString, _ := model.NewRule[string]("r", func(_ string, _ ...string) error { return nil })
  m, _ := model.New(&b, model.WithRules[B](rInt, rString))
  if err := m.Validate(context.Background()); err != nil {
    var ve *model.ValidationError
    if errors.As(err, &ve) {
      fmt.Println("-- ErrRuleOverloadNotFound example --")
      fmt.Println(ve.Error())
    }
  }
}
```
Possible fragment:
```
F: model: rule overload not found, rule_name: r, value_type: float64, available_types: int, string (rule r)
```

---

## Minimal example

```go
package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "time"
    "github.com/ygrebnov/model"
)

type Cfg struct {
    Name string        `default:"svc" validate:"nonempty"`
    Wait time.Duration `default:"500ms"`
}

func main() {
    cfg := Cfg{}
    m, err := model.New(&cfg,
        model.WithDefaults[Cfg](),
        model.WithValidation[Cfg](context.Background()), // built-ins supply nonempty automatically
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
    _ = m
    fmt.Printf("OK: %+v\n", cfg)
}
```

---

## Examples

The examples now live directly in the main package (no separate `examples` or `example` package). They are conventional Go example functions named `Example*` so they:

- Render automatically on pkg.go.dev.
- Run as part of `go test` (ensuring they compile and their output, if any, matches expectations).

### Running the examples

```bash
go test ./...                 # runs unit tests and example functions
go test -run Example ./...    # run only examples
```

Browse them:
- On pkg.go.dev: https://pkg.go.dev/github.com/ygrebnov/model
- In the repository root: search for `Example` functions inside `*.go` files.

If you had previously referenced an `examples/` directory in external docs, update that reference—examples are now inlined in the package itself for better discoverability.

---

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.