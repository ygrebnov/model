package model

import (
	"strings"
	"testing"
	"time"
)

type adInner struct {
	S string        `default:"x"`
	N int           `default:"5"`
	D time.Duration `default:"2s"`
}

type adOK struct {
	Inner adInner `default:"dive"`
	Msg   string  `default:"hello"`
}

type adBad struct {
	// unsupported literal on struct field should cause setDefaultsStruct to error
	Inner adInner `default:"oops"`
}

func TestModel_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupModel func() any // returns *Model[...]
		wantErr    string     // substring to expect in error (empty means expect nil)
		verify     func(t *testing.T, m any)
	}{
		{
			name: "nil object -> error",
			setupModel: func() any {
				var m Model[adOK]
				m.obj = nil
				return &m
			},
			wantErr: "nil object",
		},
		{
			name: "non-struct object -> error",
			setupModel: func() any {
				var m Model[int]
				v := new(int)
				*v = 1
				m.obj = v
				return &m
			},
			wantErr: "object must point to a struct",
		},
		{
			name: "success: struct with defaults applied",
			setupModel: func() any {
				var m Model[adOK]
				o := adOK{}
				m.obj = &o
				return &m
			},
			wantErr: "",
			verify: func(t *testing.T, m any) {
				mm := m.(*Model[adOK])
				if mm.obj == nil {
					t.Fatalf("obj is nil")
				}
				if mm.obj.Msg != "hello" {
					t.Fatalf("Msg default not applied: got %q want %q", mm.obj.Msg, "hello")
				}
				if mm.obj.Inner.S != "x" || mm.obj.Inner.N != 5 || mm.obj.Inner.D != 2*time.Second {
					t.Fatalf("Inner defaults not applied: %+v", mm.obj.Inner)
				}
			},
		},
		{
			name: "error: setDefaultsStruct failure is propagated",
			setupModel: func() any {
				var m Model[adBad]
				o := adBad{}
				m.obj = &o
				return &m
			},
			wantErr: "default for Inner",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			switch mm := tt.setupModel().(type) {
			case *Model[adOK]:
				err := mm.applyDefaults()
				if tt.wantErr == "" {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if tt.verify != nil {
						tt.verify(t, mm)
					}
				} else {
					if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
						t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
					}
				}
			case *Model[adBad]:
				err := mm.applyDefaults()
				if tt.wantErr == "" {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
				} else {
					if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
						t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
					}
				}
			case *Model[int]:
				err := mm.applyDefaults()
				if tt.wantErr == "" {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
				} else {
					if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
						t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
					}
				}
			default:
				t.Fatalf("unexpected model type in setup")
			}
		})
	}
}
