package updater

import "strings"

// isHexString checks if a string is a valid hexadecimal string (for commit SHAs)
func isHexString(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}
