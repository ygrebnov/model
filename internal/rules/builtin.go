package rules

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// Built-ins are always implicitly available.

const (
	// RuleMin is the canonical built-in rule name for minimum constraints.
	RuleMin = "min"
	// RuleMax is the canonical built-in rule name for maximum constraints.
	RuleMax = "max"
	// RuleNonzero is the canonical built-in rule name for non-zero constraints.
	RuleNonzero = "nonzero"
	// RuleOneOf is the canonical built-in rule name for allowed-set membership checks.
	RuleOneOf = "oneof"
	// RuleEmail is the canonical built-in rule name for email validation.
	RuleEmail = "email"
	// RuleUUID is the canonical built-in rule name for UUID validation.
	RuleUUID = "uuid"
	// RuleSemver is the canonical built-in rule name for semantic version numbers validation.
	RuleSemver = "semver"
)

const (
	// RuleStringMin is the backward-compatible alias for the string min rule name.
	RuleStringMin = RuleMin
	// RuleStringMax is the backward-compatible alias for the string max rule name.
	RuleStringMax = RuleMax
	// RuleIntMin is the backward-compatible alias for the int min rule name.
	RuleIntMin = RuleMin
	// RuleIntMax is the backward-compatible alias for the int max rule name.
	RuleIntMax = RuleMax
	// RuleInt64Min is the backward-compatible alias for the int64 min rule name.
	RuleInt64Min = RuleMin
	// RuleInt64Max is the backward-compatible alias for the int64 max rule name.
	RuleInt64Max = RuleMax
	// RuleFloat64Min is the backward-compatible alias for the float64 min rule name.
	RuleFloat64Min = RuleMin
	// RuleFloat64Max is the backward-compatible alias for the float64 max rule name.
	RuleFloat64Max = RuleMax
)

// key consists of a name and a field value type.
type key struct {
	name      string
	fieldType reflect.Type
}

// Lazy built-in rule storage.
var (
	builtInsOnce        sync.Once
	builtInMap          map[key]*Rule
	builtinStringRules  []*Rule
	builtinNumericRules []*Rule
)

const (
	paramNameAllowed = "allowed"
	paramNameLength  = "length"
	paramNameValue   = "value"

	emailCheckAtCount             = "at_count"
	emailCheckDomainHasDot        = "domain_has_dot"
	emailCheckLocalDomainNonempty = "local_domain_nonempty"
	emailCheckNoWhitespace        = "no_whitespace"

	uuidCheckFormat = "format"
	uuidCheckHex    = "hex"
	uuidLength      = 36
)

var uuidHyphenPositions = [4]int{8, 13, 18, 23}

func newRuleMissingParameterError(ruleName string) error {
	return errorc.With(
		errors.ErrRuleMissingParameter,
		errorc.String(keys.RuleName, ruleName),
	)
}

func newRuleInvalidParameterError(ruleName, paramName, paramValue string, cause error) error {
	return errorc.With(
		errors.ErrRuleInvalidParameter,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
		errorc.String(keys.RuleParamValue, paramValue),
		errorc.Error(keys.Cause, cause),
	)
}

func newRuleConstraintViolationError(ruleName string) error {
	return errorc.With(
		errors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
	)
}

func newRuleConstraintViolationWithStringParamError(ruleName, paramName, paramValue string) error {
	return errorc.With(
		errors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
		errorc.String(keys.RuleParamValue, paramValue),
	)
}

func newRuleConstraintViolationWithIntParamError(ruleName, paramName string, paramValue int) error {
	return errorc.With(
		errors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
		errorc.Int(keys.RuleParamValue, paramValue),
	)
}

func newRuleConstraintViolationWithParamNameError(ruleName, paramName string) error {
	return errorc.With(
		errors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
	)
}

func mustRule(r *Rule, err error) *Rule {
	if err != nil {
		panic(err)
	}

	return r
}

func splitEmailParts(s string) (local, domain string) {
	parts := strings.Split(s, "@")
	return parts[0], parts[1]
}

func validateBuiltinEmail(s string) error {
	if s == "" { // treat empty as error, keeping semantics similar to prior nonempty
		return newRuleConstraintViolationError(RuleEmail)
	}
	if strings.Count(s, "@") != 1 {
		return newRuleConstraintViolationWithStringParamError(RuleEmail, emailCheckAtCount, "1")
	}

	local, domain := splitEmailParts(s)
	if local == "" || domain == "" {
		return newRuleConstraintViolationWithParamNameError(RuleEmail, emailCheckLocalDomainNonempty)
	}
	if strings.ContainsAny(s, " \t\n\r") {
		return newRuleConstraintViolationWithParamNameError(RuleEmail, emailCheckNoWhitespace)
	}
	if !strings.Contains(domain, ".") { // simple domain heuristic
		return newRuleConstraintViolationWithParamNameError(RuleEmail, emailCheckDomainHasDot)
	}

	return nil
}

func validateBuiltinSemver(s string) error {
	if s == "" {
		return newRuleConstraintViolationError(RuleSemver)
	}

	//nolint:lll // a regexp from the official spec.
	e := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

	re := regexp.MustCompile(e)
	if !re.MatchString(s) {
		return newRuleConstraintViolationError(RuleSemver)
	}
	return nil
}

func isUUIDHyphenPosition(index int) bool {
	for _, position := range uuidHyphenPositions {
		if index == position {
			return true
		}
	}

	return false
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func validateBuiltinUUID(s string) error {
	// Empty is invalid; caller can omit the rule if empty is allowed.
	// Canonical form: 36 chars, 8-4-4-4-12 with hyphens, hex digits only.
	if len(s) != uuidLength {
		return newRuleConstraintViolationWithIntParamError(RuleUUID, paramNameLength, len(s))
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUUIDHyphenPosition(i) {
			if c != '-' {
				return newRuleConstraintViolationWithStringParamError(
					RuleUUID, uuidCheckFormat, "expected hyphens at 8,13,18,23",
				)
			}
			continue
		}
		if !isHexDigit(c) {
			return newRuleConstraintViolationWithStringParamError(RuleUUID, uuidCheckHex, "non-hex character")
		}
	}

	return nil
}

// string rules
func getStringMinMaxRule(
	name string,
	noop func(v int64) bool,
	compare func(a, b int) bool,
) (*Rule, error) {
	return NewRule[string](name, func(s string, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(name)
		}
		raw := strings.TrimSpace(params[0])
		v, err := strconv.ParseInt(raw, 10, 0)
		if err != nil {
			return newRuleInvalidParameterError(name, paramNameLength, raw, err)
		}
		if noop(v) { // noop as requested
			return nil
		}
		if compare(int(v), len(s)) {
			return newRuleConstraintViolationWithStringParamError(name, paramNameLength, raw)
		}
		return nil
	})
}

// min(length): requires one integer parameter. If missing -> error. If <1 -> noop.
func getStrMinRule() (*Rule, error) {
	return getStringMinMaxRule(
		RuleStringMin,
		func(v int64) bool { return v < 1 },
		func(a, b int) bool { return a > b },
	)
}

// max(length): requires one integer parameter. If missing -> error. If <0 -> noop.
func getStrMaxRule() (*Rule, error) {
	return getStringMinMaxRule(
		RuleStringMax,
		func(v int64) bool { return v < 0 },
		func(a, b int) bool { return a < b },
	)
}

// email rule: deliberately simple; not RFC 5322 exhaustive. Provides lightweight validation.
func getStrEmailRule() (*Rule, error) {
	return NewRule[string](RuleEmail, func(s string, _ ...string) error {
		return validateBuiltinEmail(s)
	})
}

// semver rule: corresponding to version 2.0.0 published at https://semver.org/spec/v2.0.0.html
func getStrSemverRule() (*Rule, error) {
	return NewRule[string](RuleSemver, func(s string, _ ...string) error {
		return validateBuiltinSemver(s)
	})
}

// oneof rule: value must match one of the provided parameters.
func getStrOneofRule() (*Rule, error) {
	return NewRule[string](RuleOneOf, func(s string, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(RuleOneOf)
		}
		for _, p := range params {
			if s == p {
				return nil
			}
		}
		// we expose the allowed set as the param value for debugging/inspection
		return newRuleConstraintViolationWithStringParamError(
			RuleOneOf,
			paramNameAllowed,
			strings.Join(params, ","),
		)
	})
}

// uuid rule: value must be a valid canonical UUID string (lower/upper hex, 8-4-4-4-12 format).
func getStrUUIDRule() (*Rule, error) {
	return NewRule[string](RuleUUID, func(s string, _ ...string) error {
		return validateBuiltinUUID(s)
	})
}

type signedNumeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type unsignedNumeric interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type floatNumeric interface {
	~float32 | ~float64
}

type numeric interface {
	signedNumeric | unsignedNumeric | floatNumeric
}

func getNumericMinMaxRule[T numeric](
	name string,
	parse func(string) (T, error),
	compare func(a, b T) bool,
) (*Rule, error) {
	return NewRule[T](name, func(n T, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(name)
		}
		raw := strings.TrimSpace(params[0])
		v, err := parse(raw)
		if err != nil {
			return newRuleInvalidParameterError(name, paramNameValue, raw, err)
		}
		if compare(n, v) {
			return newRuleConstraintViolationWithStringParamError(name, paramNameValue, raw)
		}
		return nil
	})
}

func getNumericNonzeroRule[T numeric](name string) (*Rule, error) {
	return NewRule[T](name, func(n T, _ ...string) error {
		if n == 0 {
			return newRuleConstraintViolationError(name)
		}
		return nil
	})
}

func getNumericOneofRule[T numeric](
	name string,
	parse func(string) (T, error),
) (*Rule, error) {
	return NewRule[T](name, func(n T, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(name)
		}
		for _, p := range params {
			raw := strings.TrimSpace(p)
			v, err := parse(raw)
			if err != nil {
				return newRuleInvalidParameterError(name, paramNameValue, raw, err)
			}
			if v == n {
				return nil
			}
		}
		return newRuleConstraintViolationWithStringParamError(name, paramNameAllowed, strings.Join(params, ","))
	})
}

func parseSignedValue[T signedNumeric](bitSize int) func(string) (T, error) {
	return func(s string) (T, error) {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, bitSize)
		return T(v), err
	}
}

func parseUnsignedValue[T unsignedNumeric](bitSize int) func(string) (T, error) {
	return func(s string) (T, error) {
		v, err := strconv.ParseUint(strings.TrimSpace(s), 10, bitSize)
		return T(v), err
	}
}

func parseFloatValue[T floatNumeric](bitSize int) func(string) (T, error) {
	return func(s string) (T, error) {
		v, err := strconv.ParseFloat(strings.TrimSpace(s), bitSize)
		return T(v), err
	}
}

func getSignedNumericRules[T signedNumeric](bitSize int) []*Rule {
	return []*Rule{
		mustRule(getNumericMinMaxRule[T](RuleMin, parseSignedValue[T](bitSize), func(a, b T) bool { return a < b })),
		mustRule(getNumericMinMaxRule[T](RuleMax, parseSignedValue[T](bitSize), func(a, b T) bool { return a > b })),
		mustRule(getNumericNonzeroRule[T](RuleNonzero)),
		mustRule(getNumericOneofRule[T](RuleOneOf, parseSignedValue[T](bitSize))),
	}
}

func getUnsignedNumericRules[T unsignedNumeric](bitSize int) []*Rule {
	return []*Rule{
		mustRule(getNumericMinMaxRule[T](RuleMin, parseUnsignedValue[T](bitSize), func(a, b T) bool { return a < b })),
		mustRule(getNumericMinMaxRule[T](RuleMax, parseUnsignedValue[T](bitSize), func(a, b T) bool { return a > b })),
		mustRule(getNumericNonzeroRule[T](RuleNonzero)),
		mustRule(getNumericOneofRule[T](RuleOneOf, parseUnsignedValue[T](bitSize))),
	}
}

func getFloatNumericRules[T floatNumeric](bitSize int) []*Rule {
	return []*Rule{
		mustRule(getNumericMinMaxRule[T](RuleMin, parseFloatValue[T](bitSize), func(a, b T) bool { return a < b })),
		mustRule(getNumericMinMaxRule[T](RuleMax, parseFloatValue[T](bitSize), func(a, b T) bool { return a > b })),
		mustRule(getNumericNonzeroRule[T](RuleNonzero)),
		mustRule(getNumericOneofRule[T](RuleOneOf, parseFloatValue[T](bitSize))),
	}
}

// ensureBuiltIns initializes built-in rules exactly once.
func ensureBuiltIns() {
	builtInsOnce.Do(func() {
		builtInMap = make(map[key]*Rule)

		// string rules
		builtinStringRules = []*Rule{
			mustRule(getStrMinRule()),
			mustRule(getStrMaxRule()),
			mustRule(getStrEmailRule()),
			mustRule(getStrOneofRule()),
			mustRule(getStrUUIDRule()),
			mustRule(getStrSemverRule()),
		}

		builtinNumericRules = make([]*Rule, 0, 52)
		builtinNumericRules = append(builtinNumericRules, getSignedNumericRules[int](strconv.IntSize)...)
		builtinNumericRules = append(builtinNumericRules, getSignedNumericRules[int8](8)...)
		builtinNumericRules = append(builtinNumericRules, getSignedNumericRules[int16](16)...)
		builtinNumericRules = append(builtinNumericRules, getSignedNumericRules[int32](32)...)
		builtinNumericRules = append(builtinNumericRules, getSignedNumericRules[int64](64)...)
		builtinNumericRules = append(builtinNumericRules, getUnsignedNumericRules[uint](strconv.IntSize)...)
		builtinNumericRules = append(builtinNumericRules, getUnsignedNumericRules[uint8](8)...)
		builtinNumericRules = append(builtinNumericRules, getUnsignedNumericRules[uint16](16)...)
		builtinNumericRules = append(builtinNumericRules, getUnsignedNumericRules[uint32](32)...)
		builtinNumericRules = append(builtinNumericRules, getUnsignedNumericRules[uint64](64)...)
		builtinNumericRules = append(builtinNumericRules, getUnsignedNumericRules[uintptr](strconv.IntSize)...)
		builtinNumericRules = append(builtinNumericRules, getFloatNumericRules[float32](32)...)
		builtinNumericRules = append(builtinNumericRules, getFloatNumericRules[float64](64)...)

		// fill map
		register := func(rs []*Rule) {
			for _, r := range rs {
				builtInMap[key{r.GetName(), r.getFieldType()}] = r
			}
		}
		register(builtinStringRules)
		register(builtinNumericRules)
	})
}

// lookupBuiltin returns a built-in rule by (Name,type) if present.
func lookupBuiltin(name string, t reflect.Type) (*Rule, bool) {
	ensureBuiltIns()

	r, ok := builtInMap[key{name, t}]
	if ok || t.Kind() != reflect.Ptr {
		return r, ok
	}

	r, ok = lookupBuiltin(name, t.Elem())
	if !ok {
		return nil, false
	}

	return &Rule{
		name:      name,
		fieldType: t,
		fn: func(v reflect.Value, params ...string) error {
			for v.Kind() == reflect.Ptr {
				if v.IsNil() {
					v = reflect.Zero(v.Type().Elem())
				} else {
					v = v.Elem()
				}
			}

			return r.GetValidationFn()(v, params...)
		},
	}, true
}
