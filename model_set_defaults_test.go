package model

import (
	"testing"
	"time"
)

// Minimal types for tests
type sdInner struct {
	Msg string        `default:"hi"`
	D   time.Duration `default:"2s"`
}

type sdOK struct {
	S   string  `default:"x"`
	Num int     `default:"7"`
	In  sdInner `default:"dive"`
}

type sdBad struct {
	// unsupported literal on struct field should cause an error
	In sdInner `default:"oops"`
}

func TestModel_SetDefaults_OnceGuard_RunAtMostOnce(t *testing.T) {
	// Build a model without constructor-applied defaults.
	m, err := New(&sdOK{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// 1st call: should set defaults
	if err := m.SetDefaults(); err != nil {
		t.Fatalf("SetDefaults first: %v", err)
	}
	if m.obj.S != "x" || m.obj.Num != 7 || m.obj.In.Msg != "hi" || m.obj.In.D != 2*time.Second {
		t.Fatalf("defaults not applied on first run: %+v", *m.obj)
	}

	// Manually reset fields to zero-values
	m.obj.S = ""
	m.obj.Num = 0
	m.obj.In.Msg = ""
	m.obj.In.D = 0

	// 2nd call: should NOT run applyDefaults again due to sync.Once
	if err := m.SetDefaults(); err != nil {
		t.Fatalf("SetDefaults second (should not re-run): %v", err)
	}
	// If defaults were re-applied, these would be set again; they must remain zero.
	if m.obj.S != "" || m.obj.Num != 0 || m.obj.In.Msg != "" || m.obj.In.D != 0 {
		t.Fatalf("defaults were re-applied; Once guard failed: %+v", *m.obj)
	}
}

func TestModel_SetDefaults_OnceGuard_WithDefaultsInNew(t *testing.T) {
	// Constructor applies defaults once because of WithDefaults()
	m, err := New(&sdOK{}, WithDefaults[sdOK]())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Ensure constructor applied defaults
	if m.obj.S != "x" || m.obj.Num != 7 || m.obj.In.Msg != "hi" || m.obj.In.D != 2*time.Second {
		t.Fatalf("constructor defaults not applied: %+v", *m.obj)
	}

	// Reset to zero
	m.obj.S, m.obj.Num = "", 0
	m.obj.In.Msg, m.obj.In.D = "", 0

	// Subsequent SetDefaults() should be no-op due to Once
	if err := m.SetDefaults(); err != nil {
		t.Fatalf("SetDefaults after constructor-apply: %v", err)
	}
	if m.obj.S != "" || m.obj.Num != 0 || m.obj.In.Msg != "" || m.obj.In.D != 0 {
		t.Fatalf("defaults re-applied; Once guard failed after constructor apply: %+v", *m.obj)
	}
}

func TestModel_SetDefaults_OnceGuard_ErrorOnFirstRun_NoSecondRun(t *testing.T) {
	// First run should error due to unsupported literal on struct field
	mbad, err := New(&sdBad{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err1 := mbad.SetDefaults()
	if err1 == nil {
		t.Fatalf("expected error on first SetDefaults")
	}

	// Swap to a new object of the SAME type — Once should still prevent re-running defaults
	mbad.obj = &sdBad{} // same Model instance, same type
	err2 := mbad.SetDefaults()
	if err2 != nil {
		// Note: due to Once, err2 should be nil (the once-closure isn't executed again)
		t.Fatalf("expected nil on second SetDefaults because Once prevents re-run; got: %v", err2)
	}

	// And since defaults were not re-run, the object should still be zero-valued (no defaults applied)
	if mbad.obj.In.Msg != "" || mbad.obj.In.D != 0 {
		t.Fatalf("defaults applied despite Once guard; obj.In: %+v", mbad.obj.In)
	}
}

func TestModel_SetDefaults_Guards_NilAndNonStruct(t *testing.T) {
	// nil object
	{
		var m Model[sdOK]
		m.obj = nil
		if err := m.SetDefaults(); err == nil {
			t.Fatalf("expected error for nil object")
		}
	}

	// non-struct object type (e.g., *int) — construct a Model[int]
	{
		var mInt Model[int]
		x := 42
		mInt.obj = &x
		if err := mInt.SetDefaults(); err == nil {
			t.Fatalf("expected error for non-struct object")
		}
	}
}
