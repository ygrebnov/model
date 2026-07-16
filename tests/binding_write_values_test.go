package tests

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
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
		{name: "S", value: "value"},
		{name: "PS", value: obj.PS},
		{name: "I", value: 7},
		{name: "PI", value: obj.PI},
		{name: "B", value: true},
		{name: "PB", value: obj.PB},
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
		{name: "S", value: ""},
		{name: "I", value: 0},
		{name: "B", value: false},
		{name: "PS", value: (*string)(nil)},
		{name: "M", value: map[string]int(nil)},
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
		{name: "Name", value: "app"},
		{name: "Server", value: obj.Server},
		{name: "Server.Host", value: "localhost"},
		{name: "Server.Port", value: 8080},
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
		{name: "Server", value: obj.Server},
		{name: "Server.Host", value: "localhost"},
		{name: "Server.Port", value: 8080},
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
		{name: "Server", value: (*nested)(nil)},
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
		{name: "Items[]", value: obj.Items},
		{name: "Items[].Name", value: "first"},
		{name: "Items[].Port", value: 1},
		{name: "Items[].Name", value: "second"},
		{name: "Items[].Port", value: 2},
		{name: "Array[]", value: obj.Array},
		{name: "Array[].Name", value: "array"},
		{name: "Array[].Port", value: 3},
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
		{name: "Items[]", value: obj.Items},
		{name: "Items[].Name", value: "first"},
		{name: "Items[].Name", value: "third"},
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
		{name: "Items[]", value: obj.Items},
		{name: "Items[].Name", value: "first"},
		{name: "Items[].Port", value: 1},
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

	if sink.values[0].name != "Items[]" ||
		!reflect.DeepEqual(sink.values[0].value, obj.Items) {
		t.Fatalf(
			"first write = %#v, want collection value",
			sink.values[0],
		)
	}

	if sink.values[1].name != "Items[].Name" ||
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
		{name: "Items", value: obj.Items},
		{name: "M", value: obj.M},
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
		{name: "Value", value: "text"},
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
			"Second": sinkErr,
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
		{name: "First", value: "first"},
	}

	assertWrittenValues(t, sink.values, expected)
}

func TestBindingWriteValues_NilSink(t *testing.T) {
	binding, err := model.NewBinding[Strings]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := Strings{}
	err = binding.WriteValues(&obj, nil)
	if !errors.Is(err, modelerrors.ErrInvalidValue) {
		t.Fatalf(
			"WriteValues() error = %v, want %v",
			err,
			modelerrors.ErrInvalidValue,
		)
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
