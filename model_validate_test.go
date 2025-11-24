package model

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ygrebnov/errorc"
	modelerrors "github.com/ygrebnov/model/errors"
	"github.com/ygrebnov/model/validation"
)

// --- helpers & sample rules used in tests ---
func ruleNonZeroDur(d time.Duration, _ ...string) error {
	if d == 0 {
		return errorc.With(
			modelerrors.ErrRuleNonZeroDurFailed,
			errorc.String(modelerrors.ErrorFieldRuleName, "nonZeroDur"),
		)
	}
	return nil
}

// custom rule implementing min(1) semantics for tests replaced nonempty
func ruleMin1(s string, _ ...string) error {
	if len(s) < 1 {
		return errorc.With(
			modelerrors.ErrRuleMin1Failed,
			errorc.String(modelerrors.ErrorFieldRuleName, "min(1)"),
		)
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

	type runFn func() (any, error)

	tests := []struct {
		name    string
		run     runFn
		wantErr string // substring to expect in error; empty => expect nil
		verify  func(t *testing.T, err error, m any)
	}{
		{
			name: "nil object -> error",
			run: func() (any, error) {
				var m Model[vNoTags]
				m.obj = nil
				return &m, m.validate(context.Background())
			},
			wantErr: "nil object",
		},
		{
			name: "non-struct object -> error",
			run: func() (any, error) {
				var m Model[int]
				x := 42
				m.obj = &x // *int (Elem != struct)
				return &m, m.validate(context.Background())
			},
			wantErr: "object must be a non-nil pointer to struct",
		},
		{
			name: "no tags -> ok (nil error)",
			run: func() (any, error) {
				var m Model[vNoTags]
				obj := vNoTags{A: 1, B: "x"}
				m.obj = &obj
				return &m, m.validate(context.Background())
			},
			wantErr: "",
		},
		{
			name: "rules satisfied -> ok (nil error)",
			run: func() (any, error) {
				m := &Model[vHasTags]{}
				obj := vHasTags{Name: "ok", Wait: time.Second}
				obj.Info.Note = "ok"
				m.obj = &obj
				// register rules
				min1, err := validation.NewRule("min(1)", ruleMin1) // illustrative; tag uses min(1) but rule name simplified
				if err != nil {
					return m, err
				}
				nonZeroDur, err := validation.NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					return m, err
				}
				if err := m.RegisterRules(min1, nonZeroDur); err != nil {
					return m, err
				}
				validationErr := m.validate(context.Background())
				return m, validationErr
			},
			wantErr: "",
		},
		{
			name: "rule failures -> ValidationError with multiple field errors",
			run: func() (any, error) {
				m := &Model[vHasTags]{}
				obj := vHasTags{} // Name empty, Wait zero, Info.Note empty
				m.obj = &obj
				min1, err := validation.NewRule("min(1)", ruleMin1)
				if err != nil {
					return m, err
				}
				nonZeroDur, err := validation.NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					return m, err
				}
				if err := m.RegisterRules(min1, nonZeroDur); err != nil {
					return m, err
				}
				validationErr := m.validate(context.Background())
				return m, validationErr
			},
			wantErr: "validation",
			verify: func(t *testing.T, err error, _ any) {
				var ve *validation.Error
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
				if es := by["Name"]; len(es) == 0 {
					t.Fatalf("expected error for Name")
				} else {
					// Name uses builtin string min rule; assert on builtin sentinel and metadata.
					if !errors.Is(es[0].Err, modelerrors.ErrRuleConstraintViolated) {
						t.Fatalf("expected ErrRuleConstraintViolated for Name, got %v", es[0].Err)
					}
					msg := es[0].Err.Error()
					if !strings.Contains(msg, string(modelerrors.ErrorFieldRuleName)+": min") {
						t.Errorf("expected builtin min rule name metadata for Name, got: %q", msg)
					}
					if !strings.Contains(msg, string(modelerrors.ErrorFieldRuleParamName)+": length") ||
						!strings.Contains(msg, string(modelerrors.ErrorFieldRuleParamValue)+": 1") {
						t.Errorf("expected min length metadata for Name, got: %q", msg)
					}
				}
				if es := by["Wait"]; len(es) == 0 {
					t.Fatalf("expected error for Wait")
				} else {
					if !errors.Is(es[0].Err, modelerrors.ErrRuleNonZeroDurFailed) {
						t.Fatalf("expected ErrRuleNonZeroDurFailed for Wait, got %v", es[0].Err)
					}
					if msg := es[0].Err.Error(); !strings.Contains(msg, string(modelerrors.ErrorFieldRuleName)+": nonZeroDur") {
						t.Errorf("expected rule name metadata for nonZeroDur in Wait error, got: %q", msg)
					}
				}
			},
		},
		{
			name: "unknown rule -> ValidationError with rule-not-registered message",
			run: func() (any, error) {
				type vUnknown struct {
					Alias string `validate:"doesNotExist"`
				}
				m := &Model[vUnknown]{}
				obj := vUnknown{}
				m.obj = &obj
				err := m.validate(context.Background())
				return m, err
			},
			wantErr: "rule not found",
			verify: func(t *testing.T, err error, _ any) {
				var ve *validation.Error
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
			t.Parallel()

			m, err := tt.run()
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
	var ve *validation.Error
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
