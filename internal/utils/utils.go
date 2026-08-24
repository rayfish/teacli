// Package utils contains shared utilities for the CLI
package utils

import (
	"fmt"
	"strings"
)

// parseRepoArg parses "owner/repo" string into separate owner and repo
func ParseRepoArg(arg string) (owner, repo string, err error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format: %s (expected owner/repo)", arg)
	}
	return parts[0], parts[1], nil
}

// truncate truncates a string to maxLen characters
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// parseInt parses a string to int
func ParseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// parseInt64 parses a string to int64
func ParseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
