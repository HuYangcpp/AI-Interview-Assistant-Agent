package utils

import (
	"encoding/json"
	"strings"
)

// ParseStringArrayJSON parses a JSON-encoded string array (e.g. `["a","b"]`) into []string.
// Returns nil on empty input or parse error.
func ParseStringArrayJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	return arr
}
