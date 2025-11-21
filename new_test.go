package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ygrebnov/errorc"
)

// ---- Helpers ----

// func mustPanic(t *testing.T, fn func()) (msg string) {
//	t.Helper()
//	defer func() {
//		if r := recover(); r != nil {
//			msg = fmt.Sprint(r)
//		}
//	}()
//	fn()
//	t.Fatalf("expected panic, got none")
//	return ""
// }

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
	t.Run("error: nil object", func(t *testing.T) {
		m, err := New[*int](nil)
		if m != nil {
			t.Fatalf("expected nil model")
		}
		if !errors.Is(err, ErrNilObject) {
			t.Fatalf("expected ErrNilObject, got %v", err)
		}
		if err.Error() != ErrNilObject.Error() {
			t.Fatalf("expected ErrNilObject message, got %q", err.Error())
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
		expectedError := errorc.With(ErrNotStructPtr, errorc.String(ErrorFieldObjectType, "int"))
		if err.Error() != expectedError.Error() {
			t.Fatalf("expected %q message, got %q", expectedError.Error(), err.Error())
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
			t.Fatalf("default not applied to name: %q", obj.Name)
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
		if err == nil || !strings.Contains(err.Error(), "cannot set default value") {
			t.Fatalf("expected error mentioning 'default for In', got: %v", err)
		}
	})

	t.Run("WithValidation: success when rules satisfied", func(t *testing.T) {
		obj := newOK{
			// name will be defaulted to "x" ONLY if WithDefaults is provided; here we set good values explicitly
			Name: "ok",
			Wait: 1,
			In: newInner{
				Msg: "yo",
				D:   time.Second,
			},
		}
		nonempty, err := NewRule("nonempty", ruleNonEmpty)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		nonzeroDur, err := NewRule("nonzero", ruleNonZeroDur)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		m, err := New(&obj, WithRules[newOK](nonempty, nonzeroDur))
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		if m == nil {
			t.Fatalf("expected non-nil model")
		}
	})

	t.Run("WithValidation: returns validation error", func(t *testing.T) {
		obj := newValidateBad{} // name empty
		r, err := NewRule("nonempty", ruleNonEmpty)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		m, err := New(
			&obj,
			WithRules[newValidateBad](r),
			WithValidation[newValidateBad](context.Background()),
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
			t.Fatalf("expected nonempty error for name, got: %+v", es)
		}
	})

	t.Run("WithRules: registers multiple and dispatch works (exact match)", func(t *testing.T) {
		// Use a struct with an explicit validate tag so Validate triggers the custom rule.
		type Obj struct {
			S string `validate:"nonempty"`
		}
		obj := Obj{S: ""}
		nonempty, err := NewRule[string]("nonempty", ruleNonEmpty)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		m, err := New(&obj, WithRules[Obj](nonempty))
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		// Dispatch via public Validate; expect validation error for empty S.
		if err := m.Validate(context.Background()); err == nil || !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("Validate expected rule error, got: %v", err)
		}
	})

	t.Run("newRuleAdapter: interface overload is usable (AssignableTo)", func(t *testing.T) {
		// This test is specifically about AssignableTo dispatch, so we
		// exercise the registry + rule directly instead of relying on tags.
		obj := struct{ W wrapS }{W: wrapS{v: "Z"}}
		iface, err := NewRule[myStringer]("iface", func(s myStringer, _ ...string) error {
			return fmt.Errorf("iface:%s", s.String())
		})
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		// Build a lightweight registry and call the rule directly to verify AssignableTo behavior.
		reg := newRulesRegistry()
		if err = reg.add(iface); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		// Simulate dispatch by resolving the rule for the concrete type wrapS.
		r, err := reg.get("iface", reflect.ValueOf(obj.W))
		if err != nil {
			t.Fatalf("registry.get error: %v", err)
		}
		if err = r.getValidationFn()(reflect.ValueOf(obj.W), nil...); err == nil || !strings.Contains(err.Error(), "iface:Z") {
			t.Fatalf("expected interface overload to run, got: %v", err)
		}
	})

	t.Run("WithRule error: empty name", func(t *testing.T) {
		_, err := NewRule[string]("", ruleNonEmpty)
		if err == nil || !strings.Contains(err.Error(), "non-empty name") {
			t.Fatalf("expected NewRule error for empty name, got: %v", err)
		}
	})

	t.Run("WithRule error: nil function", func(t *testing.T) {
		_, err := NewRule[string]("x", nil)
		if err == nil || !strings.Contains(err.Error(), "non-nil Fn") {
			t.Fatalf("expected NewRule error for nil function, got: %v", err)
		}
	})

	t.Run("duplicate overload registration via WithRules returns error", func(t *testing.T) {
		obj := struct{ S string }{}
		r1, err := NewRule[string]("r", func(s string, _ ...string) error { return fmt.Errorf("one") })
		if err != nil {
			t.Fatalf("NewRule r1 error: %v", err)
		}
		r2, err := NewRule[string]("r", func(s string, _ ...string) error { return fmt.Errorf("two") })
		if err != nil {
			t.Fatalf("NewRule r2 error: %v", err)
		}
		_, err = New(&obj, WithRules[struct{ S string }](r1, r2))
		if err == nil || !strings.Contains(err.Error(), "duplicate overload rule") {
			if err == nil {
				t.Fatalf("expected duplicate overload rule error, got nil")
			}
			t.Fatalf("expected duplicate overload rule error, got: %v", err)
		}
	})

	// returning error has been removed from Option signature
	//	t.Run("options: short-circuit on first error; subsequent opts not applied", func(t *testing.T) {
	//		type T struct{}
	//		obj := T{}
	//		called1 := false
	//		called2 := false
	//
	//		failOpt := Option[T](func(m *Model[T]) {
	//			called1 = true
	//			return fmt.Errorf("fail-first")
	//		})
	//		sideOpt := Option[T](func(m *Model[T]) {
	//			called2 = true
	//			m.applyDefaultsOnNew = true // visible side-effect if applied
	//			return nil
	//		})
	//
	//		m, err := New(&obj, failOpt, sideOpt)
	//		if m != nil {
	//			t.Fatalf("expected nil model on first option error")
	//		}
	//		if err == nil || !strings.Contains(err.Error(), "fail-first") {
	//			t.Fatalf("expected first option error, got %v", err)
	//		}
	//		if !called1 {
	//			t.Fatalf("expected first option to be called")
	//		}
	//		if called2 {
	//			t.Fatalf("expected second option NOT to be called after first error")
	//		}
	//	})
}
