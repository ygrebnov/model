package tests

import (
	"context"
	"testing"

	"github.com/ygrebnov/model"
)

type validateWithDefaultsSample struct {
	Name  string `default:"s" validate:"min(1)"`
	Count int    `default:"5" validate:"min(3),max(10),nonzero"`
}

func TestValidateWithDefaults_Nominal(t *testing.T) {
	obj := validateWithDefaultsSample{}
	expected := validateWithDefaultsSample{
		Name:  "s",
		Count: 5,
	}

	if err := model.ValidateWithDefaults(context.Background(), &obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj != expected {
		t.Fatalf("got %+v, want %+v", obj, expected)
	}
}

func TestValidateWithDefaults_EquivalentToDefaultsEnvThenValidate(t *testing.T) {
	t.Run("typed", func(t *testing.T) {
		t.Setenv("NAME", "typed_env")
		t.Setenv("COUNT", "7")

		gotCombined := validateWithDefaultsSample{}
		gotSplit := validateWithDefaultsSample{}

		errCombined := model.ValidateWithDefaults(context.Background(), &gotCombined)
		if errCombined != nil {
			t.Fatalf("ValidateWithDefaults returned error: %v", errCombined)
		}

		b, err := model.NewBinding[validateWithDefaultsSample]()
		if err != nil {
			t.Fatalf("NewBinding returned error: %v", err)
		}
		if err := applyBindingDefaultsAndEnv(b, &gotSplit); err != nil {
			t.Fatalf("defaults/env split returned error: %v", err)
		}
		errSplit := model.Validate(context.Background(), &gotSplit)
		if errSplit != nil {
			t.Fatalf("Validate returned error: %v", errSplit)
		}

		if gotCombined != gotSplit {
			t.Fatalf("combined=%+v split=%+v", gotCombined, gotSplit)
		}
	})
}
