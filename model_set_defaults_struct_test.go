package model

import (
	"errors"
	"reflect"
	"testing"
	"time"

	modelerrors "github.com/ygrebnov/model/errors"
)

type defInner struct {
	A string        `default:"x"`
	B int           `default:"5"`
	D time.Duration `default:"2s"`
}

type defOuter struct {
	// recurse
	In   defInner  `default:"dive"`
	PIn  *defInner `default:"dive"`
	PInt *int      `default:"dive"` // non-struct pointer -> ignore

	// collections + element recursion
	SS    []string             `default:"alloc"`
	M     map[string]defInner  `defaultElem:"dive"`
	MPtr  map[string]*defInner `default:"alloc" defaultElem:"dive"`
	Arr   []defInner           `defaultElem:"dive"`
	ArrPt []*defInner          `defaultElem:"dive"`
	S2    string               `defaultElem:"dive"` // non-collection -> ignore
	unexp string               `default:"zzz"`      // unexported -> skipped

	// literals
	S   string        `default:"hello"`
	I   int           `default:"42"`
	U   uint          `default:"7"`
	F64 float64       `default:"3.5"`
	B   bool          `default:"true"`
	D   time.Duration `default:"1s"`

	// error case (unsupported literal kind)
	BadStruct defInner `default:"oops"`
}

func newModelWithDefaults[T any](t *testing.T, obj *T) *Model[T] {
	t.Helper()
	var m Model[T]
	m.obj = obj
	if err := m.ensureBinding(); err != nil {
		t.Fatalf("unexpected error in ensureBinding: %v", err)
	}
	return &m
}

func mustSetDefaultsStruct[T any](t *testing.T, m *Model[T]) error {
	t.Helper()
	return m.service.SetDefaultsStruct(reflect.ValueOf(m.obj).Elem())
}

func assertSetDefaultStructFieldError(t *testing.T, err error, wantField string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, modelerrors.ErrSetDefault) {
		t.Fatalf("expected ErrSetDefault, got %v", err)
	}
	// We don't have direct field extraction helpers; check that error is wrapped
	// with the field name metadata using errorc's formatting convention.
	msg := err.Error()
	if msg == "" {
		t.Fatalf("expected non-empty error message")
	}
	// The error message is expected to contain the field key and name, e.g.
	// "model: cannot set default value, model.field.name: BadStruct".
	// Instead of using strings.Contains in call sites, centralize this check here.
	key := string(modelerrors.ErrorFieldFieldName) + ": " + wantField
	if !contains(msg, key) {
		t.Fatalf("expected error message to contain field key %q, got: %q", key, msg)
	}
}

// contains is a tiny local helper mirrors strings.Contains but keeps
// string-search details out of the assertion logic above.
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (func() bool {
		// simple substring scan without importing strings
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})())
}

func TestModel_SetDefaultsStruct(t *testing.T) {
	t.Run("SetDefaultsStruct table", func(t *testing.T) {
		// Define local types used only in this test suite
		type okOuter struct {
			In   defInner  `default:"dive"`
			PIn  *defInner `default:"dive"`
			PInt *int      `default:"dive"`

			SS    []string             `default:"alloc"`
			M     map[string]defInner  `defaultElem:"dive"`
			MPtr  map[string]*defInner `default:"alloc" defaultElem:"dive"`
			Arr   []defInner           `defaultElem:"dive"`
			ArrPt []*defInner          `defaultElem:"dive"`
			S2    string               `defaultElem:"dive"`

			S   string        `default:"hello"`
			I   int           `default:"42"`
			U   uint          `default:"7"`
			F64 float64       `default:"3.5"`
			B   bool          `default:"true"`
			D   time.Duration `default:"1s"`
			// unexported is skipped
			unexp string `default:"zzz"`
		}

		type allocOuter struct {
			SS []string            `default:"alloc"`
			M  map[string]defInner `default:"alloc"`
		}

		type ptrScalars struct {
			PI *int           `default:"9"`
			PD *time.Duration `default:"250ms"`
		}

		type bad struct {
			S  string   `default:"ok"`
			St defInner `default:"oops"` // unsupported kind for literal
		}

		// Table of scenarios
		tests := []struct {
			name    string
			prepare func(t *testing.T) (run func(t *testing.T))
		}{
			{
				name: "error on unsupported literal kind in BadStruct field",
				prepare: func(t *testing.T) func(t *testing.T) {
					o := defOuter{
						Arr: []defInner{
							{},
							{A: "keepA", B: 9},
						},
						ArrPt: []*defInner{
							nil,
							{},
						},
						M: map[string]defInner{
							"k1": {},
						},
						MPtr: map[string]*defInner{
							"p1": {},
						},
					}
					o.ArrPt[1] = &defInner{}
					m := newModelWithDefaults(t, &o)
					return func(t *testing.T) {
						err := mustSetDefaultsStruct(t, m)
						assertSetDefaultStructFieldError(t, err, "BadStruct")
					}
				},
			},
			{
				name: "happy path without error field",
				prepare: func(t *testing.T) func(t *testing.T) {
					o := okOuter{
						Arr: []defInner{
							{},
							{A: "keepA", B: 9},
						},
						ArrPt: []*defInner{nil, &defInner{}},
						M: map[string]defInner{
							"k1": {},
							"k2": {A: "preset"},
						},
						MPtr: map[string]*defInner{
							"p1": &defInner{},
							"p2": nil,
						},
					}
					m := newModelWithDefaults(t, &o)
					return func(t *testing.T) {
						if err := mustSetDefaultsStruct(t, m); err != nil {
							t.Fatalf("unexpected error: %v", err)
						}

						// Top-level literals
						if o.S != "hello" || o.I != 42 || o.U != 7 || o.F64 != 3.5 || o.B != true || o.D != time.Second {
							t.Fatalf("top-level literals not applied correctly: %+v", o)
						}

						// In (struct) via dive
						if o.In.A != "x" || o.In.B != 5 || o.In.D != 2*time.Second {
							t.Fatalf("dive on struct failed: %+v", o.In)
						}

						// PIn (pointer to struct) via dive + allocation
						if o.PIn == nil {
							t.Fatalf("PIn should have been allocated")
						}
						if o.PIn.A != "x" || o.PIn.B != 5 || o.PIn.D != 2*time.Second {
							t.Fatalf("dive on *struct failed: %+v", *o.PIn)
						}

						// PInt (pointer to non-struct) with dive ignored
						if o.PInt != nil {
							t.Fatalf("PInt should remain nil (dive ignored for non-struct pointer)")
						}

						// alloc: SS remains non-nil (empty)
						if o.SS == nil || len(o.SS) != 0 {
							t.Fatalf("SS should be allocated empty slice, got %#v", o.SS)
						}

						// M: element dive into values
						v1 := o.M["k1"]
						if v1.A != "x" || v1.B != 5 || v1.D != 2*time.Second {
							t.Fatalf("M[k1] defaults not applied: %+v", v1)
						}
						v2 := o.M["k2"]
						if v2.A != "preset" || v2.B != 5 || v2.D != 2*time.Second {
							t.Fatalf("M[k2] defaults not merged correctly: %+v", v2)
						}

						// MPtr: element dive into *values
						if o.MPtr["p1"] == nil {
							t.Fatalf("MPtr[p1] should not be nil")
						}
						if o.MPtr["p1"].A != "x" || o.MPtr["p1"].B != 5 || o.MPtr["p1"].D != 2*time.Second {
							t.Fatalf("MPtr[p1] defaults not applied: %+v", *o.MPtr["p1"])
						}
						// nil value remains nil (defaults do not allocate map values)
						if o.MPtr["p2"] != nil {
							t.Fatalf("MPtr[p2] should remain nil")
						}

						// Arr: element dive for value slice
						if o.Arr[0].A != "x" || o.Arr[0].B != 5 || o.Arr[0].D != 2*time.Second {
							t.Fatalf("Arr[0] defaults not applied: %+v", o.Arr[0])
						}
						if o.Arr[1].A != "keepA" || o.Arr[1].B != 9 || o.Arr[1].D != 2*time.Second {
							t.Fatalf("Arr[1] merge not correct: %+v", o.Arr[1])
						}

						// ArrPt: element dive for pointer slice; nil stays nil, non-nil is updated
						if o.ArrPt[0] != nil {
							t.Fatalf("ArrPt[0] should remain nil")
						}
						if o.ArrPt[1] == nil || o.ArrPt[1].A != "x" || o.ArrPt[1].B != 5 || o.ArrPt[1].D != 2*time.Second {
							t.Fatalf("ArrPt[1] defaults not applied: %#v", o.ArrPt[1])
						}

						// defaultElem ignored on non-collection
						if o.S2 != "hello" && o.S2 != "" {
							// S2 has no default literal; it should remain zero-value "" (or "hello" if you add a default)
							// Just ensure we didn't touch it due to defaultElem
							t.Fatalf("S2 should not be affected by defaultElem")
						}

						// unexported field must not be modified
						if o.unexp != "" {
							t.Fatalf("unexported field should be skipped, got %q", o.unexp)
						}
					}
				},
			},
			{
				name: "alloc slice/map when nil and idempotent",
				prepare: func(t *testing.T) func(t *testing.T) {
					o := allocOuter{}
					m := newModelWithDefaults(t, &o)
					return func(t *testing.T) {
						if err := mustSetDefaultsStruct(t, m); err != nil {
							t.Fatalf("unexpected err: %v", err)
						}
						if o.SS == nil || len(o.SS) != 0 {
							t.Fatalf("SS should be allocated empty slice")
						}
						if o.M == nil || len(o.M) != 0 {
							t.Fatalf("M should be allocated empty map")
						}
						// run again (idempotent)
						if err := mustSetDefaultsStruct(t, m); err != nil {
							t.Fatalf("unexpected err on second run: %v", err)
						}
						if o.SS == nil || o.M == nil {
							t.Fatalf("alloc should remain allocated")
						}
					}
				},
			},
			{
				name: "literal defaults for pointer-to-scalar: allocate when nil, no overwrite when non-zero",
				prepare: func(t *testing.T) func(t *testing.T) {
					o := ptrScalars{}
					m := newModelWithDefaults(t, &o)
					return func(t *testing.T) {
						if err := mustSetDefaultsStruct(t, m); err != nil {
							t.Fatalf("unexpected err: %v", err)
						}
						if o.PI == nil || *o.PI != 9 {
							t.Fatalf("PI should be *int(9), got %#v", o.PI)
						}
						if o.PD == nil || *o.PD != 250*time.Millisecond {
							t.Fatalf("PD should be 250ms, got %#v", o.PD)
						}
						// Non-zero should not be overwritten
						*o.PI = 5
						*o.PD = time.Second
						if err := mustSetDefaultsStruct(t, m); err != nil {
							t.Fatalf("unexpected err on second run: %v", err)
						}
						if *o.PI != 5 || *o.PD != time.Second {
							t.Fatalf("non-zero pointer scalars should not be overwritten: %v %v", *o.PI, *o.PD)
						}
					}
				},
			},
			{
				name: "error unsupported literal kind wrapped with field name St",
				prepare: func(t *testing.T) func(t *testing.T) {
					o := bad{}
					m := newModelWithDefaults(t, &o)
					return func(t *testing.T) {
						err := mustSetDefaultsStruct(t, m)
						assertSetDefaultStructFieldError(t, err, "St")
					}
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				run := tc.prepare(t)
				run(t)
			})
		}
	})
}
