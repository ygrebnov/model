package tests

import (
	"context"
	nativeerrors "errors"
	"strings"
	"testing"

	keysLib "github.com/ygrebnov/keys"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/internal/rules"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
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
	if !nativeerrors.As(err, &ve) {
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
	if !nativeerrors.As(err, &ve) {
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

	assertRuleErrorHas(t, err, errors.ErrRuleConstraintViolated, ruleName, map[keysLib.Key]string{
		keys.RuleParamName:  paramName,
		keys.RuleParamValue: paramValue,
	})
}

func assertMissingParameter(t *testing.T, err error) {
	t.Helper()

	assertRuleErrorHas(t, err, errors.ErrRuleMissingParameter, rules.RuleOneOf, nil)
}

func assertInvalidParameter(t *testing.T, err error, ruleName, paramName, paramValue string) {
	t.Helper()

	assertRuleErrorHas(t, err, errors.ErrRuleInvalidParameter, ruleName, map[keysLib.Key]string{
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
				if err := model.Validate(context.Background(), &obj); err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "string max fails when too long",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOK{S: "this string is definitely too long"}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				var vErr *validation.Error
				if !nativeerrors.As(err, &vErr) {
					t.Fatalf("expected Validate to return ValidationError, but got %T: %v", err, err)
				}

				if vErr.Len() != 1 {
					t.Fatalf("expected 1 field error, but got %d", vErr.Len())
				}
				fErr := vErr.ForField("S")
				if len(fErr) != 1 {
					t.Fatalf("expected 1 field error, but got %d", len(fErr))
				}
				if fErr[0].Path != "S" {
					t.Fatalf("expected field Path S, but got %s", fErr[0].Path)
				}
				if fErr[0].Err == nil || !errors.Is(fErr[0].Err, errors.ErrRuleConstraintViolated) {
					t.Fatalf("expected ErrRuleConstraintViolated, but got %v", fErr[0].Err)
				}
				expectedFErr := "rule constraint violated, rule.name: max, rule.param.name: length, rule.param.value: 10"
				if fErr[0].Err.Error() != expectedFErr {
					t.Fatalf("expected error message %q, but got %q", expectedFErr, fErr[0].Err.Error())
				}
			},
		},
		{
			name: "int min/max/nonzero pass",
			run: func(t *testing.T) error {
				obj := intOK{NMin: 5, NMax: 5, NZ: 1}
				if err := model.Validate(context.Background(), &obj); err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int min fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOK{NMin: 0}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleMin, "value", "1")
			},
		},
		{
			name:      "int max fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOK{NMax: 20}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleMax, "value", "10")
			},
		},
		{
			name: "int64 min/max/nonzero pass",
			run: func(t *testing.T) error {
				obj := int64OK{NMin: 5, NMax: 5, NZ: 1}
				if err := model.Validate(context.Background(), &obj); err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int64 min fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OK{NMin: 0}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleMin, "value", "1")
			},
		},
		{
			name:      "int64 max fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OK{NMax: 20}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleMax, "value", "10")
			},
		},
		{
			name: "float64 min/max/nonzero pass",
			run: func(t *testing.T) error {
				obj := float64OK{NMin: 1.5, NMax: 1.5, NZ: 2.3}
				err := model.Validate(context.Background(), &obj)
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
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleMin, "value", "0.5")
			},
		},
		{
			name:      "float64 max fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OK{NMax: 3.0}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleMax, "value", "2.5")
			},
		},

		// --- oneof: string ---
		{
			name: "string oneof passes",
			run: func(t *testing.T) error {
				obj := strOneOf{S: "green"}
				err := model.Validate(context.Background(), &obj)
				if err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "string oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOneOf{S: "yellow"}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleOneOf, "allowed", "red,green,blue")
			},
		},
		{
			name:      "string oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := strOneOfNoParams{S: "x"}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
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
				err := model.Validate(context.Background(), &obj)
				if err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOf{N: 5}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleOneOf, "allowed", "1,2,3")
			},
		},
		{
			name:      "int oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := intOneOfNoParams{N: 1}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
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
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertInvalidParameter(t, err, rules.RuleOneOf, "value", "a")
			},
		},

		// --- oneof: int64 ---
		{
			name: "int64 oneof passes",
			run: func(t *testing.T) error {
				obj := int64OneOf{N: 20}
				err := model.Validate(context.Background(), &obj)
				if err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "int64 oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOf{N: 5}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleOneOf, "allowed", "10,20,30")
			},
		},
		{
			name:      "int64 oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := int64OneOfNoParams{N: 1}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
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
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertInvalidParameter(t, err, rules.RuleOneOf, "value", "a")
			},
		},

		// --- oneof: float64 ---
		{
			name: "float64 oneof passes",
			run: func(t *testing.T) error {
				obj := float64OneOf{F: 1.0}
				err := model.Validate(context.Background(), &obj)
				if err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return nil
			},
		},
		{
			name:      "float64 oneof fails",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOf{F: 3.3}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertConstraintViolation(t, err, rules.RuleOneOf, "allowed", "0.5,1.0,2.5")
			},
		},
		{
			name:      "float64 oneof bad params",
			wantError: true,
			run: func(t *testing.T) error {
				obj := float64OneOfNoParams{F: 1.0}
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
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
				err := model.Validate(context.Background(), &obj)
				if err == nil {
					t.Fatalf("expected Validate to return an error, but got nil")
				}
				return err
			},
			checkErr: func(t *testing.T, err error) {
				assertInvalidParameter(t, err, rules.RuleOneOf, "value", "a")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			if tt.wantError && err == nil {
				// error expected
				t.Fatalf("expected error, but got nil")
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

func TestBuiltinRules_WithValidation_ExtendedNumericCoverage(t *testing.T) {
	type numericCoverageOK struct {
		I8    int8    `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		I16   int16   `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		I32   int32   `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		U     uint    `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		U8    uint8   `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		U16   uint16  `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		U32   uint32  `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		U64   uint64  `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		UP    uintptr `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		F32   float32 `validate:"min(0.5),max(2.5),nonzero,oneof(0.5,1.5,2.5)"`
		PRune rune    `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
		PByte byte    `validate:"min(3),max(10),nonzero,oneof(3,4,5)"`
	}

	t.Run("pass", func(t *testing.T) {
		obj := numericCoverageOK{
			I8: 5, I16: 5, I32: 5,
			U: 5, U8: 5, U16: 5, U32: 5, U64: 5, UP: 5,
			F32:   1.5,
			PRune: 5,
			PByte: 5,
		}

		if err := model.Validate(context.Background(), &obj); err != nil {
			t.Fatalf("Validate returned error: %v", err)
		}
	})

	t.Run("signed integer rule resolves for int8", func(t *testing.T) {
		type signedFail struct {
			V int8 `validate:"min(3)"`
		}

		err := model.Validate(context.Background(), &signedFail{V: 2})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		assertConstraintViolation(t, err, rules.RuleMin, "value", "3")
	})

	t.Run("unsigned integer rule resolves for uintptr", func(t *testing.T) {
		type unsignedFail struct {
			V uintptr `validate:"oneof(3,4,5)"`
		}

		err := model.Validate(context.Background(), &unsignedFail{V: 6})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		assertConstraintViolation(t, err, rules.RuleOneOf, "allowed", "3,4,5")
	})

	t.Run("float rule resolves for float32", func(t *testing.T) {
		type floatFail struct {
			V float32 `validate:"max(2.5)"`
		}

		err := model.Validate(context.Background(), &floatFail{V: 3.5})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		assertConstraintViolation(t, err, rules.RuleMax, "value", "2.5")
	})
}

func TestWithValidation_BuiltinsRemainValid_NoError(t *testing.T) {
	type Obj struct{ S string }
	obj := Obj{}
	if err := model.Validate(context.Background(), &obj); err != nil {
		t.Fatalf("Validate should not error for valid builtins, but got: %v", err)
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
				if !errors.Is(fe.Err, errors.ErrRuleConstraintViolated) {
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
				if !errors.Is(fe.Err, errors.ErrRuleConstraintViolated) {
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
			err := model.Validate(context.Background(), &obj)
			if tt.wantError && err == nil {
				t.Fatalf("expected Validate to return an error, but got nil")
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

func TestBuiltinStringSemver(t *testing.T) {
	type semverStruct struct {
		Semver string `validate:"semver"`
	}

	tests := []struct {
		name      string
		obj       semverStruct
		wantError bool
		checkErr  func(t *testing.T, err error)
	}{
		{
			name: "1.0.0",
			obj: semverStruct{
				Semver: "1.0.0",
			},
		},
		{
			name: "1.0.0-alpha",
			obj: semverStruct{
				Semver: "1.0.0-alpha",
			},
		},
		{
			name: "1.0.0-alpha.1",
			obj: semverStruct{
				Semver: "1.0.0-alpha.1",
			},
		},
		{
			name: "1.0.0-0.3.7",
			obj: semverStruct{
				Semver: "1.0.0-0.3.7",
			},
		},
		{
			name: "1.0.0-x.7.z.92",
			obj: semverStruct{
				Semver: "1.0.0-x.7.z.92",
			},
		},
		{
			name: "1.0.0-x-y-z.--",
			obj: semverStruct{
				Semver: "1.0.0-x-y-z.--",
			},
		},
		{
			name: "1.0.0-alpha+001",
			obj: semverStruct{
				Semver: "1.0.0-alpha+001",
			},
		},
		{
			name: "1.0.0+20130313144700",
			obj: semverStruct{
				Semver: "1.0.0+20130313144700",
			},
		},
		{
			name: "1.0.0-beta+exp.sha.5114f85",
			obj: semverStruct{
				Semver: "1.0.0-beta+exp.sha.5114f85",
			},
		},
		{
			name: "1.0.0+21AF26D3----117B344092BD",
			obj: semverStruct{
				Semver: "1.0.0+21AF26D3----117B344092BD",
			},
		},
		{
			name: "invalid semver fails",
			obj: semverStruct{
				Semver: "v1.0.0",
			},
			wantError: true,
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				fe := firstFieldErrorFor(t, err, "Semver")
				if !errors.Is(fe.Err, errors.ErrRuleConstraintViolated) {
					t.Fatalf("expected ErrRuleConstraintViolated, got %v", fe.Err)
				}
				msg := fe.Err.Error()
				if !strings.Contains(msg, string(keys.RuleName)+": semver") {
					t.Fatalf("expected semver rule metadata in field error, got %q", msg)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := tt.obj
			err := model.Validate(context.Background(), &obj)
			if tt.wantError && err == nil {
				t.Fatalf("expected Validate to return an error, but got nil")
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
