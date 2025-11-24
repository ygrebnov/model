package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	modelErrors "github.com/ygrebnov/model/errors"
	"github.com/ygrebnov/model/validation"
)

// fakeService is a lightweight test double for the internal service interface.
type fakeService struct {
	setDefaultsErr error
	validateErr    error
	rulesErr       error

	lastDefaultsValue reflect.Value
	lastValidateValue reflect.Value
	validateCtx       context.Context
}

func (f *fakeService) SetDefaultsStruct(v reflect.Value) error {
	f.lastDefaultsValue = v
	return f.setDefaultsErr
}

func (f *fakeService) AddRule(r validation.Rule) error {
	return f.rulesErr
}

func (f *fakeService) ValidateStruct(ctx context.Context, v reflect.Value, _ string, _ *validation.Error) error {
	f.validateCtx = ctx
	f.lastValidateValue = v
	return f.validateErr
}

// TestNewBinding covers constructor behavior for valid and invalid type parameters.
func TestNewBinding(t *testing.T) {
	t.Run("struct type", func(t *testing.T) {
		type sample struct{ A int }

		b, err := NewBinding[sample]()
		if err != nil {
			t.Fatalf("NewBinding[sample] unexpected error: %v", err)
		}
		if b == nil {
			t.Fatalf("NewBinding[sample] returned nil binding")
		}
	})

	t.Run("non-struct type", func(t *testing.T) {
		b, err := NewBinding[int]()
		if !errors.Is(err, modelErrors.ErrNotStructPtr) {
			t.Fatalf("NewBinding[int] expected ErrNotStructPtr, got: %v", err)
		}
		if b != nil {
			t.Fatalf("NewBinding[int] expected nil binding on error")
		}
	})
}

func TestBinding_ApplyDefaults(t *testing.T) {
	type sample struct{ A int }

	base := &fakeService{}
	b := &Binding[sample]{service: base}

	zero := &sample{}
	nilPtr := (*sample)(nil)

	tests := []struct {
		name    string
		obj     *sample
		wantErr error
	}{
		{"nil object", nilPtr, modelErrors.ErrNilObject},
		{"ok", zero, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			base.lastDefaultsValue = reflect.Value{}

			err := b.ApplyDefaults(tt.obj)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if err == nil && !base.lastDefaultsValue.IsValid() {
				t.Fatalf("expected SetDefaultsStruct to be called")
			}
		})
	}
}

func TestBinding_Validate(t *testing.T) {
	type sample struct{ A int }

	base := &fakeService{}
	b := &Binding[sample]{service: base}

	zero := &sample{}
	nilPtr := (*sample)(nil)

	tests := []struct {
		name       string
		ctx        context.Context
		obj        any
		serviceErr error
		wantErr    error
	}{
		{"nil object", context.Background(), nilPtr, nil, modelErrors.ErrNilObject},
		{"nil context", nil, zero, nil, nil},
		{"service error", context.Background(), zero, errors.New("svc"), errors.New("svc")},
		{"ok", context.Background(), zero, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base.validateErr = tt.serviceErr
			base.lastValidateValue = reflect.Value{}

			obj, _ := tt.obj.(*sample)
			err := b.Validate(tt.ctx, obj)

			if tt.serviceErr != nil {
				if !errors.Is(err, tt.serviceErr) {
					t.Fatalf("expected service error %v, got %v", tt.serviceErr, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if err == nil && !base.lastValidateValue.IsValid() {
				t.Fatalf("expected ValidateStruct to be called")
			}
			if tt.ctx == nil && base.validateCtx == nil {
				t.Fatalf("expected context to be defaulted when nil")
			}
		})
	}
}

func TestBinding_ValidateWithDefaults(t *testing.T) {
	type sample struct{ A int }

	base := &fakeService{}
	b := &Binding[sample]{service: base}

	obj := &sample{}

	t.Run("defaults error", func(t *testing.T) {
		base.setDefaultsErr = errors.New("defaults")
		base.validateErr = nil

		if err := b.ValidateWithDefaults(context.Background(), obj); !errors.Is(err, base.setDefaultsErr) {
			t.Fatalf("expected defaults error %v, got %v", base.setDefaultsErr, err)
		}
	})

	t.Run("validate error", func(t *testing.T) {
		base.setDefaultsErr = nil
		base.validateErr = errors.New("validate")

		if err := b.ValidateWithDefaults(context.Background(), obj); !errors.Is(err, base.validateErr) {
			t.Fatalf("expected validate error %v, got %v", base.validateErr, err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		base.setDefaultsErr = nil
		base.validateErr = nil

		if err := b.ValidateWithDefaults(context.Background(), obj); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestBinding_RegisterRules(t *testing.T) {
	type sample struct{ A int }

	base := &fakeService{}
	b := &Binding[sample]{service: base}

	// Create a dummy rule; we don't care about its type matching here, just AddRule errors.
	rule, err := validation.NewRule[int]("dummy", func(value int, params ...string) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error creating rule: %v", err)
	}

	t.Run("service AddRule error", func(t *testing.T) {
		base.rulesErr = errors.New("add")
		if err := b.RegisterRules(rule); !errors.Is(err, base.rulesErr) {
			t.Fatalf("expected rules error %v, got %v", base.rulesErr, err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		base.rulesErr = nil
		if err := b.RegisterRules(rule); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestBindingAndModel_Consistency(t *testing.T) {
	type Sample struct {
		// Default/validation tags mirror existing examples in the repo.
		Name  string `default:"john" validate:"required"`
		Age   int    `default:"18" validate:"min(18)"`
		Email string `validate:"email"`
	}

	// Define simple custom rules used by both Binding and Model.
	requiredString, err := validation.NewRule[string]("required", func(v string, _ ...string) error {
		if v == "" {
			return fmt.Errorf("required")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create required rule: %v", err)
	}

	minInt, err := validation.NewRule[int]("min", func(v int, params ...string) error {
		if len(params) == 0 {
			return fmt.Errorf("min: missing param")
		}
		th, err := strconv.Atoi(params[0])
		if err != nil {
			return fmt.Errorf("min: %w", err)
		}
		if v < th {
			return fmt.Errorf("min: %d < %d", v, th)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create min rule: %v", err)
	}

	emailRule, err := validation.NewRule[string]("email", func(v string, _ ...string) error {
		if v == "" {
			return fmt.Errorf("email required")
		}
		// keep it simple: just check contains '@'
		for _, ch := range v {
			if ch == '@' {
				return nil
			}
		}
		return fmt.Errorf("invalid email")
	})
	if err != nil {
		t.Fatalf("failed to create email rule: %v", err)
	}

	tests := []struct {
		name     string
		setup    func(*Sample)
		wantErr  bool
		wantName string
		wantAge  int
	}{
		{
			name: "valid after defaults",
			setup: func(s *Sample) {
				// leave zero values so defaults can populate Name and Age
				s.Email = "user@example.com"
			},
			wantErr:  false,
			wantName: "john",
			wantAge:  18,
		},
		{
			name: "invalid even after defaults",
			setup: func(s *Sample) {
				// No email -> email rule should fail
			},
			wantErr:  true,
			wantName: "john",
			wantAge:  18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Binding path
			b, err := NewBinding[Sample]()
			if err != nil {
				t.Fatalf("NewBinding[Sample] error: %v", err)
			}
			if err := b.RegisterRules(requiredString, minInt, emailRule); err != nil {
				t.Fatalf("Binding.RegisterRules error: %v", err)
			}
			bVal := &Sample{}
			if tt.setup != nil {
				setup := tt.setup
				setup(bVal)
			}

			// Model path
			mVal := &Sample{}
			if tt.setup != nil {
				setup := tt.setup
				setup(mVal)
			}
			m, err := New(mVal, WithDefaults[Sample]())
			if err != nil {
				t.Fatalf("New(Model) error: %v", err)
			}
			if err := m.RegisterRules(requiredString, minInt, emailRule); err != nil {
				t.Fatalf("Model.RegisterRules error: %v", err)
			}

			// Run Binding: defaults + validate
			if err := b.ValidateWithDefaults(context.Background(), bVal); (err != nil) != tt.wantErr {
				t.Fatalf("Binding.ValidateWithDefaults error = %v, wantErr=%v", err, tt.wantErr)
			}

			// Run Model: defaults already applied by WithDefaults; now validate
			if err := m.Validate(context.Background()); (err != nil) != tt.wantErr {
				t.Fatalf("Model.Validate error = %v, wantErr=%v", err, tt.wantErr)
			}

			// Compare resulting values
			if bVal.Name != tt.wantName || mVal.Name != tt.wantName {
				t.Fatalf("Name after defaults: binding=%q model=%q, want=%q", bVal.Name, mVal.Name, tt.wantName)
			}
			if bVal.Age != tt.wantAge || mVal.Age != tt.wantAge {
				t.Fatalf("Age after defaults: binding=%d model=%d, want=%d", bVal.Age, mVal.Age, tt.wantAge)
			}
		})
	}
}
