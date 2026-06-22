package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/validation"
)

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
	In  vInner  `validate:""` // explicit empty (no rule), but should dive due to struct recursion
	PIn *vInner `validate:""` // pointer; nil → no validation; non-nil → dive
	//nolint:unused // unexported field to test that it's skipped even with a tag
	pin *vInner `validate:"nonempty"` // unexported: must be skipped despite tag

	// Simple rules
	Name string `validate:"nonempty"`
	// params test (commas inside parens; also followed by another rule)
	Note string `validate:"withParams(a, b, c),nonempty"`

	// Unknown rule
	Alias string `validate:"doesNotExist"`

	// Ambiguity: same name registered twice for exact type
	Amb string `validate:"dup"`

	// Assignable: rule registered for fmt.Stringer, field is concrete strWrap
	Wrapped strWrap `validate:"stringerBad"`

	// Tokenizer coverage: nested parentheses and leading/trailing commas
	TokNested string `validate:"tokA((x,y)),tokB"`
	TokEdges  string `validate:",nonempty,"`

	// validateElem tokenizer coverage: nested parentheses and leading/trailing commas
	ElemTokNested []string `validateElem:"tokEA((x,y)),tokEB"`
	ElemTokEdges  []string `validateElem:",nonempty,"`
	ElemParams    []string `validateElem:"withParams(a, b, c),nonempty"`

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

func TestBinding_Validate(t *testing.T) {
	stringNonEmpty, err := validation.NewRule[string]("nonempty", ruleNonEmpty)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	stringWithParams, err := validation.NewRule[string]("withParams", ruleWithParams)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	durationNonzero, err := validation.NewRule[time.Duration]("nonzeroDuration", ruleNonzeroDuration)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	// Register interface-based rule (AssignableTo path)
	stringerBad, err := validation.NewRule[fmt.Stringer]("stringerBad", ruleStringerBad)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	// Register single rule for dup (used to be ambiguous with duplicates)
	stringDup1, err := validation.NewRule[string]("dup", ruleNonEmpty)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	// Also a rule for int to demonstrate element rules on int slices if needed
	intSlices, err := validation.NewRule[int]("intErr", ruleIntAlwaysErr)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}

	// Create Binding with all custom rules needed across subtests.
	b, err := model.NewBinding[vOuter](model.WithRules(
		stringNonEmpty,
		stringWithParams,
		durationNonzero,
		stringerBad,
		stringDup1,
		intSlices,
	))
	if err != nil {
		t.Fatalf("NewBinding error: %v", err)
	}

	t.Run("recursion, params parsing, unknown rule, ambiguity, assignable, and validateElem on slices/maps", func(t *testing.T) {
		obj := vOuter{
			// Recursion: In has validate tags; PIn initially nil (no dive); pin unexported ignored
			In: vInner{
				S: "", // nonempty → error
				D: 0,  // nonzeroDuration → error
			},
			PIn: nil,
			// Simple rules
			Name:      "",               // nonempty → error
			Note:      "",               // withParams(...) → error with params; then nonempty → error
			Alias:     "",               // unknown rule → error from applyRule
			Amb:       "",               // dup rule → ambiguity error
			Wrapped:   strWrap{v: "ok"}, // ruleStringerBad → error (we're asserting AssignableTo used)
			TokNested: "",               // triggers unknown rule errors (tokA, tokB) but exercises nested paren tokenization
			TokEdges:  "",               // leading/trailing commas around nonempty -> one nonempty error

			// validateElem tokenizer coverage
			ElemTokNested: []string{"val"}, // triggers two unknown rule applications on the same element
			ElemTokEdges:  []string{""},    // leading/trailing commas -> only nonempty applies
			ElemParams:    []string{""},    // withParams(a,b,c) and nonempty apply to element[0]

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

		err = b.Validate(context.Background(), &obj) // use non-empty path prefix to test dotted paths
		if err == nil {
			t.Fatalf("expected error, got none")
		}

		var ve *validation.Error
		if !errors.As(err, &ve) {
			t.Fatalf("expected validation errors; got none")
		}

		if ve.Empty() {
			t.Fatalf("expected validation errors; got none")
		}

		// Collect messages by field path for assertions
		by := ve.ByField()

		// Recursion into In
		if _, ok := by["In.S"]; !ok {
			t.Errorf("expected error at In.S (nonempty)")
		}
		if _, ok := by["In.D"]; !ok {
			t.Errorf("expected error at In.D (nonzeroDuration)")
		}

		// PIn nil → no entries under PIn.*
		for p := range by {
			if strings.HasPrefix(p, "PIn.") {
				t.Errorf("did not expect errors under PIn.*, got %s", p)
			}
		}

		// Unexported pin ignored
		for p := range by {
			if strings.Contains(p, ".pin") {
				t.Errorf("unexported field pin should be skipped; saw %s", p)
			}
		}

		// Simple rules
		if _, ok := by["Name"]; !ok {
			t.Errorf("expected nonempty error at Name")
		}
		// params parsing (withParams and nonempty applied)
		paramsMsgs := by["Note"]
		if len(paramsMsgs) == 0 {
			t.Errorf("expected errors for Note")
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
				t.Errorf("expected nonempty error for Note")
			}
		}

		// Ambiguity on dup (no longer ambiguous). Expect a single nonempty error.
		if es := by["Amb"]; len(es) == 0 || es[0].Rule != "dup" || !strings.Contains(es[0].Err.Error(), "must not be empty") {
			t.Errorf("expected nonempty error at Amb, got: %+v", es)
		}

		// Unknown rule applied (rule not found)
		if es := by["Alias"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "rule not found") {
			t.Errorf("expected unknown rule error at Alias, got: %+v", es)
		}

		// Assignable interface rule
		if es := by["Wrapped"]; len(es) == 0 || !strings.Contains(es[0].Err.Error(), "bad stringer") {
			t.Errorf("expected stringerBad error at Wrapped, got: %+v", es)
		}

		// Tokenizer: nested parentheses should not split inside, producing two top-level tokens (tokA(...), tokB)
		if es := by["TokNested"]; len(es) != 2 {
			t.Errorf("expected 2 errors for TokNested (tokA and tokB), got %d: %+v", len(es), es)
		} else {
			// Ensure rules are tokA and tokB (unknown rule errors)
			have := map[string]bool{"tokA": false, "tokB": false}
			for _, fe := range es {
				have[fe.Rule] = true
				if !strings.Contains(fe.Err.Error(), "rule not found") {
					t.Errorf("expected unknown rule error for %s, got %v", fe.Rule, fe.Err)
				}
			}
			if !have["tokA"] || !have["tokB"] {
				t.Errorf("expected rules tokA and tokB, got presence=%v", have)
			}
		}

		// Tokenizer: leading/trailing commas create empty tokens which must be skipped; only nonempty should apply
		if es := by["TokEdges"]; len(es) != 1 || es[0].Rule != "nonempty" {
			t.Errorf("expected exactly one nonempty error for TokEdges, got %+v", es)
		}

		// --- validateElem tokenizer coverage ---
		// ElemTokNested: nested parentheses should not split inside; two top-level tokens (tokEA(...), tokEB)
		if es := by["ElemTokNested[0]"]; len(es) != 2 {
			t.Errorf("expected 2 errors for ElemTokNested[0] (tokEA and tokEB), got %d: %+v", len(es), es)
		} else {
			have := map[string]bool{"tokEA": false, "tokEB": false}
			for _, fe := range es {
				have[fe.Rule] = true
				if !strings.Contains(fe.Err.Error(), "rule not found") {
					t.Errorf("expected unknown rule error for %s, got %v", fe.Rule, fe.Err)
				}
			}
			if !have["tokEA"] || !have["tokEB"] {
				t.Errorf("expected rules tokEA and tokEB, got presence=%v", have)
			}
		}

		// ElemTokEdges: leading/trailing commas create empty tokens which must be skipped; only nonempty should apply to the element
		if es := by["ElemTokEdges[0]"]; len(es) != 1 || es[0].Rule != "nonempty" {
			t.Errorf("expected exactly one nonempty error for ElemTokEdges[0], got %+v", es)
		}

		// ElemParams: commas inside parentheses must not split; expect withParams(a,b,c) and nonempty applied to element[0]
		if es := by["ElemParams[0]"]; len(es) != 2 {
			t.Errorf("expected 2 errors for ElemParams[0] (withParams and nonempty), got %d: %+v", len(es), es)
		} else {
			var sawParams, sawNonEmpty bool
			for _, fe := range es {
				if fe.Rule == "withParams" && strings.Contains(fe.Err.Error(), "params=a|b|c") {
					sawParams = true
				}
				if fe.Rule == "nonempty" && strings.Contains(fe.Err.Error(), "must not be empty") {
					sawNonEmpty = true
				}
			}
			if !sawParams {
				t.Errorf("expected withParams error with params=a|b|c for ElemParams[0]")
			}
			if !sawNonEmpty {
				t.Errorf("expected nonempty error for ElemParams[0]")
			}
		}

		// validateElem on slice of strings
		if es := by["Tags[0]"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Tags[0]")
		}
		if _, ok := by["Tags[1]"]; ok {
			t.Errorf("did not expect error at Tags[1] (was 'ok')")
		}
		if es := by["Tags[2]"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Tags[2]")
		}

		// validateElem dive on People slice (struct elements)
		if es := by["People[0].S"]; len(es) == 0 {
			t.Errorf("expected error at People[0].S (nonempty)")
		}
		if es := by["People[0].D"]; len(es) == 0 {
			t.Errorf("expected error at People[0].D (nonzeroDuration)")
		}
		// second element has S ok, D zero → expect only D
		if _, ok := by["People[1].S"]; ok {
			t.Errorf("did not expect error at People[1].S")
		}
		if es := by["People[1].D"]; len(es) == 0 {
			t.Errorf("expected error at People[1].D")
		}

		// misuse of dive on non-struct element slice (Numbers): error per element
		if es := by["Numbers[0]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Numbers[0], got: %+v", es)
		}
		if es := by["Numbers[1]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Numbers[1], got: %+v", es)
		}

		// Ptrs: nil pointer element produces a 'dive' misuse error (kind ptr), non-nil element gets dived
		if es := by["Ptrs[0]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at Ptrs[0], got: %+v", es)
		}
		if es := by["Ptrs[1].S"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Ptrs[1].S")
		}

		// Arrays behave like slices
		if es := by["Fixed[0].S"]; len(es) == 0 {
			t.Errorf("expected nonempty error at Fixed[0].S")
		}
		if es := by["Fixed[1].D"]; len(es) == 0 {
			t.Errorf("expected nonzeroDuration error at Fixed[1].D")
		}
		// FixedP pointer array: index 0 nil -> 'dive' misuse; index 1 non-nil -> dive into struct
		if es := by["FixedP[0]"]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at FixedP[0], got: %+v", es)
		}
		if es := by["FixedP[1].S"]; len(es) == 0 {
			t.Errorf("expected error at FixedP[1].S")
		}

		// Maps
		// Labels map[string]string with nonempty element rule
		if es := by[`Labels[a]`]; len(es) == 0 {
			t.Errorf("expected nonempty error at Labels[a]")
		}
		if _, ok := by[`Labels[b]`]; ok {
			t.Errorf("did not expect error at Labels[b]")
		}
		// Profiles map[string]vInner with dive
		if es := by[`Profiles[k1].S`]; len(es) == 0 {
			t.Errorf("expected error at Profiles[k1].S")
		}
		if es := by[`Profiles[k2].D`]; len(es) == 0 {
			t.Errorf("expected error at Profiles[k2].D")
		}
		// ProfilesP map[string]*vInner with dive
		if es := by[`ProfilesP[p1].S`]; len(es) == 0 {
			t.Errorf("expected error at ProfilesP[p1].S")
		}
		if es := by[`ProfilesP[p2]`]; len(es) == 0 || es[0].Rule != "dive" {
			t.Errorf("expected 'dive' misuse error at ProfilesP[p2], got: %+v", es)
		}

		// validateElem ignored on non-container (Ghost)
		if _, ok := by["Ghost"]; ok {
			t.Errorf("did not expect any error at Ghost due to validateElem tag (non-container)")
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

		err = b.Validate(context.Background(), &obj)
		if err == nil {
			t.Fatalf("expected error, got none")
		}

		var ve *validation.Error
		if !errors.As(err, &ve) {
			t.Fatalf("expected error to be of type validation.Error")
		}

		if ve.Empty() {
			t.Fatalf("expected validation errors; got none")
		}
		by := ve.ByField()
		if _, ok := by["PIn.S"]; !ok {
			t.Errorf("expected nonempty error at PIn.S")
		}
		if _, ok := by["PIn.D"]; !ok {
			t.Errorf("expected nonzeroDuration error at PIn.D")
		}
	})
}

// New test to ensure built-in rules are applied when Validate is called
// on a fresh Binding without any options.
func TestBinding_Validate_NoOptions_Builtins(t *testing.T) {
	t.Parallel()
	type Obj struct {
		S string `validate:"min(1)"`
	}
	obj := Obj{}
	b, err := model.NewBinding[Obj]() // no Rules
	if err != nil {
		// New should not fail just because validation isn't requested yet.
		t.Fatalf("unexpected error from New: %v", err)
	}
	// First validation should pick up built-in nonempty and fail because S is empty.
	err = b.Validate(context.Background(), &obj)
	if err == nil {
		t.Fatalf("expected validation error for empty S, got nil")
	}
	var ve *validation.Error
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if _, ok := ve.ByField()["S"]; !ok {
		t.Fatalf("expected field error for S, got: %+v", ve.ByField())
	}
	// Fix the field and validate again; should succeed.
	obj.S = "x"
	if err := b.Validate(context.Background(), &obj); err != nil {
		t.Fatalf("expected no error after fixing S, got: %v", err)
	}
}
