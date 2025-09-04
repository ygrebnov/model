package model

import "fmt"

// Built-in rule sets (no-parameter rules for now)

// String rules
func BuiltinStringRules() []Rule[string] {
	return []Rule[string]{
		{Name: "nonempty", Fn: func(s string, _ ...string) error {
			if s == "" {
				return fmt.Errorf("must not be empty")
			}
			return nil
		}},
	}
}

// Integer rules (int)
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
	}
}

// Integer rules (int64)
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
	}
}

// Float rules (float64)
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
	}
}
