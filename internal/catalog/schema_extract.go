package catalog

import (
	"encoding/json"
	"sort"
)

// SchemaArgNames returns distinct JSON Schema property keys (recursive, shallow + nested objects).
func SchemaArgNames(schemaJSON string) []string {
	if schemaJSON == "" || schemaJSON == "{}" {
		return nil
	}
	var root any
	if err := json.Unmarshal([]byte(schemaJSON), &root); err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	walkSchema(root, seen)
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func walkSchema(v any, seen map[string]struct{}) {
	switch x := v.(type) {
	case map[string]any:
		if props, ok := x["properties"].(map[string]any); ok {
			for name := range props {
				if name != "" && name != "_meta" {
					seen[name] = struct{}{}
					walkSchema(props[name], seen)
				}
			}
		}
		if items, ok := x["items"]; ok {
			walkSchema(items, seen)
		}
		for _, k := range []string{"allOf", "anyOf", "oneOf"} {
			if arr, ok := x[k].([]any); ok {
				for _, el := range arr {
					walkSchema(el, seen)
				}
			}
		}
	}
}
