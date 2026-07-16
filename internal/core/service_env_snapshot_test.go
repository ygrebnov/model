package core

import (
	"reflect"
	"testing"

	"github.com/ygrebnov/model/internal/schema"
)

func TestSnapshotEnvSource_CollectionPrefix(t *testing.T) {
	t.Setenv("M_ONE", "2")

	type config struct {
		M map[string]int `env:"M"`
	}

	compiled, err := schema.New[config]()
	if err != nil {
		t.Fatalf("schema.New() error: %v", err)
	}

	service := Service[config]{
		schema: compiled,
	}

	value, ok := service.snapshotEnvSource().Lookup("M_ONE")
	if !ok || value != "2" {
		t.Fatalf("snapshot M_ONE = %q, %t; want %q, true", value, ok, "2")
	}
}

func TestApplyEnvStruct_MapLiteral(t *testing.T) {
	type config struct {
		M map[string]int `env:"M"`
	}

	compiled, err := schema.New[config]()
	if err != nil {
		t.Fatalf("schema.New() error: %v", err)
	}

	service := Service[config]{
		schema: compiled,
		envSource: envSnapshotSource{
			"M_ONE": "2",
		},
	}
	obj := config{
		M: map[string]int{"one": 1},
	}

	if err := service.ApplyEnvStruct(reflect.ValueOf(&obj).Elem()); err != nil {
		t.Fatalf("ApplyEnvStruct() error: %v", err)
	}

	if obj.M["one"] != 2 {
		t.Fatalf("M[one] = %d, want 2", obj.M["one"])
	}
}
