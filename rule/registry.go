package rule

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Registry is a registry of validation rules.
type Registry struct {
	rules map[string][]Rule // rule name -> overloads by type
}

func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string][]Rule),
	}
}

func (r *Registry) Add(rule Rule) {
	name := rule.GetName()
	r.rules[name] = append(r.rules[name], rule)
}

func (r *Registry) Get(name string, v reflect.Value) (Rule, error) {
	rules, ok := r.rules[name]
	if !ok || len(rules) == 0 {
		return nil, fmt.Errorf("model: rule %q is not registered", name)
	}
	if !v.IsValid() {
		return nil, fmt.Errorf("model: invalid value for rule %q", name)
	}

	valueType := v.Type()
	var (
		exacts  []Rule
		assigns []Rule
	)
	for _, rr := range rules {
		//if ad.fieldType == nil || ad.fn == nil {
		//	continue
		//}
		if rr.IsOfType(valueType) {
			exacts = append(exacts, rr)
			continue
		}
		if typ.AssignableTo(ad.fieldType) {
			assigns = append(assigns, ad)
		}
	}

	switch {
	case len(exacts) == 1:
		return exacts[0], nil
	case len(exacts) > 1:
		return nil, fmt.Errorf(
			"model: rule %q is ambiguous for type %s; %d exact overloads registered",
			name,
			valueType,
			len(exacts),
		)
	case len(assigns) >= 1:
		return assigns[0], nil
	default:
		// Construct helpful message of available overload types.
		var names []string
		for _, ad := range adapters {
			if ad.fieldType != nil {
				names = append(names, ad.fieldType.String())
			}
		}
		sort.Strings(names)
		return nil, fmt.Errorf(
			"model: rule %q has no overload for type %s (available: %s)",
			name,
			typ,
			strings.Join(names, ", "),
		)
	}
}
