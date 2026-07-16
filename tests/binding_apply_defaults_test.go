package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/pkg/errors"
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
		PTD     *time.Duration  `default:"250ms"`
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
	if got.PTD == nil || *got.PTD != 250*time.Millisecond {
		t.Fatalf("PTD = %#v, want pointer to 250ms", got.PTD)
	}
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

func TestBindingApplyDefaults_PointerToStructWithDefaultsRemainsNil(
	t *testing.T,
) {
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

	if got.Server != nil {
		t.Fatalf(
			"Server = %#v, want nil without default:\"dive\"",
			got.Server,
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

func TestBindingApplyDefaults_BooleanLiteralVariants(t *testing.T) {
	type config struct {
		One  bool `default:"1"`
		True bool `default:"true"`
		T    bool `default:"t"`
		Yes  bool `default:"yes"`
		Y    bool `default:"y"`
		On   bool `default:"on"`

		Zero  *bool `default:"0"`
		False *bool `default:"false"`
		F     *bool `default:"f"`
		No    *bool `default:"no"`
		N     *bool `default:"n"`
		Off   *bool `default:"off"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyDefaults(&got); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if !got.One || !got.True || !got.T || !got.Yes || !got.Y || !got.On {
		t.Fatalf("true boolean literals were not all applied: %#v", got)
	}

	for name, value := range map[string]*bool{
		"Zero":  got.Zero,
		"False": got.False,
		"F":     got.F,
		"No":    got.No,
		"N":     got.N,
		"Off":   got.Off,
	} {
		if value == nil || *value {
			t.Fatalf("%s = %#v, want pointer to false", name, value)
		}
	}

	type invalidConfig struct {
		Value bool `default:"maybe"`
	}

	invalidBinding, err := model.NewBinding[invalidConfig]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	if err := invalidBinding.ApplyDefaults(&invalidConfig{}); !errors.Is(err, errors.ErrSetDefault) {
		t.Fatalf("ApplyDefaults() error = %v, want %v", err, errors.ErrSetDefault)
	}
}

func TestBindingApplyDefaults_InvalidUintAndFloatLiterals(t *testing.T) {
	t.Run("uint rejects negative literal", func(t *testing.T) {
		type config struct {
			Value uint `default:"-1"`
		}

		binding, err := model.NewBinding[config]()
		if err != nil {
			t.Fatalf("NewBinding() error: %v", err)
		}

		got := config{}
		err = binding.ApplyDefaults(&got)
		if !errors.Is(err, errors.ErrSetDefault) {
			t.Fatalf("ApplyDefaults() error = %v, want %v", err, errors.ErrSetDefault)
		}
		if got.Value != 0 {
			t.Fatalf("Value = %d, want 0", got.Value)
		}
	})

	t.Run("float rejects malformed literal", func(t *testing.T) {
		type config struct {
			Value float64 `default:"nope"`
		}

		binding, err := model.NewBinding[config]()
		if err != nil {
			t.Fatalf("NewBinding() error: %v", err)
		}

		got := config{}
		err = binding.ApplyDefaults(&got)
		if !errors.Is(err, errors.ErrSetDefault) {
			t.Fatalf("ApplyDefaults() error = %v, want %v", err, errors.ErrSetDefault)
		}
		if got.Value != 0 {
			t.Fatalf("Value = %v, want 0", got.Value)
		}
	})
}

func TestBindingApplyDefaults_LiteralUnsupportedKinds(t *testing.T) {
	type nested struct {
		Value int
	}

	type config struct {
		Slice         []int          `default:"unsupported"`
		Map           map[string]int `default:"unsupported"`
		PointerStruct *nested        `default:"unsupported"`
		PointerSlice  *[]int         `default:"unsupported"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	nonNilSlice := []int{}
	nonNilMap := map[string]int{}
	nonNilNested := &nested{Value: 1}

	tests := []struct {
		name   string
		object config
		verify func(t *testing.T, got config)
	}{
		{
			name:   "slice",
			object: config{},
			verify: func(t *testing.T, got config) {
				if got.Slice != nil {
					t.Fatalf("Slice = %#v, want nil", got.Slice)
				}
			},
		},
		{
			name: "map",
			object: config{
				Slice: nonNilSlice,
			},
			verify: func(t *testing.T, got config) {
				if got.Map != nil {
					t.Fatalf("Map = %#v, want nil", got.Map)
				}
			},
		},
		{
			name: "pointer to struct",
			object: config{
				Slice: nonNilSlice,
				Map:   nonNilMap,
			},
			verify: func(t *testing.T, got config) {
				if got.PointerStruct != nil {
					t.Fatalf(
						"PointerStruct = %#v, want nil",
						got.PointerStruct,
					)
				}
			},
		},
		{
			name: "pointer to slice",
			object: config{
				Slice:         nonNilSlice,
				Map:           nonNilMap,
				PointerStruct: nonNilNested,
			},
			verify: func(t *testing.T, got config) {
				if got.PointerSlice != nil {
					t.Fatalf(
						"PointerSlice = %#v, want nil",
						got.PointerSlice,
					)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := binding.ApplyDefaults(&test.object)
			if !errors.Is(err, errors.ErrSetDefault) {
				t.Fatalf(
					"ApplyDefaults() error = %v, want %v",
					err,
					errors.ErrSetDefault,
				)
			}

			test.verify(t, test.object)
		})
	}
}

func TestBindingApplyDefaults_IsIdempotent(t *testing.T) {
	type nested struct {
		Value string `default:"nested"`
	}

	type config struct {
		S      string  `default:"value"`
		P      *string `default:"pointer"`
		Nested *nested `default:"dive"`
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

func TestBindingDefaults_EnvTagOverridesJSONTag(t *testing.T) {
	type config struct {
		Name string `json:"ignored" env:"custom_name"`
	}

	t.Setenv("CUSTOM_NAME", "from-env-tag")
	t.Setenv("IGNORED", "from-json-tag")

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := applyBindingDefaultsAndEnv(binding, &got); err != nil {
		t.Fatalf("applyBindingDefaultsAndEnv() error: %v", err)
	}

	if got.Name != "from-env-tag" {
		t.Fatalf("Name = %q, want explicit env value", got.Name)
	}
}

func TestBindingDefaults_EnvPrefixAppliesToMapValues(t *testing.T) {
	type service struct {
		URL string `json:"url"`
	}

	type config struct {
		Services map[string]service `json:"services"`
	}

	t.Setenv("APP_SERVICES_API_URL", "https://prefixed.example.com")
	t.Setenv("SERVICES_API_URL", "https://unprefixed.example.com")

	binding, err := model.NewBinding[config](
		model.WithEnvPrefix("app"),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Services: map[string]service{
			"api": {},
		},
	}
	if err := applyBindingDefaultsAndEnv(binding, &got); err != nil {
		t.Fatalf("applyBindingDefaultsAndEnv() error: %v", err)
	}

	if got.Services["api"].URL != "https://prefixed.example.com" {
		t.Fatalf(
			"Services[api].URL = %q, want prefixed env value",
			got.Services["api"].URL,
		)
	}
}

func TestBindingDefaults_InvalidDurationEnvReturnsError(t *testing.T) {
	type config struct {
		Timeout time.Duration `env:"REQUEST_TIMEOUT"`
	}

	t.Setenv("REQUEST_TIMEOUT", "not-a-duration")

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Timeout: time.Second,
	}
	err = applyBindingDefaultsAndEnv(binding, &got)
	if !errors.Is(err, errors.ErrSetDefault) {
		t.Fatalf("applyBindingDefaultsAndEnv() error = %v, want %v", err, errors.ErrSetDefault)
	}

	if got.Timeout != time.Second {
		t.Fatalf("Timeout = %v, want original %v", got.Timeout, time.Second)
	}
}

func TestBinding_Defaults(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T) (run func(t *testing.T))
	}{
		{
			name: "strings ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Strings{}
				expected := Strings{
					S:  "s",
					PS: pString("s"),
				}
				expected2 := Strings{
					S:  "s2",
					PS: pString("s2"),
				}
				b, err := model.NewBinding[Strings]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualStrings(t, o, expected)

					// set env vars
					t.Setenv("S", "s2")
					t.Setenv("PS", "s2")
					b, err = model.NewBinding[Strings]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualStrings(t, o, expected2)
				}
			},
		},
		{
			name: "ints ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Ints{}
				expected := Ints{
					I:    5,
					I8:   8,
					I16:  16,
					I32:  32,
					I64:  64,
					PI:   pInt(5),
					PI8:  pInt8(8),
					PI16: pInt16(16),
					PI32: pInt32(32),
					PI64: pInt64(64),
				}
				expected2 := Ints{
					I:    6,
					I8:   9,
					I16:  17,
					I32:  33,
					I64:  65,
					PI:   pInt(6),
					PI8:  pInt8(9),
					PI16: pInt16(17),
					PI32: pInt32(33),
					PI64: pInt64(65),
				}
				b, err := model.NewBinding[Ints]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInts(t, o, expected)

					// set env vars
					t.Setenv("I", "6")
					t.Setenv("I8", "9")
					t.Setenv("I16", "17")
					t.Setenv("I32", "33")
					t.Setenv("I64", "65")
					t.Setenv("PI", "6")
					t.Setenv("PI8", "9")
					t.Setenv("PI16", "17")
					t.Setenv("PI32", "33")
					t.Setenv("PI64", "65")
					b, err = model.NewBinding[Ints]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInts(t, o, expected2)
				}
			},
		},
		{
			name: "floats ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Floats{}
				expected := Floats{
					F32:  3.2,
					F64:  0x_1FFFp-16,
					PF32: pFloat32(3.2),
					PF64: pFloat64(0x_1FFFp-16),
				}
				expected2 := Floats{
					F32:  4.2,
					F64:  5.4,
					PF32: pFloat32(4.2),
					PF64: pFloat64(5.4),
				}
				b, err := model.NewBinding[Floats]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualFloats(t, o, expected)

					// set env vars
					t.Setenv("F32", "4.2")
					t.Setenv("F64", "5.4")
					t.Setenv("PF32", "4.2")
					t.Setenv("PF64", "5.4")
					b, err = model.NewBinding[Floats]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualFloats(t, o, expected2)
				}
			},
		},
		{
			name: "bools ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Bools{}
				expected := Bools{
					B:  true,
					PB: pBool(true),
				}
				expected2 := Bools{
					B:  false,
					PB: pBool(false),
				}
				b, err := model.NewBinding[Bools]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBools(t, o, expected)

					// set env vars
					t.Setenv("B", "false")
					t.Setenv("PB", "false")
					b, err = model.NewBinding[Bools]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBools(t, o, expected2)
				}
			},
		},
		{
			name: "uints ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Uints{}
				expected := Uints{
					U:    5,
					U8:   8,
					U16:  16,
					U32:  32,
					U64:  64,
					PU:   pUint(5),
					PU8:  pUint8(8),
					PU16: pUint16(16),
					PU32: pUint32(32),
					PU64: pUint64(64),
				}
				expected2 := Uints{
					U:    6,
					U8:   9,
					U16:  17,
					U32:  33,
					U64:  65,
					PU:   pUint(6),
					PU8:  pUint8(9),
					PU16: pUint16(17),
					PU32: pUint32(33),
					PU64: pUint64(65),
				}
				b, err := model.NewBinding[Uints]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUints(t, o, expected)

					// set env vars
					t.Setenv("U", "6")
					t.Setenv("U8", "9")
					t.Setenv("U16", "17")
					t.Setenv("U32", "33")
					t.Setenv("U64", "65")
					t.Setenv("PU", "6")
					t.Setenv("PU8", "9")
					t.Setenv("PU16", "17")
					t.Setenv("PU32", "33")
					t.Setenv("PU64", "65")
					b, err = model.NewBinding[Uints]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUints(t, o, expected2)
				}
			},
		},
		{
			name: "uintptrs ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := UintPtrs{}
				expected := UintPtrs{
					UintPtr:  128,
					PUintPtr: pUintptr(128),
				}
				expected2 := UintPtrs{
					UintPtr:  256,
					PUintPtr: pUintptr(256),
				}
				b, err := model.NewBinding[UintPtrs]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUintPtrs(t, o, expected)

					// set env vars
					t.Setenv("UINTPTR", "256")
					t.Setenv("PUINTPTR", "256")
					b, err = model.NewBinding[UintPtrs]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUintPtrs(t, o, expected2)
				}
			},
		},
		{
			name: "bytes ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Bytes{}
				expected := Bytes{
					Byte:  8,
					PByte: pByte(8),
				}
				expected2 := Bytes{
					Byte:  9,
					PByte: pByte(9),
				}
				b, err := model.NewBinding[Bytes]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBytes(t, o, expected)

					// set env vars
					t.Setenv("BYTE", "9")
					t.Setenv("PBYTE", "9")
					b, err = model.NewBinding[Bytes]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBytes(t, o, expected2)
				}
			},
		},
		{
			name: "runes ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Runes{}
				expected := Runes{
					Rune:  '\U00101234',
					PRune: pRune('\U00101234'),
				}
				expected2 := Runes{
					Rune:  'Ж',
					PRune: pRune('Ж'),
				}
				b, err := model.NewBinding[Runes]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualRunes(t, o, expected)

					// set env vars
					t.Setenv("RUNE", "Ж")
					t.Setenv("PRUNE", "Ж")
					b, err = model.NewBinding[Runes]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualRunes(t, o, expected2)
				}
			},
		},
		{
			name: "complexes ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Complexes{}
				expected := Complexes{
					C64:   3 + 2i,
					C128:  6 + 4i,
					PC64:  pComplex64(3 + 2i),
					PC128: pComplex128(6 + 4i),
				}
				expected2 := Complexes{
					C64:   4 + 3i,
					C128:  7 + 5i,
					PC64:  pComplex64(4 + 3i),
					PC128: pComplex128(7 + 5i),
				}
				b, err := model.NewBinding[Complexes]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualComplexes(t, o, expected)

					// set env vars
					t.Setenv("C64", "4+3i")
					t.Setenv("C128", "7+5i")
					t.Setenv("PC64", "4+3i")
					t.Setenv("PC128", "7+5i")
					b, err = model.NewBinding[Complexes]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualComplexes(t, o, expected2)
				}
			},
		},
		{
			name: "durations ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Durations{}
				expected := Durations{
					TD:  5 * time.Second,
					D:   types.Duration(5 * time.Second),
					PTD: pTDuration(5 * time.Second),
					PD:  pDuration(types.Duration(5 * time.Second)),
				}
				expected2 := Durations{
					TD:  10 * time.Second,
					D:   types.Duration(10 * time.Second),
					PTD: pTDuration(10 * time.Second),
					PD:  pDuration(types.Duration(10 * time.Second)),
				}
				b, err := model.NewBinding[Durations]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDurations(t, o, expected)

					// set env vars
					t.Setenv("TD", "10s")
					t.Setenv("D", "10s")
					t.Setenv("PTD", "10s")
					t.Setenv("PD", "10s")
					b, err = model.NewBinding[Durations]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDurations(t, o, expected2)
				}
			},
		},
		{
			name: "default alloc ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultAlloc{}
				expected := DefaultAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "",
					Str: Strings{S: "s", PS: pString("s")}, // because of implicit dive for struct
				}
				expected2 := DefaultAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "s",
					Str: Strings{S: "str_s", PS: pString("str_s")}, // because of implicit dive for struct
				}
				b, err := model.NewBinding[DefaultAlloc]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected)

					// set env vars
					t.Setenv("SS", "ss")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					t.Setenv("STR_S", "str_s")
					t.Setenv("STR_PS", "str_s")
					b, err = model.NewBinding[DefaultAlloc]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected2)
				}
			},
		},
		{
			name: "default elem alloc ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemAlloc{}
				expected := DefaultElemAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "",
					Str: Strings{S: "s", PS: pString("s")}, // because of implicit dive for struct
				}
				expected2 := DefaultElemAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "s",
					Str: Strings{S: "str_s", PS: pString("str_s")}, // because of implicit dive for struct
				}
				b, err := model.NewBinding[DefaultElemAlloc]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemAlloc(t, o, expected)

					// set env vars
					t.Setenv("SS", "ss")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					t.Setenv("STR_S", "str_s")
					t.Setenv("STR_PS", "str_s")
					b, err = model.NewBinding[DefaultElemAlloc]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemAlloc(t, o, expected2)
				}
			},
		},
		{
			name: "dive ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Dive{}
				expected := Dive{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					},
					PStrings: &Strings{
						S:  "s",
						PS: pString("s"),
					},
					PI: nil,
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					S:  "",
				}
				expected2 := Dive{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					PI: pInt(3),
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					S:  "s",
				}
				b, err := model.NewBinding[Dive]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDive(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewBinding[Dive]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDive(t, o, expected2)
				}
			},
		},
		{
			name: "collections ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Collections{}
				expected := Collections{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					}, // "dive" is implicit for structs
					PStrings: nil, // "dive" is not implicit for pointers to structs
					M:        make(map[string]Strings),
					MP:       make(map[string]*Strings),
					A:        []Strings{},
					AP:       []*Strings{},
					SS:       []string{},
				}
				expected2 := Collections{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					SS: []string{},
				}
				b, err := model.NewBinding[Collections]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollections(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewBinding[Collections]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollections(t, o, expected2)
				}
			},
		},
		{
			name: "collections default empty ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := CollectionsDefaultEmpty{}
				expected := CollectionsDefaultEmpty{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					}, // "dive" is implicit for structs
					PStrings: nil, // "dive" is not implicit for pointers to structs
					M:        make(map[string]Strings),
					MP:       make(map[string]*Strings),
					A:        []Strings{},
					AP:       []*Strings{},
					SS:       []string{},
				}
				expected2 := CollectionsDefaultEmpty{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					SS: []string{},
				}
				b, err := model.NewBinding[CollectionsDefaultEmpty]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultEmpty(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewBinding[CollectionsDefaultEmpty]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultEmpty(t, o, expected2)
				}
			},
		},
		{
			name: "collections default element empty ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := CollectionsDefaultElemEmpty{}
				expected := CollectionsDefaultElemEmpty{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					}, // "dive" is implicit for structs
					PStrings: nil, // "dive" is not implicit for pointers to structs
					M:        make(map[string]Strings),
					MP:       make(map[string]*Strings),
					A:        []Strings{},
					AP:       []*Strings{},
					SS:       []string{},
				}
				expected2 := CollectionsDefaultElemEmpty{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					SS: []string{},
				}
				b, err := model.NewBinding[CollectionsDefaultElemEmpty]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultElemEmpty(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewBinding[CollectionsDefaultElemEmpty]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultElemEmpty(t, o, expected2)
				}
			},
		},
		{
			name: "no env tag",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := NoEnvTag{}
				expected := NoEnvTag{
					NoEnvTag: Strings{
						S:  "s",
						PS: pString("s"),
					},
				}
				expected2 := NoEnvTag{
					NoEnvTag: Strings{
						S:  "s2",
						PS: pString("s"),
					},
				}
				b, err := model.NewBinding[NoEnvTag]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualNoEnvTag(t, o, expected)

					// set env vars
					t.Setenv("NOENVTAG_S", "s2")
					b, err = model.NewBinding[NoEnvTag]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualNoEnvTag(t, o, expected2)
				}
			},
		},
		{
			name: "env prefix ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvPrefix{}
				expected := EnvPrefix{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					},
				}
				expected2 := EnvPrefix{
					Strings: Strings{
						S:  "prefixed_s",
						PS: pString("prefixed_ps"),
					},
				}
				b, err := model.NewBinding[EnvPrefix](model.WithEnvPrefix("__app_config__"))
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvPrefix(t, o, expected)

					t.Setenv("APP_CONFIG_STRINGS_S", "prefixed_s")
					t.Setenv("APP_CONFIG_STRINGS_PS", "prefixed_ps")
					b, err = model.NewBinding[EnvPrefix](model.WithEnvPrefix("__app_config__"))
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvPrefix(t, o, expected2)
				}
			},
		},
		{
			name: "env disabled is skipped",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvDisabled{}
				expected := EnvDisabled{S: "s"}
				b, err := model.NewBinding[EnvDisabled]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvDisabled(t, o, expected)

					t.Setenv("S", "s2")
					b, err = model.NewBinding[EnvDisabled]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvDisabled(t, o, expected)
				}
			},
		},
		{
			name: "json comma tag env name ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := JSONCommaTag{}
				expected := JSONCommaTag{S: "s"}
				expected2 := JSONCommaTag{S: "s2"}
				b, err := model.NewBinding[JSONCommaTag]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualJSONCommaTag(t, o, expected)

					t.Setenv("CUSTOM_S", "s2")
					b, err = model.NewBinding[JSONCommaTag]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualJSONCommaTag(t, o, expected2)
				}
			},
		},
		{
			name: "env zero values override defaults",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvZeroValues{}
				expected := EnvZeroValues{S: "s", I: 5, B: true}
				expected2 := EnvZeroValues{S: "", I: 0, B: false}
				b, err := model.NewBinding[EnvZeroValues]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvZeroValues(t, o, expected)

					t.Setenv("S", "")
					t.Setenv("I", "0")
					t.Setenv("B", "false")
					b, err = model.NewBinding[EnvZeroValues]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvZeroValues(t, o, expected2)
				}
			},
		},
		{
			name: "map literal env values ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapLiteralEnv{M: map[string]int{"one": 1}}
				expected := MapLiteralEnv{M: map[string]int{"one": 1}}
				expected2 := MapLiteralEnv{M: map[string]int{"one": 2}}
				b, err := model.NewBinding[MapLiteralEnv]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapLiteralEnv(t, o, expected)

					t.Setenv("M_ONE", "2")
					b, err = model.NewBinding[MapLiteralEnv]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapLiteralEnv(t, o, expected2)
				}
			},
		},
		{
			name: "map struct env values ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapStructEnv{M: map[string]Strings{"one": {}}}
				expected := MapStructEnv{M: map[string]Strings{"one": {S: "s", PS: pString("s")}}}
				expected2 := MapStructEnv{M: map[string]Strings{"one": {S: "s2", PS: pString("ps2")}}}
				b, err := model.NewBinding[MapStructEnv]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapStructEnv(t, o, expected)

					t.Setenv("M_ONE_S", "s2")
					t.Setenv("M_ONE_PS", "ps2")
					b, err = model.NewBinding[MapStructEnv]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapStructEnv(t, o, expected2)
				}
			},
		},
		{
			name: "map pointer struct env values ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapPtrStructEnv{M: map[string]*Strings{"one": {}, "nil": nil}}
				expected := MapPtrStructEnv{M: map[string]*Strings{"one": {S: "s", PS: pString("s")}, "nil": nil}}
				expected2 := MapPtrStructEnv{M: map[string]*Strings{"one": {S: "s2", PS: pString("ps2")}, "nil": nil}}
				b, err := model.NewBinding[MapPtrStructEnv]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapPtrStructEnv(t, o, expected)

					t.Setenv("M_ONE_S", "s2")
					t.Setenv("M_ONE_PS", "ps2")
					t.Setenv("M_NIL_S", "ignored")
					b, err = model.NewBinding[MapPtrStructEnv]()
					if err != nil {
						t.Fatal(err)
					}
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapPtrStructEnv(t, o, expected2)
				}
			},
		},
		{
			name: "default elem slice ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemSlice{Items: []Strings{{}}}
				expected := DefaultElemSlice{Items: []Strings{{S: "s", PS: pString("s")}}}
				b, err := model.NewBinding[DefaultElemSlice]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemSlice(t, o, expected)
				}
			},
		},
		{
			name: "default elem pointer slice ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemPtrSlice{Items: []*Strings{{}, nil}}
				expected := DefaultElemPtrSlice{Items: []*Strings{{S: "s", PS: pString("s")}, nil}}
				b, err := model.NewBinding[DefaultElemPtrSlice]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemPtrSlice(t, o, expected)
				}
			},
		},
		{
			name: "default elem array ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemArray{}
				expected := DefaultElemArray{Items: [1]Strings{{S: "s", PS: pString("s")}}}
				b, err := model.NewBinding[DefaultElemArray]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemArray(t, o, expected)
				}
			},
		},
		{
			name: "default elem map ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemMap{M: map[string]Strings{"one": {}}}
				expected := DefaultElemMap{M: map[string]Strings{"one": {S: "s", PS: pString("s")}}}
				b, err := model.NewBinding[DefaultElemMap]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemMap(t, o, expected)
				}
			},
		},
		{
			name: "default elem pointer map ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemPtrMap{M: map[string]*Strings{"one": {}, "nil": nil}}
				expected := DefaultElemPtrMap{M: map[string]*Strings{"one": {S: "s", PS: pString("s")}, "nil": nil}}
				b, err := model.NewBinding[DefaultElemPtrMap]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemPtrMap(t, o, expected)
				}
			},
		},
		{
			name: "default elem pointer collection ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				items := []Strings{{}}
				m := map[string]Strings{"one": {}}
				o := DefaultElemPtrCollection{Items: &items, M: &m}
				expectedItems := []Strings{{S: "s", PS: pString("s")}}
				expectedMap := map[string]Strings{"one": {S: "s", PS: pString("s")}}
				expected := DefaultElemPtrCollection{Items: &expectedItems, M: &expectedMap}
				b, err := model.NewBinding[DefaultElemPtrCollection]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemPtrCollection(t, o, expected)
				}
			},
		},
		{
			name: "default elem unsupported tag is ignored",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemUnsupported{Items: []Strings{{}}}
				expected := DefaultElemUnsupported{Items: []Strings{{}}}
				b, err := model.NewBinding[DefaultElemUnsupported]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemUnsupported(t, o, expected)
				}
			},
		},
		{
			name: "alloc no op for non nil and non collection",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := AllocNoop{SS: []string{"x"}, M: map[string]string{"k": "v"}}
				expected := AllocNoop{SS: []string{"x"}, M: map[string]string{"k": "v"}}
				b, err := model.NewBinding[AllocNoop]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualAllocNoop(t, o, expected)
				}
			},
		},
		{
			name: "dive ignored for non struct values",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DiveIgnored{}
				expected := DiveIgnored{}
				b, err := model.NewBinding[DiveIgnored]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDiveIgnored(t, o, expected)
				}
			},
		},
		{
			name: "named scalar defaults ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := NamedScalars{}
				expected := NamedScalars{S: CustomString("s"), I: CustomInt(5), B: CustomBool(true)}
				b, err := model.NewBinding[NamedScalars]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualNamedScalars(t, o, expected)
				}
			},
		},
		{
			name: "invalid env int returns set default error",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvInvalidInt{}
				b, err := model.NewBinding[EnvInvalidInt]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					t.Setenv("I", "not-int")
					b, err = model.NewBinding[EnvInvalidInt]()
					if err != nil {
						t.Fatal(err)
					}
					err = applyBindingDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
					if !errors.Is(err, errors.ErrSetDefault) {
						t.Fatalf("expected ErrSetDefault, got %v", err)
					}
				}
			},
		},
		{
			name: "invalid map env int returns set default error",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapLiteralEnv{M: map[string]int{"one": 1}}
				b, err := model.NewBinding[MapLiteralEnv]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					t.Setenv("M_ONE", "not-int")
					b, err = model.NewBinding[MapLiteralEnv]()
					if err != nil {
						t.Fatal(err)
					}
					err = applyBindingDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
					if !errors.Is(err, errors.ErrSetDefault) {
						t.Fatalf("expected ErrSetDefault, got %v", err)
					}
				}
			},
		},
		{
			name: "unexported is skipped",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Unexported{}
				expected := Unexported{}
				b, err := model.NewBinding[Unexported]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUnexported(t, o, expected)
				}
			},
		},
		{
			name: "wrapped unexported is skipped",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := WrappedUnexported{}
				expected := WrappedUnexported{}
				b, err := model.NewBinding[WrappedUnexported]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualWrappedUnexported(t, o, expected)
				}
			},
		},
		{
			name: "interface",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Interface{}
				expected := Interface{}
				b, err := model.NewBinding[Interface]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInterface(t, o, expected)
				}
			},
		},
		{
			name: "interface with value",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Interface{Interface: &Strings{}}
				expected := Interface{Interface: &Strings{}}
				b, err := model.NewBinding[Interface]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInterface(t, o, expected)
				}
			},
		},
		{
			name: "error on unsupported literal kind",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := UnsupportedLiteralKind{}
				b, err := model.NewBinding[UnsupportedLiteralKind]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					err = applyBindingDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
					if !errors.Is(err, errors.ErrSetDefault) {
						t.Fatalf("expected ErrSetDefault, got %v", err)
					}
					expectedMsg := "cannot set default value, tag.default: unsupported, field.path: unsupported, cause: default literal unsupported kind, default.literal.kind: struct"
					if err.Error() != expectedMsg {
						t.Fatalf("expected %s, got %s", expectedMsg, err.Error())
					}
				}
			},
		},
		{
			name: "nil object",
			prepare: func(t *testing.T) func(t *testing.T) {
				var o *Strings
				b, err := model.NewBinding[Strings]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = b.ApplyDefaults(o); err == nil {
						t.Fatalf("expected error, got nil")
					}
					if !errors.Is(err, errors.ErrNilObject) {
						t.Fatalf("expected ErrNilObject, got %v", err)
					}
				}
			},
		},
		{
			name: "alloc slice/map when nil and idempotent",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultAlloc{}
				expected := DefaultAlloc{
					SS: []string{},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					S:  "",
					Str: Strings{
						S:  "s",
						PS: pString("s"),
					},
				}
				b, err := model.NewBinding[DefaultAlloc]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected err: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected)
					// run again (idempotent)
					if err = applyBindingDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected err on second run: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected)
				}
			},
		},
		{
			name: "error unsupported literal kind wrapped with field name St",
			prepare: func(t *testing.T) func(t *testing.T) {
				type wrappedUnsupportedLiteralKind struct {
					W UnsupportedLiteralKind `yaml:"w" env:"W" default:"dive"`
				}
				o := wrappedUnsupportedLiteralKind{}
				b, err := model.NewBinding[wrappedUnsupportedLiteralKind]()
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					err = applyBindingDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := tc.prepare(t)
			run(t)
		})
	}
}
