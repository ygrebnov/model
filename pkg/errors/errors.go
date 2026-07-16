// Package errors defines sentinel errors returned by model operations.
//
// Callers can match these errors with errors.Is, directly or through the Is
// helper in this package.
package errors

import (
	errorsPkg "errors"

	"github.com/ygrebnov/errorc"
)

// Sentinel errors returned by model operations. Use errors.Is to match.
var (
	// ErrNilContext reports that validation received a nil context.
	ErrNilContext = errorc.New("nil context")
	// ErrNilObject reports that a required object pointer was nil.
	ErrNilObject = errorc.New("nil object")
	// ErrNilEnvSource reports that no environment snapshot source is available.
	ErrNilEnvSource = errorc.New("nil env source")
	// ErrCannotCompileSchema reports a failure while compiling struct metadata.
	ErrCannotCompileSchema = errorc.New("cannot compile schema")
	// ErrTypeParamNotStruct reports that a binding type parameter is not a struct.
	ErrTypeParamNotStruct = errorc.New("type parameter must be a struct")
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

var (
	// ErrInvalidValidateElemUsage reports validateElem on a non-collection field.
	ErrInvalidValidateElemUsage = errorc.New("validateElem can only be used on slice, array, or map fields")
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
	// ErrCannotParseInt reports an invalid signed integer literal.
	ErrCannotParseInt = errorc.New("cannot parse int")
	// ErrCannotParseUint reports an invalid unsigned integer literal.
	ErrCannotParseUint = errorc.New("cannot parse uint")
	// ErrCannotParseRuneLiteral reports an invalid rune literal.
	ErrCannotParseRuneLiteral = errorc.New("cannot parse rune literal")
	// ErrCannotParseFloat reports an invalid floating-point literal.
	ErrCannotParseFloat = errorc.New("cannot parse float")
	// ErrCannotParseComplex reports an invalid complex literal.
	ErrCannotParseComplex = errorc.New("cannot parse complex")
	// ErrCannotParseDuration reports an invalid duration literal.
	ErrCannotParseDuration = errorc.New("cannot parse duration")
)

// Is reports whether err matches target via errors.Is.
func Is(err, target error) bool {
	return errorsPkg.Is(err, target)
}

// GetBase unwraps err until it reaches the innermost wrapped error.
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
