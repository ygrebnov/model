package core

import (
	"os"
	"reflect"
	"testing"
	"time"

	fieldPkg "github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/validation"
)

// Helper types for defaults testing
type innerDef struct {
	S string `default:"x"`
	N int    `default:"42"`
}

type envNestedConfig struct {
	Host    string        `json:"host"`
	Port    int           `json:"port" default:"8080"`
	Timeout time.Duration `json:"timeout" env:"request_timeout"`
}

type envNestedConfigNoDefaults struct {
	Host    string        `json:"host"`
	Port    int           `json:"port"`
	Timeout time.Duration `json:"timeout" env:"request_timeout"`
}

type envMapValueConfig struct {
	URL     string        `json:"url"`
	Timeout time.Duration `json:"timeout"`
}

type envConfig struct {
	Name       string `json:"name"`
	Explicit   string `json:"ignored" env:"custom_name"`
	Fallback   string
	Enabled    bool                         `json:"enabled"`
	Server     envNestedConfig              `json:"server"`
	PServer    *envNestedConfig             `json:"p_server"`
	EServer    *envNestedConfigNoDefaults   `json:"e_server"`
	Services   map[string]envMapValueConfig `json:"services"`
	Numbers    []int                        `json:"numbers"`
	SkipMe     string                       `env:"-" json:"skip_me"`
	DefaultVal string                       `json:"default_val" default:"from-default"`
}

type osEnvSource struct{}

func (osEnvSource) Lookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

var _ fieldPkg.EnvSource = osEnvSource{}

func TestApplyEnvStruct_EnvironmentValues(t *testing.T) {
	tests := []struct {
		name   string
		setEnv func(t *testing.T)
		obj    envConfig
		verify func(t *testing.T, got envConfig)
	}{
		{
			name: "uses json tag as env name fallback",
			setEnv: func(t *testing.T) {
				t.Setenv("NAME", "from-env")
			},
			obj: envConfig{Name: "from-config"},
			verify: func(t *testing.T, got envConfig) {
				if got.Name != "from-env" {
					t.Fatalf("expected Name from env, got %q", got.Name)
				}
			},
		},
		{
			name: "env tag wins over json tag",
			setEnv: func(t *testing.T) {
				t.Setenv("CUSTOM_NAME", "from-env-tag")
				t.Setenv("IGNORED", "from-json-tag")
			},
			obj: envConfig{Explicit: "from-config"},
			verify: func(t *testing.T, got envConfig) {
				if got.Explicit != "from-env-tag" {
					t.Fatalf("expected Explicit from env tag, got %q", got.Explicit)
				}
			},
		},
		{
			name: "uses uppercase field name when tags are missing",
			setEnv: func(t *testing.T) {
				t.Setenv("FALLBACK", "from-field-name")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.Fallback != "from-field-name" {
					t.Fatalf("expected Fallback from field name, got %q", got.Fallback)
				}
			},
		},
		{
			name: "parses bool env values",
			setEnv: func(t *testing.T) {
				t.Setenv("ENABLED", "true")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if !got.Enabled {
					t.Fatalf("expected Enabled to be true")
				}
			},
		},
		{
			name: "environment overrides non-zero value",
			setEnv: func(t *testing.T) {
				t.Setenv("NAME", "from-env")
			},
			obj: envConfig{Name: "from-config"},
			verify: func(t *testing.T, got envConfig) {
				if got.Name != "from-env" {
					t.Fatalf("expected env to override config value, got %q", got.Name)
				}
			},
		},
		{
			name:   "default fills only zero value when env is absent",
			setEnv: func(t *testing.T) {},
			obj:    envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.DefaultVal != "from-default" {
					t.Fatalf("expected default value, got %q", got.DefaultVal)
				}
			},
		},
		{
			name: "environment overrides default tag",
			setEnv: func(t *testing.T) {
				t.Setenv("DEFAULT_VAL", "from-env")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.DefaultVal != "from-env" {
					t.Fatalf("expected env value to override default, got %q", got.DefaultVal)
				}
			},
		},
		{
			name: "environment zero value overrides default tag",
			setEnv: func(t *testing.T) {
				t.Setenv("DEFAULT_VAL", "")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.DefaultVal != "" {
					t.Fatalf("expected env zero value to override default, got %q", got.DefaultVal)
				}
			},
		},
		{
			name: "env dash tag skips field",
			setEnv: func(t *testing.T) {
				t.Setenv("SKIP_ME", "from-env")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.SkipMe != "" {
					t.Fatalf("expected SkipMe to remain empty, got %q", got.SkipMe)
				}
			},
		},
		{
			name: "pointer to struct without defaults give nil",
			obj:  envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.EServer != nil {
					t.Fatalf("expected EServer to remain nil, got %+v", got.EServer)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv != nil {
				tc.setEnv(t)
			}

			got := tc.obj
			rv := reflect.ValueOf(&got).Elem()
			if err := newService(&got).SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := newService(&got).ApplyEnvStruct(rv, osEnvSource{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.verify(t, got)
		})
	}
}

func TestApplyEnvStruct_EnvironmentNestedValues(t *testing.T) {
	tests := []struct {
		name   string
		setEnv func(t *testing.T)
		obj    envConfig
		verify func(t *testing.T, got envConfig)
	}{
		{
			name: "joins nested struct env path with underscore",
			setEnv: func(t *testing.T) {
				t.Setenv("SERVER_HOST", "localhost")
				t.Setenv("SERVER_PORT", "9090")
				t.Setenv("SERVER_REQUEST_TIMEOUT", "5s")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.Server.Host != "localhost" {
					t.Fatalf("expected Server.Host from env, got %q", got.Server.Host)
				}
				if got.Server.Port != 9090 {
					t.Fatalf("expected Server.Port from env, got %d", got.Server.Port)
				}
				if got.Server.Timeout != 5*time.Second {
					t.Fatalf("expected Server.Timeout from env, got %v", got.Server.Timeout)
				}
			},
		},
		{
			name: "nested environment zero value overrides default tag",
			setEnv: func(t *testing.T) {
				t.Setenv("SERVER_PORT", "0")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.Server.Port != 0 {
					t.Fatalf("expected nested env zero value to override default, got %d", got.Server.Port)
				}
			},
		},
		{
			name: "allocates nil pointer to struct for nested env value",
			setEnv: func(t *testing.T) {
				t.Setenv("P_SERVER_HOST", "pointer-host")
				t.Setenv("P_SERVER_PORT", "7070")
			},
			obj: envConfig{PServer: nil},
			verify: func(t *testing.T, got envConfig) {
				if got.PServer == nil {
					t.Fatalf("expected PServer to be allocated")
				}
				if got.PServer.Host != "pointer-host" {
					t.Fatalf("expected PServer.Host from env, got %q", got.PServer.Host)
				}
				if got.PServer.Port != 7070 {
					t.Fatalf("expected PServer.Port from env, got %d", got.PServer.Port)
				}
			},
		},
		{
			name: "walks map struct values using map key path segment",
			setEnv: func(t *testing.T) {
				t.Setenv("SERVICES_API_URL", "https://api.example.com")
				t.Setenv("SERVICES_API_TIMEOUT", "3s")
			},
			obj: envConfig{Services: map[string]envMapValueConfig{"api": {}}},
			verify: func(t *testing.T, got envConfig) {
				service := got.Services["api"]
				if service.URL != "https://api.example.com" {
					t.Fatalf("expected service URL from env, got %q", service.URL)
				}
				if service.Timeout != 3*time.Second {
					t.Fatalf("expected service Timeout from env, got %v", service.Timeout)
				}
			},
		},
		{
			name: "skips slices for environment traversal",
			setEnv: func(t *testing.T) {
				t.Setenv("NUMBERS_0", "99")
			},
			obj: envConfig{Numbers: []int{1, 2}},
			verify: func(t *testing.T, got envConfig) {
				if got.Numbers[0] != 1 || got.Numbers[1] != 2 {
					t.Fatalf("expected Numbers unchanged, got %#v", got.Numbers)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setEnv(t)

			got := tc.obj
			rv := reflect.ValueOf(&got).Elem()
			if err := newService(&got).SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := newService(&got).ApplyEnvStruct(rv, osEnvSource{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.verify(t, got)
		})
	}
}

func TestApplyEnvStruct_EnvironmentPrefix(t *testing.T) {
	tests := []struct {
		name      string
		envPrefix string
		setEnv    func(t *testing.T)
		obj       envConfig
		verify    func(t *testing.T, got envConfig)
	}{
		{
			name:      "uses env prefix for root field",
			envPrefix: "app",
			setEnv: func(t *testing.T) {
				t.Setenv("APP_NAME", "prefixed-name")
				t.Setenv("NAME", "unprefixed-name")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.Name != "prefixed-name" {
					t.Fatalf("expected prefixed env value, got %q", got.Name)
				}
			},
		},
		{
			name:      "uses env prefix for nested struct field",
			envPrefix: "app",
			setEnv: func(t *testing.T) {
				t.Setenv("APP_SERVER_HOST", "prefixed-host")
				t.Setenv("SERVER_HOST", "unprefixed-host")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.Server.Host != "prefixed-host" {
					t.Fatalf("expected prefixed nested env value, got %q", got.Server.Host)
				}
			},
		},
		{
			name:      "uses env prefix for map value field",
			envPrefix: "app",
			setEnv: func(t *testing.T) {
				t.Setenv("APP_SERVICES_API_URL", "https://prefixed.example.com")
				t.Setenv("SERVICES_API_URL", "https://unprefixed.example.com")
			},
			obj: envConfig{Services: map[string]envMapValueConfig{"api": {}}},
			verify: func(t *testing.T, got envConfig) {
				service := got.Services["api"]
				if service.URL != "https://prefixed.example.com" {
					t.Fatalf("expected prefixed map env value, got %q", service.URL)
				}
			},
		},
		{
			name:      "normalizes env prefix",
			envPrefix: "_my_app_",
			setEnv: func(t *testing.T) {
				t.Setenv("MY_APP_NAME", "normalized-prefix")
			},
			obj: envConfig{},
			verify: func(t *testing.T, got envConfig) {
				if got.Name != "normalized-prefix" {
					t.Fatalf("expected normalized prefix env value, got %q", got.Name)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setEnv(t)

			got := tc.obj
			rv := reflect.ValueOf(&got).Elem()
			if err := newServiceWithEnvPrefix(&got, tc.envPrefix).SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := newServiceWithEnvPrefix(&got, tc.envPrefix).ApplyEnvStruct(rv, osEnvSource{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.verify(t, got)
		})
	}
}

func TestApplyEnvStruct_EnvironmentValueErrors(t *testing.T) {
	tests := []struct {
		name   string
		setEnv func(t *testing.T)
		obj    envConfig
	}{
		{
			name: "returns error for invalid int env value",
			setEnv: func(t *testing.T) {
				t.Setenv("SERVER_PORT", "not-an-int")
			},
			obj: envConfig{},
		},
		{
			name: "returns error for invalid duration env value",
			setEnv: func(t *testing.T) {
				t.Setenv("SERVER_REQUEST_TIMEOUT", "not-a-duration")
			},
			obj: envConfig{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setEnv(t)

			got := tc.obj
			rv := reflect.ValueOf(&got).Elem()
			if err := newService(&got).SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := newService(&got).ApplyEnvStruct(rv, osEnvSource{}); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

type outerDef struct {
	Inner  innerDef
	PInner *innerDef
	N      int
	S      []string
	M      map[string]int
	PInt   *int
}

func newService[T any]() (*Service[T], error) {
	sc, err := schema.NewController[T]()
	if err != nil {
		return nil, err
	}

	return NewService[T](
		validation.NewRulesRegistry(),
		validation.NewRulesMapping(),
		sc,
		"",
	), nil
}

func newServiceWithEnvPrefix[T any](obj *T, envPrefix string) *Service[T] {
	return NewService[T](
		validation.NewRulesRegistry(),
		validation.NewRulesMapping(),
		envPrefix,
	)
}

func TestApplyDefaultTag(t *testing.T) {
	tests := []struct {
		name   string
		prep   func() (obj *outerDef, fv reflect.Value)
		act    func(*Service, reflect.Value) error
		verify func(t *testing.T, obj *outerDef)
	}{
		{
			name: "dive on struct field applies inner defaults",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{}
				rv := reflect.ValueOf(obj).Elem()
				fv := rv.FieldByName("Inner")
				return obj, fv
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "dive", "Inner")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.Inner.S != "x" || obj.Inner.N != 42 {
					t.Fatalf("expected inner defaults applied, got %+v", obj.Inner)
				}
			},
		},
		{
			name: "dive on nil *struct allocates and applies defaults",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{PInner: nil}
				rv := reflect.ValueOf(obj).Elem()
				fv := rv.FieldByName("PInner")
				return obj, fv
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "dive", "PInner")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.PInner == nil || obj.PInner.S != "x" || obj.PInner.N != 42 {
					t.Fatalf("expected allocated PInner with defaults, got %+v", obj.PInner)
				}
			},
		},
		{
			name: "dive on non-struct is no-op",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("N")
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "dive", "N")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.N != 0 {
					t.Fatalf("expected N unchanged, got %d", obj.N)
				}
			},
		},
		{
			name: "alloc on nil slice allocates empty slice",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{S: nil}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("S")
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "alloc", "S")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.S == nil || len(obj.S) != 0 {
					t.Fatalf("expected allocated empty slice, got %#v", obj.S)
				}
			},
		},
		{
			name: "alloc on nil map allocates empty map",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{M: nil}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("M")
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "alloc", "M")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.M == nil || len(obj.M) != 0 {
					t.Fatalf("expected allocated empty map, got %#v", obj.M)
				}
			},
		},
		{
			name: "literal default on nil *int allocates and sets",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{PInt: nil}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("PInt")
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "7", "PInt")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.PInt == nil || *obj.PInt != 7 {
					t.Fatalf("expected allocated *int==7, got %#v", obj.PInt)
				}
			},
		},
		{
			name: "literal default on non-zero int leaves it unchanged",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{N: 5}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("N")
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "9", "N")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.N != 5 {
					t.Fatalf("expected N to remain 5, got %d", obj.N)
				}
			},
		},
		{
			name: "literal default on zero int sets value",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{N: 0}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("N")
			},
			act: func(tb *Service, fv reflect.Value) error {
				return tb.applyDefaultTag(fv, "9", "N")
			},
			verify: func(t *testing.T, obj *outerDef) {
				if obj.N != 9 {
					t.Fatalf("expected N to be 9, got %d", obj.N)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, fv := tc.prep()
			tb := newService(obj)
			if err := tc.act(tb, fv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.verify(t, obj)
		})
	}
}

// Types and holder for applyDefaultElemTag tests
type innerElem struct {
	S string `default:"x"`
	N int    `default:"42"`
}

type elemHolder struct {
	People []innerElem
	Ptrs   []*innerElem
	Arr    [2]innerElem
	Mv     map[string]innerElem
	Mp     map[string]*innerElem
	Not    string
}

func TestApplyDefaultElemTag(t *testing.T) {
	tests := []struct {
		name   string
		prep   func() (obj *elemHolder, fv reflect.Value)
		verify func(t *testing.T, obj *elemHolder)
	}{
		{
			name: "slice of struct elements dive",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{People: []innerElem{{}, {S: "ok"}}}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("People")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.People[0].S != "x" || obj.People[0].N != 42 {
					t.Fatalf("expected defaults on People[0], got %+v", obj.People[0])
				}
				if obj.People[1].S != "ok" || obj.People[1].N != 42 {
					t.Fatalf("expected partial defaults on People[1], got %+v", obj.People[1])
				}
			},
		},
		{
			name: "slice of *struct elements dive (nil stays nil)",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Ptrs: []*innerElem{nil, {}}}
				rv := reflect.ValueOf(obj).Elem()
				// fix composite literal: replace {} with &innerElem{}
				obj.Ptrs[1] = &innerElem{}
				return obj, rv.FieldByName("Ptrs")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Ptrs[0] != nil {
					t.Fatalf("expected Ptrs[0] to remain nil")
				}
				if obj.Ptrs[1] == nil || obj.Ptrs[1].S != "x" || obj.Ptrs[1].N != 42 {
					t.Fatalf("expected defaults on Ptrs[1], got %#v", obj.Ptrs[1])
				}
			},
		},
		{
			name: "array of struct elements dive",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Arr: [2]innerElem{{}, {S: "ok"}}}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("Arr")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Arr[0].S != "x" || obj.Arr[0].N != 42 {
					t.Fatalf("expected defaults on Arr[0], got %+v", obj.Arr[0])
				}
				if obj.Arr[1].S != "ok" || obj.Arr[1].N != 42 {
					t.Fatalf("expected partial defaults on Arr[1], got %+v", obj.Arr[1])
				}
			},
		},
		{
			name: "map[string]struct values dive (copy-write-back)",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Mv: map[string]innerElem{"a": {}, "b": {S: "ok"}}}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("Mv")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Mv["a"].S != "x" || obj.Mv["a"].N != 42 {
					t.Fatalf("expected defaults on Mv[a], got %+v", obj.Mv["a"])
				}
				if obj.Mv["b"].S != "ok" || obj.Mv["b"].N != 42 {
					t.Fatalf("expected partial defaults on Mv[b], got %+v", obj.Mv["b"])
				}
			},
		},
		{
			name: "map[string]*struct values dive (in-place)",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Mp: map[string]*innerElem{"p1": {}, "p2": nil}}
				rv := reflect.ValueOf(obj).Elem()
				obj.Mp["p1"] = &innerElem{}
				return obj, rv.FieldByName("Mp")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Mp["p1"] == nil || obj.Mp["p1"].S != "x" || obj.Mp["p1"].N != 42 {
					t.Fatalf("expected defaults on Mp[p1], got %#v", obj.Mp["p1"])
				}
				if obj.Mp["p2"] != nil {
					t.Fatalf("expected Mp[p2] to remain nil")
				}
			},
		},
		{
			name: "non-collection field ignored",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Not: ""}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("Not")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Not != "" {
					t.Fatalf("expected Not unchanged, got %q", obj.Not)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, fv := tc.prep()
			tb := newService(obj)
			if err := tb.applyDefaultElemTag(fv, tagDive); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.verify(t, obj)
		})
	}
}
