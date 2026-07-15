package tests

/*
type testObj struct {
	Name string `json:"name" validate:"omitempty,min(3)"`
}

func TestValidate_omitempty(t *testing.T) {
	tests := []struct {
		name          string
		obj           testObj
		expectedError bool
	}{
		{
			name:          "empty name -> no error (omitempty)",
			obj:           testObj{Name: ""},
			expectedError: false,
		},
		{
			name:          "valid name -> ok",
			obj:           testObj{Name: "valid"},
			expectedError: false,
		},
		{
			name:          "valid name -> ok",
			obj:           testObj{Name: "va"},
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := model.Validate(context.Background(), &test.obj)
			if test.expectedError && err == nil {
				t.Fatalf("expected Validate to return an error, but got none")
			}

			if !test.expectedError && err != nil {
				t.Fatalf("expected Validate to return no error, but got: %v", err)
			}
		})
	}
}

var errNonZeroDurFailed = errorc.New("nonZeroDur rule failed")

// --- helpers & sample rules used in tests ---
func ruleNonZeroDur(d time.Duration, _ ...string) error {
	if d == 0 {
		return errorc.With(
			errNonZeroDurFailed,
			errorc.String(keys.RuleName, "nonZeroDur"),
		)
	}
	return nil
}

// custom rule implementing min(1) semantics for tests replaced nonempty
func ruleMin1(s string, _ ...string) error {
	if len(s) < 1 {
		return errorc.With(
			errorc.New("min(1) rule failed"),
			errorc.String(keys.RuleName, "min(1)"),
		)
	}
	return nil
}

// --- types under test ---

type vNoTags struct {
	A int
	B string
}

type vHasTags struct {
	Name string        `validate:"min(1)"`
	Wait time.Duration `validate:"nonZeroDur"`
	Info struct {
		Note string `validate:"min(1)"`
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	type runFn func() (any, error)

	tests := []struct {
		name    string
		run     runFn
		wantErr string // substring to expect in error; empty => expect nil
		verify  func(t *testing.T, err error, m any)
	}{
		{
			name: "nil object -> error",
			run: func() (any, error) {
				var obj *vNoTags

				return nil, model.Validate(context.Background(), obj)
			},
			wantErr: "nil object",
		},
		{
			name: "non-struct object -> error",
			run: func() (any, error) {
				x := 42
				return nil, model.Validate(context.Background(), &x)
			},
			wantErr: "type parameter must be a struct",
		},
		{
			name: "no tags -> ok (nil error)",
			run: func() (any, error) {
				obj := vNoTags{A: 1, B: "x"}
				return nil, model.Validate(context.Background(), &obj)
			},
			wantErr: "",
		},
		{
			name: "rules satisfied -> ok (nil error)",
			run: func() (any, error) {
				min1, err := model.NewRule("min(1)", ruleMin1) // illustrative; tag uses min(1) but rule name simplified
				if err != nil {
					return nil, err
				}
				nonZeroDur, err := model.NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					return nil, err
				}
				b, err := model.NewBinding[vHasTags](model.WithRules(min1, nonZeroDur))
				if err != nil {
					return b, err
				}
				obj := vHasTags{Name: "ok", Wait: time.Second}
				obj.Info.Note = "ok"
				validationErr := b.Validate(context.Background(), &obj)
				return b, validationErr
			},
			wantErr: "",
		},
		{
			name: "rule failures -> ValidationError with multiple field errors",
			run: func() (any, error) {
				min1, err := model.NewRule("min(1)", ruleMin1)
				if err != nil {
					return nil, err
				}
				nonZeroDur, err := model.NewRule("nonZeroDur", ruleNonZeroDur)
				if err != nil {
					return nil, err
				}
				b, err := model.NewBinding[vHasTags](model.WithRules(min1, nonZeroDur))
				if err != nil {
					return nil, err
				}
				obj := vHasTags{} // Name empty, Wait zero, Info.Note empty
				validationErr := b.Validate(context.Background(), &obj)
				return b, validationErr
			},
			wantErr: `- Field "Name"`,
			verify: func(t *testing.T, err error, _ any) {
				var ve *validation.Error
				if !nativeerrors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				if ve.Empty() || ve.Len() < 3 {
					t.Fatalf("expected >=3 field errors, got %d", ve.Len())
				}
				by := ve.ByField()
				for _, p := range []string{"Name", "Wait", "Info.Note"} {
					if _, ok := by[p]; !ok {
						t.Errorf("missing error for field path %q", p)
					}
				}
				if es := by["Name"]; len(es) == 0 {
					t.Fatalf("expected error for Name")
				} else {
					// Name uses builtin string min rule; assert on builtin sentinel and metadata.
					if !errors.Is(es[0].Err, errors.ErrRuleConstraintViolated) {
						t.Fatalf("expected ErrRuleConstraintViolated for Name, got %v", es[0].Err)
					}
					msg := es[0].Err.Error()
					if !strings.Contains(msg, string(keys.RuleName)+": min") {
						t.Errorf("expected builtin min rule name metadata for Name, got: %q", msg)
					}
					if !strings.Contains(msg, string(keys.RuleParamName)+": length") ||
						!strings.Contains(msg, string(keys.RuleParamValue)+": 1") {
						t.Errorf("expected min length metadata for Name, got: %q", msg)
					}
				}
				if es := by["Wait"]; len(es) == 0 {
					t.Fatalf("expected error for Wait")
				} else {
					if !errors.Is(es[0].Err, errNonZeroDurFailed) {
						t.Fatalf("expected ErrRuleNonZeroDurFailed for Wait, got %v", es[0].Err)
					}
					if msg := es[0].Err.Error(); !strings.Contains(msg, string(keys.RuleName)+": nonZeroDur") {
						t.Errorf("expected rule name metadata for nonZeroDur in Wait error, got: %q", msg)
					}
				}
			},
		},
		{
			name: "unknown rule -> ValidationError with rule-not-registered message",
			run: func() (any, error) {
				type vUnknown struct {
					Alias string `validate:"doesNotExist"`
				}
				b, err := model.NewBinding[vUnknown]()
				if err != nil {
					return nil, err
				}
				obj := vUnknown{}
				err = b.Validate(context.Background(), &obj)
				return b, err
			},
			wantErr: "rule not found",
			verify: func(t *testing.T, err error, _ any) {
				var ve *validation.Error
				if !nativeerrors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				if len(ve.ByField()["Alias"]) == 0 {
					t.Fatalf("expected Alias to have a rule-not-found error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m, err := tt.run()
			checkValidateTopError(t, err, tt.wantErr)
			if tt.verify != nil {
				tt.verify(t, err, m)
			}
		})
	}
}

func checkValidateTopError(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if wantSubstr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error containing %q, got %v", wantSubstr, err)
	}
}
*/
