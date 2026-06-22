package tests

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/pkg/types"
)

type Strings struct {
	S  string  `yaml:"s" env:"S" default:"s" validate:"min(1)"`
	PS *string `yaml:"ps,omitempty" env:"PS" default:"s" validate:"min(1)"`
}

func checkEqualStrings(t *testing.T, a, e Strings) {
	t.Helper()
	checkEqualValue(t, "S", a.S, e.S)
	checkEqualPtr(t, "PS", a.PS, e.PS)
}

func checkEqualPStrings(t *testing.T, a, e *Strings) {
	t.Helper()
	if a == nil && e == nil {
		return
	}
	if a == nil {
		t.Fatalf("PS, got nil, want: %v", *e.PS)
	}
	if e == nil {
		t.Fatalf("PS, got: %v, want nil", *a.PS)
	}
	checkEqualStrings(t, *a, *e)
}

type Ints struct {
	I    int    `yaml:"i" env:"I" default:"5" validate:"min(3),max(10)"`
	I8   int8   `yaml:"i8" env:"I8" default:"8" validate:"min(3)"`
	I16  int16  `yaml:"i16" env:"I16" default:"1_6" validate:"min(3)"`
	I32  int32  `yaml:"i32" env:"I32" default:"32" validate:"min(3)"`
	I64  int64  `yaml:"i64" env:"I64" default:"64" validate:"min(3)"`
	PI   *int   `yaml:"pi,omitempty" env:"PI" default:"5" validate:"min(3)"`
	PI8  *int8  `yaml:"pi8,omitempty" env:"PI8" default:"8" validate:"min(3)"`
	PI16 *int16 `yaml:"pi16,omitempty" env:"PI16" default:"1_6" validate:"min(3)"`
	PI32 *int32 `yaml:"pi32,omitempty" env:"PI32" default:"32" validate:"min(3)"`
	PI64 *int64 `yaml:"pi64,omitempty" env:"PI64" default:"64" validate:"min(3)"`
}

func checkEqualInts(t *testing.T, a, e Ints) {
	t.Helper()
	checkEqualValue(t, "I", a.I, e.I)
	checkEqualValue(t, "I8", a.I8, e.I8)
	checkEqualValue(t, "I16", a.I16, e.I16)
	checkEqualValue(t, "I32", a.I32, e.I32)
	checkEqualValue(t, "I64", a.I64, e.I64)

	checkEqualPtr(t, "PI", a.PI, e.PI)
	checkEqualPtr(t, "PI8", a.PI8, e.PI8)
	checkEqualPtr(t, "PI16", a.PI16, e.PI16)
	checkEqualPtr(t, "PI32", a.PI32, e.PI32)
	checkEqualPtr(t, "PI64", a.PI64, e.PI64)
}

type Floats struct {
	F32  float32  `yaml:"f32" env:"F32" default:"3.2" validate:"min(3.2)"`
	F64  float64  `yaml:"f64" env:"F64" default:"0X_1FFFP-16" validate:"min(6.4)"`
	PF32 *float32 `yaml:"pf32" env:"PF32" default:"3.2" validate:"min(3.2)"`
	PF64 *float64 `yaml:"pf64" env:"PF64" default:"0X_1FFFP-16" validate:"min(6.4)"`
}

func checkEqualFloats(t *testing.T, a, e Floats) {
	t.Helper()
	checkEqualValue(t, "F32", a.F32, e.F32)
	checkEqualValue(t, "F64", a.F64, e.F64)
	checkEqualPtr(t, "PF32", a.PF32, e.PF32)
	checkEqualPtr(t, "PF64", a.PF64, e.PF64)
}

type Bools struct {
	B  bool  `yaml:"b" env:"B" default:"true"`
	PB *bool `yaml:"pb" env:"PB" default:"true"`
}

func checkEqualBools(t *testing.T, a, e Bools) {
	t.Helper()
	checkEqualValue(t, "B", a.B, e.B)
	checkEqualPtr(t, "PB", a.PB, e.PB)
}

type Uints struct {
	U    uint    `yaml:"u" env:"U" default:"5" validate:"min(3),max(10)"`
	U8   uint8   `yaml:"u8" env:"U8" default:"8" validate:"min(3)"`
	U16  uint16  `yaml:"u16" env:"U16" default:"1_6" validate:"min(3)"`
	U32  uint32  `yaml:"u32" env:"U32" default:"32" validate:"min(3)"`
	U64  uint64  `yaml:"u64" env:"U64" default:"64" validate:"min(3)"`
	PU   *uint   `yaml:"pu" env:"PU" default:"5" validate:"min(3)"`
	PU8  *uint8  `yaml:"pu8" env:"PU8" default:"8" validate:"min(3)"`
	PU16 *uint16 `yaml:"pu16" env:"PU16" default:"1_6" validate:"min(3)"`
	PU32 *uint32 `yaml:"pu32" env:"PU32" default:"32" validate:"min(3)"`
	PU64 *uint64 `yaml:"pu64" env:"PU64" default:"64" validate:"min(3)"`
}

func checkEqualUints(t *testing.T, a, e Uints) {
	t.Helper()
	checkEqualValue(t, "U", a.U, e.U)
	checkEqualValue(t, "U8", a.U8, e.U8)
	checkEqualValue(t, "U16", a.U16, e.U16)
	checkEqualValue(t, "U32", a.U32, e.U32)
	checkEqualValue(t, "U64", a.U64, e.U64)

	checkEqualPtr(t, "PU", a.PU, e.PU)
	checkEqualPtr(t, "PU8", a.PU8, e.PU8)
	checkEqualPtr(t, "PU16", a.PU16, e.PU16)
	checkEqualPtr(t, "PU32", a.PU32, e.PU32)
	checkEqualPtr(t, "PU64", a.PU64, e.PU64)
}

type UintPtrs struct {
	UintPtr  uintptr  `yaml:"uintptr" env:"UINTPTR" default:"128" validate:"min(3)"`
	PUintPtr *uintptr `yaml:"puintptr" env:"PUINTPTR" default:"128" validate:"min(3)"`
}

func checkEqualUintPtrs(t *testing.T, a, e UintPtrs) {
	t.Helper()
	checkEqualValue(t, "UintPtr", a.UintPtr, e.UintPtr)
	checkEqualPtr(t, "PUintPtr", a.PUintPtr, e.PUintPtr)
}

type Bytes struct {
	Byte  byte  `yaml:"byte" env:"BYTE" default:"8" validate:"min(3)"`
	PByte *byte `yaml:"pbyte" env:"PBYTE" default:"8" validate:"min(3)"`
}

func checkEqualBytes(t *testing.T, a, e Bytes) {
	t.Helper()
	checkEqualValue(t, "Byte", a.Byte, e.Byte)
	checkEqualPtr(t, "PByte", a.PByte, e.PByte)
}

type Runes struct {
	Rune  rune  `yaml:"rune" env:"RUNE" default:"'\U00101234'" validate:"min(0)"`
	PRune *rune `yaml:"prune" env:"PRUNE" default:"'\U00101234'" validate:"min(0)"`
}

func checkEqualRunes(t *testing.T, a, e Runes) {
	t.Helper()
	checkEqualValue(t, "Rune", a.Rune, e.Rune)
	checkEqualPtr(t, "PRune", a.PRune, e.PRune)
}

type Complexes struct {
	C64   complex64   `yaml:"c64" env:"C64" default:"3+2i"`
	C128  complex128  `yaml:"c128" env:"C128" default:"6+4i"`
	PC64  *complex64  `yaml:"pc64" env:"PC64" default:"3+2i"`
	PC128 *complex128 `yaml:"pc128" env:"PC128" default:"6+4i"`
}

func checkEqualComplexes(t *testing.T, a, e Complexes) {
	t.Helper()
	checkEqualValue(t, "C64", a.C64, e.C64)
	checkEqualValue(t, "C128", a.C128, e.C128)
	checkEqualPtr(t, "PC64", a.PC64, e.PC64)
	checkEqualPtr(t, "PC128", a.PC128, e.PC128)
}

type Interface struct {
	Interface interface{} `yaml:"interface" env:"INTERFACE" default:"dive"`
}

func checkEqualInterface(t *testing.T, a, e Interface) {
	t.Helper()
	if !reflect.DeepEqual(a.Interface, e.Interface) {
		t.Fatal("expected interface to be equal")
	}
}

type UnsupportedLiteralKind struct {
	Unsupported Strings `yaml:"unsupported" env:"UNSUPPORTED" default:"unsupported"`
}

type Unexported struct {
	unexp string `yaml:"unexp" default:"unexported"` // unexported -> skipped
}

func checkEqualUnexported(t *testing.T, a, e Unexported) {
	t.Helper()
	if a.unexp != e.unexp {
		t.Fatalf("unexp, got: %v, want %v", a.unexp, e.unexp)
	}
}

type WrappedUnexported struct {
	unexp Strings `yaml:"unexp" default:"unexported"`
}

func checkEqualWrappedUnexported(t *testing.T, a, e WrappedUnexported) {
	t.Helper()
	checkEqualStrings(t, a.unexp, e.unexp)
}

type Durations struct {
	TD  time.Duration   `yaml:"td" env:"TD" default:"5s"`
	PTD *time.Duration  `yaml:"ptd" env:"PTD" default:"5s"`
	D   types.Duration  `yaml:"d" env:"D" default:"5s"`
	PD  *types.Duration `yaml:"pd" env:"PD" default:"5s"`
}

// TODO: mention in docs that duration string should be formatted according to time.ParseDuration requirements.

func checkEqualDurations(t *testing.T, a, e Durations) {
	t.Helper()
	checkEqualValue(t, "TD", a.TD, e.TD)
	checkEqualPtr(t, "PTD", a.PTD, e.PTD)

	checkEqualValue(t, "D", a.D, e.D)
	checkEqualPtr(t, "PD", a.PD, e.PD)
}

type DefaultAlloc struct {
	SS  []string            `yaml:"SS" env:"SS" default:"alloc"`
	M   map[string]Strings  `yaml:"M" env:"M" default:"alloc"`
	MP  map[string]*Strings `yaml:"MP" env:"MP" default:"alloc"`
	A   []Strings           `yaml:"A" env:"A" default:"alloc"`
	AP  []*Strings          `yaml:"AP" env:"AP" default:"alloc"`
	S   string              `yaml:"S" env:"S" default:"alloc"` // non-collection -> ignore
	Str Strings             `yaml:"Str" env:"STR" default:"alloc"`
}

func checkEqualDefaultAlloc(t *testing.T, a, e DefaultAlloc) {
	t.Helper()
	checkEqualStringSlices(t, "SS", a.SS, e.SS)
	checkEqualStringMaps(t, "M", a.M, e.M)
	checkEqualPStringMaps(t, "MP", a.MP, e.MP)
	checkEqualStringStructSlices(t, "A", a.A, e.A)
	checkEqualPStringStructSlices(t, "AP", a.AP, e.AP)
	checkEqualValue(t, "S", a.S, e.S)
	checkEqualStrings(t, a.Str, e.Str)
}

func checkEqualStringSlices(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length, got: %v, want %v", name, len(got), len(want))
	}
	sort.Strings(got)
	sort.Strings(want)
	for i := range got {
		checkEqualValue(t, name, got[i], want[i])
	}
}

func checkEqualStringMaps(t *testing.T, name string, got, want map[string]Strings) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length, got: %v, want %v", name, len(got), len(want))
	}
	for k, v := range got {
		ev, ok := want[k]
		if !ok {
			t.Fatalf("%s key is missing: %s", name, k)
		}
		checkEqualStrings(t, v, ev)
	}
}

func checkEqualPStringMaps(t *testing.T, name string, got, want map[string]*Strings) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length, got: %v, want %v", name, len(got), len(want))
	}
	for k, v := range got {
		ev, ok := want[k]
		if !ok {
			t.Fatalf("%s key is missing: %s", name, k)
		}
		checkEqualPStrings(t, v, ev)
	}
}

func checkEqualStringStructSlices(t *testing.T, name string, got, want []Strings) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length, got: %v, want %v", name, len(got), len(want))
	}
	sort.Slice(got, func(i, j int) bool { return got[i].S < got[j].S })
	sort.Slice(want, func(i, j int) bool { return want[i].S < want[j].S })
	for i := range got {
		checkEqualStrings(t, got[i], want[i])
	}
}

func lessPString(a, b *Strings) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	return a.S < b.S
}

func checkEqualPStringStructSlices(t *testing.T, name string, got, want []*Strings) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length, got: %v, want %v", name, len(got), len(want))
	}
	sort.Slice(got, func(i, j int) bool { return lessPString(got[i], got[j]) })
	sort.Slice(want, func(i, j int) bool { return lessPString(want[i], want[j]) })
	for i := range got {
		checkEqualPStrings(t, got[i], want[i])
	}
}

type DefaultElemAlloc struct {
	SS  []string            `yaml:"SS" env:"SS" defaultElem:"alloc"`
	M   map[string]Strings  `yaml:"M" env:"M" defaultElem:"alloc"`
	MP  map[string]*Strings `yaml:"MP" env:"MP" defaultElem:"alloc"`
	A   []Strings           `yaml:"A" env:"A" defaultElem:"alloc"`
	AP  []*Strings          `yaml:"AP" env:"AP" defaultElem:"alloc"`
	S   string              `yaml:"S" env:"S" defaultElem:"alloc"` // non-collection -> ignore
	Str Strings             `yaml:"Str" env:"STR" defaultElem:"alloc"`
}

func checkEqualDefaultElemAlloc(t *testing.T, a, e DefaultElemAlloc) {
	t.Helper()
	checkEqualStringSlices(t, "SS", a.SS, e.SS)
	checkEqualStringMaps(t, "M", a.M, e.M)
	checkEqualPStringMaps(t, "MP", a.MP, e.MP)
	checkEqualStringStructSlices(t, "A", a.A, e.A)
	checkEqualPStringStructSlices(t, "AP", a.AP, e.AP)
	checkEqualValue(t, "S", a.S, e.S)
	checkEqualStrings(t, a.Str, e.Str)
}

type Dive struct {
	Strings  Strings             `yaml:"strings" env:"STRINGS" default:"dive"`
	PStrings *Strings            `yaml:"pstrings" env:"PSTRINGS" default:"dive"`
	PI       *int                `yaml:"pi" env:"PI" default:"dive"` // non-struct pointer -> ignore
	M        map[string]Strings  `yaml:"M" env:"M" defaultElem:"dive"`
	MP       map[string]*Strings `yaml:"MP" env:"MP" defaultElem:"dive"`
	A        []Strings           `yaml:"A" env:"A" defaultElem:"dive"`
	AP       []*Strings          `yaml:"AP" env:"AP" defaultElem:"dive"`
	S        string              `yaml:"S" env:"S" defaultElem:"dive"` // non-collection -> ignore
}

func checkEqualDive(t *testing.T, a, e Dive) {
	t.Helper()
	checkEqualStrings(t, a.Strings, e.Strings)
	checkEqualPStrings(t, a.PStrings, e.PStrings)
	checkEqualPtr(t, "PI", a.PI, e.PI)
	checkEqualStringMaps(t, "M", a.M, e.M)
	checkEqualPStringMaps(t, "MP", a.MP, e.MP)
	checkEqualStringStructSlices(t, "A", a.A, e.A)
	checkEqualPStringStructSlices(t, "AP", a.AP, e.AP)
	checkEqualValue(t, "S", a.S, e.S)
}

type Collections struct {
	Strings  Strings             `yaml:"strings" env:"STRINGS"`
	PStrings *Strings            `yaml:"pstrings" env:"PSTRINGS"`
	M        map[string]Strings  `yaml:"M" env:"M"`
	MP       map[string]*Strings `yaml:"MP" env:"MP"`
	A        []Strings           `yaml:"A" env:"A"`
	AP       []*Strings          `yaml:"AP" env:"AP"`
	SS       []string            `yaml:"SS" env:"SS"`
}

func checkEqualCollections(t *testing.T, a, e Collections) {
	t.Helper()
	checkEqualStrings(t, a.Strings, e.Strings)
	checkEqualPStrings(t, a.PStrings, e.PStrings)
	checkEqualStringMaps(t, "M", a.M, e.M)
	checkEqualPStringMaps(t, "MP", a.MP, e.MP)
	checkEqualStringStructSlices(t, "A", a.A, e.A)
	checkEqualPStringStructSlices(t, "AP", a.AP, e.AP)
	checkEqualStringSlices(t, "SS", a.SS, e.SS)
}

type CollectionsDefaultEmpty struct {
	Strings  Strings             `yaml:"strings" env:"STRINGS" default:""`
	PStrings *Strings            `yaml:"pstrings" env:"PSTRINGS" default:""`
	M        map[string]Strings  `yaml:"M" env:"M" default:""`
	MP       map[string]*Strings `yaml:"MP" env:"MP" default:""`
	A        []Strings           `yaml:"A" env:"A" default:""`
	AP       []*Strings          `yaml:"AP" env:"AP" default:""`
	SS       []string            `yaml:"SS" env:"SS" default:""`
}

func checkEqualCollectionsDefaultEmpty(t *testing.T, a, e CollectionsDefaultEmpty) {
	t.Helper()
	checkEqualStrings(t, a.Strings, e.Strings)
	checkEqualPStrings(t, a.PStrings, e.PStrings)
	checkEqualStringMaps(t, "M", a.M, e.M)
	checkEqualPStringMaps(t, "MP", a.MP, e.MP)
	checkEqualStringStructSlices(t, "A", a.A, e.A)
	checkEqualPStringStructSlices(t, "AP", a.AP, e.AP)
	checkEqualStringSlices(t, "SS", a.SS, e.SS)
}

type CollectionsDefaultElemEmpty struct {
	Strings  Strings             `yaml:"strings" env:"STRINGS" defaultElem:""`
	PStrings *Strings            `yaml:"pstrings" env:"PSTRINGS" defaultElem:""`
	M        map[string]Strings  `yaml:"M" env:"M" defaultElem:""`
	MP       map[string]*Strings `yaml:"MP" env:"MP" defaultElem:""`
	A        []Strings           `yaml:"A" env:"A" defaultElem:""`
	AP       []*Strings          `yaml:"AP" env:"AP" defaultElem:""`
	SS       []string            `yaml:"SS" env:"SS" defaultElem:""`
}

func checkEqualCollectionsDefaultElemEmpty(t *testing.T, a, e CollectionsDefaultElemEmpty) {
	t.Helper()
	checkEqualStrings(t, a.Strings, e.Strings)
	checkEqualPStrings(t, a.PStrings, e.PStrings)
	checkEqualStringMaps(t, "M", a.M, e.M)
	checkEqualPStringMaps(t, "MP", a.MP, e.MP)
	checkEqualStringStructSlices(t, "A", a.A, e.A)
	checkEqualPStringStructSlices(t, "AP", a.AP, e.AP)
	checkEqualStringSlices(t, "SS", a.SS, e.SS)
}

type NoEnvTag struct {
	NoEnvTag Strings `yaml:"noenvtag"`
}

func checkEqualNoEnvTag(t *testing.T, a, e NoEnvTag) {
	t.Helper()
	checkEqualStrings(t, a.NoEnvTag, e.NoEnvTag)
}

type EnvPrefix struct {
	Strings Strings `env:"STRINGS"`
}

func checkEqualEnvPrefix(t *testing.T, got, expected EnvPrefix) {
	t.Helper()
	checkEqualStrings(t, got.Strings, expected.Strings)
}

type EnvDisabled struct {
	S string `default:"s" env:"-"`
}

func checkEqualEnvDisabled(t *testing.T, got, expected EnvDisabled) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

type JSONCommaTag struct {
	S string `json:"custom_s,omitempty" default:"s"`
}

func checkEqualJSONCommaTag(t *testing.T, got, expected JSONCommaTag) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

type EnvZeroValues struct {
	S string `default:"s"`
	I int    `default:"5"`
	B bool   `default:"true"`
}

func checkEqualEnvZeroValues(t *testing.T, got, expected EnvZeroValues) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

type MapLiteralEnv struct {
	M map[string]int `env:"M"`
}

func checkEqualMapLiteralEnv(t *testing.T, got, expected MapLiteralEnv) {
	t.Helper()
	if len(got.M) != len(expected.M) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for key, expectedValue := range expected.M {
		if got.M[key] != expectedValue {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	}
}

type MapStructEnv struct {
	M map[string]Strings `env:"M"`
}

func checkEqualMapStructEnv(t *testing.T, got, expected MapStructEnv) {
	t.Helper()
	if len(got.M) != len(expected.M) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for key, expectedValue := range expected.M {
		gotValue, ok := got.M[key]
		if !ok {
			t.Fatalf("expected key %s in %+v", key, got)
		}
		checkEqualStrings(t, gotValue, expectedValue)
	}
}

type MapPtrStructEnv struct {
	M map[string]*Strings `env:"M"`
}

func checkEqualMapPtrStructEnv(t *testing.T, got, expected MapPtrStructEnv) {
	t.Helper()
	if len(got.M) != len(expected.M) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for key, expectedValue := range expected.M {
		gotValue, ok := got.M[key]
		if !ok {
			t.Fatalf("expected key %s in %+v", key, got)
		}
		if expectedValue == nil {
			if gotValue != nil {
				t.Fatalf("expected nil value for key %s, got %+v", key, gotValue)
			}
			continue
		}
		if gotValue == nil {
			t.Fatalf("expected non-nil value for key %s", key)
		}
		checkEqualStrings(t, *gotValue, *expectedValue)
	}
}

type DefaultElemSlice struct {
	Items []Strings `defaultElem:"dive"`
}

func checkEqualDefaultElemSlice(t *testing.T, got, expected DefaultElemSlice) {
	t.Helper()
	if len(got.Items) != len(expected.Items) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for i := range expected.Items {
		checkEqualStrings(t, got.Items[i], expected.Items[i])
	}
}

type DefaultElemPtrSlice struct {
	Items []*Strings `defaultElem:"dive"`
}

func checkEqualDefaultElemPtrSlice(t *testing.T, got, expected DefaultElemPtrSlice) {
	t.Helper()
	if len(got.Items) != len(expected.Items) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for i := range expected.Items {
		if expected.Items[i] == nil {
			if got.Items[i] != nil {
				t.Fatalf("expected nil item at index %d, got %+v", i, got.Items[i])
			}
			continue
		}
		if got.Items[i] == nil {
			t.Fatalf("expected non-nil item at index %d", i)
		}
		checkEqualStrings(t, *got.Items[i], *expected.Items[i])
	}
}

type DefaultElemArray struct {
	Items [1]Strings `defaultElem:"dive"`
}

func checkEqualDefaultElemArray(t *testing.T, got, expected DefaultElemArray) {
	t.Helper()
	for i := range expected.Items {
		checkEqualStrings(t, got.Items[i], expected.Items[i])
	}
}

type DefaultElemMap struct {
	M map[string]Strings `defaultElem:"dive"`
}

func checkEqualDefaultElemMap(t *testing.T, got, expected DefaultElemMap) {
	t.Helper()
	if len(got.M) != len(expected.M) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for key, expectedValue := range expected.M {
		gotValue, ok := got.M[key]
		if !ok {
			t.Fatalf("expected key %s in %+v", key, got)
		}
		checkEqualStrings(t, gotValue, expectedValue)
	}
}

type DefaultElemPtrMap struct {
	M map[string]*Strings `defaultElem:"dive"`
}

func checkEqualDefaultElemPtrMap(t *testing.T, got, expected DefaultElemPtrMap) {
	t.Helper()
	if len(got.M) != len(expected.M) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for key, expectedValue := range expected.M {
		gotValue, ok := got.M[key]
		if !ok {
			t.Fatalf("expected key %s in %+v", key, got)
		}
		if expectedValue == nil {
			if gotValue != nil {
				t.Fatalf("expected nil value for key %s, got %+v", key, gotValue)
			}
			continue
		}
		if gotValue == nil {
			t.Fatalf("expected non-nil value for key %s", key)
		}
		checkEqualStrings(t, *gotValue, *expectedValue)
	}
}

type DefaultElemPtrCollection struct {
	Items *[]Strings          `defaultElem:"dive"`
	M     *map[string]Strings `defaultElem:"dive"`
}

func checkEqualDefaultElemPtrCollection(t *testing.T, got, expected DefaultElemPtrCollection) {
	t.Helper()
	if got.Items == nil || expected.Items == nil {
		if got.Items != expected.Items {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	} else {
		checkEqualDefaultElemSlice(t, DefaultElemSlice{Items: *got.Items}, DefaultElemSlice{Items: *expected.Items})
	}
	if got.M == nil || expected.M == nil {
		if got.M != expected.M {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	} else {
		checkEqualDefaultElemMap(t, DefaultElemMap{M: *got.M}, DefaultElemMap{M: *expected.M})
	}
}

type DefaultElemUnsupported struct {
	Items []Strings `defaultElem:"unsupported"`
}

func checkEqualDefaultElemUnsupported(t *testing.T, got, expected DefaultElemUnsupported) {
	t.Helper()
	checkEqualDefaultElemSlice(t, DefaultElemSlice{Items: got.Items}, DefaultElemSlice{Items: expected.Items})
}

type AllocNoop struct {
	S  string            `default:"alloc"`
	SS []string          `default:"alloc"`
	M  map[string]string `default:"alloc"`
}

func checkEqualAllocNoop(t *testing.T, got, expected AllocNoop) {
	t.Helper()
	if got.S != expected.S || len(got.SS) != len(expected.SS) || len(got.M) != len(expected.M) {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
	for i := range expected.SS {
		if got.SS[i] != expected.SS[i] {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	}
	for key, expectedValue := range expected.M {
		if got.M[key] != expectedValue {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	}
}

type DiveIgnored struct {
	PI *int   `default:"dive"`
	S  string `default:"dive"`
}

func checkEqualDiveIgnored(t *testing.T, got, expected DiveIgnored) {
	t.Helper()
	if got.PI != expected.PI || got.S != expected.S {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

type CustomString string

type CustomInt int

type CustomBool bool

type NamedScalars struct {
	S CustomString `default:"s"`
	I CustomInt    `default:"5"`
	B CustomBool   `default:"true"`
}

func checkEqualNamedScalars(t *testing.T, got, expected NamedScalars) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

type EnvInvalidInt struct {
	I int `env:"I"`
}

func pString(s string) *string {
	return &s
}

func pInt(i int) *int {
	return &i
}
func pInt8(i int8) *int8 {
	return &i
}
func pInt16(i int16) *int16 {
	return &i
}
func pInt32(i int32) *int32 {
	return &i
}
func pInt64(i int64) *int64 {
	return &i
}

func pFloat32(f float32) *float32 {
	return &f
}
func pFloat64(f float64) *float64 {
	return &f
}

func pBool(b bool) *bool {
	return &b
}

func pUint(i uint) *uint {
	return &i
}
func pUint8(i uint8) *uint8 {
	return &i
}
func pUint16(i uint16) *uint16 {
	return &i
}
func pUint32(i uint32) *uint32 {
	return &i
}
func pUint64(i uint64) *uint64 {
	return &i
}

func pUintptr(i uintptr) *uintptr {
	return &i
}

func pByte(i byte) *byte {
	return &i
}

func pRune(r rune) *rune {
	return &r
}

func pComplex64(c complex64) *complex64 {
	return &c
}
func pComplex128(c complex128) *complex128 {
	return &c
}

func pDuration(dur types.Duration) *types.Duration {
	return &dur
}
func pTDuration(dur time.Duration) *time.Duration {
	return &dur
}

func checkEqualValue[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s, got: %v, want: %v", name, got, want)
	}
}

func checkEqualPtr[T comparable](t *testing.T, name string, got, want *T) {
	t.Helper()

	if got == nil && want == nil {
		return
	}
	if got == nil {
		t.Fatalf("%s, got nil, want: %v", name, *want)
	}
	if want == nil {
		t.Fatalf("%s, got: %v, want nil", name, *got)
	}
	checkEqualValue(t, name, *got, *want)
}

type myStringer interface{ String() string }
type wrapS struct{ v string }

func (w wrapS) String() string { return w.v }

// nonempty for string
func ruleNonEmpty(s string, _ ...string) error {
	if s == "" {
		return errorc.New("must not be empty")
	}
	return nil
}

// withParams echoes params to prove parsing worked
func ruleWithParams(_ string, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("expected params")
	}
	// Return an error that includes params for assertion
	return fmt.Errorf("params=%s", strings.Join(params, "|"))
}

// nonzeroDuration (time.Duration or int64 underlying)
func ruleNonzeroDuration(d time.Duration, _ ...string) error {
	if d == 0 {
		return fmt.Errorf("duration must be non-zero")
	}
	return nil
}

// int rule that always errors (to populate FieldError)
func ruleIntAlwaysErr(_ int, _ ...string) error {
	return fmt.Errorf("bad int")
}

// Rule for fmt.Stringer (AssignableTo interface)
func ruleStringerBad(_ fmt.Stringer, _ ...string) error {
	return fmt.Errorf("bad stringer")
}
