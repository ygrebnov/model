package model

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// simple type with repeating fields to ensure traversal loops would run if not canceled
type ctxObj struct {
	A string `validate:"min(1)"`
	B string `validate:"min(1)"`
	C string `validate:"min(1)"`
}

func TestValidate_ContextCanceled_ReturnsEarly(t *testing.T) {
	t.Parallel()
	obj := ctxObj{A: "", B: "", C: ""}
	m, err := New(&obj) // rely on built-in min rules at Validate time
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

// Test that long-running per-element validation is canceled mid-way when the context is canceled.
func TestValidate_LongRunning_CanceledMidway(t *testing.T) {
	t.Parallel()
	// Object with many elements validated via validateElem
	type LR struct {
		Items []string `validateElem:"slow"`
	}

	var processed int32
	slowRule, err := NewRule[string]("slow", func(s string, _ ...string) error {
		time.Sleep(5 * time.Millisecond) // simulate work per element
		atomic.AddInt32(&processed, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}

	obj := LR{Items: make([]string, 200)} // many elements
	m, err := New(&obj, WithRules[LR](slowRule))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after starting validation
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err = m.Validate(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if atomic.LoadInt32(&processed) >= int32(len(obj.Items)) {
		t.Fatalf("expected to cancel before processing all elements; processed=%d total=%d", processed, len(obj.Items))
	}
}
