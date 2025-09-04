package model

import (
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
				return m.validate(), &m
			},
			wantErr: "nil object",
		},
		{
			name: "non-struct object -> error",
			run: func() (error, any) {
				var m Model[int]
				x := 42
				m.obj = &x // *int (Elem != struct)
				return m.validate(), &m
			},
			wantErr: "object must point to a struct",
		},
		{
			name: "no tags -> ok (nil error)",
			run: func() (error, any) {
				var m Model[vNoTags]
				obj := vNoTags{A: 1, B: "x"}
				m.obj = &obj
				return m.validate(), &m
			},
			wantErr: "",
		},
		{
			name: "rules satisfied -> ok (nil error)",
			run: func() (error, any) {
				m := &Model[vHasTags]{validators: make(map[string][]typedAdapter)}
				obj := vHasTags{
					Name: "ok",
					Wait: time.Second,
				}
				obj.Info.Note = "ok"
				m.obj = &obj
				// register rules
				WithRule[vHasTags, string](Rule[string]{Name: "nonempty", Fn: ruleNonEmpty})(m)
				WithRule[vHasTags, time.Duration](Rule[time.Duration]{Name: "nonZeroDur", Fn: ruleNonZeroDur})(m)
				return m.validate(), m
			},
			wantErr: "",
		},
		{
			name: "rule failures -> ValidationError with multiple field errors",
			run: func() (error, any) {
				m := &Model[vHasTags]{validators: make(map[string][]typedAdapter)}
				obj := vHasTags{
					// Name empty (violates nonempty)
					// Wait zero (violates nonZeroDur)
				}
				// nested struct field also empty (violates nonempty)
				m.obj = &obj
				// register rules
				WithRule[vHasTags, string](Rule[string]{Name: "nonempty", Fn: ruleNonEmpty})(m)
				WithRule[vHasTags, time.Duration](Rule[time.Duration]{Name: "nonZeroDur", Fn: ruleNonZeroDur})(m)
				return m.validate(), m
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
					t.Errorf("expected nonempty error for Name, got: %+v", es)
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
				m := &Model[vUnknown]{validators: make(map[string][]typedAdapter)}
				obj := vUnknown{}
				m.obj = &obj
				// no rules registered on purpose
				return m.validate(), m
			},
			wantErr: "rule \"doesNotExist\" is not registered",
			verify: func(t *testing.T, err error, _ any) {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				if len(ve.ByField()["Alias"]) == 0 {
					t.Fatalf("expected Alias to have a rule-not-registered error")
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
