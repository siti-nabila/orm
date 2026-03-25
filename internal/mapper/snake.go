package mapper

import "strings"

func toSnake(s string) string {

	var out []rune

	for i, r := range s {
		if i > 0 &&
			r >= 'A' &&
			r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, r)
	}
	return strings.ToLower(string(out))
}
