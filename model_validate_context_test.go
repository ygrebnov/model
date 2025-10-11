package model

import (
	"context"
	"errors"
	"testing"
	"time"
)

// simple type with repeating fields to ensure traversal loops would run if not canceled
type ctxObj struct {
	A string `validate:"nonempty"`
	B string `validate:"nonempty"`
	C string `validate:"nonempty"`
}

func TestValidate_ContextCanceled_ReturnsEarly(t *testing.T) {
	t.Parallel()
	obj := ctxObj{A: "", B: "", C: ""}
	m, err := New(&obj) // rely on built-in nonempty rules at Validate time
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Validate(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestNew_WithValidationContext_ValidateDuringNewCanceled(t *testing.T) {
	t.Parallel()
	obj := ctxObj{A: "", B: "", C: ""}
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	// Sleep a tick to ensure timeout expires before New runs validation
	time.Sleep(time.Millisecond)
	_, err := New(&obj,
		WithValidation[ctxObj](ctx),
	)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context deadline/cancel error, got %v", err)
	}
}
