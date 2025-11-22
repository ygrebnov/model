package rules

import (
	"context"
	"strings"
	"testing"
)

func TestBuiltinRules_WithValidation_Nominal(t *testing.T) {

	type strOK struct {
		S string `validate:"min(1)"`
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
			name: "string min(1) passes",
			run: func(t *testing.T) error {
				obj := strOK{S: "ok"}
				_, err := New(
					&obj,
					WithValidation[strOK](context.Background()),
				)
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "string min(1) fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOK{S: ""}
				_, err := New(
					&obj,
					WithValidation[strOK](context.Background()),
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
				if !strings.Contains(got, "length must be >= 1") {
					t.Fatalf("expected min length failure, got: %q", got)
				}
			},
		},
		{
			name: "int positive & nonzero pass",
			run: func(t *testing.T) error {
				obj := intOK{P: 1, NZ: 1}
				_, err := New(
					&obj,
					WithValidation[intOK](context.Background()),
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
					WithValidation[int64OK](context.Background()),
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
					WithValidation[float64OK](context.Background()),
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
					WithValidation[strOneOfOK](context.Background()),
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
					WithValidation[strOneOfBad](context.Background()),
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
				if !strings.Contains(got, "must be one of:") {
					t.Fatalf("expected oneof failure, got: %q", got)
				}
			},
		},
		{
			name:      "string oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOneOfNoParams{S: "x"}
				_, err := New(&obj, WithValidation[strOneOfNoParams](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "requires at least one parameter") {
					t.Fatalf("expected missing params error, got %v", err)
				}
			},
		},

		// --- oneof: int ---
		{
			name: "int oneof passes",
			run: func(t *testing.T) error {
				obj := intOneOfOK{N: 2}
				_, err := New(&obj, WithValidation[intOneOfOK](context.Background()))
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
				obj := intOneOfBad{N: 5}
				_, err := New(&obj, WithValidation[intOneOfBad](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "must be one of") {
					t.Fatalf("expected oneof failure, got %v", err)
				}
			},
		},
		{
			name:      "int oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOfNoParams{N: 1}
				_, err := New(&obj, WithValidation[intOneOfNoParams](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "requires at least one parameter") {
					t.Fatalf("expected missing params error, got %v", err)
				}
			},
		},
		{
			name:      "int oneof bad param type",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOfBadParam{N: 2} // use value not matching first valid param to reach invalid 'a'
				_, err := New(&obj, WithValidation[intOneOfBadParam](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "invalid oneof parameter") {
					t.Fatalf("expected invalid param error, got %v", err)
				}
			},
		},

		// --- oneof: int64 ---
		{
			name: "int64 oneof passes",
			run: func(t *testing.T) error {
				obj := int64OneOfOK{N: 20}
				_, err := New(&obj, WithValidation[int64OneOfOK](context.Background()))
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
				obj := int64OneOfBad{N: 5}
				_, err := New(&obj, WithValidation[int64OneOfBad](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "must be one of") {
					t.Fatalf("expected oneof failure, got %v", err)
				}
			},
		},
		{
			name:      "int64 oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOfNoParams{N: 1}
				_, err := New(&obj, WithValidation[int64OneOfNoParams](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "requires at least one parameter") {
					t.Fatalf("expected missing params error, got %v", err)
				}
			},
		},
		{
			name:      "int64 oneof bad param type",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOfBadParam{N: 1}
				_, err := New(&obj, WithValidation[int64OneOfBadParam](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "invalid oneof parameter") {
					t.Fatalf("expected invalid param error, got %v", err)
				}
			},
		},

		// --- oneof: float64 ---
		{
			name: "float64 oneof passes",
			run: func(t *testing.T) error {
				obj := float64OneOfOK{F: 1.0}
				_, err := New(&obj, WithValidation[float64OneOfOK](context.Background()))
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
				obj := float64OneOfBad{F: 3.3}
				_, err := New(&obj, WithValidation[float64OneOfBad](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "must be one of") {
					t.Fatalf("expected oneof failure, got %v", err)
				}
			},
		},
		{
			name:      "float64 oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOfNoParams{F: 1.0}
				_, err := New(&obj, WithValidation[float64OneOfNoParams](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "requires at least one parameter") {
					t.Fatalf("expected missing params error, got %v", err)
				}
			},
		},
		{
			name:      "float64 oneof bad param type",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOfBadParam{F: 1.0}
				_, err := New(&obj, WithValidation[float64OneOfBadParam](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "invalid oneof parameter") {
					t.Fatalf("expected invalid param error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			if tt.wantError && err == nil {
				// error expected
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkErr != nil {
				tt.checkErr(t, err)
			}
		})
	}
}
