package e2e

import (
	"fmt"
	"strings"
)

// allowedGitCommands defines the allowed git commands and their arguments
var allowedGitCommands = map[string][]string{
	"clone":  {"clone"},
	"config": {"config", "user.name", "user.email"},
	"add":    {"add", "."},
	"commit": {"commit", "-m", "--author"},
	"push":   {"push", "origin", "main"},
}

// validateGitArgs checks if the git command and its arguments are allowed
func validateGitArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no git arguments provided")
	}

	cmd := args[0]
	allowedArgs, ok := allowedGitCommands[cmd]
	if !ok {
		return fmt.Errorf("git command not allowed: %s", cmd)
	}

	// Special handling for specific commands
	switch cmd {
	case "clone":
		if len(args) != 3 || !strings.HasPrefix(args[1], "https://") {
			return fmt.Errorf("invalid clone command format")
		}
		return nil
	case "config":
		if len(args) != 3 || !strings.HasPrefix(args[1], "user.") {
			return fmt.Errorf("invalid config command format")
		}
		return nil
	}

	// Validate other commands' arguments
	for _, arg := range args {
		valid := false
		for _, allowedArg := range allowedArgs {
			if strings.HasPrefix(arg, allowedArg) {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("git argument not allowed: %s", arg)
		}
	}

	return nil
}
