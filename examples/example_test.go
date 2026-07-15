package examples

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/validation"
)

// ExampleBinding demonstrates how to use Binding[T] as a reusable
// engine for applying defaults and validation to multiple instances
// of the same type.
func ExampleBinding() {
	// Define the payload type with tags.
	type payload struct {
		ID      string `validate:"uuid"`
		Email   string `validate:"email"`
		Retries int    `validate:"min(0),max(5)"`
	}

	// Construct a reusable binding for payload.
	b, err := model.NewBinding[payload]()
	if err != nil {
		// In examples, we just print the error.
		fmt.Println("binding error:", err.Error())
		return
	}

	// Use the binding on multiple instances.
	p1 := payload{ID: "123e4567-e89b-12d3-a456-426614174000", Email: "user@example.com", Retries: 1}
	p2 := payload{ID: "not-a-uuid", Email: "bad", Retries: 10}

	ctx := context.Background()

	_ = b.ValidateWithDefaults(ctx, &p1) // p1 is valid
	if err := b.ValidateWithDefaults(ctx, &p2); err != nil {
		// In real code you would inspect *ValidationError here.
		fmt.Println("validation error:")
		fmt.Println(err.Error())
	}

	// Output: validation error:
	// - Field "id": rule "uuid": rule constraint violated (length=10)
	// - Field "email": rule "email": rule constraint violated (at_count=1)
	// - Field "retries": rule "max": rule constraint violated (value=5)
}

func ExampleValidateWithDefaults() {
	type Cfg struct {
		Name    string        `default:"svc"`
		Timeout time.Duration `default:"250ms"`
	}

	cfg := Cfg{}
	err := model.ValidateWithDefaults(context.Background(), &cfg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("ValidateWithDefaults -> name=%q timeout=%v", cfg.Name, cfg.Timeout)

	// Output: ValidateWithDefaults -> name="svc" timeout=250ms
}

func ExampleValidateWithDefaults_withRule() {
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
	err = model.ValidateWithDefaults(
		context.Background(),
		&in,
		model.WithRules(nonZeroDurationRule), // register custom rule.
	)
	if err != nil {
		var ve *validation.Error
		if errors.As(err, &ve) {
			fmt.Println("ValidateWithDefaults+WithRule -> validation error:")
			fmt.Println(ve.Error())
			return
		}
		fmt.Println("unexpected error:", err)
		return
	}
	fmt.Println("unexpected: validation passed")

	// Output: ValidateWithDefaults+WithRule -> validation error:
	// - Field "d": rule "nonzeroDur": duration must be non-zero
}

func ExampleValidateWithDefaults_withMultipleRules() {
	type Rec struct {
		Name string `validate:"nonempty"`
		Age  int    `validate:"positive,nonzero"`
	}

	nonempty, err := model.NewRule[string]("nonempty", func(s string, _ ...string) error {
		if s == "" {
			return fmt.Errorf("must not be empty")
		}
		return nil
	})
	if err != nil {
		fmt.Println("error creating nonempty rule:", err)
		return
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
	err = model.ValidateWithDefaults(
		context.Background(),
		&r,
		model.WithRules(nonempty, positive, nonzero), // batch register
	)
	if err != nil {
		var ve *validation.Error
		if errors.As(err, &ve) {
			fmt.Println("WithRules:")
			fmt.Println(ve.Error())
			return
		}
		fmt.Println("unexpected error:", err)
		return
	}
	fmt.Println("WithRules -> ok")

	// Output: WithRules:
	// - Field "name": rule "nonempty": must not be empty
	// - Field "age": rule "positive": must be > 0
	// - Field "age": rule "nonzero": must not be zero
}
