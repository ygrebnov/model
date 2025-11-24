package validation

import (
	"reflect"
	"testing"
)

func TestParseTag(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []RuleNameParams
	}{
		{
			name: "empty tag -> no rules",
			in:   "",
			want: nil,
		},
		{
			name: "dash tag -> no rules",
			in:   "-",
			want: nil,
		},
		{
			name: "leading and trailing commas are skipped",
			in:   ",email,",
			want: []RuleNameParams{{Name: "email"}},
		},
		{
			name: "whitespace around tokens and Params is trimmed",
			in:   "  foo  ,  bar ( 1 , 2 ) ",
			want: []RuleNameParams{{Name: "foo"}, {Name: "bar", Params: []string{"1", "2"}}},
		},
		{
			name: "nested parentheses do not split top-level tokens",
			in:   "tokA((x,y)),tokB",
			// Note: Params for tokA are split naively: "(x" and "y)" due to simple comma-split.
			want: []RuleNameParams{{Name: "tokA", Params: []string{"(x", "y)"}}, {Name: "tokB"}},
		},
		{
			name: "empty tokens in the middle are skipped",
			in:   "a,,b",
			want: []RuleNameParams{{Name: "a"}, {Name: "b"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseTag(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseTag(%q)\n got: %#v\nwant: %#v", tc.in, got, tc.want)
			}
		})
	}
}
