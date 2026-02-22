// Package ir defines the Intermediate Representation types and
// deterministic JSON serializer for the Agentz toolchain.
package ir

import (
	"encoding/json"
	"sort"
)

// Document is the top-level IR container.
type Document struct {
	IRVersion   string     `json:"ir_version"`
	LangVersion string     `json:"lang_version"`
	Package     Package    `json:"package"`
	Resources   []Resource `json:"resources"`
	Policies    []Policy   `json:"policies,omitempty"`
	Bindings    []Binding  `json:"bindings,omitempty"`
}

// Package holds resolved package metadata.
type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Resource is a fully-resolved resource in the IR.
type Resource struct {
	Kind       string                 `json:"kind"`
	Name       string                 `json:"name"`
	FQN        string                 `json:"fqn"`
	Attributes map[string]interface{} `json:"attributes"`
	References []string               `json:"references,omitempty"`
	Hash       string                 `json:"hash"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Policy holds resolved policy rules.
type Policy struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

// Rule is a single policy rule.
type Rule struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
	Subject  string `json:"subject"`
}

// Binding is a target adapter binding.
type Binding struct {
	Name    string                 `json:"name"`
	Adapter string                 `json:"adapter"`
	Default bool                   `json:"default,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// SortResources sorts resources by kind then name for deterministic output.
func SortResources(resources []Resource) {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Kind != resources[j].Kind {
			return resources[i].Kind < resources[j].Kind
		}
		return resources[i].Name < resources[j].Name
	})
}

// MarshalJSON produces deterministic JSON with sorted keys and 2-space indentation.
func (d *Document) MarshalJSON() ([]byte, error) {
	SortResources(d.Resources)
	type Alias Document
	return json.MarshalIndent((*Alias)(d), "", "  ")
}

// SerializeCanonical produces a canonical JSON serialization of attributes
// with sorted keys and no whitespace, suitable for hashing.
func SerializeCanonical(attrs map[string]interface{}) ([]byte, error) {
	ordered := sortedMap(attrs)
	return json.Marshal(ordered)
}

// sortedMap returns an ordered representation that json.Marshal will
// serialize with sorted keys.
func sortedMap(m map[string]interface{}) interface{} {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]keyValue, 0, len(keys))
	for _, k := range keys {
		v := m[k]
		if sub, ok := v.(map[string]interface{}); ok {
			v = sortedMap(sub)
		}
		ordered = append(ordered, keyValue{Key: k, Value: v})
	}
	return orderedMap(ordered)
}

type keyValue struct {
	Key   string
	Value interface{}
}

type orderedMap []keyValue

func (o orderedMap) MarshalJSON() ([]byte, error) {
	buf := []byte{'{'}
	for i, kv := range o {
		if i > 0 {
			buf = append(buf, ',')
		}
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = append(buf, val...)
	}
	buf = append(buf, '}')
	return buf, nil
}
