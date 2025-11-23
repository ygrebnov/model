package model

import (
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestModel_SetDefaultsStruct(t *testing.T) {
	t.Run("happy path: dive, alloc, literals, element dive for slice/map", func(t *testing.T) {
		var m Model[defOuter]
		// Prepare an object that exercises recursion and element recursion
		o := defOuter{
			// Arr has a zero element and a pre-set element (B non-zero should not be overwritten)
			Arr: []defInner{
				{},                 // should get defaults A:x, B:5, D:2s
				{A: "keepA", B: 9}, // A should remain "keepA", B stays 9, D gets 2s if zero
			},
			// ArrPt has a nil element (ignored) and a non-nil element (should dive)
			ArrPt: []*defInner{
				nil,
				{}, // typed nil literal isn’t allowed; use &defInner{} below
			},
			// Map value dive for value types
			M: map[string]defInner{
				"k1": {}, // should be dived and get defaults
			},
			// Pre-populate MPtr despite default:"alloc"; alloc should be a no-op and dive should apply
			MPtr: map[string]*defInner{
				"p1": {},
			},
			// PIn is nil; dive should allocate *defInner and set its defaults
			PIn: nil,
			// PInt is a pointer to non-struct; dive should be ignored
			PInt: nil,
		}
		// Fix ArrPt second element to a real pointer
		o.ArrPt[1] = &defInner{}
		m.obj = &o

		if err := m.ensureBinding(); err != nil {
			t.Fatalf("unexpected error in ensureBinding: %v", err)
		}

		err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem())
		if err == nil {
			t.Fatal("expected error due to BadStruct literal, got nil")
		}
		// Ensure error is the wrapped field-specific one
		if !strings.Contains(err.Error(), "cannot set default value") {
			t.Fatalf("expected error mentioning BadStruct, got: %v", err)
		}

		// Because setDefaultsStruct returns on first error, the rest of assertions in this case
		// can be flaky. To fully verify happy path, run a second object without the failing field.
	})

	t.Run("happy path without error field", func(t *testing.T) {
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

		var m Model[okOuter]
		o := okOuter{
			Arr: []defInner{
				{},
				{A: "keepA", B: 9}, // D zero -> should get 2s
			},
			ArrPt: []*defInner{nil, &defInner{}},
			M: map[string]defInner{
				"k1": {},
				"k2": {A: "preset"}, // D zero -> should get 2s
			},
			// MPtr will be non-nil due to explicit init; alloc should be no-op
			MPtr: map[string]*defInner{
				"p1": &defInner{},
				"p2": nil, // nil value: dive will skip (no allocation happens in defaults)
			},
		}
		m.obj = &o

		if err := m.ensureBinding(); err != nil {
			t.Fatalf("unexpected error in ensureBinding: %v", err)
		}

		if err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem()); err != nil {
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
	})

	t.Run("alloc: slice/map when nil; no-op when already non-nil", func(t *testing.T) {
		type allocOuter struct {
			SS []string            `default:"alloc"`
			M  map[string]defInner `default:"alloc"`
		}
		var m Model[allocOuter]
		o := allocOuter{}
		m.obj = &o

		if err := m.ensureBinding(); err != nil {
			t.Fatalf("unexpected error in ensureBinding: %v", err)
		}

		if err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem()); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if o.SS == nil || len(o.SS) != 0 {
			t.Fatalf("SS should be allocated empty slice")
		}
		if o.M == nil || len(o.M) != 0 {
			t.Fatalf("M should be allocated empty map")
		}
		// run again (idempotent)
		if err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem()); err != nil {
			t.Fatalf("unexpected err on second run: %v", err)
		}
		if o.SS == nil || o.M == nil {
			t.Fatalf("alloc should remain allocated")
		}
	})

	t.Run("literal defaults for pointer-to-scalar: allocate when nil, no overwrite when non-zero", func(t *testing.T) {
		type ptrScalars struct {
			PI *int           `default:"9"`
			PD *time.Duration `default:"250ms"`
		}
		var m Model[ptrScalars]
		o := ptrScalars{}
		m.obj = &o

		if err := m.ensureBinding(); err != nil {
			t.Fatalf("unexpected error in ensureBinding: %v", err)
		}

		if err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem()); err != nil {
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
		if err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem()); err != nil {
			t.Fatalf("unexpected err on second run: %v", err)
		}
		if *o.PI != 5 || *o.PD != time.Second {
			t.Fatalf("non-zero pointer scalars should not be overwritten: %v %v", *o.PI, *o.PD)
		}
	})

	t.Run("error: unsupported literal kind is wrapped with field name", func(t *testing.T) {
		type bad struct {
			S  string   `default:"ok"`
			St defInner `default:"oops"` // unsupported kind for literal
		}
		var m Model[bad]
		o := bad{}
		m.obj = &o

		if err := m.ensureBinding(); err != nil {
			t.Fatalf("unexpected error in ensureBinding: %v", err)
		}
		
		err := m.binding.SetDefaultsStruct(reflect.ValueOf(&o).Elem())
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "cannot set default value") {
			t.Fatalf("expected field-wrapped error mentioning St, got: %v", err)
		}
	})
}
