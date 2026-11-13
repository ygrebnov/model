[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/model)](https://pkg.go.dev/github.com/ygrebnov/model)
[![Build Status](https://github.com/ygrebnov/model/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/model/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/model)](https://goreportcard.com/report/github.com/ygrebnov/model)

# model

`model` applies defaults, typed external values, environment values, and validation rules to Go structs. A reusable `Binding[T]` compiles the schema once and can process many values of the same type.

## Install

```bash
go get github.com/ygrebnov/model
```

Requires Go 1.23.12 or later.

## Quick start

```go
package main

import (
    "context"
    "fmt"

    "github.com/ygrebnov/model"
)

type Config struct {
    Name string `default:"api" validate:"min(1)"`
    Port int    `default:"8080" validate:"min(1),max(65535)"`
}

func main() {
    binding, err := model.NewBinding[Config]()
    if err != nil {
        panic(err)
    }

    cfg := Config{}
    if err := binding.ValidateWithDefaults(context.Background(), &cfg); err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", cfg)
}
```

`ValidateWithDefaults` applies literal defaults, then environment values captured when the binding was created, then validation. Use `ApplyDefaults`, `ApplyValues`, `ApplyEnv`, `Validate`, and `WriteValues` separately when your application needs a different flow. One-off wrappers are available for defaults and validation.

## Documentation

The [model guide](https://yaroslavgrebnov.com/projects/model/) has practical examples for each operation, custom rules, error handling, and hierarchical CLI configuration. See the [API reference](https://pkg.go.dev/github.com/ygrebnov/model) for exported types and methods.

## License

BSD 3-Clause. See [LICENSE](LICENSE).
