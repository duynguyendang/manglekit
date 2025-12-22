package knowledge

import (
	"bufio"
	"os"
	"strings"
)

// Playbook represents a solution design strategy.
// Playbook represents a solution design strategy.
type Playbook struct {
	ID           string `mangle:"playbook_id"`
	Name         string
	Description  string
	CriticalNFRs []string `mangle:"nfrs"`
	Risks        []string
	ArchPattern  string `mangle:"arch_pattern"`
	RawContent   string `mangle:"-"`
}

// LoadPlaybookFromFile parses a markdown playbook file.
func LoadPlaybookFromFile(id, filepath string) (*Playbook, error) {
	contentBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)

	pb := &Playbook{
		ID:         id,
		RawContent: content,
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	var inNFRs, inRisks bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "# ") {
			pb.Name = strings.TrimPrefix(line, "# ")
			continue
		}

		if strings.Contains(line, "**Description:**") {
			parts := strings.SplitN(line, "**Description:**", 2)
			if len(parts) > 1 {
				pb.Description = strings.TrimSpace(parts[1])
			}
			continue
		}

		if strings.Contains(line, "**Core Architectural Pattern:**") {
			parts := strings.SplitN(line, "**Core Architectural Pattern:**", 2)
			if len(parts) > 1 {
				pb.ArchPattern = strings.TrimSpace(parts[1])
			}
			continue
		}

		if strings.Contains(line, "**Critical NFRs to Prioritize") {
			inNFRs = true
			inRisks = false
			continue
		}

		if strings.Contains(line, "**Key Risks & Mitigations:**") {
			inRisks = true
			inNFRs = false
			continue
		}

		// Stop flags if we hit a new section
		if strings.HasPrefix(line, "- **I.") || strings.HasPrefix(line, "- **II.") || strings.HasPrefix(line, "- **III.") {
			inNFRs = false
			inRisks = false
		}

		if inNFRs {
			if strings.Contains(line, "- **") && strings.Contains(line, ":**") {
				start := strings.Index(line, "**") + 2
				end := strings.Index(line, ":**")
				if start < end {
					nfr := line[start:end]
					pb.CriticalNFRs = append(pb.CriticalNFRs, nfr)
				}
			}
		}

		if inRisks {
			// **Risk: Scope Creep Delaying MVP.**
			if strings.Contains(line, "**Risk:") {
				start := strings.Index(line, "**Risk:") + 7
				end := strings.LastIndex(line, "**")

				// Try to find the first dot e.g. "Risk: Name."
				dotIdx := strings.Index(line[start:], ".")
				if dotIdx != -1 {
					// Using dot index relative to start
					endVal := start + dotIdx
					if endVal < end {
						// We found a dot before the closing **, use that
						// But wait, "Risk: Name.**" -> dot is usually inside?
						// The markdown is: **Risk: Name.**
						// So end of bold is just after the dot or the dot is inside.
					}
				}

				// Fallback to extraction between Risk: and .** or just **
				if start < end {
					content := line[start:end]
					// Remove trailing dot if present
					content = strings.TrimSuffix(content, ".")
					pb.Risks = append(pb.Risks, strings.TrimSpace(content))
				}
			}
		}
	}

	return pb, nil
}
