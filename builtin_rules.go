package model

import (
	"fmt"
	"strconv"
	"strings"
)

// Built-in rule sets

func BuiltinStringRules() []Rule[string] {
	return []Rule[string]{
		{Name: "nonempty", Fn: func(s string, _ ...string) error {
			if s == "" {
				return fmt.Errorf("must not be empty")
			}
			return nil
		}},
		{Name: "oneof", Fn: func(s string, params ...string) error {
			if len(params) == 0 {
				return fmt.Errorf(`oneof requires at least one parameter, e.g. validate:"oneof(red,green,blue)"`)
			}
			for _, p := range params {
				if s == p {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", strings.Join(params, ", "))
		}},
	}
}

func BuiltinIntRules() []Rule[int] {
	return []Rule[int]{
		{Name: "positive", Fn: func(n int, _ ...string) error {
			if n <= 0 {
				return fmt.Errorf("must be > 0")
			}
			return nil
		}},
		{Name: "nonzero", Fn: func(n int, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		}},
		{Name: "oneof", Fn: func(n int, params ...string) error {
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
		}},
	}
}

func BuiltinInt64Rules() []Rule[int64] {
	return []Rule[int64]{
		{Name: "positive", Fn: func(n int64, _ ...string) error {
			if n <= 0 {
				return fmt.Errorf("must be > 0")
			}
			return nil
		}},
		{Name: "nonzero", Fn: func(n int64, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		}},
		{Name: "oneof", Fn: func(n int64, params ...string) error {
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
		}},
	}
}

func BuiltinFloat64Rules() []Rule[float64] {
	return []Rule[float64]{
		{Name: "positive", Fn: func(n float64, _ ...string) error {
			if !(n > 0) {
				return fmt.Errorf("must be > 0")
			}
			return nil
		}},
		{Name: "nonzero", Fn: func(n float64, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		}},
		{Name: "oneof", Fn: func(n float64, params ...string) error {
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
		}},
	}
}
