package model

import (
	"errors"
	"strings"
	"testing"
)

type bv struct {
	Name  string  `validate:"nonempty"`
	Age   int     `validate:"positive"`
	Score float64 `validate:"nonzero"`
	ID    int64   `validate:"nonzero"`
}

func TestWithValidation_ImplicitBuiltinRulesApplied(t *testing.T) {
	obj := bv{}
	_, err := New(&obj, WithValidation[bv]())
	if err == nil {
		t.Fatalf("expected validation error due to implicit builtin rules")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %v", err)
	}
	by := ve.ByField()
	// Expect errors for all fields using builtin rules
	for _, f := range []string{"Name", "Age", "Score", "ID"} {
		if _, ok := by[f]; !ok {
			t.Fatalf("expected error for field %s; got map=%+v", f, by)
		}
	}
}

func TestWithValidation_CustomRuleOverrides_WhenRegisteredBefore(t *testing.T) {
	obj := bv{}
	customNonEmpty := Rule[string]{
		Name: "nonempty",
		Fn: func(s string, _ ...string) error {
			if s == "" {
				return errors.New("custom nonempty")
			}
			return nil
		},
	}
	_, err := New(&obj,
		WithRule[bv, string](customNonEmpty), // register BEFORE WithValidation
		WithValidation[bv](),
	)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %v", err)
	}
	msgs := ve.ByField()["Name"]
	if len(msgs) == 0 || !strings.Contains(msgs[0].Err.Error(), "custom nonempty") {
		t.Fatalf("expected custom nonempty error, got %+v", msgs)
	}
}

func TestWithValidation_CustomRuleAfter_BecomesAmbiguous(t *testing.T) {
	obj := bv{}
	customNonEmpty := Rule[string]{
		Name: "nonempty",
		Fn: func(s string, _ ...string) error {
			if s == "" {
				return errors.New("custom nonempty")
			}
			return nil
		},
	}
	_, err := New(&obj,
		WithValidation[bv](),                 // builtin nonempty for string is registered implicitly
		WithRule[bv, string](customNonEmpty), // registering AFTER creates two exact overloads
	)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %v", err)
	}
	msgs := ve.ByField()["Name"]
	if len(msgs) == 0 || !strings.Contains(msgs[0].Err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error for Name, got %+v", msgs)
	}
}
