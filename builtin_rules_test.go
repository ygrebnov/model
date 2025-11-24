package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ygrebnov/errorc"
	modelerrors "github.com/ygrebnov/model/errors"
)

func assertRuleErrorHas(t *testing.T, err, wantSentinel error, wantRule string, kv map[errorc.Key]string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantSentinel) {
		t.Fatalf("expected sentinel %v, got %v", wantSentinel, err)
	}
	msg := err.Error()
	if wantRule != "" {
		needle := string(modelerrors.ErrorFieldRuleName) + ": " + wantRule
		if !strings.Contains(msg, needle) {
			t.Fatalf("expected rule name %q in error, got %q", wantRule, msg)
		}
	}
	for k, v := range kv {
		needle := string(k) + ": " + v
		if !strings.Contains(msg, needle) {
			t.Fatalf("expected %q in error, got %q", needle, msg)
		}
	}
}

func TestBuiltinRules_WithValidation_Nominal(t *testing.T) {
	type strOK struct {
		S string `validate:"min(1),max(10)"`
	}
	type intOK struct {
		NMin int `validate:"min(1)"`
		NMax int `validate:"max(10)"`
		NZ   int `validate:"nonzero"`
	}
	type int64OK struct {
		NMin int64 `validate:"min(1)"`
		NMax int64 `validate:"max(10)"`
		NZ   int64 `validate:"nonzero"`
	}
	type float64OK struct {
		NMin float64 `validate:"min(0.5)"`
		NMax float64 `validate:"max(2.5)"`
		NZ   float64 `validate:"nonzero"`
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
			name: "string min/max pass",
			run: func(t *testing.T) error {
				obj := strOK{S: "ok"}
				_, err := New(&obj, WithValidation[strOK](context.Background()))
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "string max fails when too long",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOK{S: "this string is definitely too long"}
				_, err := New(&obj, WithValidation[strOK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "max", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "length",
					modelerrors.ErrorFieldRuleParamValue: "10",
				})
			},
		},
		{
			name: "int min/max/nonzero pass",
			run: func(t *testing.T) error {
				obj := intOK{NMin: 5, NMax: 5, NZ: 1}
				_, err := New(&obj, WithValidation[intOK](context.Background()))
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int min fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOK{NMin: 0}
				_, err := New(&obj, WithValidation[intOK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "min", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "1",
				})
			},
		},
		{
			name:      "int max fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOK{NMax: 20}
				_, err := New(&obj, WithValidation[intOK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "max", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "10",
				})
			},
		},
		{
			name: "int64 min/max/nonzero pass",
			run: func(t *testing.T) error {
				obj := int64OK{NMin: 5, NMax: 5, NZ: 1}
				_, err := New(&obj, WithValidation[int64OK](context.Background()))
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int64 min fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OK{NMin: 0}
				_, err := New(&obj, WithValidation[int64OK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "min", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "1",
				})
			},
		},
		{
			name:      "int64 max fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OK{NMax: 20}
				_, err := New(&obj, WithValidation[int64OK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "max", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "10",
				})
			},
		},
		{
			name: "float64 min/max/nonzero pass",
			run: func(t *testing.T) error {
				obj := float64OK{NMin: 1.5, NMax: 1.5, NZ: 2.3}
				_, err := New(&obj, WithValidation[float64OK](context.Background()))
				if err != nil {
					t.Fatalf("New returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "float64 min fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OK{NMin: 0.1}
				_, err := New(&obj, WithValidation[float64OK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "min", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "0.5",
				})
			},
		},
		{
			name:      "float64 max fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OK{NMax: 3.0}
				_, err := New(&obj, WithValidation[float64OK](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "max", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "2.5",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "allowed",
					modelerrors.ErrorFieldRuleParamValue: "red,green,blue",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleMissingParameter, "oneof", nil)
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "allowed",
					modelerrors.ErrorFieldRuleParamValue: "1,2,3",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleMissingParameter, "oneof", nil)
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleInvalidParameter, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "a",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "allowed",
					modelerrors.ErrorFieldRuleParamValue: "10,20,30",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleMissingParameter, "oneof", nil)
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleInvalidParameter, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "a",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "allowed",
					modelerrors.ErrorFieldRuleParamValue: "0.5,1.0,2.5",
				})
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleMissingParameter, "oneof", nil)
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
				assertRuleErrorHas(t, err, modelerrors.ErrRuleInvalidParameter, "oneof", map[errorc.Key]string{
					modelerrors.ErrorFieldRuleParamName:  "value",
					modelerrors.ErrorFieldRuleParamValue: "a",
				})
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

func TestWithValidation_BuiltinsRemainValid_NoError(t *testing.T) {
	type Obj struct{ S string }
	obj := Obj{}
	if _, err := New(&obj, WithValidation[Obj](context.Background())); err != nil {
		t.Fatalf("WithValidation should not error for valid builtins, got: %v", err)
	}
}

func TestBuiltinStringEmailAndUUID(t *testing.T) {
	type emailUUID struct {
		Email string `validate:"email"`
		ID    string `validate:"uuid"`
	}

	tests := []struct {
		name      string
		obj       emailUUID
		wantError bool
		checkErr  func(t *testing.T, err error)
	}{
		{
			name: "valid email and uuid pass",
			obj: emailUUID{
				Email: "user@example.com",
				ID:    "123e4567-e89b-12d3-a456-426614174000",
			},
		},
		{
			name: "invalid email fails",
			obj: emailUUID{
				Email: "not-an-email",
				ID:    "123e4567-e89b-12d3-a456-426614174000",
			},
			wantError: true,
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				// Expect constraint violation with rule name email.
				if !errors.Is(err, modelerrors.ErrRuleConstraintViolated) {
					t.Fatalf("expected ErrRuleConstraintViolated, got %v", err)
				}
				msg := err.Error()
				if !strings.Contains(msg, string(modelerrors.ErrorFieldRuleName)+": email") {
					t.Fatalf("expected email rule metadata in error, got %q", msg)
				}
			},
		},
		{
			name: "invalid uuid fails",
			obj: emailUUID{
				Email: "user@example.com",
				ID:    "not-a-uuid",
			},
			wantError: true,
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, modelerrors.ErrRuleConstraintViolated) {
					t.Fatalf("expected ErrRuleConstraintViolated, got %v", err)
				}
				msg := err.Error()
				if !strings.Contains(msg, string(modelerrors.ErrorFieldRuleName)+": uuid") {
					t.Fatalf("expected uuid rule metadata in error, got %q", msg)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := tt.obj
			_, err := New(&obj, WithValidation[emailUUID](context.Background()))
			if tt.wantError && err == nil {
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
