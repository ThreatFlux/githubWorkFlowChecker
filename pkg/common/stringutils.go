package common

import "strings"

// IsHexString checks if a string is a valid hexadecimal string (for commit SHAs)
func IsHexString(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// ContainsAny checks if a string contains any of the given substrings
func ContainsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ContainsAll checks if a string contains all of the given substrings
func ContainsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// TrimPrefixAny trims any of the given prefixes from the string
func TrimPrefixAny(s string, prefixes ...string) string {
	result := s
	for _, prefix := range prefixes {
		if strings.HasPrefix(result, prefix) {
			result = strings.TrimPrefix(result, prefix)
			break
		}
	}
	return result
}

// TrimSuffixAny trims any of the given suffixes from the string
func TrimSuffixAny(s string, suffixes ...string) string {
	result := s
	for _, suffix := range suffixes {
		if strings.HasSuffix(result, suffix) {
			result = strings.TrimSuffix(result, suffix)
			break
		}
	}
	return result
}

// SplitAndTrim splits a string by the given separator and trims each part
func SplitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// JoinNonEmpty joins non-empty strings with the given separator
func JoinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, part := range parts {
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, sep)
}
