package types

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	_ yaml.Unmarshaler = (*Duration)(nil)
	_ yaml.Marshaler   = Duration(0)
)

func TestDuration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "string seconds",
			input: `"5s"`,
			want:  5 * time.Second,
		},
		{
			name:  "string milliseconds",
			input: `"250ms"`,
			want:  250 * time.Millisecond,
		},
		{
			name:  "string compound duration",
			input: `"1h2m3s"`,
			want:  time.Hour + 2*time.Minute + 3*time.Second,
		},
		{
			name:  "numeric nanoseconds",
			input: `5000000000`,
			want:  5 * time.Second,
		},
		{
			name:  "zero string",
			input: `"0s"`,
			want:  0,
		},
		{
			name:  "zero numeric",
			input: `0`,
			want:  0,
		},
		{
			name:  "negative string",
			input: `"-5s"`,
			want:  -5 * time.Second,
		},
		{
			name:  "negative numeric",
			input: `-5000000000`,
			want:  -5 * time.Second,
		},
		{
			name:    "invalid string",
			input:   `"invalid"`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   `""`,
			wantErr: true,
		},
		{
			name:    "boolean",
			input:   `true`,
			wantErr: true,
		},
		{
			name:    "object",
			input:   `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Duration
			err := json.Unmarshal([]byte(tt.input), &got)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if time.Duration(got) != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, time.Duration(got))
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value Duration
		want  string
	}{
		{
			name:  "seconds",
			value: Duration(5 * time.Second),
			want:  `"5s"`,
		},
		{
			name:  "milliseconds",
			value: Duration(250 * time.Millisecond),
			want:  `"250ms"`,
		},
		{
			name:  "compound duration",
			value: Duration(time.Hour + 2*time.Minute + 3*time.Second),
			want:  `"1h2m3s"`,
		},
		{
			name:  "zero",
			value: Duration(0),
			want:  `"0s"`,
		},
		{
			name:  "negative",
			value: Duration(-5 * time.Second),
			want:  `"-5s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, string(got))
			}
		})
	}
}

func TestDuration_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := Duration(time.Hour + 30*time.Minute + 15*time.Second)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Duration
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got != original {
		t.Fatalf("expected %v, got %v", time.Duration(original), time.Duration(got))
	}
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "string seconds",
			input: "5s\n",
			want:  5 * time.Second,
		},
		{
			name:  "string milliseconds",
			input: "250ms\n",
			want:  250 * time.Millisecond,
		},
		{
			name:  "string compound duration",
			input: "1h2m3s\n",
			want:  time.Hour + 2*time.Minute + 3*time.Second,
		},
		{
			name:  "numeric nanoseconds",
			input: "5000000000\n",
			want:  5 * time.Second,
		},
		{
			name:  "zero string",
			input: "0s\n",
			want:  0,
		},
		{
			name:  "zero numeric",
			input: "0\n",
			want:  0,
		},
		{
			name:  "negative string",
			input: "-5s\n",
			want:  -5 * time.Second,
		},
		{
			name:  "negative numeric",
			input: "-5000000000\n",
			want:  -5 * time.Second,
		},
		{
			name:    "invalid string",
			input:   "invalid\n",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "''\n",
			wantErr: true,
		},
		{
			name:    "boolean",
			input:   "true\n",
			wantErr: true,
		},
		{
			name:    "float",
			input:   "1.5\n",
			wantErr: true,
		},
		{
			name:    "object",
			input:   "{}\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Duration
			err := yaml.Unmarshal([]byte(tt.input), &got)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if time.Duration(got) != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, time.Duration(got))
			}
		})
	}
}

func TestDuration_MarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value Duration
		want  string
	}{
		{
			name:  "seconds",
			value: Duration(5 * time.Second),
			want:  "5s\n",
		},
		{
			name:  "milliseconds",
			value: Duration(250 * time.Millisecond),
			want:  "250ms\n",
		},
		{
			name:  "compound duration",
			value: Duration(time.Hour + 2*time.Minute + 3*time.Second),
			want:  "1h2m3s\n",
		},
		{
			name:  "zero",
			value: Duration(0),
			want:  "0s\n",
		},
		{
			name:  "negative",
			value: Duration(-5 * time.Second),
			want:  "-5s\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := yaml.Marshal(tt.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, string(got))
			}
		})
	}
}

func TestDuration_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	original := Duration(time.Hour + 30*time.Minute + 15*time.Second)

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Duration
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got != original {
		t.Fatalf("expected %v, got %v", time.Duration(original), time.Duration(got))
	}
}

func TestDuration_Duration(t *testing.T) {
	t.Parallel()

	d := Duration(5 * time.Second)

	if got := d.Duration(); got != 5*time.Second {
		t.Fatalf("expected %v, got %v", 5*time.Second, got)
	}
}
