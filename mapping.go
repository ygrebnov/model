package model

import (
	"reflect"
	"strings"
	"sync"
)

// fieldRulesKey uniquely identifies a struct field's tag to cache parsed rules.
// It uses the parent struct type and the field index to avoid collisions
// across different structs that have the same field type or name.
// tagName distinguishes between validate and validateElem (and leaves room for others).
type fieldRulesKey struct {
	parent  reflect.Type
	index   int
	tagName string
}

// mapping holds a thread-safe cache for parsed validation rules mapping.
type mapping struct {
	c cache // map[fieldRulesKey][]ruleNameParams
}

type cache interface {
	Load(key any) (value any, ok bool)
	Store(key any, value any)
}

func newMapping() *mapping {
	return &mapping{
		c: &sync.Map{},
	}
}

func (c *mapping) get(parent reflect.Type, fieldIndex int, tagName string) ([]ruleNameParams, bool) {
	key := fieldRulesKey{parent: parent, index: fieldIndex, tagName: tagName}
	if v, ok := c.c.Load(key); ok {
		return v.([]ruleNameParams), true
	}

	return nil, false
}

func (c *mapping) add(parent reflect.Type, fieldIndex int, tagName string, parsed []ruleNameParams) {
	key := fieldRulesKey{parent: parent, index: fieldIndex, tagName: tagName}
	c.c.Store(key, parsed)
}

// ruleNameParams holds the name and params of a single validation validationRule.
type ruleNameParams struct {
	name   string
	params []string
}

// parseTag tokenizes a raw tag string (e.g., "required,min(5),max(10)") into rules.
// Behavior:
//   - Splits on top-level commas only (commas inside parentheses do not split tokens).
//   - Trims whitespace around tokens and parameters.
//   - Empty tokens (from leading/trailing commas) are skipped.
//   - Parameters are split by commas; nested parentheses inside parameters are not parsed specially.
//   - Does not support quotes or escaping inside parameters.
func parseTag(tag string) []ruleNameParams {
	var rules []ruleNameParams
	if tag == "" || tag == "-" {
		return rules
	}

	var tokens []string
	depth := 0
	start := 0
	for i, r := range tag {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				tokens = append(tokens, strings.TrimSpace(tag[start:i]))
				start = i + 1
			}
		}
	}
	// Append the last token
	if start <= len(tag) {
		tokens = append(tokens, strings.TrimSpace(tag[start:]))
	}

	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		name := tok
		var params []string
		if idx := strings.IndexRune(tok, '('); idx != -1 && strings.HasSuffix(tok, ")") {
			name = strings.TrimSpace(tok[:idx])
			inner := strings.TrimSpace(tok[idx+1 : len(tok)-1])
			if inner != "" {
				parts := strings.Split(inner, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						params = append(params, p)
					}
				}
			}
		}
		if name != "" {
			rules = append(rules, ruleNameParams{name: name, params: params})
		}
	}
	return rules
}
