package core

import (
	"os"
	"reflect"
	"testing"
	"time"

	fieldPkg "github.com/ygrebnov/model/field"
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
			svc, err := newService[envConfig]()
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if err := svc.SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := svc.ApplyEnvStruct(rv, osEnvSource{}); err != nil {
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
			svc, err := newService[envConfig]()
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if err := svc.SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := svc.ApplyEnvStruct(rv, osEnvSource{}); err != nil {
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
			svc, err := newService[envConfig]()
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if err := svc.SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := svc.ApplyEnvStruct(rv, osEnvSource{}); err != nil {
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
			svc, err := newService[envConfig]()
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if err := svc.SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := svc.ApplyEnvStruct(rv, osEnvSource{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

type outerDef struct {
	Inner  innerDef       `default:"dive"`
	PInner *innerDef      `default:"dive"`
	N      int            `default:"9"`
	S      []string       `default:"alloc"`
	M      map[string]int `default:"alloc"`
	PInt   *int           `default:"7"`
}

type nonStructDiveDef struct {
	N int `default:"dive"`
}

func TestSetDefaultsStruct_DefaultTags(t *testing.T) {
	tests := []struct {
		name   string
		obj    outerDef
		verify func(t *testing.T, got outerDef)
	}{
		{
			name: "dive on struct field applies inner defaults",
			obj:  outerDef{},
			verify: func(t *testing.T, got outerDef) {
				if got.Inner.S != "x" || got.Inner.N != 42 {
					t.Fatalf("expected inner defaults applied, got %+v", got.Inner)
				}
			},
		},
		{
			name: "dive on nil pointer to struct allocates and applies defaults",
			obj:  outerDef{PInner: nil},
			verify: func(t *testing.T, got outerDef) {
				if got.PInner == nil || got.PInner.S != "x" || got.PInner.N != 42 {
					t.Fatalf("expected allocated PInner with defaults, got %+v", got.PInner)
				}
			},
		},
		{
			name: "alloc on nil slice allocates empty slice",
			obj:  outerDef{S: nil},
			verify: func(t *testing.T, got outerDef) {
				if got.S == nil || len(got.S) != 0 {
					t.Fatalf("expected allocated empty slice, got %#v", got.S)
				}
			},
		},
		{
			name: "alloc on nil map allocates empty map",
			obj:  outerDef{M: nil},
			verify: func(t *testing.T, got outerDef) {
				if got.M == nil || len(got.M) != 0 {
					t.Fatalf("expected allocated empty map, got %#v", got.M)
				}
			},
		},
		{
			name: "literal default on nil pointer to scalar allocates and sets",
			obj:  outerDef{PInt: nil},
			verify: func(t *testing.T, got outerDef) {
				if got.PInt == nil || *got.PInt != 7 {
					t.Fatalf("expected allocated *int==7, got %#v", got.PInt)
				}
			},
		},
		{
			name: "literal default on non-zero int leaves it unchanged",
			obj:  outerDef{N: 5},
			verify: func(t *testing.T, got outerDef) {
				if got.N != 5 {
					t.Fatalf("expected N to remain 5, got %d", got.N)
				}
			},
		},
		{
			name: "literal default on zero int sets value",
			obj:  outerDef{N: 0},
			verify: func(t *testing.T, got outerDef) {
				if got.N != 9 {
					t.Fatalf("expected N to be 9, got %d", got.N)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.obj
			rv := reflect.ValueOf(&got).Elem()

			svc, err := newService[outerDef]()
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if err := svc.SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.verify(t, got)
		})
	}
}

func TestSetDefaultsStruct_DiveOnNonStructIsNoop(t *testing.T) {
	got := nonStructDiveDef{}
	rv := reflect.ValueOf(&got).Elem()

	svc, err := newService[nonStructDiveDef]()
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if err := svc.SetDefaultsStruct(rv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.N != 0 {
		t.Fatalf("expected N unchanged, got %d", got.N)
	}
}

// Types and holder for applyDefaultElemTag tests
type innerElem struct {
	S string `default:"x"`
	N int    `default:"42"`
}

type elemHolder struct {
	People []innerElem           `defaultElem:"dive"`
	Ptrs   []*innerElem          `defaultElem:"dive"`
	Arr    [2]innerElem          `defaultElem:"dive"`
	Mv     map[string]innerElem  `defaultElem:"dive"`
	Mp     map[string]*innerElem `defaultElem:"dive"`
	Not    string                `defaultElem:"dive"`
}

func TestSetDefaultsStruct_DefaultElemTags(t *testing.T) {
	tests := []struct {
		name   string
		obj    elemHolder
		verify func(t *testing.T, got elemHolder)
	}{
		{
			name: "slice of struct elements dive",
			obj:  elemHolder{People: []innerElem{{}, {S: "ok"}}},
			verify: func(t *testing.T, got elemHolder) {
				if got.People[0].S != "x" || got.People[0].N != 42 {
					t.Fatalf("expected defaults on People[0], got %+v", got.People[0])
				}
				if got.People[1].S != "ok" || got.People[1].N != 42 {
					t.Fatalf("expected partial defaults on People[1], got %+v", got.People[1])
				}
			},
		},
		{
			name: "slice of pointer to struct elements dive with nil skipped",
			obj:  elemHolder{Ptrs: []*innerElem{nil, {}}},
			verify: func(t *testing.T, got elemHolder) {
				if got.Ptrs[0] != nil {
					t.Fatalf("expected Ptrs[0] to remain nil")
				}
				if got.Ptrs[1] == nil || got.Ptrs[1].S != "x" || got.Ptrs[1].N != 42 {
					t.Fatalf("expected defaults on Ptrs[1], got %#v", got.Ptrs[1])
				}
			},
		},
		{
			name: "array of struct elements dive",
			obj:  elemHolder{Arr: [2]innerElem{{}, {S: "ok"}}},
			verify: func(t *testing.T, got elemHolder) {
				if got.Arr[0].S != "x" || got.Arr[0].N != 42 {
					t.Fatalf("expected defaults on Arr[0], got %+v", got.Arr[0])
				}
				if got.Arr[1].S != "ok" || got.Arr[1].N != 42 {
					t.Fatalf("expected partial defaults on Arr[1], got %+v", got.Arr[1])
				}
			},
		},
		{
			name: "map of struct values dive with copy write back",
			obj:  elemHolder{Mv: map[string]innerElem{"a": {}, "b": {S: "ok"}}},
			verify: func(t *testing.T, got elemHolder) {
				if got.Mv["a"].S != "x" || got.Mv["a"].N != 42 {
					t.Fatalf("expected defaults on Mv[a], got %+v", got.Mv["a"])
				}
				if got.Mv["b"].S != "ok" || got.Mv["b"].N != 42 {
					t.Fatalf("expected partial defaults on Mv[b], got %+v", got.Mv["b"])
				}
			},
		},
		{
			name: "map of pointer to struct values dive with nil skipped",
			obj:  elemHolder{Mp: map[string]*innerElem{"p1": {}, "p2": nil}},
			verify: func(t *testing.T, got elemHolder) {
				if got.Mp["p1"] == nil || got.Mp["p1"].S != "x" || got.Mp["p1"].N != 42 {
					t.Fatalf("expected defaults on Mp[p1], got %#v", got.Mp["p1"])
				}
				if got.Mp["p2"] != nil {
					t.Fatalf("expected Mp[p2] to remain nil")
				}
			},
		},
		{
			name: "defaultElem on non-collection field ignored",
			obj:  elemHolder{Not: ""},
			verify: func(t *testing.T, got elemHolder) {
				if got.Not != "" {
					t.Fatalf("expected Not unchanged, got %q", got.Not)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.obj
			rv := reflect.ValueOf(&got).Elem()

			svc, err := newService[elemHolder]()
			if err != nil {
				t.Fatalf("unexpected service error: %v", err)
			}
			if err := svc.SetDefaultsStruct(rv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.verify(t, got)
		})
	}
}
