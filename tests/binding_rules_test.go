package tests

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/validation"
)

func TestBinding_Rules(t *testing.T) {
	t.Run("add rule via WithRules", func(t *testing.T) {
		type sample struct{ A int }

		rule, err := validation.NewRule[int]("dummy", func(value int, params ...string) error { return nil })
		if err != nil {
			t.Fatalf("unexpected error creating rule: %v", err)
		}

		if _, err = model.NewBinding[sample](model.WithRules(rule)); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("WithRules: registers multiple and dispatch works (exact match)", func(t *testing.T) {
		// Use a struct with an explicit validate tag so Validate triggers the custom rule.
		type Obj struct {
			S string `validate:"nonempty"`
		}
		obj := Obj{S: ""}
		nonempty, err := validation.NewRule[string]("nonempty", ruleNonEmpty)
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		b2, err := model.NewBinding[Obj](model.WithRules(nonempty))
		if err != nil {
			t.Fatalf("New error: %v", err)
		}
		// Dispatch via public Validate; expect validation error for empty S.
		if err := b2.Validate(context.Background(), &obj); err == nil || !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("Validate expected rule error, got: %v", err)
		}
	})

	t.Run("newRuleAdapter: interface overload is usable (AssignableTo)", func(t *testing.T) {
		// This test is specifically about AssignableTo dispatch, so we
		// exercise the registry + rule directly instead of relying on tags.
		obj := struct{ W wrapS }{W: wrapS{v: "Z"}}
		iface, err := validation.NewRule[myStringer]("iface", func(s myStringer, _ ...string) error {
			return fmt.Errorf("iface:%s", s.String())
		})
		if err != nil {
			t.Fatalf("NewRule error: %v", err)
		}
		// Build a lightweight registry and call the rule directly to verify AssignableTo behavior.
		reg := validation.NewRulesRegistry()
		if err = reg.Add(iface); err != nil {
			t.Fatalf("registry.add error: %v", err)
		}
		// Simulate dispatch by resolving the rule for the concrete type wrapS.
		r, err := reg.Get("iface", reflect.ValueOf(obj.W))
		if err != nil {
			t.Fatalf("registry.get error: %v", err)
		}
		if err = r.GetValidationFn()(reflect.ValueOf(obj.W), nil...); err == nil || !strings.Contains(err.Error(), "iface:Z") {
			t.Fatalf("expected interface overload to run, got: %v", err)
		}
	})

	t.Run("duplicate overload registration via WithRules returns error", func(t *testing.T) {
		obj := struct{ S string }{}
		r1, err := validation.NewRule[string]("r", func(s string, _ ...string) error { return fmt.Errorf("one") })
		if err != nil {
			t.Fatalf("NewRule r1 error: %v", err)
		}
		r2, err := validation.NewRule[string]("r", func(s string, _ ...string) error { return fmt.Errorf("two") })
		if err != nil {
			t.Fatalf("NewRule r2 error: %v", err)
		}
		err = model.Validate(context.Background(), &obj, model.WithRules(r1, r2))
		if err == nil || !strings.Contains(err.Error(), "duplicate overload rule") {
			if err == nil {
				t.Fatalf("expected duplicate overload rule error, got nil")
			}
			t.Fatalf("expected duplicate overload rule error, got: %v", err)
		}
	})
}
