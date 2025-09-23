package model

import (
	"fmt"
	"reflect"
	"strings"
)

// RuleFunc is the signature for a validation function for a specific type.
type RuleFunc[T any] func(value T, params ...string) error

// Rule defines a named validation rule for a specific type.
type Rule[T any] struct {
	Name string
	Fn   RuleFunc[T]
}

// typedAdapter is an internal struct to hold a type-erased validation function
// along with the type it applies to.
type typedAdapter struct {
	fieldType reflect.Type
	fn        func(v reflect.Value, params ...string) error
}

// wrapRule takes a typed RuleFunc and returns a type-erased adapter.
// The adapter's func panics if the reflect.Value is not assignable to the rule's type.
func wrapRule[T any](fn RuleFunc[T]) typedAdapter {
	// Capture the static type of T even when T is an interface.
	typ := reflect.TypeOf((*T)(nil)).Elem()

	return typedAdapter{
		fieldType: typ,
		fn: func(v reflect.Value, params ...string) error {
			// Ensure the reflect.Value `v` is compatible with T.
			if v.Type() != typ {
				// Accept assignable values (including types implementing an interface T)
				if !v.Type().AssignableTo(typ) {
					// As a fallback for interface T, use Implements for clarity.
					if !(typ.Kind() == reflect.Interface && v.Type().Implements(typ)) {
						panic(fmt.Sprintf(
							"model: rule type mismatch: cannot use %s value with rule for type %s",
							v.Type(),
							typ,
						))
					}
				}
			}
			val := v.Interface().(T)
			return fn(val, params...)
		},
	}
}

// parsedRule holds the name and parameters of a single validation rule.
type parsedRule struct {
	name   string
	params []string
}

// parseRules takes a raw tag string (e.g., "required,min(5),max(10)") and splits it
// into a slice of parsedRule structs. It correctly handles parentheses and quoted parameters.
func parseRules(tag string) []parsedRule {
	var rules []parsedRule
	if tag == "" || tag == "-" {
		return rules
	}

	var tokens []string
	depth := 0
	start := 0
	for i, r := range tag {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				tokens = append(tokens, strings.TrimSpace(tag[start:i]))
				start = i + 1
			}
		}
	}
	// Append the last token
	if start <= len(tag) {
		tokens = append(tokens, strings.TrimSpace(tag[start:]))
	}

	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		name := tok
		var params []string
		if idx := strings.IndexRune(tok, '('); idx != -1 && strings.HasSuffix(tok, ")") {
			name = strings.TrimSpace(tok[:idx])
			inner := strings.TrimSpace(tok[idx+1 : len(tok)-1])
			if inner != "" {
				parts := strings.Split(inner, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						params = append(params, p)
					}
				}
			}
		}
		if name != "" {
			rules = append(rules, parsedRule{name: name, params: params})
		}
	}
	return rules
}
