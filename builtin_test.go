package model

import (
	"strings"
	"testing"
)

func TestBuiltinRules_WithValidation_Nominal(t *testing.T) {

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

	// --- oneof targets ---
	type strOneOfOK struct {
		S string `validate:"oneof(red,green,blue)"`
	}
	type strOneOfBad struct {
		S string `validate:"oneof(red,green,blue)"`
	}
	type strOneOfNoParams struct {
		S string `validate:"oneof()"`
	}

	type intOneOfOK struct {
		N int `validate:"oneof(1,2,3)"`
	}
	type intOneOfBad struct {
		N int `validate:"oneof(1,2,3)"`
	}
	type intOneOfNoParams struct {
		N int `validate:"oneof()"`
	}

	type intPositiveOnly struct {
		P int `validate:"positive"`
	}
	type intNonZeroOnly struct {
		NZ int `validate:"nonzero"`
	}
	type intOneOfBadParam struct {
		N int `validate:"oneof(1,a,3)"`
	}

	type int64OneOfOK struct {
		N int64 `validate:"oneof(10,20,30)"`
	}
	type int64OneOfBad struct {
		N int64 `validate:"oneof(10,20,30)"`
	}
	type int64OneOfNoParams struct {
		N int64 `validate:"oneof()"`
	}
	type int64PositiveOnly struct {
		P int64 `validate:"positive"`
	}
	type int64NonZeroOnly struct {
		NZ int64 `validate:"nonzero"`
	}
	type int64OneOfBadParam struct {
		N int64 `validate:"oneof(10,a,30)"`
	}

	type float64OneOfOK struct {
		F float64 `validate:"oneof(0.5,1.0,2.5)"`
	}
	type float64OneOfBad struct {
		F float64 `validate:"oneof(0.5,1.0,2.5)"`
	}
	type float64OneOfNoParams struct {
		F float64 `validate:"oneof()"`
	}
	type float64PositiveOnly struct {
		P float64 `validate:"positive"`
	}
	type float64NonZeroOnly struct {
		NZ float64 `validate:"nonzero"`
	}
	type float64OneOfBadParam struct {
		F float64 `validate:"oneof(0.5,a,2.5)"`
	}

	tests := []struct {
		name      string
		run       func(t *testing.T) error
		wantError bool
		checkErr  func(t *testing.T, err error)
	}{
		{
			name: "string nonempty passes",
			run: func(t *testing.T) error {
				obj := strOK{S: "ok"}
				_, err := New(
					&obj,
					//WithRules[strOK, string](BuiltinStringRules()),
					WithValidation[strOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "string nonempty fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOK{S: ""}
				_, err := New(
					&obj,
					//WithRules[strOK, string](BuiltinStringRules()),
					WithValidation[strOK](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must not be empty") {
					t.Fatalf("expected nonempty failure, got: %q", got)
				}
			},
		},
		{
			name: "int positive & nonzero pass",
			run: func(t *testing.T) error {
				obj := intOK{P: 1, NZ: 1}
				_, err := New(
					&obj,
					//WithRules[intOK, int](BuiltinIntRules()),
					WithValidation[intOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name: "int64 positive & nonzero pass",
			run: func(t *testing.T) error {
				obj := int64OK{P: 2, NZ: 3}
				_, err := New(
					&obj,
					//WithRules[int64OK, int64](BuiltinInt64Rules()),
					WithValidation[int64OK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name: "float64 positive & nonzero pass",
			run: func(t *testing.T) error {
				obj := float64OK{P: 0.1, NZ: 2.3}
				_, err := New(
					&obj,
					//WithRules[float64OK, float64](BuiltinFloat64Rules()),
					WithValidation[float64OK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},

		// --- oneof: string ---
		{
			name: "string oneof passes",
			run: func(t *testing.T) error {
				obj := strOneOfOK{S: "green"}
				_, err := New(
					&obj,
					//WithRules[strOneOfOK, string](BuiltinStringRules()),
					WithValidation[strOneOfOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "string oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOneOfBad{S: "yellow"}
				_, err := New(
					&obj,
					//WithRules[strOneOfBad, string](BuiltinStringRules()),
					WithValidation[strOneOfBad](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "oneof") || !strings.Contains(got, "must be one of") {
					t.Fatalf("expected oneof failure in error, got: %q", got)
				}
			},
		},
		{
			name:      "string oneof with no params fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOneOfNoParams{S: "anything"}
				_, err := New(
					&obj,
					//WithRules[strOneOfNoParams, string](BuiltinStringRules()),
					WithValidation[strOneOfNoParams](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "oneof requires at least one parameter") {
					t.Fatalf("expected parameter error, got: %q", got)
				}
			},
		},

		{
			name:      "int64 positive fails (must be > 0)",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64PositiveOnly{P: 0} // triggers positive rule
				_, err := New(
					&obj,
					//WithRules[int64PositiveOnly, int64](BuiltinInt64Rules()),
					WithValidation[int64PositiveOnly](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must be > 0") {
					t.Fatalf("expected positive failure, got: %q", got)
				}
			},
		},
		{
			name:      "int64 nonzero fails (must not be zero)",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64NonZeroOnly{NZ: 0} // triggers nonzero rule
				_, err := New(
					&obj,
					//WithRules[int64NonZeroOnly, int64](BuiltinInt64Rules()),
					WithValidation[int64NonZeroOnly](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must not be zero") {
					t.Fatalf("expected nonzero failure, got: %q", got)
				}
			},
		},
		{
			name:      "int64 oneof fails on invalid parameter",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOfBadParam{N: 20} // value irrelevant; rule errors on parsing "a"
				_, err := New(
					&obj,
					//WithRules[int64OneOfBadParam, int64](BuiltinInt64Rules()),
					WithValidation[int64OneOfBadParam](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, `invalid oneof parameter "a" for int64`) {
					t.Fatalf("expected invalid oneof parameter error, got: %q", got)
				}
			},
		},
		{
			name:      "int positive fails (must be > 0)",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intPositiveOnly{P: 0} // triggers positive rule
				_, err := New(
					&obj,
					//WithRules[intPositiveOnly, int](BuiltinIntRules()),
					WithValidation[intPositiveOnly](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must be > 0") {
					t.Fatalf("expected positive failure, got: %q", got)
				}
			},
		},
		{
			name:      "int nonzero fails (must not be zero)",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intNonZeroOnly{NZ: 0} // triggers nonzero rule
				_, err := New(
					&obj,
					//WithRules[intNonZeroOnly, int](BuiltinIntRules()),
					WithValidation[intNonZeroOnly](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must not be zero") {
					t.Fatalf("expected nonzero failure, got: %q", got)
				}
			},
		},
		{
			name:      "int oneof fails on invalid parameter",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOfBadParam{N: 2} // value doesn't matter; rule should error on parsing "a"
				_, err := New(
					&obj,
					//WithRules[intOneOfBadParam, int](BuiltinIntRules()),
					WithValidation[intOneOfBadParam](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, `invalid oneof parameter "a" for int`) {
					t.Fatalf("expected invalid oneof parameter error, got: %q", got)
				}
			},
		},
		// --- oneof: int ---
		{
			name: "int oneof passes",
			run: func(t *testing.T) error {
				obj := intOneOfOK{N: 2}
				_, err := New(
					&obj,
					//WithRules[intOneOfOK, int](BuiltinIntRules()),
					WithValidation[intOneOfOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOfBad{N: 4}
				_, err := New(
					&obj,
					//WithRules[intOneOfBad, int](BuiltinIntRules()),
					WithValidation[intOneOfBad](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must be one of") {
					t.Fatalf("expected membership error, got: %q", got)
				}
			},
		},
		{
			name:      "int oneof with no params fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOfNoParams{N: 1}
				_, err := New(
					&obj,
					//WithRules[intOneOfNoParams, int](BuiltinIntRules()),
					WithValidation[intOneOfNoParams](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "oneof requires at least one parameter") {
					t.Fatalf("expected parameter error, got: %q", got)
				}
			},
		},

		// --- oneof: int64 ---
		{
			name: "int64 oneof passes",
			run: func(t *testing.T) error {
				obj := int64OneOfOK{N: 20}
				_, err := New(
					&obj,
					//WithRules[int64OneOfOK, int64](BuiltinInt64Rules()),
					WithValidation[int64OneOfOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int64 oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOfBad{N: 99}
				_, err := New(
					&obj,
					//WithRules[int64OneOfBad, int64](BuiltinInt64Rules()),
					WithValidation[int64OneOfBad](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must be one of") {
					t.Fatalf("expected membership error, got: %q", got)
				}
			},
		},
		{
			name:      "int64 oneof with no params fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOfNoParams{N: 10}
				_, err := New(
					&obj,
					//WithRules[int64OneOfNoParams, int64](BuiltinInt64Rules()),
					WithValidation[int64OneOfNoParams](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "oneof requires at least one parameter") {
					t.Fatalf("expected parameter error, got: %q", got)
				}
			},
		},

		{
			name:      "float64 positive fails (must be > 0)",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64PositiveOnly{P: 0.0}
				_, err := New(
					&obj,
					//WithRules[float64PositiveOnly, float64](BuiltinFloat64Rules()),
					WithValidation[float64PositiveOnly](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must be > 0") {
					t.Fatalf("expected positive failure, got: %q", got)
				}
			},
		},
		{
			name:      "float64 nonzero fails (must not be zero)",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64NonZeroOnly{NZ: 0.0}
				_, err := New(
					&obj,
					//WithRules[float64NonZeroOnly, float64](BuiltinFloat64Rules()),
					WithValidation[float64NonZeroOnly](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must not be zero") {
					t.Fatalf("expected nonzero failure, got: %q", got)
				}
			},
		},
		{
			name:      "float64 oneof fails on invalid parameter",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOfBadParam{F: 1.0} // value irrelevant; rule errors parsing "a"
				_, err := New(
					&obj,
					//WithRules[float64OneOfBadParam, float64](BuiltinFloat64Rules()),
					WithValidation[float64OneOfBadParam](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, `invalid oneof parameter "a" for float64`) {
					t.Fatalf("expected invalid oneof parameter error, got: %q", got)
				}
			},
		},

		// --- oneof: float64 ---
		{
			name: "float64 oneof passes",
			run: func(t *testing.T) error {
				obj := float64OneOfOK{F: 1.0}
				_, err := New(
					&obj,
					//WithRules[float64OneOfOK, float64](BuiltinFloat64Rules()),
					WithValidation[float64OneOfOK](),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "float64 oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOfBad{F: 0.3}
				_, err := New(
					&obj,
					//WithRules[float64OneOfBad, float64](BuiltinFloat64Rules()),
					WithValidation[float64OneOfBad](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "must be one of") {
					t.Fatalf("expected membership error, got: %q", got)
				}
			},
		},
		{
			name:      "float64 oneof with no params fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOfNoParams{F: 1.0}
				_, err := New(
					&obj,
					//WithRules[float64OneOfNoParams, float64](BuiltinFloat64Rules()),
					WithValidation[float64OneOfNoParams](),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				got := strings.TrimSpace(err.Error())
				if !strings.Contains(got, "oneof requires at least one parameter") {
					t.Fatalf("expected parameter error, got: %q", got)
				}
			},
		},
	}

	// TODO: refactor, check the exact error type and messages.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.run(t)
			if tt.checkErr != nil {
				tt.checkErr(t, gotErr)
			}
		})
	}
}
