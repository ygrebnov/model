package model

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Built-in rule sets

// key consists of a name and a field value type.
type key struct {
	name      string
	fieldType reflect.Type
}

var (
	stringType  = reflect.TypeOf("")
	intType     = reflect.TypeOf(int(0))
	int64Type   = reflect.TypeOf(int64(0))
	float64Type = reflect.TypeOf(float64(0))
)

// builtInRules holds the built-in rules mapped by (name, fieldType).
var builtInRules = map[key]Rule{ // TODO: use lazy initialization
	// string rules
	{"nonempty", stringType}: BuiltinStringRules()[0],
	{"oneof", stringType}:    BuiltinStringRules()[1],
	// int rules
	{"positive", intType}: BuiltinIntRules()[0],
	{"nonzero", intType}:  BuiltinIntRules()[1],
	{"oneof", intType}:    BuiltinIntRules()[2],
	// int64 rules
	{"positive", int64Type}: BuiltinInt64Rules()[0],
	{"nonzero", int64Type}:  BuiltinInt64Rules()[1],
	{"oneof", int64Type}:    BuiltinInt64Rules()[2],
	// float64 rules
	{"positive", float64Type}: BuiltinFloat64Rules()[0],
	{"nonzero", float64Type}:  BuiltinFloat64Rules()[1],
	{"oneof", float64Type}:    BuiltinFloat64Rules()[2],
}

func BuiltinStringRules() []Rule {
	nonempty, _ := NewRule("nonempty", func(s string, _ ...string) error {
		if s == "" {
			return fmt.Errorf("must not be empty")
		}
		return nil
	})
	oneof, _ := NewRule("oneof", func(s string, params ...string) error {
		if len(params) == 0 {
			return fmt.Errorf(`oneof requires at least one parameter, e.g. validate:"oneof(red,green,blue)"`)
		}
		for _, p := range params {
			if s == p {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(params, ", "))
	})

	return []Rule{nonempty, oneof}
}

func BuiltinIntRules() []Rule {
	positive, _ := NewRule("positive", func(n int, _ ...string) error {
		if n <= 0 {
			return fmt.Errorf("must be > 0")
		}
		return nil
	})
	nonzero, _ := NewRule("nonzero", func(n int, _ ...string) error {
		if n == 0 {
			return fmt.Errorf("must not be zero")
		}
		return nil
	})
	oneof, _ := NewRule("oneof", func(n int, params ...string) error {
		if len(params) == 0 {
			return fmt.Errorf(`oneof requires at least one parameter, e.g. validate:"oneof(1,2,3)"`)
		}
		for _, p := range params {
			v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 0)
			if err != nil {
				return fmt.Errorf("invalid oneof parameter %q for int: %v", p, err)
			}
			if int(v) == n {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(params, ", "))
	})

	return []Rule{positive, nonzero, oneof}
}

func BuiltinInt64Rules() []Rule {
	positive, _ := NewRule("positive", func(n int64, _ ...string) error {
		if n <= 0 {
			return fmt.Errorf("must be > 0")
		}
		return nil
	})
	nonzero, _ := NewRule("nonzero", func(n int64, _ ...string) error {
		if n == 0 {
			return fmt.Errorf("must not be zero")
		}
		return nil
	})
	oneof, _ := NewRule("oneof", func(n int64, params ...string) error {
		if len(params) == 0 {
			return fmt.Errorf(`oneof requires at least one parameter, e.g. validate:"oneof(10,20,30)"`)
		}
		for _, p := range params {
			v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid oneof parameter %q for int64: %v", p, err)
			}
			if v == n {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(params, ", "))
	})

	return []Rule{positive, nonzero, oneof}
}

func BuiltinFloat64Rules() []Rule {
	positive, _ := NewRule("positive", func(n float64, _ ...string) error {
		if !(n > 0) {
			return fmt.Errorf("must be > 0")
		}
		return nil
	})
	nonzero, _ := NewRule("nonzero", func(n float64, _ ...string) error {
		if n == 0 {
			return fmt.Errorf("must not be zero")
		}
		return nil
	})
	oneof, _ := NewRule("oneof", func(n float64, params ...string) error {
		if len(params) == 0 {
			return fmt.Errorf(`oneof requires at least one parameter, e.g. validate:"oneof(1.5,2.0)"`)
		}
		for _, p := range params {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return fmt.Errorf("invalid oneof parameter %q for float64: %v", p, err)
			}
			if v == n {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(params, ", "))
	})

	return []Rule{positive, nonzero, oneof}
}
