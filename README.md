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

---

## Install

```bash
go get github.com/ygrebnov/model
```

---

## Why use this?

- **Simple API**: one constructor and two main methods: `SetDefaults()` and `Validate()`.
- **Predictable behavior**: defaults fill *only zero values*; validation gathers *all* issues.
- **Extensible**: register your own rules; supports interface-based rules (e.g., rules for `fmt.Stringer`).

---

## Quick start

```go
package main

import (
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
        model.WithDefaults[User](),   // apply defaults during construction
        model.WithValidation[User](), // run validation during construction
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
    _ = m.SetDefaults() // guarded by sync.Once — no double work
    _ = m.Validate()    // returns *ValidationError on failure
}
```

---

## Constructor: `New`

```go
m, err := model.New(&user,
    model.WithDefaults[User](),    // apply defaults during New()
    model.WithValidation[User](),  // run validation during New()
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

**Notes**

- `New` returns `(*Model[T], error)`.
- Misuse (nil object or pointer to a non-struct) returns an error (`ErrNilObject`, `ErrNotStructPtr`).
- Built-in rules (string / int / int64 / float64 families) are **auto-available**; no registration required.
- Register custom or overriding rules (see below) *before* `WithValidation` if you want them to apply during construction.

---

## Functional options

All options run in the order provided. If an option returns an error (e.g., attempting to register a duplicate overload for the same type & name), `New` stops and returns that error.

### `WithDefaults[T]()` — apply defaults during construction

```go
m, err := model.New(&u, model.WithDefaults[User]())
```

- Runs once per `Model` (guarded by `sync.Once`).
- Writes only zero values.

### `WithValidation[T]()` — run validation during construction

```go
m, err := model.New(&u,
    model.WithValidation[User](),
)
```

- Gathers **all** field errors; returns a `*ValidationError` on failure.
- Built-ins are always considered first for matching types.
- To override a built-in rule, register a custom rule *before* `WithValidation`:

```go
nonemptyCustom, _ := model.NewRule[string]("nonempty", func(s string, _ ...string) error {
    if strings.TrimSpace(s) == "" { return fmt.Errorf("must not be blank or whitespace") }
    return nil
})

m, err := model.New(&u,
    model.WithRules[User](nonemptyCustom), // override
    model.WithValidation[User](),
)
```

### `WithRules[T](rules ...Rule)` — register one or many rules

Create rules with `NewRule` and pass them:

```go
maxLen, _ := model.NewRule[string]("maxLen", func(s string, params ...string) error {
    if len(params) < 1 { return fmt.Errorf("maxLen requires 1 param") }
    n, _ := strconv.Atoi(params[0])
    if len(s) > n { return fmt.Errorf("must be <= %d chars", n) }
    return nil
})

m, _ := model.New(&u,
    model.WithRules[User](maxLen),
)
```

Duplicate exact overloads (same rule name & identical field type) are **rejected at registration time** with `ErrDuplicateOverloadRule`. This prevents later runtime ambiguity.

---

## Model methods

### `SetDefaults() error`

Apply `default:"…"` / `defaultElem:"…"` recursively. Safe to call multiple times (subsequent calls no-op).

### `Validate() error`

Walk fields and apply rules from `validate:"…"` / `validateElem:"…"` tags. Returns `*ValidationError` on failure.

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

## Tag syntax and parsing

Rules are parsed by a lightweight tokenizer:

- Splits only on top-level commas (commas inside parentheses are preserved).
- Trims whitespace around rule names and parameters.
- Ignores empty tokens.
- Parameters are raw strings (no quoting/escaping support). If you need richer semantics, encode (e.g. JSON) and decode inside the rule.
- Nested parentheses are not semantically interpreted beyond balancing for splitting.

Example transformations:

| Tag | Parsed |
|-----|--------|
| `validate:",nonempty,"` | `nonempty` |
| `validate:"withParams(a, b , c),nonempty"` | `withParams(a,b,c)` & `nonempty` |
| `validate:"tokA((x,y)),tokB"` | `tokA((x,y))`, `tokB` |

---

## Built-in rules

Built-in rules are always implicitly available (you do **not** need to register them):

- String: `nonempty`, `oneof(...)`
- Int / Int64 / Float64: `positive`, `nonzero`, `oneof(...)`

You can still fetch helper slices if you want to explicitly re-register (e.g., for ordering tests or to copy patterns):

```go
model.BuiltinStringRules()
model.BuiltinIntRules()
model.BuiltinInt64Rules()
model.BuiltinFloat64Rules()
```

If you register a custom rule with the same name and exact type before validation, it **overrides** the built-in for that type (registry uses your exact match first). Duplicate *exact* registrations for the same name & type are rejected.

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
if err := m.Validate(); err != nil {
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

## Behavior notes

- `SetDefaults()` is idempotent per `Model` (via `sync.Once`).
- Creating a new `Model` on the same object re-applies defaults safely (only zero values set).
- `default:"dive"` allocates nil `*struct` pointers before recursing.
- Duplicate exact rule registrations are blocked early (no runtime ambiguity errors).
- Built-ins are always available even if you never call `WithRules`.

---

## Integration example: validation failure with sorted available types

```go
package main

import (
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
        model.WithValidation[Ex](),
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
X: model: rule not found, rule_name: r, value_type: float64, available_types: int, string (rule r)
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
        model.WithValidation[Cfg](), // built-ins supply nonempty automatically
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

An examples program lives under `examples/` demonstrating each option and a validation failure. Run:

```bash
go run ./examples
```


---

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.