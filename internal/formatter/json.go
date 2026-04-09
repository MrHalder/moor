package formatter

import (
	"encoding/json"

	"github.com/ashutosh/moor/internal/scanner"
)

// FormatJSON renders port info as indented JSON.
func FormatJSON(result *scanner.ScanResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
