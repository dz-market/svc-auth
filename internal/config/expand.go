package config

import (
	"regexp"
	"strings"
)

var refPattern = regexp.MustCompile(`\$\$|\$\{[^{}]*\}`)

func expandLeaves(node any, lookup LookupFunc, missing *[]string) any {
	switch v := node.(type) {
	case string:
		return expand(v, lookup, missing)

	case map[string]any:
		for key, child := range v {
			v[key] = expandLeaves(child, lookup, missing)
		}

		return v

	case []any:
		for i, child := range v {
			v[i] = expandLeaves(child, lookup, missing)
		}

		return v

	default:
		return node
	}
}

func expand(s string, lookup LookupFunc, missing *[]string) string {
	return refPattern.ReplaceAllStringFunc(
		s, func(match string) string {
			if match == "$$" {
				return "$"
			}

			name, def, hasDefault := strings.Cut(match[2:len(match)-1], ":")

			if val, ok := lookup(name); ok && val != "" {
				return val
			}

			if hasDefault {
				return def
			}

			if name != "" {
				*missing = append(*missing, name)
			}

			return ""
		},
	)
}
