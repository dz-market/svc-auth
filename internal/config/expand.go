package config

import (
	"regexp"
	"strings"
)

var refPattern = regexp.MustCompile(`\$\$|\$\{[^{}]*\}`)

func expandNode(node any, lookup lookupFunc, missing *[]string) any {
	switch v := node.(type) {
	case string:
		return expand(v, lookup, missing)

	case map[string]any:
		for key, child := range v {
			v[key] = expandNode(child, lookup, missing)
		}

		return v

	case []any:
		for i, child := range v {
			v[i] = expandNode(child, lookup, missing)
		}

		return v

	default:
		return node
	}
}

func expand(s string, lookup lookupFunc, missing *[]string) string {
	return refPattern.ReplaceAllStringFunc(
		s, func(match string) string {
			if match == "$$" {
				return "$"
			}

			name, def, hasDefault := strings.Cut(match[2:len(match)-1], ":-")

			if name == "" {
				*missing = append(*missing, match)

				return ""
			}

			if val, ok := lookup(name); ok {
				return val
			}

			if hasDefault {
				return def
			}

			*missing = append(*missing, name)

			return ""
		},
	)
}
