package model

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ---- Helpers ----

func mustPanic(t *testing.T, fn func()) (msg string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	fn()
	t.Fatalf("expected panic, got none")
	return ""
}

type myStringer interface{ String() string }
type wrapS struct{ v string }

func (w wrapS) String() string { return w.v }

// ---- Types under test ----

type newInner struct {
	Msg string        `default:"hi" validate:"nonempty"`
	D   time.Duration `default:"2s" validate:"nonzero"`
}

type newOK struct {
	Name string   `default:"x" validate:"nonempty"`
	Wait int      `default:"3"`
	In   newInner `default:"dive"`
}

type newDefaultsBad struct {
	// unsupported literal on struct field -> setDefaultsStruct returns error
	In newInner `default:"oops"`
}

type newValidateBad struct {
	Name string `validate:"nonempty"` // empty -> error when rule registered
}

// ---- Tests ----

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("panic: nil object", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for nil obj")
			}
		}()
		_, _ = New[*int](nil) // type parameter doesn't matter here
	})

	t.Run("panic: pointer to non-struct", func(t *testing.T) {
		x := 42
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for pointer to non-struct")
			} else {
				if !strings.Contains(fmt.Sprint(r), "pointer to struct") {
					t.Fatalf("unexpected panic message: %v", r)
				}
			}
		}()
		_, _ = New(&x) // TObject = int -> *int (Elem != struct)
	})

	t.Run("WithDefaults: success applies defaults", func(t *testing.T) {
		obj := newOK{}
		m, err := New(&obj, WithDefaults[newOK]())
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		if m == nil || m.obj == nil {
			t.Fatalf("model or obj is nil")
		}
		// top level
		if obj.Name != "x" {
			t.Fatalf("default not applied to Name: %q", obj.Name)
		}
		// nested dive
		if obj.In.Msg != "hi" || obj.In.D != 2*time.Second {
			t.Fatalf("defaults not applied to nested In: %+v", obj.In)
		}
		// literal int
		if obj.Wait != 3 {
			t.Fatalf("default not applied to Wait: %d", obj.Wait)
		}
	})

	t.Run("WithDefaults: error propagated from defaults", func(t *testing.T) {
		obj := newDefaultsBad{}
		m, err := New(&obj, WithDefaults[newDefaultsBad]())
		if m != nil {
			// model is returned only if err == nil
			t.Fatalf("expected nil model on error, got non-nil")
		}
		if err == nil || !strings.Contains(err.Error(), "default for In") {
			t.Fatalf("expected error mentioning 'default for In', got: %v", err)
		}
	})

	t.Run("WithValidation: success when rules satisfied", func(t *testing.T) {
		obj := newOK{
			// Name will be defaulted to "x" ONLY if WithDefaults is provided; here we set good values explicitly
			Name: "ok",
			Wait: 1,
			In: newInner{
				Msg: "yo",
				D:   time.Second,
			},
		}
		m, err := New(
			&obj,
			WithRule[newOK, string](Rule[string]{Name: "nonempty", Fn: ruleNonEmpty}),
			WithRule[newOK, time.Duration](Rule[time.Duration]{Name: "nonzero", Fn: ruleNonZeroDur}),
			WithValidation[newOK](),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		if m == nil {
			t.Fatalf("expected non-nil model")
		}
	})

	t.Run("WithValidation: returns validation error", func(t *testing.T) {
		obj := newValidateBad{} // Name empty
		m, err := New(
			&obj,
			WithRule[newValidateBad, string](Rule[string]{Name: "nonempty", Fn: ruleNonEmpty}),
			WithValidation[newValidateBad](),
		)
		if m != nil {
			t.Fatalf("expected nil model when validation fails")
		}
		if err == nil {
			t.Fatalf("expected validation error, got nil")
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("expected *ValidationError, got %T: %v", err, err)
		}
		// Ensure it contains the expected field error
		by := ve.ByField()
		if es := by["Name"]; len(es) == 0 || es[0].Rule != "nonempty" {
			t.Fatalf("expected nonempty error for Name, got: %+v", es)
		}
	})

	t.Run("WithRules: registers multiple and dispatch works (exact match)", func(t *testing.T) {
		obj := struct{ S string }{S: ""}
		m, err := New(
			&obj,
			WithRules[struct{ S string }, string]([]Rule[string]{
				{Name: "nonempty", Fn: ruleNonEmpty},
			}),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		// Dispatch to prove adapter is wired; expect rule error (we pass empty string)
		if err := m.applyRule("nonempty", reflect.ValueOf(obj.S)); err == nil || !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("applyRule expected rule error, got: %v", err)
		}
	})

	t.Run("wrapRule: interface overload is usable (AssignableTo)", func(t *testing.T) {
		obj := struct{ W wrapS }{W: wrapS{v: "Z"}}
		m, err := New(
			&obj,
			WithRule[struct{ W wrapS }, myStringer](Rule[myStringer]{
				Name: "iface",
				Fn: func(s myStringer, _ ...string) error {
					return fmt.Errorf("iface:%s", s.String())
				},
			}),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		// Call through applyRule with a concrete type implementing the interface
		if err := m.applyRule("iface", reflect.ValueOf(obj.W)); err == nil || !strings.Contains(err.Error(), "iface:Z") {
			t.Fatalf("expected interface overload to run, got: %v", err)
		}
	})

	t.Run("WithRule panic: empty name", func(t *testing.T) {
		obj := struct{}{}
		msg := mustPanic(t, func() {
			_, _ = New(
				&obj,
				WithRule[struct{}, string](Rule[string]{Name: "", Fn: ruleNonEmpty}), // should panic during option apply
			)
		})
		if !strings.Contains(msg, "rule must have non-empty Name") {
			t.Fatalf("unexpected panic message: %q", msg)
		}
	})

	t.Run("WithRule panic: nil function", func(t *testing.T) {
		obj := struct{}{}
		msg := mustPanic(t, func() {
			_, _ = New(
				&obj,
				WithRule[struct{}, string](Rule[string]{Name: "x", Fn: nil}), // should panic during option apply
			)
		})
		if !strings.Contains(msg, "non-nil Fn") {
			t.Fatalf("unexpected panic message: %q", msg)
		}
	})

	t.Run("validators map initialized and preserves order on multiple WithRule", func(t *testing.T) {
		obj := struct{ S string }{}
		m, err := New(
			&obj,
			WithRule[struct{ S string }, string](Rule[string]{Name: "r", Fn: func(string, ...string) error { return fmt.Errorf("one") }}),
			WithRule[struct{ S string }, string](Rule[string]{Name: "r", Fn: func(string, ...string) error { return fmt.Errorf("two") }}),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		if m.validators == nil {
			t.Fatalf("validators map not initialized")
		}
		ads := m.validators["r"]
		if len(ads) != 2 {
			t.Fatalf("expected 2 adapters, got %d", len(ads))
		}
		// Verify order by calling adapter fns directly on a reflect.Value.
		// This bypasses dispatch (which would be ambiguous with 2 exact overloads).
		if err := ads[0].fn(reflect.ValueOf("x")); err == nil || !strings.Contains(err.Error(), "one") {
			t.Fatalf("expected first adapter to be 'one', got: %v", err)
		}
		if err := ads[1].fn(reflect.ValueOf("x")); err == nil || !strings.Contains(err.Error(), "two") {
			t.Fatalf("expected second adapter to be 'two', got: %v", err)
		}

		// And confirm dispatch reports ambiguity for multiple exact overloads.
		if err := m.applyRule("r", reflect.ValueOf("x")); err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("expected ambiguity error from applyRule, got: %v", err)
		}
	})
}
