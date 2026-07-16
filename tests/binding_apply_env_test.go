package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/pkg/types"
)

// TODO: check potential gap between snapshotEnvSource and collection traversal.

func TestBindingApplyEnv_ScalarsAndPointers(t *testing.T) {
	t.Setenv("S", "env-string")
	t.Setenv("PS", "env-pointer")
	t.Setenv("I", "7")
	t.Setenv("PI", "8")
	t.Setenv("F32", "4.5")
	t.Setenv("PF64", "8.25")
	t.Setenv("B", "true")
	t.Setenv("PB", "false")
	t.Setenv("U", "9")
	t.Setenv("PU64", "10")
	t.Setenv("UINTPTR", "128")
	t.Setenv("PBYTE", "12")
	t.Setenv("RUNE", "'Ж'")
	t.Setenv("PRUNE", "'λ'")
	t.Setenv("C64", "3+2i")
	t.Setenv("PC128", "6+4i")
	t.Setenv("TD", "3s")
	t.Setenv("PD", "4s")

	type config struct {
		S       string          `env:"S"`
		PS      *string         `env:"PS"`
		I       int             `env:"I"`
		PI      *int            `env:"PI"`
		F32     float32         `env:"F32"`
		PF64    *float64        `env:"PF64"`
		B       bool            `env:"B"`
		PB      *bool           `env:"PB"`
		U       uint            `env:"U"`
		PU64    *uint64         `env:"PU64"`
		UintPtr uintptr         `env:"UINTPTR"`
		PByte   *byte           `env:"PBYTE"`
		Rune    rune            `env:"RUNE"`
		PRune   *rune           `env:"PRUNE"`
		C64     complex64       `env:"C64"`
		PC128   *complex128     `env:"PC128"`
		TD      time.Duration   `env:"TD"`
		PD      *types.Duration `env:"PD"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	checkEqualValue(t, "S", got.S, "env-string")
	checkEqualPtr(t, "PS", got.PS, pString("env-pointer"))
	checkEqualValue(t, "I", got.I, 7)
	checkEqualPtr(t, "PI", got.PI, pInt(8))
	checkEqualValue(t, "F32", got.F32, float32(4.5))
	checkEqualPtr(t, "PF64", got.PF64, pFloat64(8.25))
	checkEqualValue(t, "B", got.B, true)
	checkEqualPtr(t, "PB", got.PB, pBool(false))
	checkEqualValue(t, "U", got.U, uint(9))
	checkEqualPtr(t, "PU64", got.PU64, pUint64(10))
	checkEqualValue(t, "UintPtr", got.UintPtr, uintptr(128))
	checkEqualPtr(t, "PByte", got.PByte, pByte(12))
	checkEqualValue(t, "Rune", got.Rune, rune('Ж'))
	checkEqualPtr(t, "PRune", got.PRune, pRune('λ'))
	checkEqualValue(t, "C64", got.C64, complex64(3+2i))
	checkEqualPtr(t, "PC128", got.PC128, pComplex128(6+4i))
	checkEqualValue(t, "TD", got.TD, 3*time.Second)
	checkEqualPtr(
		t,
		"PD",
		got.PD,
		pDuration(types.Duration(4*time.Second)),
	)
}

func TestBindingApplyEnv_OverridesExistingValues(t *testing.T) {
	t.Setenv("S", "from-env")
	t.Setenv("I", "9")
	t.Setenv("B", "false")

	type config struct {
		S string `env:"S"`
		I int    `env:"I"`
		B bool   `env:"B"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		S: "provided",
		I: 3,
		B: true,
	}

	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	expected := config{
		S: "from-env",
		I: 9,
		B: false,
	}

	if got != expected {
		t.Fatalf("ApplyEnv() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyEnv_UsesSnapshotCapturedAtConstruction(t *testing.T) {
	t.Setenv("S", "snapshotted")

	type config struct {
		S string `env:"S"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	t.Setenv("S", "changed-after-construction")

	got := config{}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	if got.S != "snapshotted" {
		t.Fatalf("S = %q, want %q", got.S, "snapshotted")
	}
}

func TestBindingApplyEnv_IgnoresVariableCreatedAfterConstruction(t *testing.T) {
	type config struct {
		S string `env:"BINDING_APPLY_ENV_LATE_VALUE"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	t.Setenv("BINDING_APPLY_ENV_LATE_VALUE", "late")

	got := config{S: "original"}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	if got.S != "original" {
		t.Fatalf("S = %q, want %q", got.S, "original")
	}
}

func TestBindingApplyEnv_NestedStruct(t *testing.T) {
	t.Setenv("STRINGS_S", "nested")
	t.Setenv("STRINGS_PS", "nested-pointer")

	type config struct {
		Strings Strings `env:"STRINGS"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	checkEqualStrings(t, got.Strings, Strings{
		S:  "nested",
		PS: pString("nested-pointer"),
	})
}

func TestBindingApplyEnv_PointerToStructAllocation(t *testing.T) {
	tests := []struct {
		name     string
		setEnv   func(t *testing.T)
		expected *Strings
	}{
		{
			name: "allocated when descendant value exists",
			setEnv: func(t *testing.T) {
				t.Setenv("PSTRINGS_S", "nested")
			},
			expected: &Strings{S: "nested"},
		},
		{
			name:     "remains nil when no descendant value exists",
			setEnv:   func(*testing.T) {},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			type config struct {
				PStrings *Strings `env:"PSTRINGS"`
			}

			tc.setEnv(t)

			binding, err := model.NewBinding[config]()
			if err != nil {
				t.Fatalf("NewBinding() error: %v", err)
			}

			got := config{}
			if err := binding.ApplyEnv(&got); err != nil {
				t.Fatalf("ApplyEnv() error: %v", err)
			}

			checkEqualPStrings(t, got.PStrings, tc.expected)
		})
	}
}

func TestBindingApplyEnv_Collections(t *testing.T) {
	t.Setenv("A_0_S", "slice-zero")
	t.Setenv("A_1_PS", "slice-one-pointer")
	t.Setenv("AP_0_S", "pointer-slice-zero")
	t.Setenv("M_FIRST_S", "map-first")
	t.Setenv("MP_SECOND_PS", "map-second-pointer")

	type config struct {
		A  []Strings           `env:"A"`
		AP []*Strings          `env:"AP"`
		M  map[string]Strings  `env:"M"`
		MP map[string]*Strings `env:"MP"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		A: []Strings{
			{S: "old-zero"},
			{S: "old-one"},
		},
		AP: []*Strings{
			{S: "old-pointer"},
			nil,
		},
		M: map[string]Strings{
			"first": {S: "old-map"},
		},
		MP: map[string]*Strings{
			"second": {S: "old-pointer-map"},
			"nil":    nil,
		},
	}

	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	expected := config{
		A: []Strings{
			{S: "slice-zero"},
			{
				S:  "old-one",
				PS: pString("slice-one-pointer"),
			},
		},
		AP: []*Strings{
			{S: "pointer-slice-zero"},
			nil,
		},
		M: map[string]Strings{
			"first": {S: "map-first"},
		},
		MP: map[string]*Strings{
			"second": {
				S:  "old-pointer-map",
				PS: pString("map-second-pointer"),
			},
			"nil": nil,
		},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf(
			"ApplyEnv() result = %#v, want %#v",
			got,
			expected,
		)
	}
}

func TestBindingApplyEnv_EnvPrefix(t *testing.T) {
	t.Setenv("APP_STRINGS_S", "prefixed")
	t.Setenv("STRINGS_S", "unprefixed")

	binding, err := model.NewBinding[EnvPrefix](
		model.WithEnvPrefix("APP"),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := EnvPrefix{}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	checkEqualEnvPrefix(t, got, EnvPrefix{
		Strings: Strings{S: "prefixed"},
	})
}

func TestBindingApplyEnv_EnvDisabled(t *testing.T) {
	t.Setenv("S", "must-not-be-used")

	binding, err := model.NewBinding[EnvDisabled]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := EnvDisabled{S: "original"}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	checkEqualEnvDisabled(t, got, EnvDisabled{
		S: "original",
	})
}

func TestBindingApplyEnv_EnvDisabledParentDisablesDescendants(
	t *testing.T,
) {
	t.Setenv("PARENT_S", "must-not-be-used")
	t.Setenv("S", "must-not-be-used-either")

	type config struct {
		Parent Strings `env:"-"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Parent: Strings{S: "original"},
	}

	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	checkEqualStrings(t, got.Parent, Strings{
		S: "original",
	})
}

func TestBindingApplyEnv_UsesFieldNameWithoutEnvTag(t *testing.T) {
	t.Setenv("VALUE", "fallback")

	type config struct {
		Value string
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	if got.Value != "fallback" {
		t.Fatalf("Value = %q, want %q", got.Value, "fallback")
	}
}

func TestBindingApplyEnv_NamedScalarTypes(t *testing.T) {
	t.Setenv("S", "named")
	t.Setenv("I", "7")
	t.Setenv("B", "true")

	type config struct {
		S CustomString `env:"S"`
		I CustomInt    `env:"I"`
		B CustomBool   `env:"B"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	expected := config{
		S: "named",
		I: 7,
		B: true,
	}

	if got != expected {
		t.Fatalf("ApplyEnv() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyEnv_UnsupportedFieldsAreIgnored(t *testing.T) {
	t.Setenv("INTERFACE", "ignored")
	t.Setenv("M", "ignored")

	type config struct {
		Interface interface{}    `env:"INTERFACE"`
		M         map[string]int `env:"M"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Interface: "original",
		M:         map[string]int{"one": 1},
	}

	if err := binding.ApplyEnv(&got); err != nil {
		t.Fatalf("ApplyEnv() error: %v", err)
	}

	if got.Interface != "original" {
		t.Fatalf(
			"Interface = %#v, want %q",
			got.Interface,
			"original",
		)
	}

	if len(got.M) != 1 || got.M["one"] != 1 {
		t.Fatalf("M = %#v, want unchanged map", got.M)
	}
}

func TestBindingApplyEnv_InvalidLiteralReturnsErrorAndPreservesField(
	t *testing.T,
) {
	t.Setenv("I", "not-an-int")

	binding, err := model.NewBinding[EnvInvalidInt]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := EnvInvalidInt{I: 11}

	err = binding.ApplyEnv(&got)
	if err == nil {
		t.Fatal("ApplyEnv() error = nil, want parsing error")
	}

	if got.I != 11 {
		t.Fatalf("I = %d, want original value 11", got.I)
	}
}

func TestBindingApplyEnv_NilObject(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	if err := binding.ApplyEnv(nil); err == nil {
		t.Fatal("ApplyEnv(nil) error = nil, want error")
	}
}
