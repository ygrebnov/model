package errors

import (
	stderrors "errors"
	"testing"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/keys"
)

func TestSummary(t *testing.T) {
	t.Parallel()

	customErr := stderrors.New("custom failure")

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil",
			err:  nil,
			want: "",
		},
		{
			name: "rule constraint violated",
			err:  ErrRuleConstraintViolated,
			want: "constraint violated",
		},
		{
			name: "wrapped rule constraint violated",
			err: errorc.With(
				ErrRuleConstraintViolated,
				errorc.String(keys.RuleName, "min"),
			),
			want: "constraint violated",
		},
		{
			name: "rule invalid parameter",
			err:  ErrRuleInvalidParameter,
			want: "invalid rule parameter",
		},
		{
			name: "rule missing parameter",
			err:  ErrRuleMissingParameter,
			want: "missing rule parameter",
		},
		{
			name: "rule not found",
			err:  ErrRuleNotFound,
			want: "rule not found",
		},
		{
			name: "rule overload not found",
			err:  ErrRuleOverloadNotFound,
			want: "rule is not applicable to this field type",
		},
		{
			name: "invalid value",
			err:  ErrInvalidValue,
			want: "invalid value",
		},
		{
			name: "fallback custom error",
			err:  customErr,
			want: customErr.Error(),
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := Summary(tc.err); got != tc.want {
				t.Fatalf("Summary(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
