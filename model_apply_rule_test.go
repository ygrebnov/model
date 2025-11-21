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

// another exact rule (now only used to test duplicate rejection)
func ruleExactString2(v string, _ ...string) error {
	return fmt.Errorf("exact2:string:%s", v)
}

// an int rule (used in available-types list tests)
func ruleInt(v int, _ ...string) error {
	return fmt.Errorf("int:%d", v)
}

type dummy struct{}

// newTestBindingDummy constructs a typeBinding and its backing registry/mapping
// for a dummy type, used to test applyRule behavior via the registry/get logic.
func newTestBindingDummy(t *testing.T) (*typeBinding, rulesRegistry) {
	t.Helper()
	reg := newRulesRegistry()
	mapping := newRulesMapping()
	// typ is not used by applyRule itself, but required by typeBinding.
	tb, err := buildTypeBinding(reflect.TypeOf(dummy{}), reg, mapping)
	if err != nil {
		t.Fatalf("buildTypeBinding error: %v", err)
	}
	return tb, reg
}

func TestModel_applyRule(t *testing.T) {
	// NOTE: ambiguity case removed because registry now rejects duplicate exact overloads.

	// Unregistered rule -> ErrRuleNotFound
	t.Run("unregistered rule -> error", func(t *testing.T) {
		tb, _ := newTestBindingDummy(t)
		// no rules registered
		err := tb.applyRule("nope", reflect.ValueOf("x"))
		if err == nil || !strings.Contains(err.Error(), "rule not found") {
			t.Fatalf("expected unregistered-rule error, got: %v", err)
		}
	})

	t.Run("invalid reflect.Value -> error", func(t *testing.T) {
		// invalid reflect.Value is handled by registry.get; construct a registry and call directly.
		_, reg := newTestBindingDummy(t)
		var invalid reflect.Value // zero Value is invalid
		_, err := reg.get("r", invalid)
		if err == nil || !strings.Contains(err.Error(), "invalid value") {
			t.Fatalf("expected invalid-value error, got: %v", err)
		}
	})

	t.Run("exact match -> calls exact overload", func(t *testing.T) {
		tb, reg := newTestBindingDummy(t)
		pick, err := NewRule[string]("pick", ruleExactString)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		if err = reg.add(pick); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		err = tb.applyRule("pick", reflect.ValueOf("hi"))
		if err == nil || !strings.Contains(err.Error(), "exact:string:hi") {
			t.Fatalf("expected exact overload to run, got: %v", err)
		}
	})

	t.Run("exact preferred over assignable (interface) -> picks exact", func(t *testing.T) {
		tb, reg := newTestBindingDummy(t)
		interfaceOverload, err := NewRule[stringer]("pick", ruleIfaceStringer)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		exactOverload, err := NewRule[string]("pick", ruleExactString)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		if err = reg.add(interfaceOverload); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		if err = reg.add(exactOverload); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		err = tb.applyRule("pick", reflect.ValueOf("yo"))
		if err == nil || !strings.Contains(err.Error(), "exact:string:yo") {
			t.Fatalf("expected EXACT overload chosen, got: %v", err)
		}
	})

	t.Run("assignable (interface) match -> calls interface overload", func(t *testing.T) {
		tb, reg := newTestBindingDummy(t)
		iface, err := NewRule[stringer]("iface", ruleIfaceStringer)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		if err = reg.add(iface); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		v := sw{s: "wrapped"}
		err = tb.applyRule("iface", reflect.ValueOf(v))
		if err == nil || !strings.Contains(err.Error(), "assign:stringer:wrapped") {
			t.Fatalf("expected interface overload to run, got: %v", err)
		}
	})

	t.Run("duplicate exact overloads -> registration error", func(t *testing.T) {
		_, reg := newTestBindingDummy(t)
		exactString1, err := NewRule[string]("dup", ruleExactString)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		exactString2, err := NewRule[string]("dup", ruleExactString2)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		if err = reg.add(exactString1); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		err = reg.add(exactString2)
		if err == nil || !strings.Contains(err.Error(), "duplicate overload rule") {
			t.Fatalf("expected duplicate overload error, got: %v", err)
		}
	})

	t.Run("no matching overload -> error lists available types", func(t *testing.T) {
		tb, reg := newTestBindingDummy(t)
		stringOverload, err := NewRule[string]("r", ruleExactString)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		intOverload, err := NewRule[int]("r", ruleInt)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		if err = reg.add(stringOverload); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		if err = reg.add(intOverload); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		// Trigger no-overload with a float
		err = tb.applyRule("r", reflect.ValueOf(3.14))
		if err == nil || !strings.Contains(err.Error(), "rule overload not found") {
			t.Fatalf("expected no-overload rule-not-found error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "available_types: int, string") {
			t.Fatalf("expected available_types list, got: %v", err)
		}
	})

	t.Run("available types list is sorted deterministically", func(t *testing.T) {
		tb, reg := newTestBindingDummy(t)
		stringOverload, err := NewRule[string]("sorted", ruleExactString)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		intOverload, err := NewRule[int]("sorted", ruleInt)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		if err = reg.add(stringOverload); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		if err = reg.add(intOverload); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		err = tb.applyRule("sorted", reflect.ValueOf(1.23))
		if err == nil {
			t.Fatalf("expected error")
		}
		msg := err.Error()
		// Expect available list to be sorted as "int, string"
		if !strings.Contains(msg, "available_types: int, string") {
			t.Fatalf("expected sorted available list in error, got: %q", msg)
		}
	})
}
