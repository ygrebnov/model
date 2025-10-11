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

// --- types under test ---

// no tags -> validate should return nil
type vNoTags struct {
	A int
	B string
}

// tags, will be satisfied/violated in different scenarios
type vHasTags struct {
	Name string        `validate:"nonempty"`
	Wait time.Duration `validate:"nonZeroDur"`
	Info struct {
		Note string `validate:"nonempty"`
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
				m := &Model[vHasTags]{rulesMapping: newRulesMapping(), rulesRegistry: newRulesRegistry()}
				obj := vHasTags{
					Name: "ok",
					Wait: time.Second,
				}
				obj.Info.Note = "ok"
				m.obj = &obj
				// register rules
				nonempty, err := NewRule("nonempty", ruleNonEmpty)
				if err != nil {
					t.Fatalf("NewRule error: %v", err)
				}
				nonZeroDur, err := NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					t.Fatalf("NewRule error: %v", err)
				}
				if err = m.RegisterRules(nonempty, nonZeroDur); err != nil {
					t.Fatalf("RegisterRules error: %v", err)
				}
				return m.validate(context.Background()), m
			},
			wantErr: "",
		},
		{
			name: "rule failures -> ValidationError with multiple field errors",
			run: func() (error, any) {
				m := &Model[vHasTags]{rulesMapping: newRulesMapping(), rulesRegistry: newRulesRegistry()}
				obj := vHasTags{
					// name empty (violates nonempty)
					// Wait zero (violates nonZeroDur)
				}
				// nested struct field also empty (violates nonempty)
				m.obj = &obj
				// register rules
				nonempty, err := NewRule("nonempty", ruleNonEmpty)
				if err != nil {
					t.Fatalf("NewRule error: %v", err)
				}
				nonZeroDur, err := NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					t.Fatalf("NewRule error: %v", err)
				}
				if err = m.RegisterRules(nonempty, nonZeroDur); err != nil {
					t.Fatalf("RegisterRules error: %v", err)
				}
				return m.validate(context.Background()), m
			},
			wantErr: "validation", // we’ll assert concrete type & fields in verify
			verify: func(t *testing.T, err error, _ any) {
				// type assertion to *ValidationError
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				if ve.Empty() || ve.Len() < 3 {
					t.Fatalf("expected >=3 field errors, got %d", ve.Len())
				}
				// ensure important paths exist
				by := ve.ByField()
				wantPaths := []string{"Name", "Wait", "Info.Note"}
				for _, p := range wantPaths {
					if _, ok := by[p]; !ok {
						t.Errorf("missing error for field path %q", p)
					}
				}
				// check representative messages
				if es := by["Name"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "must not be empty") {
					t.Errorf("expected nonempty error for name, got: %+v", es)
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
				m := &Model[vUnknown]{rulesMapping: newRulesMapping(), rulesRegistry: newRulesRegistry()}
				obj := vUnknown{}
				m.obj = &obj
				// no rules registered on purpose
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
		tt := tt
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
		S string `validate:"nonempty"`
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
