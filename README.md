[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/model)](https://pkg.go.dev/github.com/ygrebnov/model)
[![Build Status](https://github.com/ygrebnov/model/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/model/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/model)](https://goreportcard.com/report/github.com/ygrebnov/model)

# model — defaults & validation for Go structs

`model` is a tiny helper that binds a **Model** to your struct. It can:

- **Set defaults** from struct tags like `default:"…"` and `defaultElem:"…"`.
- **Validate** fields using named rules from `validate:"…"` and `validateElem:"…"`.
- Accumulate all issues into a single **ValidationError** (no fail-fast).
- Recurse through nested structs, pointers, slices/arrays, and map values.

It’s designed to be **small, explicit, and type-safe** (uses generics). You register rules with friendly helpers and `model` handles traversal, dispatch, and error reporting.

---

## Install

```bash
go get github.com/ygrebnov/model
```

---

## Why use this?

- **Simple API**: one constructor and two main methods: `SetDefaults()` and `Validate()`.
- **Predictable behavior**: defaults fill *only zero values*; validation gathers *all* issues.
- **Extensible**: register your own rules; support interface-based rules (e.g., `fmt.Stringer`).

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
    Home     Address       `default:"dive"`            // recurse into nested struct
    Aliases  []string      `validateElem:"nonempty"`   // validate each element
    Profiles map[string]Address `default:"alloc" defaultElem:"dive"`
}

func main() {
    u := User{Aliases: []string{"", "ok"}} // Tags will flag index 0 as empty

    m, err := model.New(
        &u,
        model.WithRules[User, string](model.BuiltinStringRules()),
        model.WithRules[User, int](model.BuiltinIntRules()),
        model.WithDefaults[User](),   // apply defaults in constructor
        model.WithValidation[User](), // validate in constructor
    )
    if err != nil {
        // When validation fails, err is *model.ValidationError
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

    // You can also call them manually later:
    _ = m.SetDefaults()     // guarded by sync.Once — runs only once per Model
    _ = m.Validate()        // returns *ValidationError on failure
}
```

---

## Constructor: `New`

```go
m, err := model.New(&user,
    // You can mix and match options:
    model.WithDefaults[User](),            // apply defaults during New()
    model.WithValidation[User](),          // run Validate() during New()
    model.WithRule[User, string](model.Rule[string]{ // register a single rule
        Name: "nonempty",
        Fn: func(s string, _ ...string) error {
            if s == "" { return fmt.Errorf("must not be empty") }
            return nil
        },
    }),
    model.WithRules[User, int](model.BuiltinIntRules()), // register a batch of rules
)
if err != nil {
    // If WithValidation is used, err can be *model.ValidationError.
}
```

**Notes**

- `New` returns `(*Model[T], error)`.
- Misuse (nil object or pointer to a non-struct) **panics** to enforce invariants.
- Errors from `WithDefaults` / `WithValidation` are **returned**.

---

## Functional options

### `WithDefaults[T]()` — apply defaults during construction

```go
m, err := model.New(&u, model.WithDefaults[User]())
```

- Runs once per `Model` (guarded by `sync.Once`).
- Fills only zero values; non-zero values are left intact.

### `WithValidation[T]()` — run validation during construction

```go
m, err := model.New(&u,
    model.WithRules[User, string](model.BuiltinStringRules()),
    model.WithValidation[User](),
)
```

- Make sure the needed rules are registered before validation.
- Returns a `*ValidationError` on failure.
- Built-in rules are registered implicitly: for `string` (nonempty, oneof), `int` (positive, nonzero, oneof), `int64` (positive, nonzero, oneof), and `float64` (positive, nonzero, oneof). You no longer need to call `WithRules` for these built-ins.
- Option order matters for overrides:
  - To override a built-in rule (e.g., a custom `nonempty` for `string`), register your rule with `WithRule` BEFORE `WithValidation`.
  - If you register AFTER `WithValidation`, there will be two exact overloads and validation will produce an "ambiguous" error for that rule/type.

### `WithRule[TObject, TField](Rule[TField])` — register a single rule

```go
m, _ := model.New(&u,
    model.WithRule[User, string](model.Rule[string]{
        Name: "nonempty",
        Fn: func(s string, _ ...string) error {
            if s == "" { return fmt.Errorf("must not be empty") }
            return nil
        },
    }),
)

// Interface rule example (AssignableTo):
type stringer interface{ String() string }
model.WithRule[User, stringer](model.Rule[stringer]{
    Name: "stringerBad",
    Fn: func(s stringer, _ ...string) error {
        return fmt.Errorf("bad stringer: %s", s.String())
    },
})(m)
```

> **Note**: WithValidation now registers built-in rules implicitly. `WithRule` and `WithRules` are still useful to add your own rules, to register rules for additional types or interfaces, or to intentionally override built-ins (place them BEFORE `WithValidation`).

### `WithRules[TObject, TField]([]Rule[TField])` — register many at once

```go
m, _ := model.New(&u,
    model.WithRules[User, string](model.BuiltinStringRules()),
    model.WithRules[User, float64](model.BuiltinFloat64Rules()),
)
```

---

## Model methods

### `SetDefaults() error`

Apply `default:"…"` / `defaultElem:"…"` recursively. Guarded by `sync.Once`.

```go
if err := m.SetDefaults(); err != nil {
    // e.g., a bad literal like default:"oops" on a struct field
    log.Println("defaults error:", err)
}
```

### `Validate() error`

Walk fields and apply rules from `validate:"…"` / `validateElem:"…"`.

```go
if err := m.Validate(); err != nil {
    var ve *model.ValidationError
    if errors.As(err, &ve) {
        for field, issues := range ve.ByField() {
            for _, fe := range issues {
                fmt.Printf("%s: %s\n", field, fe.Err)
            }
        }
    }
}
```

---

## Struct tags (how it works)

### Defaults: `default:"…"` and `defaultElem:"…"`

- **Literals**: strings, bools, ints/uints, floats, `time.Duration` (e.g., `1h30m`).
- **`dive`**: recurse into a struct or `*struct` field and set its defaults.
- **`alloc`**: allocate an empty `slice`/`map` when `nil`.
- **`defaultElem:"dive"`**: recurse into struct **elements** (slice/array) or **map values**.

```go
type Config struct {
    Addr    string        `default:"0.0.0.0"`
    Port    int           `default:"8080"`
    Backoff time.Duration `default:"250ms"`

    Limit *int `default:"5"` // pointer-to-scalar allocated & set if nil

    TLS struct {
        Enabled bool   `default:"true"`
        CAFile  string `default:"/etc/ssl/ca.pem"`
    } `default:"dive"`

    Labels  map[string]string `default:"alloc"`
    Servers []Server          `defaultElem:"dive"`
    Peers   map[string]Peer   `default:"alloc" defaultElem:"dive"`
}
```

> Defaults write only zero values. Non-zero values are preserved.

### Validation: `validate:"…"` and `validateElem:"…"`

- Multiple rules: `validate:"nonempty,min(3),max(10)"`.
- Params are strings: `rule(p1,p2,…)` — parse them inside your rule.
- **`validateElem`** applies to each element (slice/array) or value (map).
- Special rule name **`dive`**: recurse into element structs. If an element is not a struct (or is a nil pointer), a **misuse** error is recorded under rule `"dive"`.

```go
type Input struct {
    Name   string        `validate:"nonempty"`
    Delay  time.Duration `validate:"nonzeroDur"`
    Tags   []string      `validateElem:"nonempty"`
    Nodes  []Node        `validateElem:"dive"`
    ByName map[string]Node `validateElem:"dive"`
}
```

---

## Tag syntax and parsing

Tag values are simple, human-friendly strings parsed with a lightweight tokenizer:

- Multiple rules are separated by commas at the top level, e.g. `validate:"nonempty,min(3),max(10)"`.
- Parentheses group parameters: `rule(p1,p2,...)`. Commas inside parentheses do not split top-level tokens.
- Whitespace around rule names and parameters is trimmed.
- Empty tokens from leading/trailing commas are ignored: `",nonempty,"` -> only `nonempty`.
- Parameters are split by commas without special handling for quotes or escaping. This means quoted strings and commas inside quotes are not supported.
- Nested parentheses inside parameters are not parsed specially; they will be included in parameter tokens as-is.

Examples:

- `validate:"foo( a , b ),bar"` -> rules: `foo` with params `["a","b"]`, and `bar`.
- `validate:"tokA((x,y)),tokB"` -> rules: `tokA` with params `["(x","y)"]`, and `tokB`.
- `validate:",nonempty,"` -> rules: only `nonempty`.

If you need richer parameter parsing (quotes, escaping, nested structures), consider encoding parameters (e.g., JSON) and decoding them inside your rule.

---

## Built-in rules

Quick starts for common checks:

```go
model.BuiltinStringRules()  // nonempty
model.BuiltinIntRules()     // positive, nonzero
model.BuiltinInt64Rules()   // positive, nonzero
model.BuiltinFloat64Rules() // positive, nonzero

m, _ := model.New(&u,
    model.WithRules[User, string](model.BuiltinStringRules()),
    model.WithRules[User, int](model.BuiltinIntRules()),
)
```

---

## Custom rules (with parameters)

```go
// e.g., validate:"minLen(3)"
func minLenRule(s string, params ...string) error {
    if len(params) < 1 { return fmt.Errorf("minLen requires 1 param") }
    n, err := strconv.Atoi(params[0])
    if err != nil { return fmt.Errorf("minLen: bad param: %w", err) }
    if len(s) < n { return fmt.Errorf("must be at least %d chars", n) }
    return nil
}

type Payload struct { Body string `validate:"minLen(3)"` }

p := Payload{Body: "xy"}
m, _ := model.New(&p,
    model.WithRule[Payload, string](model.Rule[string]{Name: "minLen", Fn: minLenRule}),
)
if err := m.Validate(); err != nil {
    fmt.Println(err) // "Body: must be at least 3 chars (rule minLen)"
}
```

Interface rules are supported too:

```go
type stringer interface{ String() string }
model.WithRule[YourType, stringer](model.Rule[stringer]{
    Name: "stringerOk",
    Fn: func(s stringer, _ ...string) error {
        if s.String() == "" { return fmt.Errorf("empty") }
        return nil
    },
})(m)
```

---

## Error types

### FieldError

Represents a single failure.

```go
fe := model.FieldError{Path: "User.Name", Rule: "nonempty", Err: fmt.Errorf("must not be empty")}
fmt.Println(fe.Error()) // "User.Name: must not be empty (rule nonempty)"

b, _ := fe.MarshalJSON() // {"path":"User.Name","rule":"nonempty","message":"must not be empty"}
```

### ValidationError

Accumulates many `FieldError`s.

```go
var ve *model.ValidationError
if errors.As(err, &ve) {
    fmt.Println(ve.Len(), "issues")
    fmt.Println(ve.Fields())   // ["Name", "Tags[0]", …]
    fmt.Println(ve.ForField("Name"))
    fmt.Println(ve.ByField())  // map[string][]FieldError
    fmt.Println(ve.Unwrap())   // errors.Join of underlying causes

    b, _ := json.MarshalIndent(ve, "", "  ")
    fmt.Println(string(b))
}
```

### Deterministic ordering of available types in error messages

When a field has a `validate:"ruleName"` but no matching overload is registered for its type, the error includes a list of available overload types, sorted alphabetically for deterministic output. This makes messages stable across runs and easier to test.

Example fragment: `(available: int, string)`

---

## Behavior notes

- `SetDefaults()` is idempotent per `Model` (guarded by `sync.Once`).
- Creating a new `Model` for the same object pointer can apply defaults again — safe because only zero values are filled.
- `default:"dive"` auto-allocates `*struct` pointers when nil. For collections, use `default:"alloc"` to allocate.
- `validateElem:"dive"` recurses into struct elements and records a **misuse** error for non-struct or nil pointer elements/values.

---

## Integration example: validation failure with sorted available types

The following example triggers a validation error because we register `"r"` for `string` and `int`, but the field is `float64`. Note how the `(available: ...)` list is sorted.

```go
package main

import (
    "errors"
    "fmt"

    "github.com/ygrebnov/model"
)

type Ex struct {
    // float64 has validate:"r" but we only register string and int overloads
    X float64 `validate:"r"`
}

func main() {
    ex := Ex{X: 3.14}

    // Dummy rules that would pass if types matched; they won't be used here
    rStr := model.Rule[string]{Name: "r", Fn: func(_ string, _ ...string) error { return nil }}
    rInt := model.Rule[int]{Name: "r", Fn: func(_ int, _ ...string) error { return nil }}

    m, err := model.New(&ex,
        model.WithRule[Ex, string](rStr),
        model.WithRule[Ex, int](rInt),
        model.WithValidation[Ex](), // will fail: no overload for float64
    )
    if err != nil {
        var ve *model.ValidationError
        if errors.As(err, &ve) {
            // Print the human-friendly aggregated error
            fmt.Println(ve.Error())
            // Or iterate fields:
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

Possible output (formatted for readability):

```
Ex.X: model: rule "r" has no overload for type float64 (available: int, string) (rule r)
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
        model.WithRules[Cfg, string](model.BuiltinStringRules()),
        model.WithDefaults[Cfg](),
        model.WithValidation[Cfg](),
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

    _ = m // model bound to cfg
    fmt.Printf("OK: %+v\n", cfg)
}
```

---

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.