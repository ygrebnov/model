package rule

import (
	"reflect"
	"sync"
)

// fieldRulesKey uniquely identifies a struct field's tag to cache parsed rules.
// It uses the parent struct type and the field index to avoid collisions
// across different structs that have the same field type or Name.
// tagName distinguishes between validate and validateElem (and leaves room for others).
type fieldRulesKey struct {
	parent  reflect.Type
	index   int
	tagName string
}

var fieldRulesCache sync.Map // map[fieldRulesKey][]Metadata

// parseRulesCached returns parsed rules for the given field tag using a cache.
// rawTag is still required to parse on a cache miss, but is not part of the key
// because struct tags are static for a given compiled type.
func parseRulesCached(parent reflect.Type, fieldIndex int, tagName, rawTag string) []Metadata {
	key := fieldRulesKey{parent: parent, index: fieldIndex, tagName: tagName}
	if v, ok := fieldRulesCache.Load(key); ok {
		return v.([]Metadata)
	}
	parsed := ParseTag(rawTag)
	// Store even empty result to avoid repeated parsing of empty/"-" tags.
	fieldRulesCache.Store(key, parsed)
	return parsed
}

// Cache holds a thread-safe cache for parsed validation rules.
type Cache struct {
	c cache // map[fieldRulesKey][]Metadata
}

type cache interface {
	Load(key any) (value any, ok bool)
	Store(key any, value any)
}

func NewCache() *Cache {
	return &Cache{
		c: &sync.Map{},
	}
}

func (c *Cache) Get(parent reflect.Type, fieldIndex int, tagName string) ([]Metadata, bool) {
	key := fieldRulesKey{parent: parent, index: fieldIndex, tagName: tagName}
	if v, ok := c.c.Load(key); ok {
		return v.([]Metadata), true
	}

	return nil, false
}

func (c *Cache) Put(parent reflect.Type, fieldIndex int, tagName string, parsed []Metadata) {
	key := fieldRulesKey{parent: parent, index: fieldIndex, tagName: tagName}
	c.c.Store(key, parsed)
}
