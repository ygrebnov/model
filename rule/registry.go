package rule

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Registry is a registry of validation rules.
type Registry struct {
	mu    sync.RWMutex
	rules map[string][]Rule // rule name -> overloads by type
}

func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string][]Rule),
	}
}

func (r *Registry) Add(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := rule.GetName()
	r.rules[name] = append(r.rules[name], rule)
}

// Get returns the best-matching overload of rule `name` for the given field value.
// Selection strategy:
//  1. Prefer exact type match (v.Type() == fieldType).
//  2. Otherwise accept AssignableTo matches (interfaces, named types), preferring the first declared.
//  3. Otherwise, if no matches, fetch a built-in rule if available.
//  4. If no matches, return a descriptive error listing available overload types.
//  5. If multiple exact matches (shouldn't happen), return an ambiguity error.
func (r *Registry) Get(name string, v reflect.Value) (Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !v.IsValid() {
		return nil, fmt.Errorf("model: invalid value for rule %q", name)
	}

	valueType := v.Type()

	rules, ok := r.rules[name]
	builtinRule, hasBuiltin := builtInRules[key{name, valueType}] // TODO: move inside switch

	if (!ok || len(rules) == 0) && !hasBuiltin {
		return nil, fmt.Errorf("model: rule %q is not registered", name)
	}

	var (
		exacts  []Rule
		assigns []Rule
	)
	for _, rr := range rules {
		//if ad.fieldType == nil || ad.fn == nil {
		//	continue
		//}
		// TODO: consider valueType nil, skip?
		if rr.IsOfType(valueType) {
			exacts = append(exacts, rr)
			continue
		}
		if rr.IsAssignableTo(valueType) {
			assigns = append(assigns, rr)
		}
	}

	switch {
	case len(exacts) == 1:
		return exacts[0], nil
	case len(exacts) > 1:
		// TODO: check for duplicates in Add to prevent this
		return nil, fmt.Errorf(
			"model: rule %q is ambiguous for type %s; %d exact overloads registered",
			name,
			valueType,
			len(exacts),
		)
	case len(assigns) >= 1:
		return assigns[0], nil
	case hasBuiltin:
		return builtinRule, nil
	default:
		// Construct helpful message of available overload types.
		return nil, fmt.Errorf(
			"model: rule %q has no overload for type %s (available: %s)",
			name,
			valueType,
			strings.Join(getFieldTypesNames(rules), ", "),
		)
	}
}

func getFieldTypesNames(rules []Rule) []string {
	var names []string
	for _, rr := range rules {
		filedTypeName := rr.GetFieldTypeName()
		if filedTypeName != "" {
			names = append(names, filedTypeName)
		}
	}
	sort.Strings(names) // TODO: replace with slices.Sort

	return names
}
