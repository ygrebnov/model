package model

import (
	"reflect"
	"testing"
)

func TestParseRules_Edges(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []parsedRule
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
			in:   ",nonempty,",
			want: []parsedRule{{name: "nonempty"}},
		},
		{
			name: "whitespace around tokens and params is trimmed",
			in:   "  foo  ,  bar ( 1 , 2 ) ",
			want: []parsedRule{{name: "foo"}, {name: "bar", params: []string{"1", "2"}}},
		},
		{
			name: "nested parentheses do not split top-level tokens",
			in:   "tokA((x,y)),tokB",
			// Note: params for tokA are split naively: "(x" and "y)" due to simple comma-split.
			want: []parsedRule{{name: "tokA", params: []string{"(x", "y)"}}, {name: "tokB"}},
		},
		{
			name: "empty tokens in the middle are skipped",
			in:   "a,,b",
			want: []parsedRule{{name: "a"}, {name: "b"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRules(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseRules(%q)\n got: %#v\nwant: %#v", tc.in, got, tc.want)
			}
		})
	}
}
