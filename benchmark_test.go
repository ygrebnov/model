package model

import (
	"context"
	"testing"

	"github.com/ygrebnov/model/validation"
)

// Benchmark helper struct
type benchStruct struct {
	S string `validate:"min(1)"`
	I int    `validate:"positive,nonzero"`
	D int64  `validate:"nonzero"`
}

// mediumRule is a medium-level CPU and memory usage rule used for benchmarking.
// It performs a moderate amount of work on the input string: allocations,
// iterations, and some branching.
func mediumRule(s string, _ ...string) error {
	// Allocate a buffer proportional to input length
	buf := make([]byte, 0, len(s)*2)
	// Simple transformation with some branching
	for i := 0; i < 100; i++ {
		for j := 0; j < len(s); j++ {
			c := s[j]
			if c%2 == 0 {
				buf = append(buf, c^0x1)
			} else {
				buf = append(buf, c^0x2)
			}
		}
	}
	if len(buf) > 0 {
		_ = buf[0]
	}
	return nil
}

// mediumStruct is a medium-size struct used for validation benchmarks.
// Several fields reuse the same mediumRule validation.
type mediumStruct struct {
	F1  string `validate:"medium"`
	F2  string `validate:"medium"`
	F3  string `validate:"medium"`
	F4  string `validate:"medium"`
	F5  string `validate:"medium"`
	F6  string `validate:"medium"`
	F7  string `validate:"medium"`
	F8  string `validate:"medium"`
	F9  string `validate:"medium"`
	F10 string `validate:"medium"`
	F11 string `validate:"medium"`
	F12 string `validate:"medium"`
	F13 string `validate:"medium"`
	F14 string `validate:"medium"`
	F15 string `validate:"medium"`
	F16 string `validate:"medium"`
	F17 string `validate:"medium"`
	F18 string `validate:"medium"`
	F19 string `validate:"medium"`
	F20 string `validate:"medium"`
}

// BenchmarkMediumValidate measures CPU and memory usage of validating a medium-size struct
// using a medium-complexity custom rule applied to many fields.
func BenchmarkMediumValidate(b *testing.B) {
	// Prepare a medium-sized object
	obj := mediumStruct{
		F1:  "some medium length string for benchmarking",
		F2:  "another medium length string for benchmarking",
		F3:  "third medium length string for benchmarking",
		F4:  "fourth medium length string for benchmarking",
		F5:  "fifth medium length string for benchmarking",
		F6:  "sixth medium length string for benchmarking",
		F7:  "seventh medium length string for benchmarking",
		F8:  "eighth medium length string for benchmarking",
		F9:  "ninth medium length string for benchmarking",
		F10: "tenth medium length string for benchmarking",
		F11: "eleventh medium length string for benchmarking",
		F12: "twelfth medium length string for benchmarking",
		F13: "thirteenth medium length string for benchmarking",
		F14: "fourteenth medium length string for benchmarking",
		F15: "fifteenth medium length string for benchmarking",
		F16: "sixteenth medium length string for benchmarking",
		F17: "seventeenth medium length string for benchmarking",
		F18: "eighteenth medium length string for benchmarking",
		F19: "nineteenth medium length string for benchmarking",
		F20: "twentieth medium length string for benchmarking",
	}

	rule, err := validation.NewRule[string]("medium", mediumRule)
	if err != nil {
		b.Fatalf("failed to create medium rule: %v", err)
	}

	m, err := New(&obj, WithRules[mediumStruct](rule), WithValidation[mediumStruct](context.Background()))
	if err != nil {
		b.Fatalf("failed to create model: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := m.Validate(context.Background()); err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}
}
