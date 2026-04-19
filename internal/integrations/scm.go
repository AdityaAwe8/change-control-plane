package integrations

import (
	"context"
	"regexp"
	"strings"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var scmIssueKeyPattern = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)

type SCMRepository struct {
	Provider      string         `json:"provider,omitempty"`
	ExternalID    string         `json:"external_id,omitempty"`
	Namespace     string         `json:"namespace,omitempty"`
	Owner         string         `json:"owner,omitempty"`
	Name          string         `json:"name"`
	FullName      string         `json:"full_name"`
	HTMLURL       string         `json:"html_url"`
	DefaultBranch string         `json:"default_branch"`
	Private       bool           `json:"private"`
	Archived      bool           `json:"archived"`
	Metadata      types.Metadata `json:"metadata,omitempty"`
}

type SCMChangedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status,omitempty"`
	Additions int    `json:"additions,omitempty"`
	Deletions int    `json:"deletions,omitempty"`
	Changes   int    `json:"changes,omitempty"`
}

type SCMWebhookChange struct {
	Repository SCMRepository    `json:"repository"`
	Summary    string           `json:"summary"`
	Branch     string           `json:"branch,omitempty"`
	Tag        string           `json:"tag,omitempty"`
	CommitSHA  string           `json:"commit_sha,omitempty"`
	ChangeType string           `json:"change_type"`
	FileCount  int              `json:"file_count"`
	Files      []SCMChangedFile `json:"files,omitempty"`
	IssueKeys  []string         `json:"issue_keys,omitempty"`
	Approvers  []string         `json:"approvers,omitempty"`
	Reviewers  []string         `json:"reviewers,omitempty"`
	Labels     []string         `json:"labels,omitempty"`
	Metadata   types.Metadata   `json:"metadata,omitempty"`
}

type SCMWebhookResult struct {
	Operation     string            `json:"operation"`
	Summary       string            `json:"summary"`
	Details       []string          `json:"details,omitempty"`
	ResourceCount int               `json:"resource_count"`
	Change        *SCMWebhookChange `json:"change,omitempty"`
}

type SCMWebhookRegistration struct {
	Provider        string         `json:"provider"`
	ScopeIdentifier string         `json:"scope_identifier,omitempty"`
	CallbackURL     string         `json:"callback_url"`
	ExternalHookID  string         `json:"external_hook_id,omitempty"`
	Status          string         `json:"status"`
	DeliveryHealth  string         `json:"delivery_health,omitempty"`
	Details         []string       `json:"details,omitempty"`
	Metadata        types.Metadata `json:"metadata,omitempty"`
}

type SCMClient interface {
	TestConnection(ctx context.Context, scope string) ([]string, error)
	DiscoverRepositories(ctx context.Context, scope string) ([]SCMRepository, error)
}

func stringValue(value any) string {
	result, _ := value.(string)
	return strings.TrimSpace(result)
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(value)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func extractSCMIssueKeys(values ...string) []string {
	joined := strings.ToUpper(strings.Join(values, "\n"))
	matches := scmIssueKeyPattern.FindAllString(joined, -1)
	result := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		match = strings.TrimSpace(match)
		if match == "" {
			continue
		}
		if _, exists := seen[match]; exists {
			continue
		}
		seen[match] = struct{}{}
		result = append(result, match)
	}
	return result
}

func flattenCommitFiles(added, removed, modified []string) []SCMChangedFile {
	files := make([]SCMChangedFile, 0, len(added)+len(removed)+len(modified))
	for _, name := range added {
		files = append(files, SCMChangedFile{Filename: name, Status: "added"})
	}
	for _, name := range removed {
		files = append(files, SCMChangedFile{Filename: name, Status: "removed"})
	}
	for _, name := range modified {
		files = append(files, SCMChangedFile{Filename: name, Status: "modified"})
	}
	return files
}

func uniqueSCMFiles(files []SCMChangedFile) []SCMChangedFile {
	result := make([]SCMChangedFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		name := strings.TrimSpace(file.Filename)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, file)
	}
	return result
}
