package model

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ----- Test types -----

// Implements fmt.Stringer to test AssignableTo(interface) path via rules registered for an interface type.
type strWrap struct{ v string }

func (s strWrap) String() string { return s.v }

// Inner struct with own validate tags, used for dive recursion tests.
type vInner struct {
	S string        `validate:"nonempty"`
	D time.Duration `validate:"nonzeroDuration"`
}

// Struct under test with a variety of fields/tags.
type vOuter struct {
	// Recursion targets
	In  vInner  `validate:""`         // explicit empty (no rule), but should dive due to struct recursion
	PIn *vInner `validate:""`         // pointer; nil → no validation; non-nil → dive
	pin *vInner `validate:"nonempty"` // unexported: must be skipped despite tag

	// Simple rules
	Name string `validate:"nonempty"`
	// Params test (commas inside parens; also followed by another rule)
	Note string `validate:"withParams(a, b, c),nonempty"`

	// Unknown rule
	Alias string `validate:"doesNotExist"`

	// Ambiguity: same name registered twice for exact type
	Amb string `validate:"dup"`

	// Assignable: rule registered for fmt.Stringer, field is concrete strWrap
	Wrapped strWrap `validate:"stringerBad"`

	// Element validation: slices/arrays
	Tags    []string   `validateElem:"nonempty"`
	People  []vInner   `validateElem:"dive"`
	Numbers []int      `validateElem:"dive"` // misuse of dive on non-struct → error per element
	Ptrs    []*vInner  `validateElem:"dive"` // nil/non-nil pointer elements
	Fixed   [2]vInner  `validateElem:"dive"`
	FixedP  [2]*vInner `validateElem:"dive"`

	// Element validation: maps (values validated)
	Labels    map[string]string  `validateElem:"nonempty"`
	Profiles  map[string]vInner  `validateElem:"dive"`
	ProfilesP map[string]*vInner `validateElem:"dive"`

	// Non-container with validateElem (should be ignored)
	Ghost string `validateElem:"nonempty"`
}

// ----- Helpers to register rules for tests -----

// nonempty for string
func ruleNonEmpty(s string, _ ...string) error {
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	return nil
}

// withParams echoes params to prove parsing worked
func ruleWithParams(s string, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("expected params")
	}
	// Return an error that includes params for assertion
	return fmt.Errorf("params=%s", strings.Join(params, "|"))
}

// nonzeroDuration (time.Duration or int64 underlying)
func ruleNonzeroDuration(d time.Duration, _ ...string) error {
	if d == 0 {
		return fmt.Errorf("duration must be non-zero")
	}
	return nil
}

// int rule that always errors (to populate FieldError)
func ruleIntAlwaysErr(_ int, _ ...string) error {
	return fmt.Errorf("bad int")
}

// Rule for fmt.Stringer (AssignableTo interface)
func ruleStringerBad(_ fmt.Stringer, _ ...string) error {
	return fmt.Errorf("bad stringer")
}

func TestModel_validateStruct(t *testing.T) {
	// Build a model and register rules needed across subtests.
	m := &Model[vOuter]{validators: make(map[string][]typedAdapter)}

	// Register string rules
	WithRule[vOuter, string](Rule[string]{Name: "nonempty", Fn: ruleNonEmpty})(m)
	WithRule[vOuter, string](Rule[string]{Name: "withParams", Fn: ruleWithParams})(m)

	// Register time.Duration rule
	WithRule[vOuter, time.Duration](Rule[time.Duration]{Name: "nonzeroDuration", Fn: ruleNonzeroDuration})(m)

	// Register interface-based rule (AssignableTo path)
	WithRule[vOuter, fmt.Stringer](Rule[fmt.Stringer]{Name: "stringerBad", Fn: ruleStringerBad})(m)

	// Register ambiguous rule (same name & type twice) → exact duplicates trigger ambiguity
	WithRule[vOuter, string](Rule[string]{Name: "dup", Fn: ruleNonEmpty})(m)
	WithRule[vOuter, string](Rule[string]{Name: "dup", Fn: ruleNonEmpty})(m)

	// Also a rule for int to demonstrate element rules on int slices if needed
	WithRule[vOuter, int](Rule[int]{Name: "intErr", Fn: ruleIntAlwaysErr})(m)

	t.Run("recursion, params parsing, unknown rule, ambiguity, assignable, and validateElem on slices/maps", func(t *testing.T) {
		obj := vOuter{
			// Recursion: In has validate tags; PIn initially nil (no dive); pin unexported ignored
			In: vInner{
				S: "", // nonempty → error
				D: 0,  // nonzeroDuration → error
			},
			PIn: nil,
			// Simple rules
			Name:    "",               // nonempty → error
			Note:    "",               // withParams(...) → error with params; then nonempty → error
			Alias:   "",               // unknown rule → error from applyRule
			Amb:     "",               // dup rule → ambiguity error
			Wrapped: strWrap{v: "ok"}, // ruleStringerBad → error (we're asserting AssignableTo used)
			// validateElem on slices/arrays
			Tags:    []string{"", "ok", ""},                   // nonempty applied per element
			People:  []vInner{{S: "", D: 0}, {S: "ok", D: 0}}, // dive into elements: apply inner rules
			Numbers: []int{1, 0},                              // misuse of dive (non-struct) → error per element
			Ptrs:    []*vInner{nil, {}},                       // nil stays; non-nil empty struct gets dived
			Fixed:   [2]vInner{{}, {S: "ok", D: 0}},           // array handled like slice
			FixedP:  [2]*vInner{nil, {}},
			// validateElem on maps (values)
			Labels: map[string]string{
				"a": "",
				"b": "ok",
			},
			Profiles: map[string]vInner{
				"k1": {S: "", D: 0},
				"k2": {S: "ok", D: 0},
			},
			ProfilesP: map[string]*vInner{
				"p1": {},
				"p2": nil,
			},
			// Ghost should not produce errors (validateElem ignored on non-container)
			Ghost: "",
		}

		// fix composite literals for pointers (cannot use {} directly)
		obj.Ptrs[1] = &vInner{}
		obj.FixedP[1] = &vInner{}

		ve := &ValidationError{}
		rv := reflect.ValueOf(&obj).Elem()
		m.validateStruct(rv, "Root", ve) // use non-empty path prefix to test dotted paths

		if ve.Empty() {
			t.Fatalf("expected validation errors; got none")
		}

		// Collect messages by field path for assertions
		by := ve.ByField()

		// Recursion into In
		if _, ok := by["Root.In.S"]; !ok {
			t.Errorf("expected error at Root.In.S (nonempty)")
		}
		if _, ok := by["Root.In.D"]; !ok {
			t.Errorf("expected error at Root.In.D (nonzeroDuration)")
		}

		// PIn nil → no entries under Root.PIn.*
		for p := range by {
			if strings.HasPrefix(p, "Root.PIn.") {
				t.Errorf("did not expect errors under Root.PIn.*, got %s", p)
			}
		}

		// Unexported pin ignored
		for p := range by {
			if strings.Contains(p, ".pin") {
				t.Errorf("unexported field pin should be skipped; saw %s", p)
			}
		}

		// Simple rules
		if _, ok := by["Root.Name"]; !ok {
			t.Errorf("expected nonempty error at Root.Name")
		}
		// Params parsing (withParams and nonempty applied)
		paramsMsgs := by["Root.Note"]
		if len(paramsMsgs) == 0 {
			t.Errorf("expected errors for Root.Note")
		} else {
			// Ensure the params were captured
			foundParams := false
			foundNonEmpty := false
			for _, fe := range paramsMsgs {
				if strings.Contains(fe.Err.Error(), "params=a|b|c") {
					foundParams = true
				}
				if strings.Contains(fe.Err.Error(), "must not be empty") {
					foundNonEmpty = true
				}
			}
			if !foundParams {
				t.Errorf("expected withParams error containing params=a|b|c")
			}
			if !foundNonEmpty {
				t.Errorf("expected nonempty error for Root.Note")
			}
		}

		// Unknown rule applied
		if es := by["Root.Alias"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "is not registered") {
			t.Errorf("expected unknown rule error at Root.Alias, got: %+v", es)
		}

		// Ambiguity on dup
		if es := by["Root.Amb"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "ambiguous") {
			t.Errorf("expected ambiguity error at Root.Amb, got: %+v", es)
		}

		// Assignable interface rule
		if es := by["Root.Wrapped"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "bad stringer") {
			t.Errorf("expected stringerBad error at Root.Wrapped, got: %+v", es)
		}

		// validateElem on slice of strings
		if es := by["Root.Tags[0]"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Root.Tags[0]")
		}
		if _, ok := by["Root.Tags[1]"]; ok {
			t.Errorf("did not expect error at Root.Tags[1] (was 'ok')")
		}
		if es := by["Root.Tags[2]"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Root.Tags[2]")
		}

		// validateElem dive on People slice (struct elements)
		if es := by["Root.People[0].S"]; len(es) == 0 {
			t.Errorf("expected error at Root.People[0].S (nonempty)")
		}
		if es := by["Root.People[0].D"]; len(es) == 0 {
			t.Errorf("expected error at Root.People[0].D (nonzeroDuration)")
		}
		// second element has S ok, D zero → expect only D
		if _, ok := by["Root.People[1].S"]; ok {
			t.Errorf("did not expect error at Root.People[1].S")
		}
		if es := by["Root.People[1].D"]; len(es) == 0 {
			t.Errorf("expected error at Root.People[1].D")
		}

		// misuse of dive on non-struct element slice (Numbers): error per element
		if es := by["Root.Numbers[0]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Root.Numbers[0], got: %+v", es)
		}
		if es := by["Root.Numbers[1]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Root.Numbers[1], got: %+v", es)
		}

		// Ptrs: nil pointer element produces a 'dive' misuse error (kind ptr), non-nil element gets dived
		if es := by["Root.Ptrs[0]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Root.Ptrs[0], got: %+v", es)
		}
		if es := by["Root.Ptrs[1].S"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Root.Ptrs[1].S")
		}

		// Arrays behave like slices
		if es := by["Root.Fixed[0].S"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Root.Fixed[0].S")
		}
		if es := by["Root.Fixed[1].D"]; len(es) == 0 {
			t.Errorf("expected nonzeroDuration error at Root.Fixed[1].D")
		}
		// FixedP pointer array: index 0 nil -> 'dive' misuse; index 1 non-nil -> dive into struct
		if es := by["Root.FixedP[0]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Root.FixedP[0], got: %+v", es)
		}
		if es := by["Root.FixedP[1].S"]; len(es) == 0 {
			t.Errorf("expected error at Root.FixedP[1].S")
		}

		// Maps
		// Labels map[string]string with nonempty element rule
		if es := by[`Root.Labels[a]`]; len(es) == 0 {
			t.Errorf("expected nonempty error at Root.Labels[a]")
		}
		if _, ok := by[`Root.Labels[b]`]; ok {
			t.Errorf("did not expect error at Root.Labels[b]")
		}
		// Profiles map[string]vInner with dive
		if es := by[`Root.Profiles[k1].S`]; len(es) == 0 {
			t.Errorf("expected error at Root.Profiles[k1].S")
		}
		if es := by[`Root.Profiles[k2].D`]; len(es) == 0 {
			t.Errorf("expected error at Root.Profiles[k2].D")
		}
		// ProfilesP map[string]*vInner with dive
		if es := by[`Root.ProfilesP[p1].S`]; len(es) == 0 {
			t.Errorf("expected error at Root.ProfilesP[p1].S")
		}
		if es := by[`Root.ProfilesP[p2]`]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Root.ProfilesP[p2], got: %+v", es)
		}

		// validateElem ignored on non-container (Ghost)
		if _, ok := by["Root.Ghost"]; ok {
			t.Errorf("did not expect any error at Root.Ghost due to validateElem tag (non-container)")
		}
	})

	// Covers: pointer-to-struct recursion branch (fv.Kind()==Ptr && fv.Elem().Kind()==Struct)
	// We set PIn (a *vInner) to a non-nil value so validateStruct recurses into it.
	// Expect errors for inner fields according to their tags.
	// This specifically targets the `else if fv.Elem().Kind() == reflect.Struct` path.

	t.Run("pointer-to-struct field recurses", func(t *testing.T) {
		obj := vOuter{
			PIn: &vInner{S: "", D: 0}, // both violate rules in vInner
		}
		ve := &ValidationError{}
		rv := reflect.ValueOf(&obj).Elem()
		m.validateStruct(rv, "Root", ve)

		if ve.Empty() {
			t.Fatalf("expected validation errors; got none")
		}
		by := ve.ByField()
		if _, ok := by["Root.PIn.S"]; !ok {
			t.Errorf("expected nonempty error at Root.PIn.S")
		}
		if _, ok := by["Root.PIn.D"]; !ok {
			t.Errorf("expected nonzeroDuration error at Root.PIn.D")
		}
	})
}
