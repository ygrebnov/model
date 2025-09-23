package model

import (
	"reflect"
	"testing"
)

// Helper types for defaults testing
type innerDef struct {
	S string `default:"x"`
	N int    `default:"42"`
}

type outerDef struct {
	Inner  innerDef
	PInner *innerDef
	N      int
	S      []string
	M      map[string]int
	PInt   *int
}

func TestApplyDefaultTag(t *testing.T) {
	m := &Model[struct{}]{}

	tests := []struct {
		name   string
		prep   func() (obj *outerDef, fv reflect.Value)
		act    func(*Model[struct{}], reflect.Value) error
		verify func(t *testing.T, obj *outerDef)
	}{
		{
			name: "dive on struct field applies inner defaults",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{}
				rv := reflect.ValueOf(obj).Elem()
				fv := rv.FieldByName("Inner")
				return obj, fv
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "dive", "Inner") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.Inner.S != "x" || obj.Inner.N != 42 {
					t.Fatalf("expected inner defaults applied, got %+v", obj.Inner)
				}
			},
		},
		{
			name: "dive on nil *struct allocates and applies defaults",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{PInner: nil}
				rv := reflect.ValueOf(obj).Elem()
				fv := rv.FieldByName("PInner")
				return obj, fv
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "dive", "PInner") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.PInner == nil || obj.PInner.S != "x" || obj.PInner.N != 42 {
					t.Fatalf("expected allocated PInner with defaults, got %+v", obj.PInner)
				}
			},
		},
		{
			name: "dive on non-struct is no-op",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("N")
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "dive", "N") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.N != 0 {
					t.Fatalf("expected N unchanged, got %d", obj.N)
				}
			},
		},
		{
			name: "alloc on nil slice allocates empty slice",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{S: nil}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("S")
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "alloc", "S") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.S == nil || len(obj.S) != 0 {
					t.Fatalf("expected allocated empty slice, got %#v", obj.S)
				}
			},
		},
		{
			name: "alloc on nil map allocates empty map",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{M: nil}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("M")
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "alloc", "M") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.M == nil || len(obj.M) != 0 {
					t.Fatalf("expected allocated empty map, got %#v", obj.M)
				}
			},
		},
		{
			name: "literal default on nil *int allocates and sets",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{PInt: nil}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("PInt")
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "7", "PInt") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.PInt == nil || *obj.PInt != 7 {
					t.Fatalf("expected allocated *int==7, got %#v", obj.PInt)
				}
			},
		},
		{
			name: "literal default on non-zero int leaves it unchanged",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{N: 5}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("N")
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "9", "N") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.N != 5 {
					t.Fatalf("expected N to remain 5, got %d", obj.N)
				}
			},
		},
		{
			name: "literal default on zero int sets value",
			prep: func() (*outerDef, reflect.Value) {
				obj := &outerDef{N: 0}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("N")
			},
			act: func(m *Model[struct{}], fv reflect.Value) error { return m.applyDefaultTag(fv, "9", "N") },
			verify: func(t *testing.T, obj *outerDef) {
				if obj.N != 9 {
					t.Fatalf("expected N to be 9, got %d", obj.N)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, fv := tc.prep()
			if err := tc.act(m, fv); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.verify(t, obj)
		})
	}
}

// Types and holder for applyDefaultElemTag tests
type innerElem struct {
	S string `default:"x"`
	N int    `default:"42"`
}

type elemHolder struct {
	People []innerElem
	Ptrs   []*innerElem
	Arr    [2]innerElem
	Mv     map[string]innerElem
	Mp     map[string]*innerElem
	Not    string
}

func TestApplyDefaultElemTag(t *testing.T) {
	m := &Model[struct{}]{}

	tests := []struct {
		name   string
		prep   func() (obj *elemHolder, fv reflect.Value)
		verify func(t *testing.T, obj *elemHolder)
	}{
		{
			name: "slice of struct elements dive",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{People: []innerElem{{}, {S: "ok"}}}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("People")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.People[0].S != "x" || obj.People[0].N != 42 {
					t.Fatalf("expected defaults on People[0], got %+v", obj.People[0])
				}
				if obj.People[1].S != "ok" || obj.People[1].N != 42 {
					t.Fatalf("expected partial defaults on People[1], got %+v", obj.People[1])
				}
			},
		},
		{
			name: "slice of *struct elements dive (nil stays nil)",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Ptrs: []*innerElem{nil, {}}}
				rv := reflect.ValueOf(obj).Elem()
				// fix composite literal: replace {} with &innerElem{}
				obj.Ptrs[1] = &innerElem{}
				return obj, rv.FieldByName("Ptrs")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Ptrs[0] != nil {
					t.Fatalf("expected Ptrs[0] to remain nil")
				}
				if obj.Ptrs[1] == nil || obj.Ptrs[1].S != "x" || obj.Ptrs[1].N != 42 {
					t.Fatalf("expected defaults on Ptrs[1], got %#v", obj.Ptrs[1])
				}
			},
		},
		{
			name: "array of struct elements dive",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Arr: [2]innerElem{{}, {S: "ok"}}}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("Arr")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Arr[0].S != "x" || obj.Arr[0].N != 42 {
					t.Fatalf("expected defaults on Arr[0], got %+v", obj.Arr[0])
				}
				if obj.Arr[1].S != "ok" || obj.Arr[1].N != 42 {
					t.Fatalf("expected partial defaults on Arr[1], got %+v", obj.Arr[1])
				}
			},
		},
		{
			name: "map[string]struct values dive (copy-write-back)",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Mv: map[string]innerElem{"a": {}, "b": {S: "ok"}}}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("Mv")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Mv["a"].S != "x" || obj.Mv["a"].N != 42 {
					t.Fatalf("expected defaults on Mv[a], got %+v", obj.Mv["a"])
				}
				if obj.Mv["b"].S != "ok" || obj.Mv["b"].N != 42 {
					t.Fatalf("expected partial defaults on Mv[b], got %+v", obj.Mv["b"])
				}
			},
		},
		{
			name: "map[string]*struct values dive (in-place)",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Mp: map[string]*innerElem{"p1": {}, "p2": nil}}
				rv := reflect.ValueOf(obj).Elem()
				obj.Mp["p1"] = &innerElem{}
				return obj, rv.FieldByName("Mp")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Mp["p1"] == nil || obj.Mp["p1"].S != "x" || obj.Mp["p1"].N != 42 {
					t.Fatalf("expected defaults on Mp[p1], got %#v", obj.Mp["p1"])
				}
				if obj.Mp["p2"] != nil {
					t.Fatalf("expected Mp[p2] to remain nil")
				}
			},
		},
		{
			name: "non-collection field ignored",
			prep: func() (*elemHolder, reflect.Value) {
				obj := &elemHolder{Not: ""}
				rv := reflect.ValueOf(obj).Elem()
				return obj, rv.FieldByName("Not")
			},
			verify: func(t *testing.T, obj *elemHolder) {
				if obj.Not != "" {
					t.Fatalf("expected Not unchanged, got %q", obj.Not)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, fv := tc.prep()
			if err := m.applyDefaultElemTag(fv, "dive"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.verify(t, obj)
		})
	}
}
