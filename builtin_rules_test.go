package model

import (
	"testing"
)

func TestBuiltinRules_WithValidation_Nominal(t *testing.T) {
	t.Parallel()

	type strOK struct {
		S string `validate:"nonempty"`
	}
	type intOK struct {
		P  int `validate:"positive"`
		NZ int `validate:"nonzero"`
	}
	type int64OK struct {
		P  int64 `validate:"positive"`
		NZ int64 `validate:"nonzero"`
	}
	type float64OK struct {
		P  float64 `validate:"positive"`
		NZ float64 `validate:"nonzero"`
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "string nonempty passes",
			run: func(t *testing.T) {
				obj := strOK{S: "ok"}
				m, err := New(
					&obj,
					WithRules[strOK, string](BuiltinStringRules()),
					WithValidation[strOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				if m == nil || m.obj == nil {
					t.Fatalf("expected non-nil model and obj")
				}
			},
		},
		{
			name: "int positive & nonzero pass",
			run: func(t *testing.T) {
				obj := intOK{P: 1, NZ: 1}
				m, err := New(
					&obj,
					WithRules[intOK, int](BuiltinIntRules()),
					WithValidation[intOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				if m == nil || m.obj == nil {
					t.Fatalf("expected non-nil model and obj")
				}
			},
		},
		{
			name: "int64 positive & nonzero pass",
			run: func(t *testing.T) {
				obj := int64OK{P: 2, NZ: 3}
				m, err := New(
					&obj,
					WithRules[int64OK, int64](BuiltinInt64Rules()),
					WithValidation[int64OK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				if m == nil || m.obj == nil {
					t.Fatalf("expected non-nil model and obj")
				}
			},
		},
		{
			name: "float64 positive & nonzero pass",
			run: func(t *testing.T) {
				obj := float64OK{P: 0.1, NZ: 2.3}
				m, err := New(
					&obj,
					WithRules[float64OK, float64](BuiltinFloat64Rules()),
					WithValidation[float64OK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				if m == nil || m.obj == nil {
					t.Fatalf("expected non-nil model and obj")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, tt.run)
	}
}
