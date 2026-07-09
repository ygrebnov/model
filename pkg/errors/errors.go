package errors

import (
	errorsPkg "errors"

	"github.com/ygrebnov/errorc"
)

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	ErrNilContext = errorc.New("nil context")
	// ErrNilObject reports that a required object pointer was nil.
	ErrNilObject    = errorc.New("nil object")
	ErrNilEnvSource = errorc.New("nil env source")
	// ErrNotStructPtr reports that a value expected to be a non-nil pointer to a struct was not.
	ErrNotStructPtr        = errorc.New("object must be a non-nil pointer to struct")
	ErrCannotCompileSchema = errorc.New("cannot compile schema")
	ErrTypeParamNotStruct  = errorc.New("type parameter must be a struct")
	// ErrTypeMismatch reports that a value's type did not match the expected type.
	ErrTypeMismatch = errorc.New("type mismatch")
	// ErrInvalidRule reports that a rule definition is missing a name or validation function.
	ErrInvalidRule = errorc.New("rule must have non-empty name and non-nil function")
	// ErrRuleTypeMismatch reports that a rule was applied to an incompatible field type.
	ErrRuleTypeMismatch = errorc.New("rule type mismatch")
	// ErrDuplicateOverloadRule reports that a rule overload for the same field type was registered twice.
	ErrDuplicateOverloadRule = errorc.New("duplicate overload rule")
	// ErrRuleNotFound reports that no rule exists for the requested name.
	ErrRuleNotFound = errorc.New("rule not found")
	// ErrRuleOverloadNotFound reports that a rule exists by name but not for the requested field type.
	ErrRuleOverloadNotFound = errorc.New("rule overload not found")
	// ErrInvalidValue reports that a provided reflect.Value or runtime value is invalid.
	ErrInvalidValue = errorc.New("invalid value")
	// ErrAmbiguousRule reports that more than one equally suitable rule overload matched.
	ErrAmbiguousRule = errorc.New("ambiguous rule")
	// ErrSetDefault reports a failure while applying a default value.
	ErrSetDefault = errorc.New("cannot set default value")
	// ErrDefaultLiteralUnsupportedKind reports that a default literal was used with an unsupported field kind.
	ErrDefaultLiteralUnsupportedKind = errorc.New("default literal unsupported kind")
)

// Validation rule argument and parameter errors
var (
	// ErrRuleMissingParameter reports that a rule requiring parameters was invoked without them.
	ErrRuleMissingParameter = errorc.New("rule parameter is required")
	// ErrRuleInvalidParameter reports that a rule parameter could not be parsed or understood.
	ErrRuleInvalidParameter = errorc.New("rule parameter is invalid")
	// ErrRuleConstraintViolated reports that a rule was evaluated successfully but the value failed it.
	ErrRuleConstraintViolated = errorc.New("rule constraint violated")
)

var (
	ErrCannotParseInt         = errorc.New("cannot parse int")
	ErrCannotParseUint        = errorc.New("cannot parse uint")
	ErrCannotParseRuneLiteral = errorc.New("cannot parse rune literal")
	ErrCannotParseFloat       = errorc.New("cannot parse float")
	ErrCannotParseComplex     = errorc.New("cannot parse complex")
	ErrCannotParseDuration    = errorc.New("cannot parse duration")
)

// Is reports whether err matches target via errors.Is.
func Is(err, target error) bool {
	return errorsPkg.Is(err, target)
}

func GetBase(err error) error {
	if err == nil {
		return nil
	}

	for {
		base := errorsPkg.Unwrap(err)
		if base == nil {
			return err
		}

		err = base
	}
}
