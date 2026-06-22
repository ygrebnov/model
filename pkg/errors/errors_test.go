package errors

import (
	stdErrors "errors"
	"fmt"
	"testing"

	"github.com/ygrebnov/errorc"
)

func TestGetBase(t *testing.T) {
	baseErr := stdErrors.New("base")
	otherErr := stdErrors.New("other")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "nil error",
			err:  nil,
			want: nil,
		},
		{
			name: "plain error",
			err:  baseErr,
			want: baseErr,
		},
		{
			name: "single wrapped error",
			err:  fmt.Errorf("wrap: %w", baseErr),
			want: baseErr,
		},
		{
			name: "multiple wrapped error",
			err:  fmt.Errorf("level 2: %w", fmt.Errorf("level 1: %w", baseErr)),
			want: baseErr,
		},
		{
			name: "wrapped sentinel error",
			err:  fmt.Errorf("constructor failed: %w", ErrNilObject),
			want: ErrNilObject,
		},
		{
			name: "different base error",
			err:  fmt.Errorf("wrap: %w", otherErr),
			want: otherErr,
		},
		{

			name: "errorc field wrapper returns sentinel base",
			err:  errorc.With(ErrRuleConstraintViolated, errorc.String("field", "value")),
			want: ErrRuleConstraintViolated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetBase(tc.err)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
