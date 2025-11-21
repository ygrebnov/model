package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// --- helpers & sample rules used in tests ---
func ruleNonZeroDur(d time.Duration, _ ...string) error {
	if d == 0 {
		return fmt.Errorf("duration must be non-zero")
	}
	return nil
}

// custom rule implementing min(1) semantics for tests replaced nonempty
func ruleMin1(s string, _ ...string) error {
	if len(s) < 1 {
		return fmt.Errorf("length must be >= 1 (got %d)", len(s))
	}
	return nil
}

// --- types under test ---

type vNoTags struct {
	A int
	B string
}

type vHasTags struct {
	Name string        `validate:"min(1)"`
	Wait time.Duration `validate:"nonZeroDur"`
	Info struct {
		Note string `validate:"min(1)"`
	}
}

func TestModel_validate(t *testing.T) {
	t.Parallel()

	type runFn func() (error, any)

	tests := []struct {
		name    string
		run     runFn
		wantErr string // substring to expect in error; empty => expect nil
		verify  func(t *testing.T, err error, m any)
	}{
		{
			name: "nil object -> error",
			run: func() (error, any) {
				var m Model[vNoTags]
				m.obj = nil
				return m.validate(context.Background()), &m
			},
			wantErr: "nil object",
		},
		{
			name: "non-struct object -> error",
			run: func() (error, any) {
				var m Model[int]
				x := 42
				m.obj = &x // *int (Elem != struct)
				return m.validate(context.Background()), &m
			},
			wantErr: "object must be a non-nil pointer to struct",
		},
		{
			name: "no tags -> ok (nil error)",
			run: func() (error, any) {
				var m Model[vNoTags]
				obj := vNoTags{A: 1, B: "x"}
				m.obj = &obj
				return m.validate(context.Background()), &m
			},
			wantErr: "",
		},
		{
			name: "rules satisfied -> ok (nil error)",
			run: func() (error, any) {
				m := &Model[vHasTags]{}
				obj := vHasTags{Name: "ok", Wait: time.Second}
				obj.Info.Note = "ok"
				m.obj = &obj
				// register rules
				min1, err := NewRule("min(1)", ruleMin1) // illustrative; tag uses min(1) but rule name simplified
				if err != nil {
					return err, m
				}
				nonZeroDur, err := NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					return err, m
				}
				if err = m.RegisterRules(min1, nonZeroDur); err != nil {
					return err, m
				}
				return m.validate(context.Background()), m
			},
			wantErr: "",
		},
		{
			name: "rule failures -> ValidationError with multiple field errors",
			run: func() (error, any) {
				m := &Model[vHasTags]{}
				obj := vHasTags{} // Name empty, Wait zero, Info.Note empty
				m.obj = &obj
				min1, err := NewRule("min(1)", ruleMin1)
				if err != nil {
					return err, m
				}
				nonZeroDur, err := NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					return err, m
				}
				if err = m.RegisterRules(min1, nonZeroDur); err != nil {
					return err, m
				}
				return m.validate(context.Background()), m
			},
			wantErr: "validation",
			verify: func(t *testing.T, err error, _ any) {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				if ve.Empty() || ve.Len() < 3 {
					t.Fatalf("expected >=3 field errors, got %d", ve.Len())
				}
				by := ve.ByField()
				for _, p := range []string{"Name", "Wait", "Info.Note"} {
					if _, ok := by[p]; !ok {
						t.Errorf("missing error for field path %q", p)
					}
				}
				if es := by["Name"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "length must be >= 1") {
					t.Errorf("expected min(1) error for name, got: %+v", es)
				}
				if es := by["Wait"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "non-zero") {
					t.Errorf("expected nonZeroDur error for Wait, got: %+v", es)
				}
			},
		},
		{
			name: "unknown rule -> ValidationError with rule-not-registered message",
			run: func() (error, any) {
				type vUnknown struct {
					Alias string `validate:"doesNotExist"`
				}
				m := &Model[vUnknown]{}
				obj := vUnknown{}
				m.obj = &obj
				return m.validate(context.Background()), m
			},
			wantErr: "rule not found",
			verify: func(t *testing.T, err error, _ any) {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				if len(ve.ByField()["Alias"]) == 0 {
					t.Fatalf("expected Alias to have a rule-not-found error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, m := tt.run()
			checkValidateTopError(t, err, tt.wantErr)
			if tt.verify != nil {
				tt.verify(t, err, m)
			}
		})
	}
}

// New test to ensure built-in rules are applied when Validate is called on a fresh Model without any options.
func TestModel_Validate_NoOptions_Builtins(t *testing.T) {
	t.Parallel()
	type Obj struct {
		S string `validate:"min(1)"`
	}
	obj := Obj{}
	m, err := New(&obj) // no WithValidation, no WithRules
	if err != nil {
		// New should not fail just because validation isn't requested yet.
		t.Fatalf("unexpected error from New: %v", err)
	}
	// First validation should pick up built-in nonempty and fail because S is empty.
	err = m.Validate(context.Background())
	if err == nil {
		t.Fatalf("expected validation error for empty S, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if _, ok := ve.ByField()["S"]; !ok {
		t.Fatalf("expected field error for S, got: %+v", ve.ByField())
	}
	// Fix the field and validate again; should succeed.
	obj.S = "x"
	if err := m.Validate(context.Background()); err != nil {
		t.Fatalf("expected no error after fixing S, got: %v", err)
	}
}

func checkValidateTopError(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if wantSubstr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error containing %q, got %v", wantSubstr, err)
	}
}
