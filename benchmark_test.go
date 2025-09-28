package model

import (
	"sync"
	"testing"
	"time"
)

// Benchmark helper struct
type benchStruct struct {
	S string `validate:"nonempty"`
	I int    `validate:"positive,nonzero"`
	D int64  `validate:"nonzero"`
}

// resetBuiltinsForBench provides a way to re-run lazy init cost in benchmarks.
// Not part of the public API; used only in benchmarks.
func resetBuiltinsForBench() {
	// Reinitialize the sync.Once by assigning a new zero value.
	builtInsOnce = sync.Once{}
	builtInMap = nil
	builtinStringRules = nil
	builtinIntRules = nil
	builtinInt64Rules = nil
	builtinFloat64Rules = nil
}

// BenchmarkBuiltinColdStart measures first validation triggering lazy initialization.
func BenchmarkBuiltinColdStart(b *testing.B) {
	resetBuiltinsForBench()
	obj := benchStruct{S: "abc", I: 1, D: int64(time.Second)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ := New(&obj, WithValidation[benchStruct]())
		_ = m.Validate()
	}
}

// BenchmarkBuiltinWarm measures validation when built-ins already initialized.
func BenchmarkBuiltinWarm(b *testing.B) {
	// Force init once
	obj := benchStruct{S: "abc", I: 1, D: 1}
	m, _ := New(&obj, WithValidation[benchStruct]())
	_ = m.Validate()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Validate()
	}
}

// BenchmarkValidateNoBuiltins ensures path with custom rule only (simulate by using custom type field).
func BenchmarkValidateNoBuiltins(b *testing.B) {
	type custom struct {
		V string `validate:"cRule"`
	}
	cr, _ := NewRule[string]("cRule", func(s string, _ ...string) error { return nil })
	obj := custom{V: "x"}
	m, _ := New(&obj, WithRules[custom](cr), WithValidation[custom]())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Validate()
	}
}
