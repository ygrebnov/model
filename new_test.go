package model

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ygrebnov/model/rule"
)

// ---- Helpers ----

//func mustPanic(t *testing.T, fn func()) (msg string) {
//	t.Helper()
//	defer func() {
//		if r := recover(); r != nil {
//			msg = fmt.Sprint(r)
//		}
//	}()
//	fn()
//	t.Fatalf("expected panic, got none")
//	return ""
//}

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

	t.Run("error: nil object", func(t *testing.T) {
		m, err := New[*int](nil)
		if m != nil {
			t.Fatalf("expected nil model")
		}
		if !errors.Is(err, ErrNilObject) {
			t.Fatalf("expected ErrNilObject, got %v", err)
		}
	})

	t.Run("error: pointer to non-struct", func(t *testing.T) {
		x := 42
		m, err := New(&x) // TObject = int -> *int (Elem != struct)
		if m != nil {
			t.Fatalf("expected nil model")
		}
		if !errors.Is(err, ErrNotStructPtr) {
			t.Fatalf("expected ErrNotStructPtr, got %v", err)
		}
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
			WithRule[newOK, string]("nonempty", ruleNonEmpty),
			WithRule[newOK, time.Duration]("nonzero", ruleNonZeroDur),
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
			WithRule[newValidateBad, string]("nonempty", ruleNonEmpty),
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
		r, _ := rule.NewRule[string]("nonempty", ruleNonEmpty)
		m, err := New(
			&obj,
			WithRules[struct{ S string }, string]([]rule.Rule{r}),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		// Dispatch to prove adapter is wired; expect rule error (we pass empty string)
		if err := m.applyRule("nonempty", reflect.ValueOf(obj.S)); err == nil || !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("applyRule expected rule error, got: %v", err)
		}
	})

	t.Run("newRuleAdapter: interface overload is usable (AssignableTo)", func(t *testing.T) {
		obj := struct{ W wrapS }{W: wrapS{v: "Z"}}
		m, err := New(
			&obj,
			WithRule[struct{ W wrapS }, myStringer](
				"iface",
				func(s myStringer, _ ...string) error {
					return fmt.Errorf("iface:%s", s.String())
				},
			),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		// Call through applyRule with a concrete type implementing the interface
		if err := m.applyRule("iface", reflect.ValueOf(obj.W)); err == nil || !strings.Contains(err.Error(), "iface:Z") {
			t.Fatalf("expected interface overload to run, got: %v", err)
		}
	})

	t.Run("WithRule error: empty name", func(t *testing.T) {
		obj := struct{}{}
		m, err := New(
			&obj,
			WithRule[struct{}, string]("", ruleNonEmpty),
		)
		if m != nil {
			t.Fatalf("expected nil model on option error")
		}
		if err == nil || !strings.Contains(err.Error(), "non-empty Name") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("WithRule error: nil function", func(t *testing.T) {
		obj := struct{}{}
		m, err := New(
			&obj,
			WithRule[struct{}, string]("x", nil),
		)
		if m != nil {
			t.Fatalf("expected nil model on option error")
		}
		if err == nil || !strings.Contains(err.Error(), "non-nil Fn") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validators map initialized and preserves order on multiple WithRule", func(t *testing.T) {
		obj := struct{ S string }{}
		m, err := New(
			&obj,
			WithRule[struct{ S string }, string]("r", func(string, ...string) error { return fmt.Errorf("one") }),
			WithRule[struct{ S string }, string]("r", func(string, ...string) error { return fmt.Errorf("two") }),
		)
		if err != nil {
			t.Fatalf("New error: %v", err)
		}

		// validators have been removed.
		//if m.validators == nil {
		//	t.Fatalf("validators map not initialized")
		//}
		//ads := m.validators["r"]
		//if len(ads) != 2 {
		//	t.Fatalf("expected 2 adapters, got %d", len(ads))
		//}
		//// Verify order by calling adapter fns directly on a reflect.Value.
		//// This bypasses dispatch (which would be ambiguous with 2 exact overloads).
		//if err := ads[0].Fn(reflect.ValueOf("x")); err == nil || !strings.Contains(err.Error(), "one") {
		//	t.Fatalf("expected first adapter to be 'one', got: %v", err)
		//}
		//if err := ads[1].Fn(reflect.ValueOf("x")); err == nil || !strings.Contains(err.Error(), "two") {
		//	t.Fatalf("expected second adapter to be 'two', got: %v", err)
		//}

		// And confirm dispatch reports ambiguity for multiple exact overloads.
		if err := m.applyRule("r", reflect.ValueOf("x")); err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("expected ambiguity error from applyRule, got: %v", err)
		}
	})

	t.Run("options: short-circuit on first error; subsequent opts not applied", func(t *testing.T) {
		type T struct{}
		obj := T{}
		called1 := false
		called2 := false

		failOpt := Option[T](func(m *Model[T]) error {
			called1 = true
			return fmt.Errorf("fail-first")
		})
		sideOpt := Option[T](func(m *Model[T]) error {
			called2 = true
			m.applyDefaultsOnNew = true // visible side-effect if applied
			return nil
		})

		m, err := New(&obj, failOpt, sideOpt)
		if m != nil {
			t.Fatalf("expected nil model on first option error")
		}
		if err == nil || !strings.Contains(err.Error(), "fail-first") {
			t.Fatalf("expected first option error, got %v", err)
		}
		if !called1 {
			t.Fatalf("expected first option to be called")
		}
		if called2 {
			t.Fatalf("expected second option NOT to be called after first error")
		}
	})
}
