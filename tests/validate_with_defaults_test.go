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

func TestValidateWithDefaultsAny_Nominal(t *testing.T) {
	obj := validateWithDefaultsSample{}
	expected := validateWithDefaultsSample{
		Name:  "s",
		Count: 5,
	}

	if err := model.ValidateWithDefaultsAny(context.Background(), &obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj != expected {
		t.Fatalf("got %+v, want %+v", obj, expected)
	}
}

func TestValidateWithDefaults_EquivalentToSetDefaultsThenValidate(t *testing.T) {
	t.Run("typed", func(t *testing.T) {
		t.Setenv("NAME", "typed_env")
		t.Setenv("COUNT", "7")

		gotCombined := validateWithDefaultsSample{}
		gotSplit := validateWithDefaultsSample{}

		errCombined := model.ValidateWithDefaults(context.Background(), &gotCombined)
		if errCombined != nil {
			t.Fatalf("ValidateWithDefaults returned error: %v", errCombined)
		}

		if err := model.SetDefaults(&gotSplit); err != nil {
			t.Fatalf("SetDefaults returned error: %v", err)
		}
		errSplit := model.Validate(context.Background(), &gotSplit)
		if errSplit != nil {
			t.Fatalf("Validate returned error: %v", errSplit)
		}

		if gotCombined != gotSplit {
			t.Fatalf("combined=%+v split=%+v", gotCombined, gotSplit)
		}
	})

	t.Run("any", func(t *testing.T) {
		t.Setenv("NAME", "any_env")
		t.Setenv("COUNT", "8")

		gotCombined := validateWithDefaultsSample{}
		gotSplit := validateWithDefaultsSample{}

		errCombined := model.ValidateWithDefaultsAny(context.Background(), &gotCombined)
		if errCombined != nil {
			t.Fatalf("ValidateWithDefaultsAny returned error: %v", errCombined)
		}

		if err := model.SetDefaultsAny(&gotSplit); err != nil {
			t.Fatalf("SetDefaultsAny returned error: %v", err)
		}
		errSplit := model.ValidateAny(context.Background(), &gotSplit)
		if errSplit != nil {
			t.Fatalf("ValidateAny returned error: %v", errSplit)
		}

		if gotCombined != gotSplit {
			t.Fatalf("combined=%+v split=%+v", gotCombined, gotSplit)
		}
	})
}
