package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	keysLib "github.com/ygrebnov/keys"
	"github.com/ygrebnov/model/constants"
	modelerrors "github.com/ygrebnov/model/errors"
	"github.com/ygrebnov/model/keys"
	"github.com/ygrebnov/model/validation"
)

func assertRuleErrorHas(t *testing.T, err, wantSentinel error, wantRule string, kv map[keysLib.Key]string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantSentinel) {
		t.Fatalf("expected sentinel %v, got %v", wantSentinel, err)
	}

	var ve *validation.Error
	if !errors.As(err, &ve) {
		t.Fatalf("expected *validation.Error, got %T: %v", err, err)
	}

	for _, field := range ve.Fields() {
		for _, fe := range ve.ForField(field) {
			if !errors.Is(fe.Err, wantSentinel) {
				continue
			}

			msg := fe.Err.Error()
			if wantRule != "" {
				needle := string(keys.RuleName) + ": " + wantRule
				if !strings.Contains(msg, needle) {
					continue
				}
			}

			matched := true
			for k, v := range kv {
				needle := string(k) + ": " + v
				if !strings.Contains(msg, needle) {
					matched = false
					break
				}
			}
			if matched {
				return
			}
		}
	}

	t.Fatalf("expected structured field error with sentinel %v, rule %q and kv %+v, got %v", wantSentinel, wantRule, kv, err)
}

func firstFieldErrorFor(t *testing.T, err error, path string) validation.FieldError {
	t.Helper()

	var ve *validation.Error
	if !errors.As(err, &ve) {
		t.Fatalf("expected *validation.Error, got %T: %v", err, err)
	}

	es := ve.ByField()[path]
	if len(es) == 0 {
		t.Fatalf("expected field error for %q, got %v", path, ve.ByField())
	}

	return es[0]
}

func assertConstraintViolation(t *testing.T, err error, ruleName, paramName, paramValue string) {
	t.Helper()

	assertRuleErrorHas(t, err, modelerrors.ErrRuleConstraintViolated, ruleName, map[keysLib.Key]string{
		keys.RuleParamName:  paramName,
		keys.RuleParamValue: paramValue,
	})
}

func assertMissingParameter(t *testing.T, err error) {
	t.Helper()

	assertRuleErrorHas(t, err, modelerrors.ErrRuleMissingParameter, constants.RuleOneOf, nil)
}

func assertInvalidParameter(t *testing.T, err error, ruleName, paramName, paramValue string) {
	t.Helper()

	assertRuleErrorHas(t, err, modelerrors.ErrRuleInvalidParameter, ruleName, map[keysLib.Key]string{
		keys.RuleParamName:  paramName,
		keys.RuleParamValue: paramValue,
	})
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
	type strOneOf struct {
		S string `validate:"oneof(red,green,blue)"`
	}
	type strOneOfNoParams struct {
		S string `validate:"oneof()"`
	}

	type intOneOf struct {
		N int `validate:"oneof(1,2,3)"`
	}
	type intOneOfNoParams struct {
		N int `validate:"oneof()"`
	}
	type intOneOfBadParam struct {
		N int `validate:"oneof(1,a,3)"`
	}

	type int64OneOf struct {
		N int64 `validate:"oneof(10,20,30)"`
	}
	type int64OneOfNoParams struct {
		N int64 `validate:"oneof()"`
	}
	type int64OneOfBadParam struct {
		N int64 `validate:"oneof(10,a,30)"`
	}

	type float64OneOf struct {
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
				assertConstraintViolation(t, err, constants.RuleMax, "length", "10")
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
				assertConstraintViolation(t, err, constants.RuleMin, "value", "1")
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
				assertConstraintViolation(t, err, constants.RuleMax, "value", "10")
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
				assertConstraintViolation(t, err, constants.RuleMin, "value", "1")
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
				assertConstraintViolation(t, err, constants.RuleMax, "value", "10")
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
				assertConstraintViolation(t, err, constants.RuleMin, "value", "0.5")
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
				assertConstraintViolation(t, err, constants.RuleMax, "value", "2.5")
			},
		},

		// --- oneof: string ---
		{
			name: "string oneof passes",
			run: func(t *testing.T) error {
				obj := strOneOf{S: "green"}
				_, err := New(
					&obj,
					WithValidation[strOneOf](context.Background()),
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
				obj := strOneOf{S: "yellow"}
				_, err := New(
					&obj,
					WithValidation[strOneOf](context.Background()),
				)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, constants.RuleOneOf, "allowed", "red,green,blue")
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
				assertMissingParameter(t, err)
			},
		},

		// --- oneof: int ---
		{
			name: "int oneof passes",
			run: func(t *testing.T) error {
				obj := intOneOf{N: 2}
				_, err := New(&obj, WithValidation[intOneOf](context.Background()))
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
				obj := intOneOf{N: 5}
				_, err := New(&obj, WithValidation[intOneOf](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, constants.RuleOneOf, "allowed", "1,2,3")
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
				assertMissingParameter(t, err)
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
				assertInvalidParameter(t, err, constants.RuleOneOf, "value", "a")
			},
		},

		// --- oneof: int64 ---
		{
			name: "int64 oneof passes",
			run: func(t *testing.T) error {
				obj := int64OneOf{N: 20}
				_, err := New(&obj, WithValidation[int64OneOf](context.Background()))
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
				obj := int64OneOf{N: 5}
				_, err := New(&obj, WithValidation[int64OneOf](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, constants.RuleOneOf, "allowed", "10,20,30")
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
				assertMissingParameter(t, err)
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
				assertInvalidParameter(t, err, constants.RuleOneOf, "value", "a")
			},
		},

		// --- oneof: float64 ---
		{
			name: "float64 oneof passes",
			run: func(t *testing.T) error {
				obj := float64OneOf{F: 1.0}
				_, err := New(&obj, WithValidation[float64OneOf](context.Background()))
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
				obj := float64OneOf{F: 3.3}
				_, err := New(&obj, WithValidation[float64OneOf](context.Background()))
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, constants.RuleOneOf, "allowed", "0.5,1.0,2.5")
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
				assertMissingParameter(t, err)
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
				assertInvalidParameter(t, err, constants.RuleOneOf, "value", "a")
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
				fe := firstFieldErrorFor(t, err, "Email")
				if !errors.Is(fe.Err, modelerrors.ErrRuleConstraintViolated) {
					t.Fatalf("expected ErrRuleConstraintViolated, got %v", fe.Err)
				}
				msg := fe.Err.Error()
				if !strings.Contains(msg, string(keys.RuleName)+": email") {
					t.Fatalf("expected email rule metadata in field error, got %q", msg)
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
				fe := firstFieldErrorFor(t, err, "ID")
				if !errors.Is(fe.Err, modelerrors.ErrRuleConstraintViolated) {
					t.Fatalf("expected ErrRuleConstraintViolated, got %v", fe.Err)
				}
				msg := fe.Err.Error()
				if !strings.Contains(msg, string(keys.RuleName)+": uuid") {
					t.Fatalf("expected uuid rule metadata in field error, got %q", msg)
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
