package gen

import (
	"os"

	"github.com/duynguyendang/manglekit-wip/internal/resources/icl"
)

// GetICLContent returns the In-Context Learning content.
// If userPath is provided, it reads from that file.
// Otherwise, it returns the embedded golden rules.
func GetICLContent(userPath string) (string, error) {
	if userPath != "" {
		content, err := os.ReadFile(userPath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return icl.GetGoldenRules(), nil
}
