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

	if _, ok := by["first.name"]; !ok {
		t.Fatalf("expected validation error at first.name, got: %+v", by)
	}
	if _, ok := by["first.next.name"]; !ok {
		t.Fatalf("expected validation error at first.next.name, got: %+v", by)
	}

	for path := range by {
		if strings.HasPrefix(path, "first.next.next") {
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

	if _, ok := by["first.name"]; !ok {
		t.Fatalf("expected validation error at first.name, got: %+v", by)
	}
	if _, ok := by["second.name"]; !ok {
		t.Fatalf("expected validation error at second.name, got: %+v", by)
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

	if _, ok := by["items[0].name"]; !ok { // actual key "items[][0].name"
		t.Fatalf("expected validation error at items[0].name, got: %+v", by)
	}
	if _, ok := by["items[0].next.name"]; !ok {
		t.Fatalf("expected validation error at items[0].next.name, got: %+v", by)
	}

	for path := range by {
		if strings.HasPrefix(path, "items[0].next.next") {
			t.Fatalf("unexpected recursive validation past cycle boundary at %s", path)
		}
	}
}
