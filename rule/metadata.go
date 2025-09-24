package rule

import "strings"

// Metadata holds the Name and ParamNames of a single validation Rule.
type Metadata struct {
	Name       string
	ParamNames []string
}

// ParseTag tokenizes a raw tag string (e.g., "required,min(5),max(10)") into rules.
// Behavior:
//   - Splits on top-level commas only (commas inside parentheses do not split tokens).
//   - Trims whitespace around tokens and parameters.
//   - Empty tokens (from leading/trailing commas) are skipped.
//   - Parameters are split by commas; nested parentheses inside parameters are not parsed specially.
//   - Does not support quotes or escaping inside parameters.
func ParseTag(tag string) []Metadata {
	var rules []Metadata
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
			rules = append(rules, Metadata{Name: name, ParamNames: params})
		}
	}
	return rules
}
