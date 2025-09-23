package model

import (
	"reflect"
	"testing"
)

// Tests for ensureRule behavior and compatibility with WithValidation.

func TestEnsureRule_InvalidRule_Errors(t *testing.T) {
	m := &Model[struct{}]{validators: make(map[string][]typedAdapter)}

	// Empty name
	if err := ensureRule[struct{}, string](m, Rule[string]{Name: "", Fn: func(string, ...string) error { return nil }}); err == nil {
		t.Fatalf("expected error for empty rule name")
	}
	// Nil function
	if err := ensureRule[struct{}, string](m, Rule[string]{Name: "x", Fn: nil}); err == nil {
		t.Fatalf("expected error for nil rule function")
	}
}

func TestWithValidation_BuiltinsRemainValid_NoError(t *testing.T) {
	type Obj struct{ S string }
	obj := Obj{}
	if _, err := New(&obj, WithValidation[Obj]()); err != nil {
		t.Fatalf("WithValidation should not error for valid builtins, got: %v", err)
	}
}

func TestEnsureRule_ValidRule_AppendsAdapter(t *testing.T) {
	m := &Model[struct{}]{validators: make(map[string][]typedAdapter)}
	r := Rule[string]{
		Name: "customX",
		Fn:   func(s string, _ ...string) error { return nil },
	}
	if err := ensureRule[struct{}, string](m, r); err != nil {
		t.Fatalf("unexpected error from ensureRule: %v", err)
	}
	ads := m.validators["customX"]
	if len(ads) != 1 {
		t.Fatalf("expected 1 adapter appended, got %d", len(ads))
	}
	ad := ads[0]
	if ad.fn == nil {
		t.Fatalf("expected non-nil adapter fn")
	}
	if ad.fieldType == nil || ad.fieldType != reflect.TypeOf("") {
		t.Fatalf("unexpected fieldType: %#v", ad.fieldType)
	}
}

func TestEnsureRule_DuplicateExact_NoAppend(t *testing.T) {
	m := &Model[struct{}]{validators: make(map[string][]typedAdapter)}
	r := Rule[string]{
		Name: "dup",
		Fn:   func(s string, _ ...string) error { return nil },
	}
	if err := ensureRule[struct{}, string](m, r); err != nil {
		t.Fatalf("unexpected error on first ensureRule: %v", err)
	}
	if got := len(m.validators["dup"]); got != 1 {
		t.Fatalf("after first add, expected 1 adapter, got %d", got)
	}
	// Call ensureRule again with the same name and type; should NOT append a duplicate
	if err := ensureRule[struct{}, string](m, r); err != nil {
		t.Fatalf("unexpected error on second ensureRule: %v", err)
	}
	if got := len(m.validators["dup"]); got != 1 {
		t.Fatalf("after second add, expected still 1 adapter (no duplicate), got %d", got)
	}
	// Sanity: adapter type remains string
	if m.validators["dup"][0].fieldType != reflect.TypeOf("") {
		t.Fatalf("unexpected adapter fieldType: %v", m.validators["dup"][0].fieldType)
	}
}
