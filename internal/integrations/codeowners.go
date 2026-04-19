package integrations

import (
	"strings"
)

var codeownersCandidatePaths = []string{
	".github/CODEOWNERS",
	"CODEOWNERS",
	"docs/CODEOWNERS",
}

type CodeownersRule struct {
	Pattern string   `json:"pattern"`
	Owners  []string `json:"owners"`
}

type RepositoryOwnershipImport struct {
	Provider string           `json:"provider,omitempty"`
	Source   string           `json:"source,omitempty"`
	Status   string           `json:"status"`
	FilePath string           `json:"file_path,omitempty"`
	Ref      string           `json:"ref,omitempty"`
	Revision string           `json:"revision,omitempty"`
	Owners   []string         `json:"owners,omitempty"`
	Rules    []CodeownersRule `json:"rules,omitempty"`
	Error    string           `json:"error,omitempty"`
}

func ParseCODEOWNERS(content string) []CodeownersRule {
	lines := strings.Split(content, "\n")
	rules := make([]CodeownersRule, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(stripInlineComment(line))
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		owners := make([]string, 0, len(fields)-1)
		for _, owner := range fields[1:] {
			owner = strings.TrimSpace(owner)
			if owner == "" {
				continue
			}
			owners = append(owners, owner)
		}
		if len(owners) == 0 {
			continue
		}
		rules = append(rules, CodeownersRule{
			Pattern: strings.TrimSpace(fields[0]),
			Owners:  owners,
		})
	}
	return rules
}

func CollectCodeownersOwners(rules []CodeownersRule) []string {
	result := make([]string, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		for _, owner := range rule.Owners {
			owner = strings.TrimSpace(owner)
			if owner == "" {
				continue
			}
			if _, exists := seen[owner]; exists {
				continue
			}
			seen[owner] = struct{}{}
			result = append(result, owner)
		}
	}
	return result
}

func stripInlineComment(line string) string {
	for index, r := range line {
		if r == '#' && (index == 0 || line[index-1] == ' ' || line[index-1] == '\t') {
			return line[:index]
		}
	}
	return line
}
