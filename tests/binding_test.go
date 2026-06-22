package tests

import (
	"errors"
	"testing"

	"github.com/ygrebnov/model"
	modelErrors "github.com/ygrebnov/model/pkg/errors"
)

func TestNewBinding(t *testing.T) {
	t.Run("struct type", func(t *testing.T) {
		b, err := model.NewBinding[Strings]()
		if err != nil {
			t.Fatalf("NewBinding[Strings] unexpected error: %v", err)
		}
		if b == nil {
			t.Fatalf("NewBinding[Strings] returned nil binding")
		}
	})

	t.Run("pointer to struct type", func(t *testing.T) {
		b, err := model.NewBinding[*Strings]()
		if !errors.Is(err, modelErrors.ErrTypeParamNotStruct) {
			t.Fatalf("NewBinding[*Strings] expected ErrTypeParamNotStruct, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewBinding[*Strings] expected nil binding on error")
		}
	})

	t.Run("int type", func(t *testing.T) {
		b, err := model.NewBinding[int]()
		if !errors.Is(err, modelErrors.ErrTypeParamNotStruct) {
			t.Fatalf("NewBinding[int] expected ErrTypeParamNotStruct, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewBinding[int] expected nil binding on error")
		}
	})

	t.Run("interface", func(t *testing.T) {
		b, err := model.NewBinding[interface{}]()
		if !errors.Is(err, modelErrors.ErrTypeParamNotStruct) {
			t.Fatalf("NewBinding[interface{}] expected ErrTypeParamNotStruct, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewBinding[interface{}] expected nil binding on error")
		}
	})
}

func TestNewDynamicBinding(t *testing.T) {
	t.Run("struct type", func(t *testing.T) {
		b, err := model.NewDynamicBinding(Strings{})
		if err == nil {
			t.Fatalf("NewDynamicBinding expected error to return errror, but got nil")
		}
		if !errors.Is(err, modelErrors.ErrNotStructPtr) {
			t.Fatalf("NewDynamicBinding expected ErrNotStructPtr, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewDynamicBinding expected nil binding on error")
		}
	})

	t.Run("pointer to struct type", func(t *testing.T) {
		b, err := model.NewDynamicBinding(&Strings{})
		if err != nil {
			t.Fatalf("NewDynamicBinding unexpected error: %v", err)
		}
		if b == nil {
			t.Fatalf("NewDynamicBinding returned nil binding")
		}
	})

	t.Run("int type", func(t *testing.T) {
		b, err := model.NewDynamicBinding(0)
		if !errors.Is(err, modelErrors.ErrNotStructPtr) {
			t.Fatalf("NewDynamicBinding expected ErrNotStructPtr, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewDynamicBinding expected nil binding on error")
		}
	})

	t.Run("interface", func(t *testing.T) {
		b, err := model.NewDynamicBinding(interface{}(nil))
		if !errors.Is(err, modelErrors.ErrNilObject) {
			t.Fatalf("NewDynamicBinding expected ErrNilObject, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewDynamicBinding expected nil binding on error")
		}
	})
}
