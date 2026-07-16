package tests

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

type mapValueSource struct {
	values map[string]any
	errFor map[string]error
	calls  []string
}

func (s *mapValueSource) Get(name string) (any, bool, error) {
	s.calls = append(s.calls, name)

	if err, ok := s.errFor[name]; ok {
		return nil, false, err
	}

	value, ok := s.values[name]
	return value, ok, nil
}

var _ field.ValueSource = (*mapValueSource)(nil)

func TestBindingApplyValues_AssignsScalarAndPointerValues(t *testing.T) {
	type config struct {
		S  string
		PS *string
		I  int
		PI *int
		B  bool
		PB *bool
	}

	source := &mapValueSource{
		values: map[string]any{
			"S":  "provided",
			"PS": "pointer-value",
			"I":  7,
			"PI": 8,
			"B":  true,
			"PB": false,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	checkEqualValue(t, "S", got.S, "provided")
	checkEqualPtr(t, "PS", got.PS, pString("pointer-value"))
	checkEqualValue(t, "I", got.I, 7)
	checkEqualPtr(t, "PI", got.PI, pInt(8))
	checkEqualValue(t, "B", got.B, true)
	checkEqualPtr(t, "PB", got.PB, pBool(false))
}

func TestBindingApplyValues_OverridesExistingValues(t *testing.T) {
	type config struct {
		S string
		I int
		B bool
	}

	source := &mapValueSource{
		values: map[string]any{
			"S": "replacement",
			"I": 11,
			"B": false,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		S: "original",
		I: 5,
		B: true,
	}

	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	expected := config{
		S: "replacement",
		I: 11,
		B: false,
	}

	if got != expected {
		t.Fatalf("ApplyValues() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyValues_ConvertsConvertibleValues(t *testing.T) {
	type customString string
	type customInt int

	type config struct {
		S customString
		I customInt
	}

	source := &mapValueSource{
		values: map[string]any{
			"S": "converted",
			"I": int32(17),
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	expected := config{
		S: "converted",
		I: 17,
	}

	if got != expected {
		t.Fatalf("ApplyValues() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyValues_NilResetsFieldToZeroValue(t *testing.T) {
	type config struct {
		S  string
		PS *string
		M  map[string]int
	}

	source := &mapValueSource{
		values: map[string]any{
			"S":  nil,
			"PS": nil,
			"M":  nil,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		S:  "original",
		PS: pString("original-pointer"),
		M:  map[string]int{"one": 1},
	}

	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	if got.S != "" {
		t.Fatalf("S = %q, want empty string", got.S)
	}

	if got.PS != nil {
		t.Fatalf("PS = %v, want nil", *got.PS)
	}

	if got.M != nil {
		t.Fatalf("M = %#v, want nil", got.M)
	}
}

func TestBindingApplyValues_MissingValuesLeaveObjectUnchanged(t *testing.T) {
	type config struct {
		S string
		I int
	}

	source := &mapValueSource{
		values: map[string]any{},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		S: "original",
		I: 9,
	}

	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	expected := config{
		S: "original",
		I: 9,
	}

	if got != expected {
		t.Fatalf("ApplyValues() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyValues_NestedStruct(t *testing.T) {
	type nested struct {
		Host string
		Port int
	}

	type config struct {
		Server nested
	}

	source := &mapValueSource{
		values: map[string]any{
			"Server.Host": "localhost",
			"Server.Port": 8080,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	expected := config{
		Server: nested{
			Host: "localhost",
			Port: 8080,
		},
	}

	if got != expected {
		t.Fatalf("ApplyValues() result = %+v, want %+v", got, expected)
	}
}

func TestBindingApplyValues_PointerToStructAllocation(t *testing.T) {
	type nested struct {
		Host string
		Port int
	}

	type config struct {
		Server *nested
	}

	tests := []struct {
		name     string
		values   map[string]any
		expected *nested
	}{
		{
			name: "allocates when descendant value exists",
			values: map[string]any{
				"Server.Host": "localhost",
			},
			expected: &nested{
				Host: "localhost",
			},
		},
		{
			name:     "remains nil when no descendant value exists",
			values:   map[string]any{},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			binding, err := model.NewBinding[config]()
			if err != nil {
				t.Fatalf("NewBinding() error: %v", err)
			}

			got := config{}
			source := &mapValueSource{
				values: tc.values,
			}

			if err := binding.ApplyValues(&got, source); err != nil {
				t.Fatalf("ApplyValues() error: %v", err)
			}

			if !reflect.DeepEqual(got.Server, tc.expected) {
				t.Fatalf(
					"Server = %#v, want %#v",
					got.Server,
					tc.expected,
				)
			}
		})
	}
}

func TestBindingApplyValues_RecursivePointerProbeStopsAtCycleBoundary(
	t *testing.T,
) {
	type node struct {
		Value string
		Next  *node
	}

	type config struct {
		Root *node
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	tests := []struct {
		name     string
		values   map[string]any
		expected *node
	}{
		{
			name:     "empty source leaves root nil",
			values:   map[string]any{},
			expected: nil,
		},
		{
			name: "first-level value allocates root",
			values: map[string]any{
				"Root.Value": "provided",
			},
			expected: &node{
				Value: "provided",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := config{}
			source := &mapValueSource{
				values: test.values,
			}

			if err := binding.ApplyValues(&got, source); err != nil {
				t.Fatalf("ApplyValues() error: %v", err)
			}

			if !reflect.DeepEqual(got.Root, test.expected) {
				t.Fatalf("Root = %#v, want %#v", got.Root, test.expected)
			}
		})
	}
}

func TestBindingApplyValues_DirectPointerToStructValue(t *testing.T) {
	type nested struct {
		Host string
	}

	type config struct {
		Server *nested
	}

	expectedServer := &nested{
		Host: "direct",
	}

	source := &mapValueSource{
		values: map[string]any{
			"Server": expectedServer,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	if got.Server != expectedServer {
		t.Fatalf(
			"Server = %#v, want original provided pointer %#v",
			got.Server,
			expectedServer,
		)
	}
}

func TestBindingApplyValues_DirectCollectionValues(t *testing.T) {
	type config struct {
		Items []string
		M     map[string]int
	}

	expectedItems := []string{"one", "two"}
	expectedMap := map[string]int{
		"one": 1,
		"two": 2,
	}

	source := &mapValueSource{
		values: map[string]any{
			"Items[]": expectedItems,
			"M[]":     expectedMap,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	if !reflect.DeepEqual(got.Items, expectedItems) {
		t.Fatalf(
			"Items = %#v, want %#v",
			got.Items,
			expectedItems,
		)
	}

	if !reflect.DeepEqual(got.M, expectedMap) {
		t.Fatalf(
			"M = %#v, want %#v",
			got.M,
			expectedMap,
		)
	}
}

func TestBindingApplyValues_DoesNotApplyCollectionElementPaths(
	t *testing.T,
) {
	type item struct {
		Name string
	}

	type config struct {
		Items []item
	}

	source := &mapValueSource{
		values: map[string]any{
			"Items[].Name": "replacement",
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		Items: []item{
			{Name: "first"},
			{Name: "second"},
		},
	}

	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	expected := config{
		Items: []item{
			{Name: "first"},
			{Name: "second"},
		},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf(
			"ApplyValues() result = %#v, want unchanged %#v",
			got,
			expected,
		)
	}
}

func TestBindingApplyValues_SourceReceivesCompiledSchemaNames(
	t *testing.T,
) {
	type nested struct {
		Value string
	}

	type config struct {
		Top    string
		Nested nested
		Items  []nested
	}

	source := &mapValueSource{
		values: map[string]any{},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	expectedCalls := []string{
		"Top",
		"Nested",
		"Nested.Value",
		"Items[]",
		"Items[].Value",
	}

	if !reflect.DeepEqual(source.calls, expectedCalls) {
		t.Fatalf(
			"source calls = %#v, want %#v",
			source.calls,
			expectedCalls,
		)
	}
}

func TestBindingApplyValues_DistinguishesCaseSensitiveExportedNames(
	t *testing.T,
) {
	type config struct {
		URL string
		Url string //nolint:stylecheck
	}

	source := &mapValueSource{
		values: map[string]any{
			"URL": "initialism",
			"Url": "title-case",
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{}
	if err := binding.ApplyValues(&got, source); err != nil {
		t.Fatalf("ApplyValues() error: %v", err)
	}

	if got.URL != "initialism" || got.Url != "title-case" {
		t.Fatalf("ApplyValues() result = %+v, want both fields assigned", got)
	}
}

func TestBindingApplyValues_SourceErrorStopsApplication(t *testing.T) {
	type config struct {
		First  string
		Second string
	}

	sourceErr := errors.New("provider failure")

	source := &mapValueSource{
		values: map[string]any{
			"First": "replacement",
		},
		errFor: map[string]error{
			"Second": sourceErr,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		First:  "original-first",
		Second: "original-second",
	}

	err = binding.ApplyValues(&got, source)
	if err == nil {
		t.Fatal("ApplyValues() error = nil, want source error")
	}

	if !errors.Is(err, sourceErr) {
		t.Fatalf(
			"ApplyValues() error = %v, want wrapped %v",
			err,
			sourceErr,
		)
	}

	expected := config{
		First:  "original-first",
		Second: "original-second",
	}

	if got != expected {
		t.Fatalf(
			"object was partially modified: got %+v, want %+v",
			got,
			expected,
		)
	}
}

func TestBindingApplyValues_TypeMismatchReturnsError(t *testing.T) {
	type config struct {
		I int
	}

	source := &mapValueSource{
		values: map[string]any{
			"I": "not-an-int",
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		I: 42,
	}

	err = binding.ApplyValues(&got, source)
	if err == nil {
		t.Fatal("ApplyValues() error = nil, want type mismatch")
	}
	if !errors.Is(err, modelerrors.ErrTypeMismatch) {
		t.Fatalf(
			"ApplyValues() error = %v, want %v",
			err,
			modelerrors.ErrTypeMismatch,
		)
	}

	if got.I != 42 {
		t.Fatalf("I = %d, want original value 42", got.I)
	}
}

func TestBindingApplyValues_EarlierAssignmentsRemainOnLaterTypeMismatch(
	t *testing.T,
) {
	type config struct {
		First  string
		Second int
	}

	source := &mapValueSource{
		values: map[string]any{
			"First":  "applied",
			"Second": "not-an-int",
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := config{
		First:  "original",
		Second: 7,
	}

	err = binding.ApplyValues(&got, source)
	if err == nil {
		t.Fatal("ApplyValues() error = nil, want type mismatch")
	}

	if got.First != "applied" {
		t.Fatalf(
			"First = %q, want already-applied value",
			got.First,
		)
	}

	if got.Second != 7 {
		t.Fatalf(
			"Second = %d, want original value 7",
			got.Second,
		)
	}
}

func TestBindingApplyValues_NilSource(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	got := Strings{}
	if err := binding.ApplyValues(&got, nil); err == nil {
		t.Fatal("ApplyValues() error = nil, want nil-source error")
	}
}

func TestBindingApplyValues_NilObject(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	source := &mapValueSource{
		values: map[string]any{},
	}

	if err := binding.ApplyValues(nil, source); err == nil {
		t.Fatal("ApplyValues(nil) error = nil, want nil-object error")
	}
}
