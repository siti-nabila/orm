package lock

import "strings"

func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func BuildKeyLock(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, v := range parts {
		v = Normalize(v)
		if v != "" {
			clean = append(clean, v)
		}
	}
	return strings.Join(clean, ":")
}
