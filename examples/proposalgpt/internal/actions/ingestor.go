package actions

import (
	"fmt"
	"os"
)

// IngestRFP reads the content of an RFP markdown file.
func IngestRFP(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read RFP file: %w", err)
	}
	return string(content), nil
}
