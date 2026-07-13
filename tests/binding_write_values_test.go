package tests

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/field"
)

type writtenValue struct {
	name  string
	value any
}

type recordingValueSink struct {
	values []writtenValue
	errFor map[string]error
}

func (s *recordingValueSink) Set(name string, value any) error {
	if err, ok := s.errFor[name]; ok {
		return err
	}

	s.values = append(s.values, writtenValue{
		name:  name,
		value: value,
	})

	return nil
}

var _ field.ValueSink = (*recordingValueSink)(nil)

func TestBindingWriteValues_ScalarsAndPointers(t *testing.T) {
	type config struct {
		S  string
		PS *string
		I  int
		PI *int
		B  bool
		PB *bool
	}

	obj := config{
		S:  "value",
		PS: pString("pointer-value"),
		I:  7,
		PI: pInt(8),
		B:  true,
		PB: pBool(false),
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "s", value: "value"},
		{name: "ps", value: obj.PS},
		{name: "i", value: 7},
		{name: "pi", value: obj.PI},
		{name: "b", value: true},
		{name: "pb", value: obj.PB},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_WritesZeroAndNilValues(t *testing.T) {
	type config struct {
		S  string
		I  int
		B  bool
		PS *string
		M  map[string]int
	}

	obj := config{}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "s", value: ""},
		{name: "i", value: 0},
		{name: "b", value: false},
		{name: "ps", value: (*string)(nil)},
		{name: "m", value: map[string]int(nil)},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_NestedStruct(t *testing.T) {
	type nested struct {
		Host string
		Port int
	}

	type config struct {
		Name   string
		Server nested
	}

	obj := config{
		Name: "app",
		Server: nested{
			Host: "localhost",
			Port: 8080,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "name", value: "app"},
		{name: "server", value: obj.Server},
		{name: "server.host", value: "localhost"},
		{name: "server.port", value: 8080},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_NonNilPointerToStruct(t *testing.T) {
	type nested struct {
		Host string
		Port int
	}

	type config struct {
		Server *nested
	}

	obj := config{
		Server: &nested{
			Host: "localhost",
			Port: 8080,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "server", value: obj.Server},
		{name: "server.host", value: "localhost"},
		{name: "server.port", value: 8080},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_NilPointerToStructIsWrittenButNotTraversed(
	t *testing.T,
) {
	type nested struct {
		Host string
	}

	type config struct {
		Server *nested
	}

	obj := config{}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "server", value: (*nested)(nil)},
	}

	assertWrittenValues(t, sink.values, expected)

	if obj.Server != nil {
		t.Fatalf("Server = %#v, want nil", obj.Server)
	}
}

func TestBindingWriteValues_SliceAndArrayStructElements(t *testing.T) {
	type item struct {
		Name string
		Port int
	}

	type config struct {
		Items []item
		Array [1]item
	}

	obj := config{
		Items: []item{
			{Name: "first", Port: 1},
			{Name: "second", Port: 2},
		},
		Array: [1]item{
			{Name: "array", Port: 3},
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "items[]", value: obj.Items},
		{name: "items[].name", value: "first"},
		{name: "items[].port", value: 1},
		{name: "items[].name", value: "second"},
		{name: "items[].port", value: 2},
		{name: "array[]", value: obj.Array},
		{name: "array[].name", value: "array"},
		{name: "array[].port", value: 3},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_PointerSliceElements(t *testing.T) {
	type item struct {
		Name string
	}

	type config struct {
		Items []*item
	}

	obj := config{
		Items: []*item{
			{Name: "first"},
			nil,
			{Name: "third"},
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "items[]", value: obj.Items},
		{name: "items[].name", value: "first"},
		{name: "items[].name", value: "third"},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_MapStructValues(t *testing.T) {
	type item struct {
		Name string
		Port int
	}

	type config struct {
		Items map[string]item
	}

	obj := config{
		Items: map[string]item{
			"one": {
				Name: "first",
				Port: 1,
			},
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "items[]", value: obj.Items},
		{name: "items[].name", value: "first"},
		{name: "items[].port", value: 1},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_MapPointerValuesSkipNilEntries(t *testing.T) {
	type item struct {
		Name string
	}

	type config struct {
		Items map[string]*item
	}

	obj := config{
		Items: map[string]*item{
			"one": {Name: "first"},
			"nil": nil,
		},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	if len(sink.values) != 2 {
		t.Fatalf(
			"written value count = %d, want 2: %#v",
			len(sink.values),
			sink.values,
		)
	}

	if sink.values[0].name != "items[]" ||
		!reflect.DeepEqual(sink.values[0].value, obj.Items) {
		t.Fatalf(
			"first write = %#v, want collection value",
			sink.values[0],
		)
	}

	if sink.values[1].name != "items[].name" ||
		sink.values[1].value != "first" {
		t.Fatalf(
			"second write = %#v, want non-nil map element field",
			sink.values[1],
		)
	}
}

func TestBindingWriteValues_ScalarCollectionsAreWrittenWithoutElementTraversal(
	t *testing.T,
) {
	type config struct {
		Items []string
		M     map[string]int
	}

	obj := config{
		Items: []string{"one", "two"},
		M:     map[string]int{"one": 1},
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "items", value: obj.Items},
		{name: "m", value: obj.M},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_InterfaceValue(t *testing.T) {
	type config struct {
		Value any
	}

	obj := config{
		Value: "text",
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(&obj, sink); err != nil {
		t.Fatalf("WriteValues() error: %v", err)
	}

	expected := []writtenValue{
		{name: "value", value: "text"},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_SinkErrorStopsTraversal(t *testing.T) {
	type config struct {
		First  string
		Second string
		Third  string
	}

	sinkErr := errors.New("sink failure")
	sink := &recordingValueSink{
		errFor: map[string]error{
			"second": sinkErr,
		},
	}

	obj := config{
		First:  "first",
		Second: "second",
		Third:  "third",
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	err = binding.WriteValues(&obj, sink)
	if err == nil {
		t.Fatal("WriteValues() error = nil, want sink error")
	}

	if !errors.Is(err, sinkErr) {
		t.Fatalf(
			"WriteValues() error = %v, want wrapped %v",
			err,
			sinkErr,
		)
	}

	expected := []writtenValue{
		{name: "first", value: "first"},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_NilSink(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := Strings{}
	if err := binding.WriteValues(&obj, nil); err == nil {
		t.Fatal("WriteValues() error = nil, want nil-sink error")
	}
}

func TestBindingWriteValues_NilObject(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	sink := &recordingValueSink{}
	if err := binding.WriteValues(nil, sink); err == nil {
		t.Fatal("WriteValues(nil) error = nil, want nil-object error")
	}
}

func assertWrittenValues(
	t *testing.T,
	got []writtenValue,
	want []writtenValue,
) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf(
			"written value count = %d, want %d\ngot:  %#v\nwant: %#v",
			len(got),
			len(want),
			got,
			want,
		)
	}

	for i := range want {
		if got[i].name != want[i].name {
			t.Fatalf(
				"write[%d].name = %q, want %q",
				i,
				got[i].name,
				want[i].name,
			)
		}

		if !reflect.DeepEqual(got[i].value, want[i].value) {
			t.Fatalf(
				"write[%d].value = %#v, want %#v",
				i,
				got[i].value,
				want[i].value,
			)
		}
	}
}
