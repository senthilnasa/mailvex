package template

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	varTagRegex      = regexp.MustCompile(`\{\{\{([a-zA-Z0-9_\.]+)\}\}\}|\{\{\&([a-zA-Z0-9_\.]+)\}\}|\{\{([a-zA-Z0-9_\.]+)\}\}`)
	sectionOpenRegex = regexp.MustCompile(`\{\{#([a-zA-Z0-9_\.]+)\}\}`)
	sectionCloseTag  = func(name string) string { return fmt.Sprintf("{{/%s}}", name) }
	invertedOpenTag  = func(name string) string { return fmt.Sprintf("{{^%s}}", name) }
)

type Options struct {
	StrictVariables bool // if true, returns error when a variable is missing
	AutoEscape      bool // if true, double curly {{var}} is HTML escaped
}

// Render processes a template string with given variables map and options
func Render(tmpl string, vars map[string]interface{}, opts Options) (string, error) {
	if tmpl == "" {
		return "", nil
	}

	result := tmpl

	// 1. Process sections {{#name}} ... {{/name}} and inverted {{^name}} ... {{/name}}
	// We handle sections recursively or sequentially
	for {
		loc := sectionOpenRegex.FindStringSubmatchIndex(result)
		if loc == nil {
			break
		}

		secName := result[loc[2]:loc[3]]
		closeTag := sectionCloseTag(secName)
		closeIdx := strings.Index(result[loc[1]:], closeTag)
		if closeIdx == -1 {
			return "", fmt.Errorf("unclosed section {{#%s}} in template", secName)
		}

		secStart := loc[0]
		contentStart := loc[1]
		contentEnd := loc[1] + closeIdx
		secEnd := contentEnd + len(closeTag)

		secContent := result[contentStart:contentEnd]
		val, exists := lookupVar(vars, secName)

		var renderedSec strings.Builder
		if exists && val != nil {
			switch v := val.(type) {
			case bool:
				if v {
					sub, err := Render(secContent, vars, opts)
					if err != nil {
						return "", err
					}
					renderedSec.WriteString(sub)
				}
			case []map[string]interface{}:
				for _, item := range v {
					// Merge child vars with parent
					merged := make(map[string]interface{})
					for pk, pv := range vars {
						merged[pk] = pv
					}
					for ck, cv := range item {
						merged[ck] = cv
					}
					sub, err := Render(secContent, merged, opts)
					if err != nil {
						return "", err
					}
					renderedSec.WriteString(sub)
				}
			case []interface{}:
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						merged := make(map[string]interface{})
						for pk, pv := range vars {
							merged[pk] = pv
						}
						for ck, cv := range m {
							merged[ck] = cv
						}
						sub, err := Render(secContent, merged, opts)
						if err != nil {
							return "", err
						}
						renderedSec.WriteString(sub)
					}
				}
			case string:
				if v != "" {
					sub, err := Render(secContent, vars, opts)
					if err != nil {
						return "", err
					}
					renderedSec.WriteString(sub)
				}
			default:
				sub, err := Render(secContent, vars, opts)
				if err != nil {
					return "", err
				}
				renderedSec.WriteString(sub)
			}
		}

		result = result[:secStart] + renderedSec.String() + result[secEnd:]
	}

	// 2. Process variable tags
	var renderErr error
	rendered := varTagRegex.ReplaceAllStringFunc(result, func(match string) string {
		if renderErr != nil {
			return match
		}

		var varName string
		isRaw := false

		if strings.HasPrefix(match, "{{{") && strings.HasSuffix(match, "}}}") {
			varName = strings.TrimSpace(match[3 : len(match)-3])
			isRaw = true
		} else if strings.HasPrefix(match, "{{&") && strings.HasSuffix(match, "}}") {
			varName = strings.TrimSpace(match[3 : len(match)-2])
			isRaw = true
		} else if strings.HasPrefix(match, "{{") && strings.HasSuffix(match, "}}") {
			varName = strings.TrimSpace(match[2 : len(match)-2])
		}

		val, exists := lookupVar(vars, varName)
		if !exists || val == nil {
			if opts.StrictVariables {
				renderErr = fmt.Errorf("template_variable_missing: %s", varName)
				return match
			}
			return ""
		}

		strVal := fmt.Sprintf("%v", val)
		if opts.AutoEscape && !isRaw {
			return html.EscapeString(strVal)
		}
		return strVal
	})

	if renderErr != nil {
		return "", renderErr
	}

	return rendered, nil
}

func lookupVar(vars map[string]interface{}, key string) (interface{}, bool) {
	if vars == nil {
		return nil, false
	}

	parts := strings.Split(key, ".")
	var current interface{} = vars

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	return current, true
}
