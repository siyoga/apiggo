package codegen

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"
)

// pascal converts an arbitrary identifier (snake_case, kebab-case, camelCase,
// spaced) to PascalCase suitable for an exported Go identifier.
func pascal(s string) string {
	var b strings.Builder
	upNext := true
	for _, r := range s {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.' || r == '/':
			upNext = true
		case unicode.IsUpper(r):
			b.WriteRune(r) // preserve camelCase word boundary
			upNext = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if upNext {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(r)
			}
			upNext = false
		default:
			upNext = true
		}
	}

	out := b.String()
	if out == "" {
		return "X"
	}
	if unicode.IsDigit(rune(out[0])) {
		out = "X" + out
	}
	return out
}

// componentName maps a $ref ("#/components/schemas/User") to its Go type name.
func componentName(ref string) string {
	seg := ref
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		seg = ref[i+1:]
	}
	return pascal(seg)
}

// statusWord renders an HTTP status code as a PascalCase word ("NotFound").
func statusWord(code int) string {
	if t := http.StatusText(code); t != "" {
		return pascal(t)
	}
	return "Status" + strconv.Itoa(code)
}

// enumValueName builds the constant name for an enum member.
func enumValueName(typeName string, v any) string {
	return typeName + pascal(toStr(v))
}

func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		return ""
	}
}
