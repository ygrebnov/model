package model

import (
	"reflect"
	"testing"
)

// helper: a no-op ruleFn
func noopRuleFn(_ reflect.Value, _ ...string) error { return nil }

type rrDummy struct{}

func TestModel_registerRuleAdapter(t *testing.T) {
	t.Parallel()

	strType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)

	tests := []struct {
		name          string
		startMapNil   bool
		existingKey   string
		existingCount int
		regName       string
		ad            ruleAdapter
		wantChanged   bool
		wantKey       string
		wantLen       int
		wantTypes     []reflect.Type // expected field types for that key in order
	}{
		{
			name:        "empty name -> no change",
			startMapNil: true,
			regName:     "",
			ad:          ruleAdapter{fieldType: strType, fn: noopRuleFn},
			wantChanged: false,
		},
		{
			name:        "nil fn -> no change",
			startMapNil: true,
			regName:     "rule",
			ad:          ruleAdapter{fieldType: strType, fn: nil},
			wantChanged: false,
		},
		{
			name:        "nil fieldType -> no change",
			startMapNil: true,
			regName:     "rule",
			ad:          ruleAdapter{fieldType: nil, fn: noopRuleFn},
			wantChanged: false,
		},
		{
			name:        "valid input initializes map and inserts",
			startMapNil: true,
			regName:     "ruleA",
			ad:          ruleAdapter{fieldType: strType, fn: noopRuleFn},
			wantChanged: true,
			wantKey:     "ruleA",
			wantLen:     1,
			wantTypes:   []reflect.Type{strType},
		},
		{
			name:          "append to existing key preserves order",
			startMapNil:   false,
			existingKey:   "ruleB",
			existingCount: 1, // pre-seed with one adapter
			regName:       "ruleB",
			ad:            ruleAdapter{fieldType: intType, fn: noopRuleFn},
			wantChanged:   true,
			wantKey:       "ruleB",
			wantLen:       2,
			wantTypes:     []reflect.Type{strType, intType}, // existing string, then appended int
		},
		{
			name:          "insert under new key while map already exists",
			startMapNil:   false,
			existingKey:   "ruleC",
			existingCount: 1,
			regName:       "ruleD",
			ad:            ruleAdapter{fieldType: intType, fn: noopRuleFn},
			wantChanged:   true,
			wantKey:       "ruleD",
			wantLen:       1,
			wantTypes:     []reflect.Type{intType},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Model[rrDummy]{}

			// Optionally pre-seed the validators map
			if !tc.startMapNil {
				m.validators = make(map[string][]ruleAdapter)
				if tc.existingKey != "" && tc.existingCount > 0 {
					// seed with one adapter: fieldType string
					seed := ruleAdapter{fieldType: strType, fn: noopRuleFn}
					m.validators[tc.existingKey] = []ruleAdapter{seed}
				}
			}

			// Snapshot before
			var beforeNil bool
			var beforeLen int
			if m.validators == nil {
				beforeNil = true
			} else if tc.wantKey != "" {
				beforeLen = len(m.validators[tc.wantKey])
			}

			// Call registerRuleAdapter
			m.registerRuleAdapter(tc.regName, tc.ad)

			// Assertions
			if !tc.wantChanged {
				// Must be no change at all
				if tc.startMapNil {
					if m.validators != nil {
						t.Fatalf("validators map should remain nil")
					}
				} else {
					// If map existed, its size and contents under existingKey should be unchanged
					if m.validators == nil {
						t.Fatalf("validators map should not become nil")
					}
					if tc.existingKey != "" {
						got := m.validators[tc.existingKey]
						if len(got) != tc.existingCount {
							t.Fatalf("expected existing count %d, got %d", tc.existingCount, len(got))
						}
						// sanity: type remains the seeded strType
						if len(got) > 0 && got[0].fieldType != strType {
							t.Fatalf("seeded fieldType changed")
						}
					} else if beforeNil {
						// no existingKey but map existed means no-op is still fine
						_ = beforeLen
					}
				}
				return
			}

			// Changed cases
			if m.validators == nil {
				t.Fatalf("validators map must be initialized")
			}
			adapters := m.validators[tc.wantKey]
			if len(adapters) != tc.wantLen {
				t.Fatalf("expected %d adapter(s) under key %q, got %d", tc.wantLen, tc.wantKey, len(adapters))
			}
			// Check order of types under the key
			for i, wantT := range tc.wantTypes {
				if adapters[i].fieldType != wantT {
					t.Fatalf("at index %d, want fieldType %v, got %v", i, wantT, adapters[i].fieldType)
				}
				if adapters[i].fn == nil {
					t.Fatalf("adapter fn must be non-nil at index %d", i)
				}
			}
		})
	}
}
