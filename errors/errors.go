package errors

import (
	errorsPkg "errors"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/constants"
)

var namespace = errorc.Namespace(constants.Namespace)

const (
	messageNilObject                     = "nil object"
	messageNotStructPtr                  = "object must be a non-nil pointer to struct"
	messageInvalidRule                   = "rule must have non-empty name and non-nil function"
	messageRuleTypeMismatch              = "rule type mismatch"
	messageDuplicateOverloadRule         = "duplicate overload rule"
	messageRuleNotFound                  = "rule not found"
	messageRuleOverloadNotFound          = "rule overload not found"
	messageInvalidValue                  = "invalid value"
	messageAmbiguousRule                 = "ambiguous rule"
	messageSetDefault                    = "cannot set default value"
	messageDefaultLiteralUnsupportedKind = "default literal unsupported kind"
	messageRuleMissingParameter          = "rule parameter is required"
	messageRuleInvalidParameter          = "rule parameter is invalid"
	messageRuleConstraintViolated        = "rule constraint violated"
)

const (
	summaryRuleConstraintViolated = "constraint violated"
	summaryRuleInvalidParameter   = "invalid rule parameter"
	summaryRuleMissingParameter   = "missing rule parameter"
	summaryRuleOverloadNotFound   = "rule is not applicable to this field type"
)

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	// ErrNilObject reports that a required object pointer was nil.
	ErrNilObject = namespace.NewError(messageNilObject)
	// ErrNotStructPtr reports that a value expected to be a non-nil pointer to a struct was not.
	ErrNotStructPtr = namespace.NewError(messageNotStructPtr)
	// ErrInvalidRule reports that a rule definition is missing a name or validation function.
	ErrInvalidRule = namespace.NewError(messageInvalidRule)
	// ErrRuleTypeMismatch reports that a rule was applied to an incompatible field type.
	ErrRuleTypeMismatch = namespace.NewError(messageRuleTypeMismatch)
	// ErrDuplicateOverloadRule reports that a rule overload for the same field type was registered twice.
	ErrDuplicateOverloadRule = namespace.NewError(messageDuplicateOverloadRule)
	// ErrRuleNotFound reports that no rule exists for the requested name.
	ErrRuleNotFound = namespace.NewError(messageRuleNotFound)
	// ErrRuleOverloadNotFound reports that a rule exists by name but not for the requested field type.
	ErrRuleOverloadNotFound = namespace.NewError(messageRuleOverloadNotFound)
	// ErrInvalidValue reports that a provided reflect.Value or runtime value is invalid.
	ErrInvalidValue = namespace.NewError(messageInvalidValue)
	// ErrAmbiguousRule reports that more than one equally suitable rule overload matched.
	ErrAmbiguousRule = namespace.NewError(messageAmbiguousRule)
	// ErrSetDefault reports a failure while applying a default value.
	ErrSetDefault = namespace.NewError(messageSetDefault)
	// ErrDefaultLiteralUnsupportedKind reports that a default literal was used with an unsupported field kind.
	ErrDefaultLiteralUnsupportedKind = namespace.NewError(messageDefaultLiteralUnsupportedKind)

	// Validation rule argument and parameter errors
	// ErrRuleMissingParameter reports that a rule requiring parameters was invoked without them.
	ErrRuleMissingParameter = namespace.NewError(messageRuleMissingParameter)
	// ErrRuleInvalidParameter reports that a rule parameter could not be parsed or understood.
	ErrRuleInvalidParameter = namespace.NewError(messageRuleInvalidParameter)
	// ErrRuleConstraintViolated reports that a rule was evaluated successfully but the value failed it.
	ErrRuleConstraintViolated = namespace.NewError(messageRuleConstraintViolated)
)

// Is reports whether err matches target via errors.Is.
func Is(err, target error) bool {
	return errorsPkg.Is(err, target)
}

// Summary returns a concise human-readable summary for known model errors.
// It is intended for presentation helpers that need stable display text
// without duplicating sentinel-to-message mapping logic.
func Summary(err error) string {
	if err == nil {
		return ""
	}

	if summary, ok := knownSummary(err); ok {
		return summary
	}

	return err.Error()
}

func knownSummary(err error) (string, bool) {
	switch {
	case errorsPkg.Is(err, ErrRuleConstraintViolated):
		return summaryRuleConstraintViolated, true
	case errorsPkg.Is(err, ErrRuleInvalidParameter):
		return summaryRuleInvalidParameter, true
	case errorsPkg.Is(err, ErrRuleMissingParameter):
		return summaryRuleMissingParameter, true
	case errorsPkg.Is(err, ErrRuleNotFound):
		return messageRuleNotFound, true
	case errorsPkg.Is(err, ErrRuleOverloadNotFound):
		return summaryRuleOverloadNotFound, true
	case errorsPkg.Is(err, ErrInvalidValue):
		return messageInvalidValue, true
	default:
		return "", false
	}
}
