package model

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/ygrebnov/model/rule"
)

type Model[TObject any] struct {
	mu                 sync.RWMutex
	once               sync.Once
	obj                *TObject
	rulesCache         rulesCache
	validators         map[string][]ruleAdapter // per-model registry: rule name -> overloads by type
	applyDefaultsOnNew bool
	validateOnNew      bool
}

type rulesCache interface {
	Get(parent reflect.Type, fieldIndex int, tagName string) ([]rule.Metadata, bool)
	Put(parent reflect.Type, fieldIndex int, tagName string, parsed []rule.Metadata)
}

// registerRuleAdapter registers/overwrites a rule overload at runtime (internal use).
func (m *Model[TObject]) registerRuleAdapter(name string, ad ruleAdapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if name == "" || ad.fn == nil || ad.fieldType == nil {
		return
	}
	if m.validators == nil {
		m.validators = make(map[string][]ruleAdapter)
	}
	m.validators[name] = append(m.validators[name], ad)
}

// getRuleAdapters retrieves all overloads for a rule name.
func (m *Model[TObject]) getRuleAdapters(name string) []ruleAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validators[name]
}

// applyRule dispatches to the best-matching overload of rule `name` for the given field value.
// Selection strategy:
//  1. Prefer exact type match (v.Type() == fieldType).
//  2. Otherwise accept AssignableTo matches (interfaces, named types), preferring the first declared.
//  3. If no matches, return a descriptive error listing available overload types.
//  4. If multiple exact matches (shouldn't happen), return an ambiguity error.
func (m *Model[TObject]) applyRule(name string, v reflect.Value, params ...string) error {
	adapters := m.getRuleAdapters(name)
	if len(adapters) == 0 {
		return fmt.Errorf("model: rule %q is not registered", name)
	}
	if !v.IsValid() {
		return fmt.Errorf("model: invalid value for rule %q", name)
	}

	typ := v.Type()
	var (
		exacts  []ruleAdapter
		assigns []ruleAdapter
	)
	for _, ad := range adapters {
		if ad.fieldType == nil || ad.fn == nil {
			continue
		}
		if typ == ad.fieldType {
			exacts = append(exacts, ad)
			continue
		}
		if typ.AssignableTo(ad.fieldType) {
			assigns = append(assigns, ad)
		}
	}

	switch {
	case len(exacts) == 1:
		return exacts[0].fn(v, params...)
	case len(exacts) > 1:
		return fmt.Errorf(
			"model: rule %q is ambiguous for type %s; %d exact overloads registered",
			name,
			typ,
			len(exacts),
		)
	case len(assigns) >= 1:
		return assigns[0].fn(v, params...)
	default:
		// Construct helpful message of available overload types.
		var names []string
		for _, ad := range adapters {
			if ad.fieldType != nil {
				names = append(names, ad.fieldType.String())
			}
		}
		sort.Strings(names)
		return fmt.Errorf(
			"model: rule %q has no overload for type %s (available: %s)",
			name,
			typ,
			strings.Join(names, ", "),
		)
	}
}

// rootStructValue validates that m.obj is a non-nil pointer to a struct and returns the struct value.
// The phase string is used in error messages (e.g., "Validate", "SetDefaults").
func (m *Model[TObject]) rootStructValue(phase string) (reflect.Value, error) {
	if m.obj == nil {
		return reflect.Value{}, fmt.Errorf("model: %s: nil object", phase)
	}
	rv := reflect.ValueOf(m.obj)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		// effectively unreachable due to generic type constraint, left for completeness
		return reflect.Value{}, fmt.Errorf("model: %s: object must be a non-nil pointer to struct; got %s", phase, rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("model: %s: object must point to a struct; got %s", phase, rv.Kind())
	}
	return rv, nil
}
