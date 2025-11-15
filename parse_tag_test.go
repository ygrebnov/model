package model

import (
	"reflect"
	"testing"
)

func TestParseTag(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []ruleNameParams
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
			want: []ruleNameParams{{name: "email"}},
		},
		{
			name: "whitespace around tokens and params is trimmed",
			in:   "  foo  ,  bar ( 1 , 2 ) ",
			want: []ruleNameParams{{name: "foo"}, {name: "bar", params: []string{"1", "2"}}},
		},
		{
			name: "nested parentheses do not split top-level tokens",
			in:   "tokA((x,y)),tokB",
			// Note: params for tokA are split naively: "(x" and "y)" due to simple comma-split.
			want: []ruleNameParams{{name: "tokA", params: []string{"(x", "y)"}}, {name: "tokB"}},
		},
		{
			name: "empty tokens in the middle are skipped",
			in:   "a,,b",
			want: []ruleNameParams{{name: "a"}, {name: "b"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTag(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseTag(%q)\n got: %#v\nwant: %#v", tc.in, got, tc.want)
			}
		})
	}
}
