package tests

import (
	"testing"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/types"
)

func TestDynamicBinding_Defaults(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T) (run func(t *testing.T))
	}{
		{
			name: "strings ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Strings{}
				expected := Strings{
					S:  "s",
					PS: pString("s"),
				}
				expected2 := Strings{
					S:  "s2",
					PS: pString("s2"),
				}
				b, err := model.NewDynamicBinding(&Strings{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualStrings(t, o, expected)

					// set env vars
					t.Setenv("S", "s2")
					t.Setenv("PS", "s2")
					b, err = model.NewDynamicBinding(&Strings{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualStrings(t, o, expected2)
				}
			},
		},
		{
			name: "ints ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Ints{}
				expected := Ints{
					I:    5,
					I8:   8,
					I16:  16,
					I32:  32,
					I64:  64,
					PI:   pInt(5),
					PI8:  pInt8(8),
					PI16: pInt16(16),
					PI32: pInt32(32),
					PI64: pInt64(64),
				}
				expected2 := Ints{
					I:    6,
					I8:   9,
					I16:  17,
					I32:  33,
					I64:  65,
					PI:   pInt(6),
					PI8:  pInt8(9),
					PI16: pInt16(17),
					PI32: pInt32(33),
					PI64: pInt64(65),
				}
				b, err := model.NewDynamicBinding(&Ints{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInts(t, o, expected)

					// set env vars
					t.Setenv("I", "6")
					t.Setenv("I8", "9")
					t.Setenv("I16", "17")
					t.Setenv("I32", "33")
					t.Setenv("I64", "65")
					t.Setenv("PI", "6")
					t.Setenv("PI8", "9")
					t.Setenv("PI16", "17")
					t.Setenv("PI32", "33")
					t.Setenv("PI64", "65")
					b, err = model.NewDynamicBinding(&Ints{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInts(t, o, expected2)
				}
			},
		},
		{
			name: "floats ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Floats{}
				expected := Floats{
					F32:  3.2,
					F64:  0x_1FFFp-16,
					PF32: pFloat32(3.2),
					PF64: pFloat64(0x_1FFFp-16),
				}
				expected2 := Floats{
					F32:  4.2,
					F64:  5.4,
					PF32: pFloat32(4.2),
					PF64: pFloat64(5.4),
				}
				b, err := model.NewDynamicBinding(&Floats{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualFloats(t, o, expected)

					// set env vars
					t.Setenv("F32", "4.2")
					t.Setenv("F64", "5.4")
					t.Setenv("PF32", "4.2")
					t.Setenv("PF64", "5.4")
					b, err = model.NewDynamicBinding(&Floats{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualFloats(t, o, expected2)
				}
			},
		},
		{
			name: "bools ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Bools{}
				expected := Bools{
					B:  true,
					PB: pBool(true),
				}
				expected2 := Bools{
					B:  false,
					PB: pBool(false),
				}
				b, err := model.NewDynamicBinding(&Bools{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBools(t, o, expected)

					// set env vars
					t.Setenv("B", "false")
					t.Setenv("PB", "false")
					b, err = model.NewDynamicBinding(&Bools{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBools(t, o, expected2)
				}
			},
		},
		{
			name: "uints ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Uints{}
				expected := Uints{
					U:    5,
					U8:   8,
					U16:  16,
					U32:  32,
					U64:  64,
					PU:   pUint(5),
					PU8:  pUint8(8),
					PU16: pUint16(16),
					PU32: pUint32(32),
					PU64: pUint64(64),
				}
				expected2 := Uints{
					U:    6,
					U8:   9,
					U16:  17,
					U32:  33,
					U64:  65,
					PU:   pUint(6),
					PU8:  pUint8(9),
					PU16: pUint16(17),
					PU32: pUint32(33),
					PU64: pUint64(65),
				}
				b, err := model.NewDynamicBinding(&Uints{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUints(t, o, expected)

					// set env vars
					t.Setenv("U", "6")
					t.Setenv("U8", "9")
					t.Setenv("U16", "17")
					t.Setenv("U32", "33")
					t.Setenv("U64", "65")
					t.Setenv("PU", "6")
					t.Setenv("PU8", "9")
					t.Setenv("PU16", "17")
					t.Setenv("PU32", "33")
					t.Setenv("PU64", "65")
					b, err = model.NewDynamicBinding(&Uints{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUints(t, o, expected2)
				}
			},
		},
		{
			name: "uintptrs ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := UintPtrs{}
				expected := UintPtrs{
					UintPtr:  128,
					PUintPtr: pUintptr(128),
				}
				expected2 := UintPtrs{
					UintPtr:  256,
					PUintPtr: pUintptr(256),
				}
				b, err := model.NewDynamicBinding(&UintPtrs{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUintPtrs(t, o, expected)

					// set env vars
					t.Setenv("UINTPTR", "256")
					t.Setenv("PUINTPTR", "256")
					b, err = model.NewDynamicBinding(&UintPtrs{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUintPtrs(t, o, expected2)
				}
			},
		},
		{
			name: "bytes ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Bytes{}
				expected := Bytes{
					Byte:  8,
					PByte: pByte(8),
				}
				expected2 := Bytes{
					Byte:  9,
					PByte: pByte(9),
				}
				b, err := model.NewDynamicBinding(&Bytes{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBytes(t, o, expected)

					// set env vars
					t.Setenv("BYTE", "9")
					t.Setenv("PBYTE", "9")
					b, err = model.NewDynamicBinding(&Bytes{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualBytes(t, o, expected2)
				}
			},
		},
		{
			name: "runes ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Runes{}
				expected := Runes{
					Rune:  '\U00101234',
					PRune: pRune('\U00101234'),
				}
				expected2 := Runes{
					Rune:  'Ж',
					PRune: pRune('Ж'),
				}
				b, err := model.NewDynamicBinding(&Runes{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualRunes(t, o, expected)

					// set env vars
					t.Setenv("RUNE", "Ж")
					t.Setenv("PRUNE", "Ж")
					b, err = model.NewDynamicBinding(&Runes{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualRunes(t, o, expected2)
				}
			},
		},
		{
			name: "complexes ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Complexes{}
				expected := Complexes{
					C64:   3 + 2i,
					C128:  6 + 4i,
					PC64:  pComplex64(3 + 2i),
					PC128: pComplex128(6 + 4i),
				}
				expected2 := Complexes{
					C64:   4 + 3i,
					C128:  7 + 5i,
					PC64:  pComplex64(4 + 3i),
					PC128: pComplex128(7 + 5i),
				}
				b, err := model.NewDynamicBinding(&Complexes{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualComplexes(t, o, expected)

					// set env vars
					t.Setenv("C64", "4+3i")
					t.Setenv("C128", "7+5i")
					t.Setenv("PC64", "4+3i")
					t.Setenv("PC128", "7+5i")
					b, err = model.NewDynamicBinding(&Complexes{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualComplexes(t, o, expected2)
				}
			},
		},
		{
			name: "durations ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Durations{}
				expected := Durations{
					TD:  5 * time.Second,
					D:   types.Duration(5 * time.Second),
					PTD: pTDuration(5 * time.Second),
					PD:  pDuration(types.Duration(5 * time.Second)),
				}
				expected2 := Durations{
					TD:  10 * time.Second,
					D:   types.Duration(10 * time.Second),
					PTD: pTDuration(10 * time.Second),
					PD:  pDuration(types.Duration(10 * time.Second)),
				}
				b, err := model.NewDynamicBinding(&Durations{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDurations(t, o, expected)

					// set env vars
					t.Setenv("TD", "10s")
					t.Setenv("D", "10s")
					t.Setenv("PTD", "10s")
					t.Setenv("PD", "10s")
					b, err = model.NewDynamicBinding(&Durations{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDurations(t, o, expected2)
				}
			},
		},
		{
			name: "default alloc ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultAlloc{}
				expected := DefaultAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "",
					Str: Strings{S: "s", PS: pString("s")}, // because of implicit dive for struct
				}
				expected2 := DefaultAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "s",
					Str: Strings{S: "str_s", PS: pString("str_s")}, // because of implicit dive for struct
				}
				b, err := model.NewDynamicBinding(&DefaultAlloc{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected)

					// set env vars
					t.Setenv("SS", "ss")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					t.Setenv("STR_S", "str_s")
					t.Setenv("STR_PS", "str_s")
					b, err = model.NewDynamicBinding(&DefaultAlloc{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected2)
				}
			},
		},
		{
			name: "default elem alloc ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemAlloc{}
				expected := DefaultElemAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "",
					Str: Strings{S: "s", PS: pString("s")}, // because of implicit dive for struct
				}
				expected2 := DefaultElemAlloc{
					SS:  []string{},
					M:   make(map[string]Strings),
					MP:  make(map[string]*Strings),
					A:   []Strings{},
					AP:  []*Strings{},
					S:   "s",
					Str: Strings{S: "str_s", PS: pString("str_s")}, // because of implicit dive for struct
				}
				b, err := model.NewDynamicBinding(&DefaultElemAlloc{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemAlloc(t, o, expected)

					// set env vars
					t.Setenv("SS", "ss")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					t.Setenv("STR_S", "str_s")
					t.Setenv("STR_PS", "str_s")
					b, err = model.NewDynamicBinding(&DefaultElemAlloc{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemAlloc(t, o, expected2)
				}
			},
		},
		{
			name: "dive ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Dive{}
				expected := Dive{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					},
					PStrings: &Strings{
						S:  "s",
						PS: pString("s"),
					},
					PI: nil,
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					S:  "",
				}
				expected2 := Dive{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					PI: pInt(3),
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					S:  "s",
				}
				b, err := model.NewDynamicBinding(&Dive{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDive(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewDynamicBinding(&Dive{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDive(t, o, expected2)
				}
			},
		},
		{
			name: "collections ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Collections{}
				expected := Collections{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					}, // "dive" is implicit for structs
					PStrings: nil, // "dive" is not implicit for pointers to structs
					M:        make(map[string]Strings),
					MP:       make(map[string]*Strings),
					A:        []Strings{},
					AP:       []*Strings{},
					SS:       []string{},
				}
				expected2 := Collections{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					SS: []string{},
				}
				b, err := model.NewDynamicBinding(&Collections{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollections(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewDynamicBinding(&Collections{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollections(t, o, expected2)
				}
			},
		},
		{
			name: "collections default empty ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := CollectionsDefaultEmpty{}
				expected := CollectionsDefaultEmpty{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					}, // "dive" is implicit for structs
					PStrings: nil, // "dive" is not implicit for pointers to structs
					M:        make(map[string]Strings),
					MP:       make(map[string]*Strings),
					A:        []Strings{},
					AP:       []*Strings{},
					SS:       []string{},
				}
				expected2 := CollectionsDefaultEmpty{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					SS: []string{},
				}
				b, err := model.NewDynamicBinding(&CollectionsDefaultEmpty{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultEmpty(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewDynamicBinding(&CollectionsDefaultEmpty{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultEmpty(t, o, expected2)
				}
			},
		},
		{
			name: "collections default element empty ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := CollectionsDefaultElemEmpty{}
				expected := CollectionsDefaultElemEmpty{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					}, // "dive" is implicit for structs
					PStrings: nil, // "dive" is not implicit for pointers to structs
					M:        make(map[string]Strings),
					MP:       make(map[string]*Strings),
					A:        []Strings{},
					AP:       []*Strings{},
					SS:       []string{},
				}
				expected2 := CollectionsDefaultElemEmpty{
					Strings: Strings{
						S:  "strings_s",
						PS: pString("strings_ps"),
					},
					PStrings: &Strings{
						S:  "pstrings_s",
						PS: pString("pstrings_ps"),
					},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					SS: []string{},
				}
				b, err := model.NewDynamicBinding(&CollectionsDefaultElemEmpty{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultElemEmpty(t, o, expected)

					// set env vars
					t.Setenv("STRINGS_S", "strings_s")
					t.Setenv("STRINGS_PS", "strings_ps")
					t.Setenv("PSTRINGS_S", "pstrings_s")
					t.Setenv("PSTRINGS_PS", "pstrings_ps")
					t.Setenv("PI", "3")
					t.Setenv("M", "m")
					t.Setenv("MP", "mp")
					t.Setenv("A", "a")
					t.Setenv("AP", "ap")
					t.Setenv("S", "s")
					b, err = model.NewDynamicBinding(&CollectionsDefaultElemEmpty{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualCollectionsDefaultElemEmpty(t, o, expected2)
				}
			},
		},
		{
			name: "no env tag",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := NoEnvTag{}
				expected := NoEnvTag{
					NoEnvTag: Strings{
						S:  "s",
						PS: pString("s"),
					},
				}
				expected2 := NoEnvTag{
					NoEnvTag: Strings{
						S:  "s2",
						PS: pString("s"),
					},
				}
				b, err := model.NewDynamicBinding(&NoEnvTag{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualNoEnvTag(t, o, expected)

					// set env vars
					t.Setenv("NOENVTAG_S", "s2")
					b, err = model.NewDynamicBinding(&NoEnvTag{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualNoEnvTag(t, o, expected2)
				}
			},
		},
		{
			name: "env prefix ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvPrefix{}
				expected := EnvPrefix{
					Strings: Strings{
						S:  "s",
						PS: pString("s"),
					},
				}
				expected2 := EnvPrefix{
					Strings: Strings{
						S:  "prefixed_s",
						PS: pString("prefixed_ps"),
					},
				}
				b, err := model.NewDynamicBinding(&EnvPrefix{}, model.WithEnvPrefix("__app_config__"))
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvPrefix(t, o, expected)

					t.Setenv("APP_CONFIG_STRINGS_S", "prefixed_s")
					t.Setenv("APP_CONFIG_STRINGS_PS", "prefixed_ps")
					b, err = model.NewDynamicBinding(&EnvPrefix{}, model.WithEnvPrefix("__app_config__"))
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvPrefix(t, o, expected2)
				}
			},
		},
		{
			name: "env disabled is skipped",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvDisabled{}
				expected := EnvDisabled{S: "s"}
				b, err := model.NewDynamicBinding(&EnvDisabled{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvDisabled(t, o, expected)

					t.Setenv("S", "s2")
					b, err = model.NewDynamicBinding(&EnvDisabled{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvDisabled(t, o, expected)
				}
			},
		},
		{
			name: "json comma tag env name ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := JSONCommaTag{}
				expected := JSONCommaTag{S: "s"}
				expected2 := JSONCommaTag{S: "s2"}
				b, err := model.NewDynamicBinding(&JSONCommaTag{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualJSONCommaTag(t, o, expected)

					t.Setenv("CUSTOM_S", "s2")
					b, err = model.NewDynamicBinding(&JSONCommaTag{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualJSONCommaTag(t, o, expected2)
				}
			},
		},
		{
			name: "env zero values override defaults",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvZeroValues{}
				expected := EnvZeroValues{S: "s", I: 5, B: true}
				expected2 := EnvZeroValues{S: "", I: 0, B: false}
				b, err := model.NewDynamicBinding(&EnvZeroValues{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvZeroValues(t, o, expected)

					t.Setenv("S", "")
					t.Setenv("I", "0")
					t.Setenv("B", "false")
					b, err = model.NewDynamicBinding(&EnvZeroValues{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualEnvZeroValues(t, o, expected2)
				}
			},
		},
		{
			name: "map literal env values ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapLiteralEnv{M: map[string]int{"one": 1}}
				expected := MapLiteralEnv{M: map[string]int{"one": 1}}
				expected2 := MapLiteralEnv{M: map[string]int{"one": 2}}
				b, err := model.NewDynamicBinding(&MapLiteralEnv{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapLiteralEnv(t, o, expected)

					t.Setenv("M_ONE", "2")
					b, err = model.NewDynamicBinding(&MapLiteralEnv{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapLiteralEnv(t, o, expected2)
				}
			},
		},
		{
			name: "map struct env values ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapStructEnv{M: map[string]Strings{"one": {}}}
				expected := MapStructEnv{M: map[string]Strings{"one": {S: "s", PS: pString("s")}}}
				expected2 := MapStructEnv{M: map[string]Strings{"one": {S: "s2", PS: pString("ps2")}}}
				b, err := model.NewDynamicBinding(&MapStructEnv{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapStructEnv(t, o, expected)

					t.Setenv("M_ONE_S", "s2")
					t.Setenv("M_ONE_PS", "ps2")
					b, err = model.NewDynamicBinding(&MapStructEnv{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapStructEnv(t, o, expected2)
				}
			},
		},
		{
			name: "map pointer struct env values ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapPtrStructEnv{M: map[string]*Strings{"one": {}, "nil": nil}}
				expected := MapPtrStructEnv{M: map[string]*Strings{"one": {S: "s", PS: pString("s")}, "nil": nil}}
				expected2 := MapPtrStructEnv{M: map[string]*Strings{"one": {S: "s2", PS: pString("ps2")}, "nil": nil}}
				b, err := model.NewDynamicBinding(&MapPtrStructEnv{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapPtrStructEnv(t, o, expected)

					t.Setenv("M_ONE_S", "s2")
					t.Setenv("M_ONE_PS", "ps2")
					t.Setenv("M_NIL_S", "ignored")
					b, err = model.NewDynamicBinding(&MapPtrStructEnv{})
					if err != nil {
						t.Fatal(err)
					}
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualMapPtrStructEnv(t, o, expected2)
				}
			},
		},
		{
			name: "default elem slice ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemSlice{Items: []Strings{{}}}
				expected := DefaultElemSlice{Items: []Strings{{S: "s", PS: pString("s")}}}
				b, err := model.NewDynamicBinding(&DefaultElemSlice{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemSlice(t, o, expected)
				}
			},
		},
		{
			name: "default elem pointer slice ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemPtrSlice{Items: []*Strings{{}, nil}}
				expected := DefaultElemPtrSlice{Items: []*Strings{{S: "s", PS: pString("s")}, nil}}
				b, err := model.NewDynamicBinding(&DefaultElemPtrSlice{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemPtrSlice(t, o, expected)
				}
			},
		},
		{
			name: "default elem array ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemArray{}
				expected := DefaultElemArray{Items: [1]Strings{{S: "s", PS: pString("s")}}}
				b, err := model.NewDynamicBinding(&DefaultElemArray{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemArray(t, o, expected)
				}
			},
		},
		{
			name: "default elem map ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemMap{M: map[string]Strings{"one": {}}}
				expected := DefaultElemMap{M: map[string]Strings{"one": {S: "s", PS: pString("s")}}}
				b, err := model.NewDynamicBinding(&DefaultElemMap{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemMap(t, o, expected)
				}
			},
		},
		{
			name: "default elem pointer map ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemPtrMap{M: map[string]*Strings{"one": {}, "nil": nil}}
				expected := DefaultElemPtrMap{M: map[string]*Strings{"one": {S: "s", PS: pString("s")}, "nil": nil}}
				b, err := model.NewDynamicBinding(&DefaultElemPtrMap{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemPtrMap(t, o, expected)
				}
			},
		},
		{
			name: "default elem pointer collection ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				items := []Strings{{}}
				m := map[string]Strings{"one": {}}
				o := DefaultElemPtrCollection{Items: &items, M: &m}
				expectedItems := []Strings{{S: "s", PS: pString("s")}}
				expectedMap := map[string]Strings{"one": {S: "s", PS: pString("s")}}
				expected := DefaultElemPtrCollection{Items: &expectedItems, M: &expectedMap}
				b, err := model.NewDynamicBinding(&DefaultElemPtrCollection{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemPtrCollection(t, o, expected)
				}
			},
		},
		{
			name: "default elem unsupported tag is ignored",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultElemUnsupported{Items: []Strings{{}}}
				expected := DefaultElemUnsupported{Items: []Strings{{}}}
				b, err := model.NewDynamicBinding(&DefaultElemUnsupported{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDefaultElemUnsupported(t, o, expected)
				}
			},
		},
		{
			name: "alloc no op for non nil and non collection",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := AllocNoop{SS: []string{"x"}, M: map[string]string{"k": "v"}}
				expected := AllocNoop{SS: []string{"x"}, M: map[string]string{"k": "v"}}
				b, err := model.NewDynamicBinding(&AllocNoop{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualAllocNoop(t, o, expected)
				}
			},
		},
		{
			name: "dive ignored for non struct values",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DiveIgnored{}
				expected := DiveIgnored{}
				b, err := model.NewDynamicBinding(&DiveIgnored{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualDiveIgnored(t, o, expected)
				}
			},
		},
		{
			name: "named scalar defaults ok",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := NamedScalars{}
				expected := NamedScalars{S: CustomString("s"), I: CustomInt(5), B: CustomBool(true)}
				b, err := model.NewDynamicBinding(&NamedScalars{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualNamedScalars(t, o, expected)
				}
			},
		},
		{
			name: "invalid env int returns set default error",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := EnvInvalidInt{}
				b, err := model.NewDynamicBinding(&EnvInvalidInt{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					t.Setenv("I", "not-int")
					b, err = model.NewDynamicBinding(&EnvInvalidInt{})
					if err != nil {
						t.Fatal(err)
					}
					err = applyDynamicDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
					if !errors.Is(err, errors.ErrSetDefault) {
						t.Fatalf("expected ErrSetDefault, got %v", err)
					}
				}
			},
		},
		{
			name: "invalid map env int returns set default error",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := MapLiteralEnv{M: map[string]int{"one": 1}}
				b, err := model.NewDynamicBinding(&MapLiteralEnv{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					t.Setenv("M_ONE", "not-int")
					b, err = model.NewDynamicBinding(&MapLiteralEnv{})
					if err != nil {
						t.Fatal(err)
					}
					err = applyDynamicDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
					if !errors.Is(err, errors.ErrSetDefault) {
						t.Fatalf("expected ErrSetDefault, got %v", err)
					}
				}
			},
		},
		{
			name: "unexported is skipped",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Unexported{}
				expected := Unexported{}
				b, err := model.NewDynamicBinding(&Unexported{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualUnexported(t, o, expected)
				}
			},
		},
		{
			name: "wrapped unexported is skipped",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := WrappedUnexported{}
				expected := WrappedUnexported{}
				b, err := model.NewDynamicBinding(&WrappedUnexported{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualWrappedUnexported(t, o, expected)
				}
			},
		},
		{
			name: "interface",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Interface{}
				expected := Interface{}
				b, err := model.NewDynamicBinding(&Interface{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInterface(t, o, expected)
				}
			},
		},
		{
			name: "interface with value",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := Interface{Interface: &Strings{}}
				expected := Interface{Interface: &Strings{}}
				b, err := model.NewDynamicBinding(&Interface{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					checkEqualInterface(t, o, expected)
				}
			},
		},
		{
			name: "error on unsupported literal kind",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := UnsupportedLiteralKind{}
				b, err := model.NewDynamicBinding(&UnsupportedLiteralKind{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					err = applyDynamicDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
					if !errors.Is(err, errors.ErrSetDefault) {
						t.Fatalf("expected ErrSetDefault, got %v", err)
					}
					expectedMsg := "cannot set default value, field.name: Unsupported, cause: default literal unsupported kind, default.literal.kind: struct"
					if err.Error() != expectedMsg {
						t.Fatalf("expected %s, got %s", expectedMsg, err.Error())
					}
				}
			},
		},
		{
			name: "nil object",
			prepare: func(t *testing.T) func(t *testing.T) {
				var o *Strings
				b, err := model.NewDynamicBinding(&Strings{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = b.ApplyDefaults(o); err == nil {
						t.Fatalf("expected error, got nil")
					}
					if !errors.Is(err, errors.ErrNotStructPtr) {
						t.Fatalf("expected ErrNilObject, got %v", err)
					}
				}
			},
		},
		{
			name: "alloc slice/map when nil and idempotent",
			prepare: func(t *testing.T) func(t *testing.T) {
				o := DefaultAlloc{}
				expected := DefaultAlloc{
					SS: []string{},
					M:  make(map[string]Strings),
					MP: make(map[string]*Strings),
					A:  []Strings{},
					AP: []*Strings{},
					S:  "",
					Str: Strings{
						S:  "s",
						PS: pString("s"),
					},
				}
				b, err := model.NewDynamicBinding(&DefaultAlloc{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected err: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected)
					// run again (idempotent)
					if err = applyDynamicDefaultsAndEnv(b, &o); err != nil {
						t.Fatalf("unexpected err on second run: %v", err)
					}
					checkEqualDefaultAlloc(t, o, expected)
				}
			},
		},
		{
			name: "error unsupported literal kind wrapped with field name St",
			prepare: func(t *testing.T) func(t *testing.T) {
				type wrappedUnsupportedLiteralKind struct {
					W UnsupportedLiteralKind `yaml:"w" env:"W" default:"dive"`
				}
				o := wrappedUnsupportedLiteralKind{}
				b, err := model.NewDynamicBinding(&wrappedUnsupportedLiteralKind{})
				if err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					err = applyDynamicDefaultsAndEnv(b, &o)
					if err == nil {
						t.Fatal("expected error")
					}
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
}
