package model

import "github.com/ygrebnov/model/validation"

type options struct {
	envPrefix string
	rules     []validation.Rule
}

// Option customizes Binding, DynamicBinding or wrappers behavior.
type Option func(o *options)

// WithEnvPrefix adds non-empty environment variables names prefix.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		if prefix != "" {
			o.envPrefix = prefix
		}
	}
}

// WithRules registers one or many named custom validation rules during binding construction.
//
// See the validation.Rule type and validation.NewRule function for details on creating rules.
func WithRules(rules ...validation.Rule) Option {
	return func(o *options) {
		o.rules = rules
	}
}
