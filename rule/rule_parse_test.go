package rule

import (
	"reflect"
	"testing"
)

func TestParseRules_Edges(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Metadata
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
			want: []Metadata{{Name: "nonempty"}},
		},
		{
			name: "whitespace around tokens and ParamNames is trimmed",
			in:   "  foo  ,  bar ( 1 , 2 ) ",
			want: []Metadata{{Name: "foo"}, {Name: "bar", ParamNames: []string{"1", "2"}}},
		},
		{
			name: "nested parentheses do not split top-level tokens",
			in:   "tokA((x,y)),tokB",
			// Note: ParamNames for tokA are split naively: "(x" and "y)" due to simple comma-split.
			want: []Metadata{{Name: "tokA", ParamNames: []string{"(x", "y)"}}, {Name: "tokB"}},
		},
		{
			name: "empty tokens in the middle are skipped",
			in:   "a,,b",
			want: []Metadata{{Name: "a"}, {Name: "b"}},
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
