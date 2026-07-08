package schema

import (
	"errors"
	"reflect"
	"testing"

	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

func TestCompile(t *testing.T) {
	type credentials struct {
		User     string `json:"user_name,omitempty" validate:"required"`
		Password string `env:"db_password" validate:"required"`
	}

	type database struct {
		Host        string      `json:"host" default:"localhost"`
		Credentials credentials `json:"credentials" default:"dive"`
		Skipped     string      `env:"-"`
		Fallback    string      `json:"-"`
		hidden      string
	}

	type config struct {
		Database database      `json:"database"`
		Ptr      *database     `json:"ptr"`
		Items    []credentials `json:"items" defaultElem:"dive" validateElem:"dive"`
		Disabled credentials   `env:"-"`
	}

	s, err := Compile(reflect.TypeOf(config{}))
	if err != nil {
		t.Fatalf("Compile returned unexpected error: %v", err)
	}

	if s.Type != reflect.TypeOf(config{}) {
		t.Fatalf("Schema.Type = %v, want %v", s.Type, reflect.TypeOf(config{}))
	}

	databaseNode, ok := s.Lookup("Database")
	if !ok {
		t.Fatal("Lookup(Database) did not find a node")
	}
	if databaseNode.JSONName != "database" {
		t.Fatalf("Database.JSONName = %q, want %q", databaseNode.JSONName, "database")
	}
	if !databaseNode.EnvEnabled {
		t.Fatal("Database.EnvEnabled = false, want true")
	}
	if !reflect.DeepEqual(databaseNode.EnvPath, []string{"DATABASE"}) {
		t.Fatalf("Database.EnvPath = %v, want %v", databaseNode.EnvPath, []string{"DATABASE"})
	}

	passwordNode, ok := s.Lookup("Database.Credentials.Password")
	if !ok {
		t.Fatal("Lookup(Database.Credentials.Password) did not find a node")
	}
	if passwordNode.EnvName != "DB_PASSWORD" {
		t.Fatalf("Password.EnvName = %q, want %q", passwordNode.EnvName, "DB_PASSWORD")
	}
	if !reflect.DeepEqual(passwordNode.EnvPath, []string{"DATABASE", "CREDENTIALS", "DB_PASSWORD"}) {
		t.Fatalf("Password.EnvPath = %v, want %v", passwordNode.EnvPath, []string{"DATABASE", "CREDENTIALS", "DB_PASSWORD"})
	}
	if passwordNode.ValidateTag != "required" {
		t.Fatalf("Password.ValidateTag = %q, want %q", passwordNode.ValidateTag, "required")
	}

	skippedNode, ok := s.Lookup("Database.Skipped")
	if !ok {
		t.Fatal("Lookup(Database.Skipped) did not find a node")
	}
	if skippedNode.EnvEnabled {
		t.Fatal("Skipped.EnvEnabled = true, want false")
	}
	if skippedNode.EnvPath != nil {
		t.Fatalf("Skipped.EnvPath = %v, want nil", skippedNode.EnvPath)
	}

	fallbackNode, ok := s.Lookup("Database.Fallback")
	if !ok {
		t.Fatal("Lookup(Database.Fallback) did not find a node")
	}
	if fallbackNode.EnvName != "FALLBACK" {
		t.Fatalf("Fallback.EnvName = %q, want %q", fallbackNode.EnvName, "FALLBACK")
	}

	if _, ok := s.Lookup("Database.hidden"); ok {
		t.Fatal("Lookup(Database.hidden) found an unexported field")
	}

	ptrNode, ok := s.Lookup("Ptr")
	if !ok {
		t.Fatal("Lookup(Ptr) did not find a node")
	}
	if _, ok := ptrNode.Child("Credentials"); !ok {
		t.Fatal("Ptr.Child(Credentials) did not find nested pointer child")
	}

	itemsNode, ok := s.Lookup("Items")
	if !ok {
		t.Fatal("Lookup(Items) did not find a node")
	}
	if !itemsNode.IsCollection() {
		t.Fatal("Items.IsCollection() = false, want true")
	}
	if len(itemsNode.Children) != 0 {
		t.Fatalf("len(Items.Children) = %d, want 0", len(itemsNode.Children))
	}
	elemType, ok := itemsNode.CollectionElementType()
	if !ok || elemType != reflect.TypeOf(credentials{}) {
		t.Fatalf("Items.CollectionElementType() = (%v, %v), want (%v, true)", elemType, ok, reflect.TypeOf(credentials{}))
	}
	if itemsNode.DefaultElemTag != "dive" {
		t.Fatalf("Items.DefaultElemTag = %q, want %q", itemsNode.DefaultElemTag, "dive")
	}
	if itemsNode.ValidateElemTag != "dive" {
		t.Fatalf("Items.ValidateElemTag = %q, want %q", itemsNode.ValidateElemTag, "dive")
	}

	disabledPasswordNode, ok := s.Lookup("Disabled.Password")
	if !ok {
		t.Fatal("Lookup(Disabled.Password) did not find a node")
	}
	if disabledPasswordNode.EnvEnabled {
		t.Fatal("Disabled.Password.EnvEnabled = true, want false")
	}
	if disabledPasswordNode.EnvPath != nil {
		t.Fatalf("Disabled.Password.EnvPath = %v, want nil", disabledPasswordNode.EnvPath)
	}
}

func TestCompileRecursiveType(t *testing.T) {
	type recursive struct {
		Name string
		Next *recursive `default:"dive"`
	}

	s, err := Compile(reflect.TypeOf(recursive{}))
	if err != nil {
		t.Fatalf("Compile returned unexpected error: %v", err)
	}

	nextNode, ok := s.Lookup("Next")
	if !ok {
		t.Fatal("Lookup(Next) did not find a node")
	}
	if !nextNode.Recursive {
		t.Fatal("Next.Recursive = false, want true")
	}
	if len(nextNode.Children) != 0 {
		t.Fatalf("len(Next.Children) = %d, want 0", len(nextNode.Children))
	}
}

func TestCompileRejectsNonStruct(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "nil", typ: nil},
		{name: "scalar", typ: reflect.TypeOf(1)},
		{name: "interface", typ: reflect.TypeOf((*error)(nil)).Elem()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Compile(tc.typ)
			if !errors.Is(err, modelerrors.ErrTypeParamNotStruct) {
				t.Fatalf("Compile error = %v, want %v", err, modelerrors.ErrTypeParamNotStruct)
			}
			if s != nil {
				t.Fatalf("Compile returned schema = %#v, want nil", s)
			}
		})
	}
}
