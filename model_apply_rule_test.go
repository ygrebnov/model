package model

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// a concrete type implementing fmt.Stringer-like behavior (we won't import fmt here)
type sw struct{ s string }

func (w sw) String() string { return w.s }

// --- helpers ---

// exact rule that returns a tagged error so we can see which overload ran
func ruleExactString(v string, _ ...string) error {
	return fmt.Errorf("exact:string:%s", v)
}

// interface/assignable rule that returns a tagged error
type stringer interface{ String() string }

func ruleIfaceStringer(v stringer, _ ...string) error {
	return fmt.Errorf("assign:stringer:%s", v.String())
}

// another exact rule for ambiguity checks
func ruleExactString2(v string, _ ...string) error {
	return fmt.Errorf("exact2:string:%s", v)
}

// an int rule (used in available-types list tests)
func ruleInt(v int, _ ...string) error {
	return fmt.Errorf("int:%d", v)
}

type dummy struct{}

func TestModel_applyRule(t *testing.T) {
	t.Parallel()

	t.Run("unregistered rule -> error", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		err := m.applyRule("nope", reflect.ValueOf("x"))
		if err == nil || !strings.Contains(err.Error(), `rule "nope" is not registered`) {
			t.Fatalf("expected unregistered-rule error, got: %v", err)
		}
	})

	t.Run("invalid reflect.Value -> error", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		// Register something so we don't hit the 'not registered' branch
		WithRule[dummy, string](Rule[string]{Name: "r", Fn: ruleExactString})(m)

		var invalid reflect.Value // zero Value is invalid
		err := m.applyRule("r", invalid)
		if err == nil || !strings.Contains(err.Error(), `invalid value`) {
			t.Fatalf("expected invalid-value error, got: %v", err)
		}
	})

	t.Run("exact match -> calls exact overload", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		WithRule[dummy, string](Rule[string]{Name: "pick", Fn: ruleExactString})(m)
		err := m.applyRule("pick", reflect.ValueOf("hi"))
		if err == nil || !strings.Contains(err.Error(), "exact:string:hi") {
			t.Fatalf("expected exact overload to run, got: %v", err)
		}
	})

	t.Run("exact preferred over assignable (interface) -> picks exact", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		// Register interface overload
		WithRule[dummy, stringer](Rule[stringer]{Name: "pick", Fn: ruleIfaceStringer})(m)
		// Register exact overload for string
		WithRule[dummy, string](Rule[string]{Name: "pick", Fn: ruleExactString})(m)

		err := m.applyRule("pick", reflect.ValueOf("yo"))
		if err == nil || !strings.Contains(err.Error(), "exact:string:yo") {
			t.Fatalf("expected EXACT overload chosen, got: %v", err)
		}
	})

	t.Run("assignable (interface) match -> calls interface overload", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		WithRule[dummy, stringer](Rule[stringer]{Name: "iface", Fn: ruleIfaceStringer})(m)

		v := sw{s: "wrapped"}
		err := m.applyRule("iface", reflect.ValueOf(v))
		if err == nil || !strings.Contains(err.Error(), "assign:stringer:wrapped") {
			t.Fatalf("expected interface overload to run, got: %v", err)
		}
	})

	t.Run("multiple exact overloads -> ambiguous error", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		WithRule[dummy, string](Rule[string]{Name: "dup", Fn: ruleExactString})(m)
		WithRule[dummy, string](Rule[string]{Name: "dup", Fn: ruleExactString2})(m)

		err := m.applyRule("dup", reflect.ValueOf("x"))
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("expected ambiguity error, got: %v", err)
		}
	})

	t.Run("no matching overload -> error lists available types", func(t *testing.T) {
		m := &Model[dummy]{validators: make(map[string][]typedAdapter)}
		// Register overloads for string and int, but we'll pass a float
		WithRule[dummy, string](Rule[string]{Name: "r", Fn: ruleExactString})(m)
		WithRule[dummy, int](Rule[int]{Name: "r", Fn: ruleInt})(m)

		err := m.applyRule("r", reflect.ValueOf(3.14))
		if err == nil || !strings.Contains(err.Error(), "has no overload for type float64") {
			t.Fatalf("expected no-overload error, got: %v", err)
		}
		// Should include available types list
		if !strings.Contains(err.Error(), "string") || !strings.Contains(err.Error(), "int") {
			t.Fatalf("expected available types in message, got: %v", err)
		}
	})
}
