package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/rule"
)

// Example 1: WithDefaults — apply defaults during construction
func exampleWithDefaults() {
	type Cfg struct {
		Name    string        `default:"svc"`
		Timeout time.Duration `default:"250ms"`
	}

	cfg := Cfg{}
	m, err := model.New(&cfg, model.WithDefaults[Cfg]())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	_ = m
	fmt.Printf("WithDefaults -> Name=%q Timeout=%v\n", cfg.Name, cfg.Timeout)
}

// Example 2: WithValidation + WithRule — register a custom rule and fail validation
func exampleWithValidationAndWithRule_error() {
	type Input struct {
		// Use a custom rule name for a non-builtin type (time.Duration)
		D time.Duration `validate:"nonzeroDur"`
	}

	r := rule.Rule[time.Duration]{
		Name: "nonzeroDur",
		Fn: func(d time.Duration, _ ...string) error {
			if d == 0 {
				return fmt.Errorf("duration must be non-zero")
			}
			return nil
		},
	}

	in := Input{} // D is zero -> should fail validation
	m, err := model.New(&in,
		model.WithRule[Input, time.Duration](r), // register custom rule
		model.WithValidation[Input](),           // run validation during New()
	)
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			fmt.Println("WithValidation+WithRule -> validation error:")
			fmt.Println(ve.Error())
			return
		}
		fmt.Println("unexpected error:", err)
		return
	}
	_ = m
	fmt.Println("unexpected: validation passed")
}

// Example 3: WithRule — register a single rule and validate later
func exampleWithRule() {
	type Doc struct {
		Title string `validate:"nonempty"`
	}
	// You can rely on built-in string rules implicitly via WithValidation, but
	// here we demonstrate a custom single-rule registration for clarity.
	r := rule.Rule[string]{
		Name: "nonempty",
		Fn: func(s string, _ ...string) error {
			if s == "" {
				return fmt.Errorf("must not be empty")
			}
			return nil
		},
	}
	d := Doc{}
	m, err := model.New(&d, model.WithRule[Doc, string](r))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := m.Validate(); err != nil {
		fmt.Println("WithRule ->", err)
		return
	}
	fmt.Println("WithRule -> ok")
}

// Example 4: WithRules — register multiple rules at once
func exampleWithRules() {
	type Rec struct {
		Age int `validate:"positive,nonzero"`
	}
	rules := []rule.Rule[int]{
		{Name: "positive", Fn: func(n int, _ ...string) error {
			if n <= 0 {
				return fmt.Errorf("must be > 0")
			}
			return nil
		}},
		{Name: "nonzero", Fn: func(n int, _ ...string) error {
			if n == 0 {
				return fmt.Errorf("must not be zero")
			}
			return nil
		}},
	}
	r := Rec{Age: 0}
	m, err := model.New(&r,
		model.WithRules[Rec, int](rules), // batch register
		model.WithValidation[Rec](),      // run validation
	)
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			fmt.Println("WithRules ->", ve.Error())
			return
		}
		fmt.Println("unexpected error:", err)
		return
	}
	_ = m
	fmt.Println("WithRules -> ok")
}

func main() {
	exampleWithDefaults()
	exampleWithValidationAndWithRule_error()
	exampleWithRule()
	exampleWithRules()
}
