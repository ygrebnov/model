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
