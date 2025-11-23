package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ygrebnov/model/validation"
)

func ExampleNew_withDefaults() {
	type Cfg struct {
		Name    string        `default:"svc"`
		Timeout time.Duration `default:"250ms"`
	}

	cfg := Cfg{}
	m, err := New(&cfg, WithDefaults[Cfg]())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	_ = m
	fmt.Printf("WithDefaults -> name=%q timeout=%v", cfg.Name, cfg.Timeout)

	// Output: WithDefaults -> name="svc" timeout=250ms
}

func ExampleNew_withWithRuleAndValidation() {
	type Input struct {
		// Use a custom rule name for a non-builtin type (time.Duration)
		D time.Duration `validate:"nonzeroDur"`
	}

	in := Input{} // D is zero -> should fail validation
	nonZeroDurationRule, err := validation.NewRule[time.Duration]("nonzeroDur",
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
	m, err := New(
		&in,
		WithRules[Input](nonZeroDurationRule), // register custom rule.
		WithValidation[Input](context.Background()), // run validation in New().
	)
	if err != nil {
		var ve *validation.Error
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

	// Output: WithValidation+WithRule -> validation error:
	// - Field "D": rule "nonzeroDur": duration must be non-zero
}

func ExampleNew_withRuleAndLaterValidation() {
	type Doc struct {
		Title string `validate:"nonempty"`
	}
	// You can rely on built-in string rules implicitly via WithValidation, but
	// here we demonstrate a custom single-rule registration for clarity.
	// We will register a "nonempty" rule for strings, but will run validation manually later.
	// Note: if you register a rule with the same name as a built-in rule, your custom rule
	// will override the built-in one.
	d := Doc{}
	nonEmptyRule, err := validation.NewRule[string]("nonempty",
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
	m, err := New(&d, WithRules[Doc](nonEmptyRule))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if err = m.Validate(context.Background()); err != nil {
		fmt.Println("WithRule: validation error:")
		fmt.Println(err)
		return
	}
	fmt.Println("WithRule -> ok")

	// Output: WithRule: validation error:
	// - Field "Title": rule "nonempty": must not be empty
}

func ExampleNew_withMultipleRules() {
	type Rec struct {
		Age int `validate:"positive,nonzero"`
	}

	positive, err := validation.NewRule[int]("positive", func(n int, _ ...string) error {
		if n <= 0 {
			return fmt.Errorf("must be > 0")
		}
		return nil
	})
	if err != nil {
		fmt.Println("error creating positive rule:", err)
		return
	}
	nonzero, err := validation.NewRule[int]("nonzero", func(n int, _ ...string) error {
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
	m, err := New(&r,
		WithRules[Rec](positive, nonzero),         // batch register
		WithValidation[Rec](context.Background()), // run validation
	)
	if err != nil {
		var ve *validation.Error
		if errors.As(err, &ve) {
			fmt.Println("WithRules:", ve.Error())
			return
		}
		fmt.Println("unexpected error:", err)
		return
	}
	_ = m
	fmt.Println("WithRules -> ok")

	// Output: WithRules: validation failed:
	//   - Field "Age": rule "positive": must be > 0
	//   - Field "Age": rule "nonzero": must not be zero
}
