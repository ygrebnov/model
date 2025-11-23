package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Built-ins are always implicitly available.

// key consists of a name and a field value type.
type key struct {
	name      string
	fieldType reflect.Type
}

// Lazy built-in rule storage.
var (
	builtInsOnce        sync.Once
	builtInMap          map[key]Rule
	builtinStringRules  []Rule
	builtinIntRules     []Rule
	builtinInt64Rules   []Rule
	builtinFloat64Rules []Rule
)

// ensureBuiltIns initializes built-in rules exactly once.
func ensureBuiltIns() {
	builtInsOnce.Do(func() {
		builtInMap = make(map[key]Rule)

		// string rules
		// min(length): requires one integer parameter. If missing -> error. If <1 -> noop.
		minStr, _ := NewRule[string]("min", func(s string, params ...string) error {
			if len(params) == 0 {
				return fmt.Errorf(`min requires a length parameter, e.g. validate:"min(1)"`)
			}
			v, err := strconv.ParseInt(strings.TrimSpace(params[0]), 10, 0)
			if err != nil {
				return fmt.Errorf("invalid min length parameter %q: %v", params[0], err)
			}
			if v < 1 { // noop as requested
				return nil
			}
			if int(v) > len(s) { // length too small
				return fmt.Errorf("length must be >= %d (got %d)", v, len(s))
			}
			return nil
		})
		// email rule: deliberately simple; not RFC 5322 exhaustive. Provides lightweight validation.
		emailStr, _ := NewRule[string]("email", func(s string, _ ...string) error {
			if s == "" { // treat empty as error, keeping semantics similar to prior nonempty
				return fmt.Errorf("must be a valid email address")
			}
			if strings.Count(s, "@") != 1 {
				return fmt.Errorf("must contain exactly one @")
			}
			parts := strings.Split(s, "@")
			local, domain := parts[0], parts[1]
			if local == "" || domain == "" {
				return fmt.Errorf("local and domain parts must be non-empty")
			}
			if strings.ContainsAny(s, " \t\n\r") {
				return fmt.Errorf("must not contain whitespace")
			}
			if !strings.Contains(domain, ".") { // simple domain heuristic
				return fmt.Errorf("domain must contain a dot")
			}
			return nil
		})
		oneofStr, _ := NewRule[string]("oneof", func(s string, params ...string) error {
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
		builtinStringRules = []Rule{emailStr, minStr, oneofStr}

		// int rules
		positiveInt, _ := NewRule[int]("positive", func(n int, _ ...string) error {
			if n <= 0 {
				return fmt.Errorf("must be > 0")
			}
			return nil
		})
		nonzeroInt, _ := NewRule[int]("nonzero", func(n int, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		})
		oneofInt, _ := NewRule[int]("oneof", func(n int, params ...string) error {
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
		builtinIntRules = []Rule{positiveInt, nonzeroInt, oneofInt}

		// int64 rules
		positiveInt64, _ := NewRule[int64]("positive", func(n int64, _ ...string) error {
			if n <= 0 {
				return fmt.Errorf("must be > 0")
			}
			return nil
		})
		nonzeroInt64, _ := NewRule[int64]("nonzero", func(n int64, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		})
		oneofInt64, _ := NewRule[int64]("oneof", func(n int64, params ...string) error {
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
		builtinInt64Rules = []Rule{positiveInt64, nonzeroInt64, oneofInt64}

		// float64 rules
		positiveFloat64, _ := NewRule[float64]("positive", func(n float64, _ ...string) error {
			if !(n > 0) {
				return fmt.Errorf("must be > 0")
			}
			return nil
		})
		nonzeroFloat64, _ := NewRule[float64]("nonzero", func(n float64, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		})
		oneofFloat64, _ := NewRule[float64]("oneof", func(n float64, params ...string) error {
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
		builtinFloat64Rules = []Rule{positiveFloat64, nonzeroFloat64, oneofFloat64}

		// fill map
		register := func(rs []Rule) {
			for _, r := range rs {
				builtInMap[key{r.GetName(), r.GetFieldType()}] = r
			}
		}
		register(builtinStringRules)
		register(builtinIntRules)
		register(builtinInt64Rules)
		register(builtinFloat64Rules)
	})
}

// lookupBuiltin returns a built-in rule by (Name,type) if present.
func lookupBuiltin(name string, t reflect.Type) (Rule, bool) {
	ensureBuiltIns()
	r, ok := builtInMap[key{name, t}]
	return r, ok
}
