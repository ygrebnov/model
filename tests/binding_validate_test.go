package tests

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
	"github.com/ygrebnov/model/validation"
)

type validateInner struct {
	Name string        `validate:"nonempty"`
	Wait time.Duration `validate:"nonzeroDuration"`
}

type validateOuter struct {
	Title string `validate:"nonempty"`
	Note  string `validate:"omitempty,min(3)"`

	Inner    validateInner
	InnerPtr *validateInner

	Tags   []string                  `validateElem:"nonempty"`
	Fixed  [2]string                 `validateElem:"nonempty"`
	Labels map[string]string         `validateElem:"nonempty"`
	Items  []validateInner           `validateElem:"dive"`
	Ptrs   []*validateInner          `validateElem:"dive"`
	ByName map[string]validateInner  `validateElem:"dive"`
	ByPtr  map[string]*validateInner `validateElem:"dive"`
}

func TestBindingValidate_EndToEnd(t *testing.T) {
	nonempty, err := model.NewRule[string](
		"nonempty",
		validateNonempty,
	)
	if err != nil {
		t.Fatalf("NewRule(nonempty) error: %v", err)
	}

	nonzeroDuration, err := model.NewRule[time.Duration](
		"nonzeroDuration",
		validateNonzeroDuration,
	)
	if err != nil {
		t.Fatalf("NewRule(nonzeroDuration) error: %v", err)
	}

	binding, err := model.NewBinding[validateOuter](
		model.WithRules(
			nonempty,
			nonzeroDuration,
		),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := validateOuter{
		Title: "",
		Note:  "",
		Inner: validateInner{},
		InnerPtr: &validateInner{
			Name: "ok",
			Wait: 0,
		},
		Tags:  []string{"", "ok", ""},
		Fixed: [2]string{"ok", ""},
		Labels: map[string]string{
			"bad": "",
			"ok":  "value",
		},
		Items: []validateInner{
			{},
			{
				Name: "ok",
				Wait: 0,
			},
		},
		Ptrs: []*validateInner{
			nil,
			{},
		},
		ByName: map[string]validateInner{
			"first": {},
			"second": {
				Name: "ok",
				Wait: 0,
			},
		},
		ByPtr: map[string]*validateInner{
			"nil": nil,
			"set": {},
		},
	}

	err = binding.Validate(context.Background(), &obj)
	ve := requireValidationError(t, err)
	byField := ve.ByField()

	assertFieldRule(t, byField, "title", "nonempty")

	// omitempty suppresses min(3) for an empty string.
	assertNoFieldError(t, byField, "note")

	// Ordinary nested structs and non-nil pointers are traversed automatically.
	assertFieldRule(t, byField, "inner.name", "nonempty")
	assertFieldRule(t, byField, "inner.wait", "nonzeroDuration")
	assertNoFieldError(t, byField, "innerptr.name")
	assertFieldRule(t, byField, "innerptr.wait", "nonzeroDuration")

	// validateElem applies ordinary rules to scalar collection elements.
	assertFieldRule(t, byField, "tags[0]", "nonempty")
	assertNoFieldError(t, byField, "tags[1]")
	assertFieldRule(t, byField, "tags[2]", "nonempty")

	assertNoFieldError(t, byField, "fixed[0]")
	assertFieldRule(t, byField, "fixed[1]", "nonempty")

	assertFieldRule(t, byField, "labels[bad]", "nonempty")
	assertNoFieldError(t, byField, "labels[ok]")

	// validateElem:dive traverses struct collection elements.
	assertFieldRule(t, byField, "items[0].name", "nonempty")
	assertFieldRule(t, byField, "items[0].wait", "nonzeroDuration")
	assertNoFieldError(t, byField, "items[1].name")
	assertFieldRule(t, byField, "items[1].wait", "nonzeroDuration")

	// Nil pointer elements are reported as dive errors; non-nil elements are
	// traversed normally.
	assertFieldRule(t, byField, "ptrs[0]", "dive")
	assertFieldRule(t, byField, "ptrs[1].name", "nonempty")
	assertFieldRule(t, byField, "ptrs[1].wait", "nonzeroDuration")

	assertFieldRule(
		t,
		byField,
		"byname[first].name",
		"nonempty",
	)
	assertFieldRule(
		t,
		byField,
		"byname[first].wait",
		"nonzeroDuration",
	)
	assertNoFieldError(
		t,
		byField,
		"byname[second].name",
	)
	assertFieldRule(
		t,
		byField,
		"byname[second].wait",
		"nonzeroDuration",
	)

	assertFieldRule(t, byField, "byptr[nil]", "dive")
	assertFieldRule(
		t,
		byField,
		"byptr[set].name",
		"nonempty",
	)
	assertFieldRule(
		t,
		byField,
		"byptr[set].wait",
		"nonzeroDuration",
	)
}

func TestBindingValidate_ValidObject(t *testing.T) {
	nonempty, err := model.NewRule[string](
		"nonempty",
		validateNonempty,
	)
	if err != nil {
		t.Fatalf("NewRule(nonempty) error: %v", err)
	}

	nonzeroDuration, err := model.NewRule[time.Duration](
		"nonzeroDuration",
		validateNonzeroDuration,
	)
	if err != nil {
		t.Fatalf("NewRule(nonzeroDuration) error: %v", err)
	}

	binding, err := model.NewBinding[validateOuter](
		model.WithRules(
			nonempty,
			nonzeroDuration,
		),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := validateOuter{
		Title: "valid",
		Note:  "note",
		Inner: validateInner{
			Name: "inner",
			Wait: time.Second,
		},
		InnerPtr: nil,
		Tags:     []string{"one", "two"},
		Fixed:    [2]string{"one", "two"},
		Labels: map[string]string{
			"one": "value",
		},
		Items: []validateInner{
			{
				Name: "one",
				Wait: time.Second,
			},
		},
		Ptrs: []*validateInner{
			{
				Name: "one",
				Wait: time.Second,
			},
		},
		ByName: map[string]validateInner{
			"one": {
				Name: "one",
				Wait: time.Second,
			},
		},
		ByPtr: map[string]*validateInner{
			"one": {
				Name: "one",
				Wait: time.Second,
			},
		},
	}

	if err := binding.Validate(
		context.Background(),
		&obj,
	); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
}

func TestBindingValidate_Builtins(t *testing.T) {
	type config struct {
		Name  string  `validate:"min(3),max(5)"`
		Count int     `validate:"min(2),max(4)"`
		Score float64 `validate:"nonzero"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Name:  "ab",
		Count: 5,
		Score: 0,
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)
	byField := ve.ByField()

	assertFieldRule(t, byField, "name", "min")
	assertFieldRule(t, byField, "count", "max")
	assertFieldRule(t, byField, "score", "nonzero")
}

func TestBindingValidate_OmitEmpty(t *testing.T) {
	type config struct {
		Name string `validate:"omitempty,min(3)"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	tests := []struct {
		name     string
		value    string
		wantRule string
	}{
		{
			name:  "empty value is skipped",
			value: "",
		},
		{
			name:  "valid value passes",
			value: "valid",
		},
		{
			name:     "short non-empty value fails",
			value:    "ab",
			wantRule: "min",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obj := config{
				Name: test.value,
			}

			err := binding.Validate(context.Background(), &obj)
			if test.wantRule == "" {
				if err != nil {
					t.Fatalf("Validate() error: %v", err)
				}

				return
			}

			validationErr := requireValidationError(t, err)
			assertFieldRule(
				t,
				validationErr.ByField(),
				"name",
				test.wantRule,
			)
		})
	}
}

func TestBindingValidate_ValidateElemBuiltins(t *testing.T) {
	type config struct {
		Names  []string `validateElem:"min(2)"`
		Counts []int    `validateElem:"min(2)"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Names:  []string{"x", "valid"},
		Counts: []int{1, 2},
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)
	byField := ve.ByField()

	assertFieldRule(t, byField, "names[0]", "min")
	assertNoFieldError(t, byField, "names[1]")

	assertFieldRule(t, byField, "counts[0]", "min")
	assertNoFieldError(t, byField, "counts[1]")
}

func TestBindingValidate_ValidateElemOmitempty(t *testing.T) {
	type config struct {
		Names []string `validateElem:"omitempty,min(3)"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Names: []string{
			"",
			"ab",
			"valid",
		},
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)
	byField := ve.ByField()

	assertNoFieldError(t, byField, "names[0]")
	assertFieldRule(t, byField, "names[1]", "min")
	assertNoFieldError(t, byField, "names[2]")
}

func TestBindingValidate_CustomRuleAssignableToInterface(t *testing.T) {
	type config struct {
		Value stringerValue `validate:"stringer"`
	}

	rule, err := model.NewRule[fmt.Stringer](
		"stringer",
		func(value fmt.Stringer, _ ...string) error {
			if value.String() != "valid" {
				return errorc.New("invalid stringer")
			}

			return nil
		},
	)
	if err != nil {
		t.Fatalf("NewRule() error: %v", err)
	}

	binding, err := model.NewBinding[config](
		model.WithRules(rule),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Value: stringerValue("invalid"),
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)

	assertFieldRule(
		t,
		ve.ByField(),
		"value",
		"stringer",
	)
}

type stringerValue string

func (v stringerValue) String() string {
	return string(v)
}

func TestBindingValidate_CustomRuleReceivesParameters(t *testing.T) {
	type config struct {
		Value string `validate:"params(one,two,three)"`
	}

	rule, err := model.NewRule[string](
		"params",
		func(_ string, params ...string) error {
			if strings.Join(params, "|") != "one|two|three" {
				return fmt.Errorf(
					"unexpected params: %v",
					params,
				)
			}

			return errorc.New("parameter rule failed")
		},
	)
	if err != nil {
		t.Fatalf("NewRule() error: %v", err)
	}

	binding, err := model.NewBinding[config](
		model.WithRules(rule),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Value: "value",
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)
	fieldErrors := ve.ByField()["value"]

	if len(fieldErrors) != 1 {
		t.Fatalf(
			"value error count = %d, want 1: %+v",
			len(fieldErrors),
			fieldErrors,
		)
	}

	if got := strings.Join(
		fieldErrors[0].Params,
		"|",
	); got != "one|two|three" {
		t.Fatalf(
			"field error params = %q, want %q",
			got,
			"one|two|three",
		)
	}
}

func TestBindingValidate_MultipleRulesAccumulate(t *testing.T) {
	type config struct {
		Value string `validate:"first,second"`
	}

	first, err := model.NewRule[string](
		"first",
		func(string, ...string) error {
			return errorc.New("first failed")
		},
	)
	if err != nil {
		t.Fatalf("NewRule(first) error: %v", err)
	}

	second, err := model.NewRule[string](
		"second",
		func(string, ...string) error {
			return errorc.New("second failed")
		},
	)
	if err != nil {
		t.Fatalf("NewRule(second) error: %v", err)
	}

	binding, err := model.NewBinding[config](
		model.WithRules(first, second),
	)
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Value: "value",
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)
	fieldErrors := ve.ByField()["value"]

	if len(fieldErrors) != 2 {
		t.Fatalf(
			"value error count = %d, want 2: %+v",
			len(fieldErrors),
			fieldErrors,
		)
	}

	assertFieldRule(t, ve.ByField(), "value", "first")
	assertFieldRule(t, ve.ByField(), "value", "second")
}

func TestBindingValidate_NilPointerToStructIsSkipped(t *testing.T) {
	type nested struct {
		Name string `validate:"min(1)"`
	}

	type config struct {
		Nested *nested
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Nested: nil,
	}

	if err := binding.Validate(
		context.Background(),
		&obj,
	); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if obj.Nested != nil {
		t.Fatalf(
			"Validate() allocated Nested: %#v",
			obj.Nested,
		)
	}
}

func TestBindingValidate_EmptyAndNilCollections(t *testing.T) {
	type nested struct {
		Name string `validate:"min(1)"`
	}

	type config struct {
		Names []string          `validateElem:"min(1)"`
		Map   map[string]string `validateElem:"min(1)"`
		Items []*nested         `validateElem:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	tests := []struct {
		name string
		obj  config
	}{
		{
			name: "nil collections",
			obj:  config{},
		},
		{
			name: "empty collections",
			obj: config{
				Names: []string{},
				Map:   map[string]string{},
				Items: []*nested{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obj := test.obj

			if err := binding.Validate(
				context.Background(),
				&obj,
			); err != nil {
				t.Fatalf("Validate() error: %v", err)
			}
		})
	}
}

func TestBindingValidate_FailsEarlyForUnknownRule(t *testing.T) {
	type config struct {
		Value string `validate:"missing"`
	}

	_, err := model.NewBinding[config]()
	if err == nil {
		t.Fatal(
			"NewBinding() error = nil, want unknown-rule error",
		)
	}

	if !stderrors.Is(err, errors.ErrRuleNotFound) {
		t.Fatalf(
			"NewBinding() error = %v, want ErrRuleNotFound",
			err,
		)
	}
}

func TestBindingValidate_FailsEarlyForUnknownElementRule(
	t *testing.T,
) {
	type config struct {
		Values []string `validateElem:"missing"`
	}

	_, err := model.NewBinding[config]()
	if err == nil {
		t.Fatal(
			"NewBinding() error = nil, want unknown-rule error",
		)
	}

	if !stderrors.Is(err, errors.ErrRuleNotFound) {
		t.Fatalf(
			"NewBinding() error = %v, want ErrRuleNotFound",
			err,
		)
	}
}

func TestBindingValidate_FailsEarlyForInapplicableRule(
	t *testing.T,
) {
	type config struct {
		Value int `validate:"textRule"`
	}

	textRule, err := model.NewRule[string](
		"textRule",
		func(string, ...string) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("NewRule() error: %v", err)
	}

	_, err = model.NewBinding[config](
		model.WithRules(textRule),
	)
	if err == nil {
		t.Fatal(
			"NewBinding() error = nil, want overload error",
		)
	}

	if !stderrors.Is(
		err,
		errors.ErrRuleOverloadNotFound,
	) {
		t.Fatalf(
			"NewBinding() error = %v, "+
				"want ErrRuleOverloadNotFound",
			err,
		)
	}
}

func TestBindingValidate_FailsEarlyForInapplicableElementRule(
	t *testing.T,
) {
	type config struct {
		Values []int `validateElem:"textRule"`
	}

	textRule, err := model.NewRule[string](
		"textRule",
		func(string, ...string) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("NewRule() error: %v", err)
	}

	_, err = model.NewBinding[config](
		model.WithRules(textRule),
	)
	if err == nil {
		t.Fatal(
			"NewBinding() error = nil, want overload error",
		)
	}

	if !stderrors.Is(
		err,
		errors.ErrRuleOverloadNotFound,
	) {
		t.Fatalf(
			"NewBinding() error = %v, "+
				"want ErrRuleOverloadNotFound",
			err,
		)
	}
}

func TestBindingValidate_FailsEarlyForValidateElemOnNonCollection(
	t *testing.T,
) {
	type config struct {
		Value string `validateElem:"min(1)"`
	}

	_, err := model.NewBinding[config]()
	if err == nil {
		t.Fatal(
			"NewBinding() error = nil, " +
				"want validateElem usage error",
		)
	}

	if !stderrors.Is(
		err,
		errors.ErrInvalidValidateElemUsage,
	) {
		t.Fatalf(
			"NewBinding() error = %v, "+
				"want ErrInvalidValidateElemUsage",
			err,
		)
	}
}

func TestBindingValidate_FailsEarlyForDuplicateOverload(
	t *testing.T,
) {
	type config struct {
		Value string `validate:"custom"`
	}

	first, err := model.NewRule[string](
		"custom",
		func(string, ...string) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("NewRule(first) error: %v", err)
	}

	second, err := model.NewRule[string](
		"custom",
		func(string, ...string) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("NewRule(second) error: %v", err)
	}

	_, err = model.NewBinding[config](
		model.WithRules(first, second),
	)
	if err == nil {
		t.Fatal(
			"NewBinding() error = nil, " +
				"want duplicate-overload error",
		)
	}

	if !stderrors.Is(
		err,
		errors.ErrDuplicateOverloadRule,
	) {
		t.Fatalf(
			"NewBinding() error = %v, "+
				"want ErrDuplicateOverloadRule",
			err,
		)
	}
}

func TestBindingValidate_DiveOnScalarElementsReportsRuntimeErrors(
	t *testing.T,
) {
	type config struct {
		Values []int `validateElem:"dive"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{
		Values: []int{1, 2},
	}

	ve := requireValidationError(
		t,
		binding.Validate(context.Background(), &obj),
	)

	assertFieldRule(
		t,
		ve.ByField(),
		"values[0]",
		"dive",
	)
	assertFieldRule(
		t,
		ve.ByField(),
		"values[1]",
		"dive",
	)
}

func TestBindingValidate_ContextAndObjectErrors(t *testing.T) {
	type config struct {
		Name string `validate:"min(1)"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	obj := config{}

	if err := binding.Validate(
		nil,
		&obj,
	); !stderrors.Is(err, errors.ErrNilContext) {
		t.Fatalf(
			"Validate(nil context) error = %v, "+
				"want ErrNilContext",
			err,
		)
	}

	if err := binding.Validate(
		context.Background(),
		nil,
	); !stderrors.Is(err, errors.ErrNilObject) {
		t.Fatalf(
			"Validate(nil object) error = %v, "+
				"want ErrNilObject",
			err,
		)
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	if err := binding.Validate(
		ctx,
		&obj,
	); !stderrors.Is(err, context.Canceled) {
		t.Fatalf(
			"Validate(cancelled context) error = %v, "+
				"want context.Canceled",
			err,
		)
	}
}

func TestBindingValidate_ConcurrentUseWithDifferentObjects(
	t *testing.T,
) {
	type config struct {
		Name string `validate:"min(2)"`
	}

	binding, err := model.NewBinding[config]()
	if err != nil {
		t.Fatalf("NewBinding() error: %v", err)
	}

	const workers = 32

	var wg sync.WaitGroup

	errorsCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			obj := config{
				Name: "valid",
			}

			if index%2 == 0 {
				obj.Name = "x"
			}

			err := binding.Validate(
				context.Background(),
				&obj,
			)

			if index%2 == 0 {
				if err == nil {
					errorsCh <- fmt.Errorf(
						"worker %d: expected validation error",
						index,
					)
					return
				}

				var ve *validation.Error
				if !stderrors.As(err, &ve) {
					errorsCh <- fmt.Errorf(
						"worker %d: expected "+
							"validation.Error, got %T",
						index,
						err,
					)
				}

				return
			}

			if err != nil {
				errorsCh <- fmt.Errorf(
					"worker %d: unexpected error: %w",
					index,
					err,
				)
			}
		}(i)
	}

	wg.Wait()
	close(errorsCh)

	for err := range errorsCh {
		t.Error(err)
	}
}

func validateNonempty(
	value string,
	_ ...string,
) error {
	if value == "" {
		return errorc.With(
			errorc.New("must not be empty"),
			errorc.String(
				keys.RuleName,
				"nonempty",
			),
		)
	}

	return nil
}

func validateNonzeroDuration(
	value time.Duration,
	_ ...string,
) error {
	if value == 0 {
		return errorc.With(
			errorc.New("duration must not be zero"),
			errorc.String(
				keys.RuleName,
				"nonzeroDuration",
			),
		)
	}

	return nil
}

func requireValidationError(
	t *testing.T,
	err error,
) *validation.Error {
	t.Helper()

	if err == nil {
		t.Fatal(
			"Validate() error = nil, want validation error",
		)
	}

	var ve *validation.Error
	if !stderrors.As(err, &ve) {
		t.Fatalf(
			"Validate() error type = %T, "+
				"want *validation.Error: %v",
			err,
			err,
		)
	}

	if ve.Empty() {
		t.Fatal("validation.Error is empty")
	}

	return ve
}

func assertFieldRule(
	t *testing.T,
	byField map[string][]validation.FieldError,
	path string,
	rule string,
) {
	t.Helper()

	fieldErrors := byField[path]
	if len(fieldErrors) == 0 {
		t.Fatalf(
			"expected error at %q for rule %q; "+
				"available fields: %s",
			path,
			rule,
			fieldNames(byField),
		)
	}

	for _, fieldErr := range fieldErrors {
		if fieldErr.Rule == rule {
			return
		}
	}

	t.Fatalf(
		"errors at %q do not contain rule %q: %+v",
		path,
		rule,
		fieldErrors,
	)
}

func assertNoFieldError(
	t *testing.T,
	byField map[string][]validation.FieldError,
	path string,
) {
	t.Helper()

	if fieldErrors := byField[path]; len(fieldErrors) != 0 {
		t.Fatalf(
			"unexpected errors at %q: %+v",
			path,
			fieldErrors,
		)
	}
}

func fieldNames(
	byField map[string][]validation.FieldError,
) string {
	names := make([]string, 0, len(byField))

	for name := range byField {
		names = append(names, name)
	}

	return strings.Join(names, ", ")
}
