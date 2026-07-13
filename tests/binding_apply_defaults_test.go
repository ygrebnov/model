package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/pkg/types"
)

func TestBindingApplyDefaults_ScalarsAndPointers(t *testing.T) {
	type config struct {
		S       string          `default:"value"`
		PS      *string         `default:"pointer"`
		I       int             `default:"7"`
		PI      *int            `default:"8"`
		F32     float32         `default:"4.5"`
		PF64    *float64        `default:"8.25"`
		B       bool            `default:"true"`
		PB      *bool           `default:"false"`
		U       uint            `default:"9"`
		PU64    *uint64         `default:"10"`
		UintPtr uintptr         `default:"128"`
		PByte   *byte           `default:"12"`
		Rune    rune            `default:"'Ж'"`
		PRune   *rune           `default:"'λ'"`
		C64     complex64       `default:"3+2i"`
		PC128   *complex128     `default:"6+4i"`
		TD      time.Duration   `default:"3s"`
		PD      *types.Duration `default:"4s"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	checkEqualValue(t, "S", got.S, "value")
	checkEqualPtr(t, "PS", got.PS, pString("pointer"))
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

func TestBindingApplyDefaults_PreservesNonZeroValues(t *testing.T) {
	type config struct {
		S string  `default:"default"`
		I int     `default:"7"`
		B bool    `default:"true"`
		P *string `default:"default-pointer"`
	}

	provided := pString("provided-pointer")
	got := config{
		S: "provided",
		I: 11,
		B: true,
		P: provided,
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	expected := config{
		S: "provided",
		I: 11,
		B: true,
		P: provided,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("ApplyDefaults() result = %#v, want %#v", got, expected)
	}
}

func TestBindingApplyDefaults_ZeroBoolReceivesDefault(t *testing.T) {
	type config struct {
		Enabled bool `default:"true"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if !got.Enabled {
		t.Fatal("Enabled = false, want true")
	}
}

func TestBindingApplyDefaults_NestedStruct(t *testing.T) {
	type nested struct {
		Host string `default:"localhost"`
		Port int    `default:"8080"`
	}

	type config struct {
		Server nested
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	expected := config{
		Server: nested{
			Host: "localhost",
			Port: 8080,
		},
	}

	if got != expected {
		t.Fatalf("ApplyDefaults() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyDefaults_PointerToStructAllocation(t *testing.T) {
	type nested struct {
		Host string `default:"localhost"`
	}

	type config struct {
		Server *nested
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Server == nil {
		t.Fatal("Server = nil, want allocated nested struct")
	}

	if got.Server.Host != "localhost" {
		t.Fatalf(
			"Server.Host = %q, want %q",
			got.Server.Host,
			"localhost",
		)
	}
}

func TestBindingApplyDefaults_PointerToStructWithoutDefaultsRemainsNil(
	t *testing.T,
) {
	type nested struct {
		Host string
	}

	type config struct {
		Server *nested
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Server != nil {
		t.Fatalf("Server = %#v, want nil", got.Server)
	}
}

func TestBindingApplyDefaults_DiveAllocatesPointerToStruct(t *testing.T) {
	type nested struct {
		Host string
	}

	type config struct {
		Server *nested `default:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Server == nil {
		t.Fatal("Server = nil, want allocated zero nested struct")
	}

	if got.Server.Host != "" {
		t.Fatalf(
			"Server.Host = %q, want empty string",
			got.Server.Host,
		)
	}
}

func TestBindingApplyDefaults_AllocInitializesNilCollections(t *testing.T) {
	type config struct {
		Items []string          `default:"alloc"`
		M     map[string]string `default:"alloc"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Items == nil {
		t.Fatal("Items = nil, want allocated empty slice")
	}

	if len(got.Items) != 0 {
		t.Fatalf("len(Items) = %d, want 0", len(got.Items))
	}

	if got.M == nil {
		t.Fatal("M = nil, want allocated empty map")
	}

	if len(got.M) != 0 {
		t.Fatalf("len(M) = %d, want 0", len(got.M))
	}
}

func TestBindingApplyDefaults_AllocPreservesExistingCollections(t *testing.T) {
	type config struct {
		Items []string          `default:"alloc"`
		M     map[string]string `default:"alloc"`
	}

	items := []string{"existing"}
	m := map[string]string{"key": "value"}

	got := config{
		Items: items,
		M:     m,
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if !reflect.DeepEqual(got.Items, items) {
		t.Fatalf("Items = %#v, want %#v", got.Items, items)
	}

	if !reflect.DeepEqual(got.M, m) {
		t.Fatalf("M = %#v, want %#v", got.M, m)
	}
}

func TestBindingApplyDefaults_DefaultElemDiveSlice(t *testing.T) {
	type item struct {
		Name string `default:"default-name"`
		Port int    `default:"8080"`
	}

	type config struct {
		Items []item `defaultElem:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: []item{
			{},
			{
				Name: "provided",
				Port: 9000,
			},
		},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	expected := config{
		Items: []item{
			{
				Name: "default-name",
				Port: 8080,
			},
			{
				Name: "provided",
				Port: 9000,
			},
		},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("ApplyDefaults() result = %#v, want %#v", got, expected)
	}
}

func TestBindingApplyDefaults_DefaultElemDivePointerSlice(t *testing.T) {
	type item struct {
		Name string `default:"default-name"`
	}

	type config struct {
		Items []*item `defaultElem:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: []*item{
			{},
			nil,
			{Name: "provided"},
		},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Items[0] == nil ||
		got.Items[0].Name != "default-name" {
		t.Fatalf(
			"Items[0] = %#v, want defaulted item",
			got.Items[0],
		)
	}

	if got.Items[1] != nil {
		t.Fatalf("Items[1] = %#v, want nil", got.Items[1])
	}

	if got.Items[2] == nil ||
		got.Items[2].Name != "provided" {
		t.Fatalf(
			"Items[2] = %#v, want provided item",
			got.Items[2],
		)
	}
}

func TestBindingApplyDefaults_DefaultElemDiveArray(t *testing.T) {
	type item struct {
		Name string `default:"default-name"`
	}

	type config struct {
		Items [2]item `defaultElem:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: [2]item{
			{},
			{Name: "provided"},
		},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	expected := config{
		Items: [2]item{
			{Name: "default-name"},
			{Name: "provided"},
		},
	}

	if got != expected {
		t.Fatalf("ApplyDefaults() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyDefaults_MapValuesAreTraversed(t *testing.T) {
	type item struct {
		Name string `default:"default-name"`
		Port int    `default:"8080"`
	}

	type config struct {
		Items map[string]item
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: map[string]item{
			"empty": {},
			"provided": {
				Name: "provided",
				Port: 9000,
			},
		},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	expected := config{
		Items: map[string]item{
			"empty": {
				Name: "default-name",
				Port: 8080,
			},
			"provided": {
				Name: "provided",
				Port: 9000,
			},
		},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("ApplyDefaults() result = %#v, want %#v", got, expected)
	}
}

func TestBindingApplyDefaults_MapPointerValuesAreTraversed(t *testing.T) {
	type item struct {
		Name string `default:"default-name"`
	}

	type config struct {
		Items map[string]*item
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: map[string]*item{
			"empty":    {},
			"nil":      nil,
			"provided": {Name: "provided"},
		},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Items["empty"] == nil ||
		got.Items["empty"].Name != "default-name" {
		t.Fatalf(
			"Items[empty] = %#v, want defaulted item",
			got.Items["empty"],
		)
	}

	if got.Items["nil"] != nil {
		t.Fatalf(
			"Items[nil] = %#v, want nil",
			got.Items["nil"],
		)
	}

	if got.Items["provided"] == nil ||
		got.Items["provided"].Name != "provided" {
		t.Fatalf(
			"Items[provided] = %#v, want provided item",
			got.Items["provided"],
		)
	}
}

func TestBindingApplyDefaults_SliceWithoutDefaultElemDiveIsNotTraversed(
	t *testing.T,
) {
	type item struct {
		Name string `default:"default-name"`
	}

	type config struct {
		Items []item
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: []item{{}},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.Items[0].Name != "" {
		t.Fatalf(
			"Items[0].Name = %q, want unchanged empty string",
			got.Items[0].Name,
		)
	}
}

func TestBindingApplyDefaults_UnsupportedDirectiveIsIgnoredForNonLiteralKinds(
	t *testing.T,
) {
	type config struct {
		S     string         `default:"alloc"`
		Items []string       `default:"dive"`
		M     map[string]int `default:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if got.S != "" {
		t.Fatalf("S = %q, want empty string", got.S)
	}

	if got.Items != nil {
		t.Fatalf("Items = %#v, want nil", got.Items)
	}

	if got.M != nil {
		t.Fatalf("M = %#v, want nil", got.M)
	}
}

func TestBindingApplyDefaults_InvalidLiteralReturnsError(t *testing.T) {
	type config struct {
		I int `default:"not-an-int"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		I: 11,
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf(
			"ApplyDefaults() on non-zero field error: %v",
			err,
		)
	}

	if got.I != 11 {
		t.Fatalf("I = %d, want preserved 11", got.I)
	}

	got = config{}

	if err := binding.ApplyDefaults(&got); err == nil {
		t.Fatal("ApplyDefaults() error = nil, want invalid literal error")
	}
}

func TestBindingApplyDefaults_IsIdempotent(t *testing.T) {
	type nested struct {
		Value string `default:"nested"`
	}

	type config struct {
		S      string  `default:"value"`
		P      *string `default:"pointer"`
		Nested *nested
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("first ApplyDefaults() error: %v", err)
	}

	first := config{
		S: got.S,
		P: pString(*got.P),
		Nested: &nested{
			Value: got.Nested.Value,
		},
	}

	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("second ApplyDefaults() error: %v", err)
	}

	if !reflect.DeepEqual(got, first) {
		t.Fatalf(
			"second ApplyDefaults() result = %#v, want %#v",
			got,
			first,
		)
	}
}

func TestBindingApplyDefaults_NilObject(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	if err := binding.ApplyDefaults(nil); err == nil {
		t.Fatal("ApplyDefaults(nil) error = nil, want error")
	}
}
