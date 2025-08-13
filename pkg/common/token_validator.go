package common

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// Token type prefixes
	TokenTypePATClassic     = "ghp"
	TokenTypePATFineGrained = "github_pat"
	TokenTypeOAuth          = "gho"
	TokenTypeInstallation   = "ghs"
	TokenTypeUserToServer   = "ghu"
)

var (
	// Token format patterns
	patClassicPattern      = regexp.MustCompile(`^ghp_[A-Za-z0-9]{36}$`)
	patFineGrainedPattern  = regexp.MustCompile(`^github_pat_[A-Za-z0-9_]{82}$`)
	oauthTokenPattern      = regexp.MustCompile(`^gho_[A-Za-z0-9]{36}$`)
	installTokenPattern    = regexp.MustCompile(`^ghs_[A-Za-z0-9]{36}$`)
	userServerTokenPattern = regexp.MustCompile(`^ghu_[A-Za-z0-9]{36}$`)
)

// TokenInfo contains information about a validated token
type TokenInfo struct {
	Type   string
	Valid  bool
	Reason string
}

// ValidateGitHubToken validates the format of a GitHub token
func ValidateGitHubToken(token string) (*TokenInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Remove any whitespace
	token = strings.TrimSpace(token)

	// Check length bounds (min 40 for classic PAT, max ~95 for fine-grained PAT)
	if len(token) < 40 || len(token) > 100 {
		return nil, fmt.Errorf("token length %d is outside valid range (40-100)", len(token))
	}

	// Check for different token types
	switch {
	case strings.HasPrefix(token, "github_pat_"):
		if patFineGrainedPattern.MatchString(token) {
			return &TokenInfo{
				Type:  TokenTypePATFineGrained,
				Valid: true,
			}, nil
		}
		return nil, fmt.Errorf("invalid fine-grained PAT format")

	case strings.HasPrefix(token, "ghp_"):
		if patClassicPattern.MatchString(token) {
			return &TokenInfo{
				Type:  TokenTypePATClassic,
				Valid: true,
			}, nil
		}
		return nil, fmt.Errorf("invalid classic PAT format")

	case strings.HasPrefix(token, "gho_"):
		if oauthTokenPattern.MatchString(token) {
			return &TokenInfo{
				Type:  TokenTypeOAuth,
				Valid: true,
			}, nil
		}
		return nil, fmt.Errorf("invalid OAuth token format")

	case strings.HasPrefix(token, "ghs_"):
		if installTokenPattern.MatchString(token) {
			return &TokenInfo{
				Type:  TokenTypeInstallation,
				Valid: true,
			}, nil
		}
		return nil, fmt.Errorf("invalid installation token format")

	case strings.HasPrefix(token, "ghu_"):
		if userServerTokenPattern.MatchString(token) {
			return &TokenInfo{
				Type:  TokenTypeUserToServer,
				Valid: true,
			}, nil
		}
		return nil, fmt.Errorf("invalid user-to-server token format")
	}

	return nil, fmt.Errorf("token does not match any known GitHub token format")
}

// IsValidGitHubToken is a simple validation check
func IsValidGitHubToken(token string) bool {
	_, err := ValidateGitHubToken(token)
	return err == nil
}
