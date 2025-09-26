package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/ygrebnov/model"
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
	fmt.Printf("WithDefaults -> name=%q Timeout=%v\n", cfg.Name, cfg.Timeout)
}

// Example 2: WithValidation + WithRule — register a custom rule and fail validation
func exampleWithValidationAndWithRule_error() {
	type Input struct {
		// Use a custom rule name for a non-builtin type (time.Duration)
		D time.Duration `validate:"nonzeroDur"`
	}

	in := Input{} // D is zero -> should fail validation
	nonZeroDurationRule, err := model.NewRule[time.Duration]("nonzeroDur",
		func(d time.Duration, _ ...string) error {
			if d == 0 {
				return fmt.Errorf("duration must be non-zero")
			}
			return nil
		},
	)
	if err != nil {
		fmt.Println("error creating rule:", err)
		return
	}
	m, err := model.New(
		&in, model.WithRules[Input](nonZeroDurationRule), // register custom rule.
		model.WithValidation[Input](), // run validation in New().
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
	// We will register a "nonempty" rule for strings, but will run validation manually later.
	// Note: if you register a rule with the same name as a built-in rule, your custom rule
	// will override the built-in one.
	d := Doc{}
	nonEmptyRule, err := model.NewRule[string]("nonempty",
		func(s string, _ ...string) error {
			if s == "" {
				return fmt.Errorf("must not be empty")
			}
			return nil
		},
	)
	if err != nil {
		fmt.Println("error creating rule:", err)
		return
	}
	// Register the rule via WithRule (single rule) instead of WithRules (batch).
	// Note: WithRule does NOT imply WithValidation, so validation is NOT run automatically during New().
	m, err := model.New(&d, model.WithRules[Doc](nonEmptyRule))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if err = m.Validate(); err != nil {
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

	positive, err := model.NewRule[int]("positive", func(n int, _ ...string) error {
		if n <= 0 {
			return fmt.Errorf("must be > 0")
		}
		return nil
	})
	if err != nil {
		fmt.Println("error creating positive rule:", err)
		return
	}
	nonzero, err := model.NewRule[int]("nonzero", func(n int, _ ...string) error {
		if n == 0 {
			return fmt.Errorf("must not be zero")
		}
		return nil
	})
	if err != nil {
		fmt.Println("error creating nonzero rule:", err)
		return
	}

	r := Rec{Age: 0}
	m, err := model.New(&r,
		model.WithRules[Rec](positive, nonzero), // batch register
		model.WithValidation[Rec](),             // run validation
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
