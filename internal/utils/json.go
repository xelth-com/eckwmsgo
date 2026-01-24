package utils

import (
	"strings"
)

// SanitizeJSON cleans raw AI output to extract valid JSON
// It removes Markdown code blocks (```json ... ```) and whitespace
func SanitizeJSON(input string) string {
	cleaned := strings.TrimSpace(input)

	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
	}

	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}

	return strings.TrimSpace(cleaned)
}
