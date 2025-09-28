package model

import "errors"

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	ErrNilObject             = errors.New("model: nil object")
	ErrNotStructPtr          = errors.New("model: object must be a pointer to struct")
	ErrDuplicateOverloadRule = errors.New("model: duplicate overload rule")
	ErrRuleNotFound          = errors.New("model: rule not found")
	ErrRuleOverloadNotFound  = errors.New("model: rule overload not found")
	ErrInvalidValue          = errors.New("model: invalid value")
	ErrAmbiguousRule         = errors.New("model: ambiguous rule")
)
