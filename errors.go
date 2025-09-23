package model

import "errors"

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	ErrNilObject    = errors.New("model: nil object")
	ErrNotStructPtr = errors.New("model: object must be a pointer to struct")
)
