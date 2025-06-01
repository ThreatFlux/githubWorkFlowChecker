package common

import (
	"strings"
	"testing"
)

func TestValidateGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectValid bool
		expectType  string
		expectError string
	}{
		// Valid tokens
		{
			name:        "valid classic PAT",
			token:       "ghp_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
			expectType:  TokenTypePATClassic,
		},
		{
			name:        "valid fine-grained PAT",
			token:       "github_pat_11AAAAAA0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			expectValid: true,
			expectType:  TokenTypePATFineGrained,
		},
		{
			name:        "valid OAuth token",
			token:       "gho_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
			expectType:  TokenTypeOAuth,
		},
		{
			name:        "valid installation token",
			token:       "ghs_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
			expectType:  TokenTypeInstallation,
		},
		{
			name:        "valid user-to-server token",
			token:       "ghu_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
			expectType:  TokenTypeUserToServer,
		},

		// Invalid tokens - empty/whitespace
		{
			name:        "empty token",
			token:       "",
			expectValid: false,
			expectError: "token cannot be empty",
		},
		{
			name:        "whitespace only token",
			token:       "   ",
			expectValid: false,
			expectError: "token length 0 is outside valid range",
		},

		// Invalid tokens - length issues
		{
			name:        "token too short",
			token:       "short",
			expectValid: false,
			expectError: "token length 5 is outside valid range (40-100)",
		},
		{
			name:        "token too long",
			token:       strings.Repeat("a", 101),
			expectValid: false,
			expectError: "token length 101 is outside valid range (40-100)",
		},

		// Invalid tokens - format issues
		{
			name:        "invalid classic PAT format - invalid chars",
			token:       "ghp_16C7e42F292c6912E7710c838347Ae178B!!",
			expectValid: false,
			expectError: "invalid classic PAT format",
		},
		{
			name:        "invalid classic PAT format - wrong length",
			token:       "ghp_16C7e42F292c6912E7710c838347Ae178B4abc", // 42 chars
			expectValid: false,
			expectError: "invalid classic PAT format",
		},
		{
			name:        "invalid fine-grained PAT format - wrong suffix length",
			token:       "github_pat_11AAAAAA0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			expectValid: false,
			expectError: "invalid fine-grained PAT format",
		},
		{
			name:        "invalid fine-grained PAT format - invalid chars",
			token:       "github_pat_11AAAAAA0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA!!!",
			expectValid: false,
			expectError: "invalid fine-grained PAT format",
		},
		{
			name:        "invalid OAuth token format - wrong length",
			token:       "gho_16C7e42F292c6912E7710c838347Ae178B4abc", // 42 chars
			expectValid: false,
			expectError: "invalid OAuth token format",
		},
		{
			name:        "invalid OAuth token format - invalid chars",
			token:       "gho_16C7e42F292c6912E7710c838347Ae178B!!",
			expectValid: false,
			expectError: "invalid OAuth token format",
		},
		{
			name:        "invalid installation token format - wrong length",
			token:       "ghs_16C7e42F292c6912E7710c838347Ae178B4abc", // 42 chars
			expectValid: false,
			expectError: "invalid installation token format",
		},
		{
			name:        "invalid user-to-server token format - wrong length",
			token:       "ghu_16C7e42F292c6912E7710c838347Ae178B4abc", // 42 chars
			expectValid: false,
			expectError: "invalid user-to-server token format",
		},

		// Invalid tokens - unknown prefix
		{
			name:        "unknown prefix format",
			token:       "ghx_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: false,
			expectError: "token does not match any known GitHub token format",
		},
		{
			name:        "random string",
			token:       "not-a-token-format-1234567890abcdef1234567890",
			expectValid: false,
			expectError: "token does not match any known GitHub token format",
		},

		// Edge cases
		{
			name:        "token with leading/trailing whitespace",
			token:       "  ghp_16C7e42F292c6912E7710c838347Ae178B4a  ",
			expectValid: true,
			expectType:  TokenTypePATClassic,
		},
		{
			name:        "classic PAT",
			token:       "ghp_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
			expectType:  TokenTypePATClassic,
		},
		{
			name:        "maximum length fine-grained PAT",
			token:       "github_pat_11AAAAAA0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			expectValid: true,
			expectType:  TokenTypePATFineGrained,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenInfo, err := ValidateGitHubToken(tt.token)

			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected valid token, got error: %v", err)
					return
				}
				if tokenInfo == nil {
					t.Errorf("Expected TokenInfo, got nil")
					return
				}
				if tokenInfo.Type != tt.expectType {
					t.Errorf("Expected token type %s, got %s", tt.expectType, tokenInfo.Type)
				}
				if !tokenInfo.Valid {
					t.Errorf("Expected token to be valid, got false")
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid token, got nil")
					return
				}
				if tt.expectError != "" && !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
				}
			}
		})
	}
}

func TestIsValidGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectValid bool
	}{
		{
			name:        "valid classic PAT",
			token:       "ghp_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
		},
		{
			name:        "valid OAuth token",
			token:       "gho_16C7e42F292c6912E7710c838347Ae178B4a",
			expectValid: true,
		},
		{
			name:        "invalid token",
			token:       "invalid-token",
			expectValid: false,
		},
		{
			name:        "empty token",
			token:       "",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidGitHubToken(tt.token)
			if result != tt.expectValid {
				t.Errorf("Expected %v, got %v for token validation", tt.expectValid, result)
			}
		})
	}
}

func TestTokenInfoStruct(t *testing.T) {
	// Test TokenInfo structure
	tokenInfo := &TokenInfo{
		Type:   TokenTypePATClassic,
		Valid:  true,
		Reason: "test reason",
	}

	if tokenInfo.Type != TokenTypePATClassic {
		t.Errorf("Expected Type %s, got %s", TokenTypePATClassic, tokenInfo.Type)
	}
	if !tokenInfo.Valid {
		t.Errorf("Expected Valid to be true, got false")
	}
	if tokenInfo.Reason != "test reason" {
		t.Errorf("Expected Reason 'test reason', got %s", tokenInfo.Reason)
	}
}

func TestTokenTypeConstants(t *testing.T) {
	// Verify token type constants are as expected
	expectedTypes := map[string]string{
		TokenTypePATClassic:     "ghp",
		TokenTypePATFineGrained: "github_pat",
		TokenTypeOAuth:          "gho",
		TokenTypeInstallation:   "ghs",
		TokenTypeUserToServer:   "ghu",
	}

	for constant, expected := range expectedTypes {
		if constant != expected {
			t.Errorf("Expected constant %s to equal %s", constant, expected)
		}
	}
}

// Benchmark tests for performance
func BenchmarkValidateGitHubToken(b *testing.B) {
	tokens := []string{
		"ghp_16C7e42F292c6912E7710c838347Ae178B4a",                                                      // Classic PAT
		"github_pat_11AAAAAA0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // Fine-grained PAT
		"gho_16C7e42F292c6912E7710c838347Ae178B4a",                                                      // OAuth
		"invalid-token", // Invalid
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, token := range tokens {
			_, _ = ValidateGitHubToken(token)
		}
	}
}

func BenchmarkIsValidGitHubToken(b *testing.B) {
	token := "ghp_16C7e42F292c6912E7710c838347Ae178B4a"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidGitHubToken(token)
	}
}
