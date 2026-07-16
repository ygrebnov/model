package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/validation"
)

func TestBinding_Rules(t *testing.T) {
	t.Run("add rule via WithRules", func(t *testing.T) {
		type sample struct{ A int }

		rule, err := model.NewRule[int]("dummy", func(value int, params ...string) error { return nil })
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
		nonempty, err := model.NewRule[string]("nonempty", ruleNonEmpty)
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

	t.Run("duplicate overload registration via WithRules returns error", func(t *testing.T) {
		obj := struct{ S string }{}
		r1, err := model.NewRule[string]("r", func(s string, _ ...string) error { return fmt.Errorf("one") })
		if err != nil {
			t.Fatalf("NewRule r1 error: %v", err)
		}
		r2, err := model.NewRule[string]("r", func(s string, _ ...string) error { return fmt.Errorf("two") })
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

	t.Run("implicit builtin rules are applied", func(t *testing.T) {
		type sample struct {
			Name  string  `validate:"email"`
			Age   int     `validate:"min(1)"`
			Score float64 `validate:"nonzero"`
			ID    int64   `validate:"nonzero"`
		}

		binding, err := model.NewBinding[sample]()
		if err != nil {
			t.Fatalf("NewBinding() error: %v", err)
		}

		err = binding.Validate(context.Background(), &sample{})
		var validationErr *validation.Error
		if !errors.As(err, &validationErr) {
			t.Fatalf("Validate() error = %v, want *validation.Error", err)
		}

		for _, field := range []string{"name", "age", "score", "id"} {
			if _, ok := validationErr.ByField()[field]; !ok {
				t.Fatalf(
					"expected validation error for %q, got: %+v",
					field,
					validationErr.ByField(),
				)
			}
		}
	})

	t.Run("custom rule overrides builtin rule", func(t *testing.T) {
		type sample struct {
			Name string `validate:"email"`
		}

		customEmail, err := model.NewRule[string](
			"email",
			func(value string, _ ...string) error {
				return errors.New("custom email: " + value)
			},
		)
		if err != nil {
			t.Fatalf("NewRule() error: %v", err)
		}

		binding, err := model.NewBinding[sample](
			model.WithRules(customEmail),
		)
		if err != nil {
			t.Fatalf("NewBinding() error: %v", err)
		}

		err = binding.Validate(context.Background(), &sample{})
		var validationErr *validation.Error
		if !errors.As(err, &validationErr) {
			t.Fatalf("Validate() error = %v, want *validation.Error", err)
		}

		fieldErrors := validationErr.ByField()["name"]
		if len(fieldErrors) == 0 ||
			!strings.Contains(fieldErrors[0].Err.Error(), "custom email") {
			t.Fatalf("expected custom email error, got: %+v", fieldErrors)
		}
	})
}
