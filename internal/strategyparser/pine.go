package strategyparser

import (
	"fmt"
	"strconv"
	"strings"
)

// Param describes one parsed Pine input.
type Param struct {
	Name    string
	Type    string
	Default any
	Min     *float64
	Max     *float64
	Step    *float64
	Options []string
	Label   string
}

// ParsePineInputs extracts input.* declarations from Pine Script.
func ParsePineInputs(content string) ([]Param, error) {
	lines := strings.Split(content, "\n")
	params := make([]Param, 0, 32)

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") || !strings.Contains(line, "= input.") {
			continue
		}
		p, ok, err := parseInputLine(line)
		if err != nil {
			return nil, err
		}
		if ok {
			params = append(params, p)
		}
	}
	return params, nil
}

func parseInputLine(line string) (Param, bool, error) {
	idx := strings.Index(line, "= input.")
	if idx < 0 {
		return Param{}, false, nil
	}
	name := strings.TrimSpace(line[:idx])
	if name == "" {
		return Param{}, false, nil
	}

	right := strings.TrimSpace(line[idx+len("= input."):])
	open := strings.Index(right, "(")
	if open < 0 {
		return Param{}, false, nil
	}
	typ := strings.TrimSpace(right[:open])
	if typ == "" {
		return Param{}, false, nil
	}

	argStart := open + 1
	argEnd := findMatchingParen(right, open)
	if argEnd < 0 {
		return Param{}, false, fmt.Errorf("unbalanced input parentheses in line: %s", line)
	}
	argsRaw := right[argStart:argEnd]
	args := splitTopLevel(argsRaw, ',')
	if len(args) == 0 {
		return Param{}, false, nil
	}

	p := Param{
		Name: strings.TrimSpace(name),
		Type: strings.TrimSpace(typ),
	}

	p.Default = parseDefault(strings.TrimSpace(args[0]), p.Type)
	if len(args) > 1 {
		p.Label = trimQuotes(strings.TrimSpace(args[1]))
	}

	for _, token := range args[2:] {
		part := strings.TrimSpace(token)
		if part == "" || !strings.Contains(part, "=") {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		switch key {
		case "minval":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				p.Min = &f
			}
		case "maxval":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				p.Max = &f
			}
		case "step":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				p.Step = &f
			}
		case "options":
			p.Options = parseOptionsArray(val)
		}
	}

	return p, true, nil
}

func findMatchingParen(s string, open int) int {
	depth := 0
	inQuotes := false
	escaped := false
	for i := open; i < len(s); i++ {
		ch := s[i]
		if inQuotes {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inQuotes = false
			}
			continue
		}
		if ch == '"' {
			inQuotes = true
			continue
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopLevel(s string, sep byte) []string {
	var out []string
	start := 0
	parenDepth := 0
	bracketDepth := 0
	inQuotes := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuotes {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inQuotes = false
			}
			continue
		}

		switch ch {
		case '"':
			inQuotes = true
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case sep:
			if parenDepth == 0 && bracketDepth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

func parseDefault(v string, typ string) any {
	v = strings.TrimSpace(v)
	switch typ {
	case "int":
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	case "float":
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	case "bool":
		if n, err := strconv.ParseBool(v); err == nil {
			return n
		}
	case "string":
		return trimQuotes(v)
	}
	return trimQuotes(v)
}

func parseOptionsArray(v string) []string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(v, "["), "]"))
	if body == "" {
		return nil
	}
	parts := splitTopLevel(body, ',')
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := trimQuotes(strings.TrimSpace(p))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func trimQuotes(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"`)
	return strings.TrimSpace(v)
}
