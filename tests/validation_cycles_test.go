package tests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/validation"
)

type cycleNode struct {
	Name string     `validate:"nonempty"`
	Next *cycleNode `validate:""`
}

type cycleRoot struct {
	First  *cycleNode   `validate:""`
	Second *cycleNode   `validate:""`
	Items  []*cycleNode `validateElem:"dive"`
}

func newCycleBinding(t *testing.T) *model.Binding[cycleRoot] {
	t.Helper()

	nonempty, err := model.NewRule[string]("nonempty", ruleNonEmpty)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}

	b, err := model.NewBinding[cycleRoot](model.WithRules(nonempty))
	if err != nil {
		t.Fatalf("NewBinding error: %v", err)
	}

	return b
}

func validationByField(t *testing.T, err error) map[string][]validation.FieldError {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var ve *validation.Error
	if !errors.As(err, &ve) {
		t.Fatalf("expected *validation.Error, got %T", err)
	}

	return ve.ByField()
}

func TestBinding_Validate_SkipsPointerCycles(t *testing.T) {
	b := newCycleBinding(t)

	first := &cycleNode{}
	second := &cycleNode{}
	first.Next = second
	second.Next = first

	obj := cycleRoot{First: first}
	by := validationByField(t, b.Validate(context.Background(), &obj))

	if _, ok := by["First.Name"]; !ok {
		t.Fatalf("expected validation error at First.Name, got: %+v", by)
	}
	if _, ok := by["First.Next.Name"]; !ok {
		t.Fatalf("expected validation error at First.Next.Name, got: %+v", by)
	}

	for path := range by {
		if strings.HasPrefix(path, "First.Next.Next") {
			t.Fatalf("unexpected recursive validation past cycle boundary at %s", path)
		}
	}
}

func TestBinding_Validate_RevisitsSharedNodesByPath(t *testing.T) {
	b := newCycleBinding(t)

	shared := &cycleNode{}
	shared.Next = shared

	obj := cycleRoot{
		First:  shared,
		Second: shared,
	}

	by := validationByField(t, b.Validate(context.Background(), &obj))

	if _, ok := by["First.Name"]; !ok {
		t.Fatalf("expected validation error at First.Name, got: %+v", by)
	}
	if _, ok := by["Second.Name"]; !ok {
		t.Fatalf("expected validation error at Second.Name, got: %+v", by)
	}
}

func TestBinding_ValidateElemDive_SkipsPointerCycles(t *testing.T) {
	b := newCycleBinding(t)

	first := &cycleNode{}
	second := &cycleNode{}
	first.Next = second
	second.Next = first

	obj := cycleRoot{
		Items: []*cycleNode{first},
	}

	by := validationByField(t, b.Validate(context.Background(), &obj))

	if _, ok := by["Items[0].Name"]; !ok { // actual key "Items[][0].Name"
		t.Fatalf("expected validation error at Items[0].Name, got: %+v", by)
	}
	if _, ok := by["Items[0].Next.Name"]; !ok {
		t.Fatalf("expected validation error at Items[0].Next.Name, got: %+v", by)
	}

	for path := range by {
		if strings.HasPrefix(path, "Items[0].Next.Next") {
			t.Fatalf("unexpected recursive validation past cycle boundary at %s", path)
		}
	}
}
