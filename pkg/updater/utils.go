package updater

import "github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"

// isHexString checks if a string is a valid hexadecimal string (for commit SHAs)
func isHexString(s string) bool {
	return common.IsHexString(s)
}
