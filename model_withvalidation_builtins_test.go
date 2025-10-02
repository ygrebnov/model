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
	customNonempty, err := NewRule[string]("nonempty", func(s string, _ ...string) error {
		if s == "" {
			return errors.New("custom nonempty")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	_, err = New(&obj, WithRules[bv](customNonempty), WithValidation[bv]())
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
