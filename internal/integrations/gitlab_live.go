package integrations

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type GitLabRepository = SCMRepository
type GitLabChangedFile = SCMChangedFile
type GitLabWebhookChange = SCMWebhookChange
type GitLabWebhookResult = SCMWebhookResult

type GitLabClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewGitLabClient(baseURL, token string) GitLabClient {
	return GitLabClient{
		baseURL: strings.TrimRight(valueOrDefault(strings.TrimSpace(baseURL), "https://gitlab.com/api/v4"), "/"),
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func ValidateGitLabWebhookToken(secret, token string) bool {
	trimmedSecret := strings.TrimSpace(secret)
	trimmedToken := strings.TrimSpace(token)
	if trimmedSecret == "" || trimmedToken == "" {
		return false
	}
	return hmac.Equal([]byte(trimmedSecret), []byte(trimmedToken))
}

func (c GitLabClient) TestConnection(ctx context.Context, scope string) ([]string, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Username string `json:"username"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	details := []string{fmt.Sprintf("connected to %s", c.baseURL)}
	if payload.Username != "" {
		details = append(details, fmt.Sprintf("resolved gitlab principal %s", payload.Username))
	} else if payload.Name != "" {
		details = append(details, fmt.Sprintf("resolved gitlab principal %s", payload.Name))
	}
	if trimmedScope := strings.TrimSpace(scope); trimmedScope != "" {
		groupBody, err := c.doJSON(ctx, http.MethodGet, "/groups/"+url.PathEscape(trimmedScope), nil)
		if err != nil {
			return nil, err
		}
		var groupPayload struct {
			FullPath string `json:"full_path"`
			Name     string `json:"name"`
		}
		if err := json.Unmarshal(groupBody, &groupPayload); err == nil {
			details = append(details, fmt.Sprintf("resolved gitlab scope %s", valueOrDefault(groupPayload.FullPath, groupPayload.Name)))
		}
	}
	return details, nil
}

func (c GitLabClient) DiscoverRepositories(ctx context.Context, scope string) ([]SCMRepository, error) {
	path := "/projects?membership=true&per_page=100&simple=true"
	if trimmedScope := strings.TrimSpace(scope); trimmedScope != "" {
		path = "/groups/" + url.PathEscape(trimmedScope) + "/projects?per_page=100&simple=true&include_subgroups=true"
	}
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var payload []struct {
		ID               int    `json:"id"`
		Name             string `json:"name"`
		PathWithNS       string `json:"path_with_namespace"`
		WebURL           string `json:"web_url"`
		DefaultBranch    string `json:"default_branch"`
		Archived         bool   `json:"archived"`
		Visibility       string `json:"visibility"`
		NamespaceDetails struct {
			FullPath string `json:"full_path"`
			Path     string `json:"path"`
			Name     string `json:"name"`
		} `json:"namespace"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	items := make([]SCMRepository, 0, len(payload))
	for _, item := range payload {
		items = append(items, SCMRepository{
			Provider:      "gitlab",
			ExternalID:    strconv.Itoa(item.ID),
			Namespace:     valueOrDefault(item.NamespaceDetails.FullPath, item.NamespaceDetails.Path),
			Owner:         valueOrDefault(item.NamespaceDetails.Path, item.NamespaceDetails.Name),
			Name:          item.Name,
			FullName:      valueOrDefault(item.PathWithNS, item.Name),
			HTMLURL:       item.WebURL,
			DefaultBranch: valueOrDefault(item.DefaultBranch, "main"),
			Private:       strings.EqualFold(item.Visibility, "private"),
			Archived:      item.Archived,
			Metadata: types.Metadata{
				"visibility": item.Visibility,
			},
		})
	}
	return items, nil
}

func (c GitLabClient) LoadCODEOWNERS(ctx context.Context, repository SCMRepository) (RepositoryOwnershipImport, error) {
	projectID := strings.TrimSpace(repository.ExternalID)
	if projectID == "" {
		return RepositoryOwnershipImport{
			Provider: "gitlab",
			Source:   "codeowners",
			Status:   "unavailable",
			Ref:      valueOrDefault(repository.DefaultBranch, "main"),
			Error:    "gitlab project external id is required for CODEOWNERS import",
		}, nil
	}
	ref := valueOrDefault(repository.DefaultBranch, "main")
	for _, candidate := range codeownersCandidatePaths {
		content, revision, found, err := c.readRepositoryFile(ctx, projectID, candidate, ref)
		if err != nil {
			return RepositoryOwnershipImport{}, err
		}
		if !found {
			continue
		}
		rules := ParseCODEOWNERS(content)
		return RepositoryOwnershipImport{
			Provider: "gitlab",
			Source:   "codeowners",
			Status:   "imported",
			FilePath: candidate,
			Ref:      ref,
			Revision: revision,
			Owners:   CollectCodeownersOwners(rules),
			Rules:    rules,
		}, nil
	}
	return RepositoryOwnershipImport{
		Provider: "gitlab",
		Source:   "codeowners",
		Status:   "not_found",
		Ref:      ref,
	}, nil
}

func (c GitLabClient) EnsureGroupWebhook(ctx context.Context, group, callbackURL, secret string) (SCMWebhookRegistration, error) {
	trimmedGroup := strings.TrimSpace(group)
	if trimmedGroup == "" {
		return SCMWebhookRegistration{}, fmt.Errorf("gitlab webhook registration requires group or namespace scope")
	}
	if strings.TrimSpace(secret) == "" {
		return SCMWebhookRegistration{}, fmt.Errorf("gitlab webhook registration requires a webhook secret")
	}
	body, err := c.doJSON(ctx, http.MethodGet, "/groups/"+url.PathEscape(trimmedGroup)+"/hooks", nil)
	if err != nil {
		return SCMWebhookRegistration{}, err
	}
	var hooks []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &hooks); err != nil {
		return SCMWebhookRegistration{}, err
	}
	request := map[string]any{
		"url":                    callbackURL,
		"token":                  secret,
		"enable_ssl_verification": true,
		"push_events":            true,
		"merge_requests_events":  true,
		"tag_push_events":        true,
		"releases_events":        true,
	}
	registration := SCMWebhookRegistration{
		Provider:        "gitlab",
		ScopeIdentifier: trimmedGroup,
		CallbackURL:     callbackURL,
		Status:          "registered",
		DeliveryHealth:  "unknown",
	}
	for _, hook := range hooks {
		if strings.EqualFold(strings.TrimSpace(hook.URL), strings.TrimSpace(callbackURL)) {
			if _, err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("/groups/%s/hooks/%d", url.PathEscape(trimmedGroup), hook.ID), request); err != nil {
				return SCMWebhookRegistration{}, err
			}
			registration.ExternalHookID = strconv.Itoa(hook.ID)
			registration.Details = []string{"existing gitlab group webhook updated"}
			return registration, nil
		}
	}
	createdBody, err := c.doJSON(ctx, http.MethodPost, "/groups/"+url.PathEscape(trimmedGroup)+"/hooks", request)
	if err != nil {
		return SCMWebhookRegistration{}, err
	}
	var created struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(createdBody, &created); err == nil && created.ID > 0 {
		registration.ExternalHookID = strconv.Itoa(created.ID)
	}
	registration.Details = []string{"gitlab group webhook registered automatically"}
	return registration, nil
}

func (c GitLabClient) MergeRequestChanges(ctx context.Context, projectID string, iid int) ([]SCMChangedFile, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%d/changes", url.PathEscape(strings.TrimSpace(projectID)), iid)
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Changes []struct {
			OldPath     string `json:"old_path"`
			NewPath     string `json:"new_path"`
			NewFile     bool   `json:"new_file"`
			RenamedFile bool   `json:"renamed_file"`
			DeletedFile bool   `json:"deleted_file"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	files := make([]SCMChangedFile, 0, len(payload.Changes))
	for _, change := range payload.Changes {
		name := strings.TrimSpace(valueOrDefault(change.NewPath, change.OldPath))
		if name == "" {
			continue
		}
		status := "modified"
		switch {
		case change.NewFile:
			status = "added"
		case change.DeletedFile:
			status = "removed"
		case change.RenamedFile:
			status = "renamed"
		}
		files = append(files, SCMChangedFile{Filename: name, Status: status})
	}
	return uniqueSCMFiles(files), nil
}

func ParseGitLabWebhook(event, deliveryID string, body []byte, fetchMergeRequestChanges func(projectID string, iid int) ([]SCMChangedFile, error)) (SCMWebhookResult, error) {
	switch strings.TrimSpace(event) {
	case "Push Hook":
		return parseGitLabPushWebhook(deliveryID, body)
	case "Merge Request Hook":
		return parseGitLabMergeRequestWebhook(deliveryID, body, fetchMergeRequestChanges)
	case "Tag Push Hook":
		return parseGitLabTagPushWebhook(deliveryID, body)
	case "Release Hook":
		return parseGitLabReleaseWebhook(deliveryID, body)
	default:
		return SCMWebhookResult{
			Operation: "gitlab.webhook." + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(event), " ", "_")),
			Summary:   fmt.Sprintf("ignored unsupported gitlab event %s", strings.TrimSpace(event)),
			Details:   []string{"event payload recorded for audit only"},
		}, nil
	}
}

func parseGitLabPushWebhook(deliveryID string, body []byte) (SCMWebhookResult, error) {
	var payload struct {
		Ref         string `json:"ref"`
		Before      string `json:"before"`
		After       string `json:"after"`
		CheckoutSHA string `json:"checkout_sha"`
		ObjectKind  string `json:"object_kind"`
		UserName    string `json:"user_name"`
		UserUser    string `json:"user_username"`
		ProjectID   int    `json:"project_id"`
		Project     struct {
			Name          string `json:"name"`
			WebURL        string `json:"web_url"`
			DefaultBranch string `json:"default_branch"`
			PathWithNS    string `json:"path_with_namespace"`
			Namespace     string `json:"namespace"`
		} `json:"project"`
		Commits []struct {
			ID       string   `json:"id"`
			Message  string   `json:"message"`
			Title    string   `json:"title"`
			Added    []string `json:"added"`
			Removed  []string `json:"removed"`
			Modified []string `json:"modified"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SCMWebhookResult{}, err
	}
	files := make([]SCMChangedFile, 0)
	for _, commit := range payload.Commits {
		files = append(files, flattenCommitFiles(commit.Added, commit.Removed, commit.Modified)...)
	}
	files = uniqueSCMFiles(files)
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	tag := strings.TrimPrefix(payload.Ref, "refs/tags/")
	summary := ""
	if len(payload.Commits) > 0 {
		latest := payload.Commits[len(payload.Commits)-1]
		summary = strings.TrimSpace(valueOrDefault(latest.Title, latest.Message))
	}
	if summary == "" {
		summary = fmt.Sprintf("Push to %s", valueOrDefault(branch, payload.Ref))
	}
	return SCMWebhookResult{
		Operation:     "gitlab.webhook.push",
		Summary:       fmt.Sprintf("processed gitlab push webhook for %s", payload.Project.PathWithNS),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("files=%d", len(files)), fmt.Sprintf("ref=%s", payload.Ref)},
		ResourceCount: len(files),
		Change: &SCMWebhookChange{
			Repository: SCMRepository{
				Provider:      "gitlab",
				ExternalID:    strconv.Itoa(payload.ProjectID),
				Namespace:     payload.Project.Namespace,
				Owner:         payload.Project.Namespace,
				Name:          payload.Project.Name,
				FullName:      payload.Project.PathWithNS,
				HTMLURL:       payload.Project.WebURL,
				DefaultBranch: valueOrDefault(payload.Project.DefaultBranch, "main"),
			},
			Summary:    summary,
			Branch:     branch,
			Tag:        tag,
			CommitSHA:  valueOrDefault(payload.After, payload.CheckoutSHA),
			ChangeType: "push",
			FileCount:  len(files),
			Files:      files,
			IssueKeys:  extractSCMIssueKeys(summary, payload.Ref),
			Metadata: types.Metadata{
				"delivery_id":   deliveryID,
				"before":        payload.Before,
				"after":         payload.After,
				"checkout_sha":  payload.CheckoutSHA,
				"object_kind":   payload.ObjectKind,
				"user_name":     payload.UserName,
				"user_username": payload.UserUser,
			},
		},
	}, nil
}

func parseGitLabMergeRequestWebhook(deliveryID string, body []byte, fetchMergeRequestChanges func(projectID string, iid int) ([]SCMChangedFile, error)) (SCMWebhookResult, error) {
	var payload struct {
		ObjectKind string `json:"object_kind"`
		Project    struct {
			ID            int    `json:"id"`
			Name          string `json:"name"`
			WebURL        string `json:"web_url"`
			DefaultBranch string `json:"default_branch"`
			PathWithNS    string `json:"path_with_namespace"`
			Namespace     string `json:"namespace"`
		} `json:"project"`
		ObjectAttributes struct {
			IID          int    `json:"iid"`
			Title        string `json:"title"`
			Description  string `json:"description"`
			SourceBranch string `json:"source_branch"`
			TargetBranch string `json:"target_branch"`
			Action       string `json:"action"`
			State        string `json:"state"`
			URL          string `json:"url"`
			MergeStatus  string `json:"merge_status"`
			LastCommit   struct {
				ID string `json:"id"`
			} `json:"last_commit"`
		} `json:"object_attributes"`
		Labels []struct {
			Title string `json:"title"`
		} `json:"labels"`
		Assignees []struct {
			Username string `json:"username"`
		} `json:"assignees"`
		Reviewers []struct {
			Username string `json:"username"`
		} `json:"reviewers"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SCMWebhookResult{}, err
	}
	files := make([]SCMChangedFile, 0)
	if fetchMergeRequestChanges != nil && payload.Project.ID > 0 && payload.ObjectAttributes.IID > 0 {
		if fetched, err := fetchMergeRequestChanges(strconv.Itoa(payload.Project.ID), payload.ObjectAttributes.IID); err == nil {
			files = fetched
		}
	}
	reviewers := make([]string, 0, len(payload.Assignees)+len(payload.Reviewers))
	for _, assignee := range payload.Assignees {
		if strings.TrimSpace(assignee.Username) != "" {
			reviewers = append(reviewers, assignee.Username)
		}
	}
	for _, reviewer := range payload.Reviewers {
		if strings.TrimSpace(reviewer.Username) != "" {
			reviewers = append(reviewers, reviewer.Username)
		}
	}
	labels := make([]string, 0, len(payload.Labels))
	for _, label := range payload.Labels {
		if strings.TrimSpace(label.Title) != "" {
			labels = append(labels, label.Title)
		}
	}
	approvers := []string{}
	if strings.EqualFold(payload.ObjectAttributes.State, "merged") && strings.TrimSpace(payload.User.Username) != "" {
		approvers = append(approvers, payload.User.Username)
	}
	return SCMWebhookResult{
		Operation:     "gitlab.webhook.merge_request",
		Summary:       fmt.Sprintf("processed gitlab merge request webhook for %s", payload.Project.PathWithNS),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("action=%s", payload.ObjectAttributes.Action), fmt.Sprintf("files=%d", len(files))},
		ResourceCount: maxInt(1, len(files)),
		Change: &SCMWebhookChange{
			Repository: SCMRepository{
				Provider:      "gitlab",
				ExternalID:    strconv.Itoa(payload.Project.ID),
				Namespace:     payload.Project.Namespace,
				Owner:         payload.Project.Namespace,
				Name:          payload.Project.Name,
				FullName:      payload.Project.PathWithNS,
				HTMLURL:       payload.Project.WebURL,
				DefaultBranch: valueOrDefault(payload.Project.DefaultBranch, "main"),
			},
			Summary:    fmt.Sprintf("MR !%d %s", payload.ObjectAttributes.IID, strings.TrimSpace(payload.ObjectAttributes.Title)),
			Branch:     payload.ObjectAttributes.SourceBranch,
			CommitSHA:  payload.ObjectAttributes.LastCommit.ID,
			ChangeType: "merge_request",
			FileCount:  maxInt(1, len(files)),
			Files:      files,
			IssueKeys:  extractSCMIssueKeys(payload.ObjectAttributes.Title, payload.ObjectAttributes.Description, payload.ObjectAttributes.SourceBranch),
			Approvers:  approvers,
			Reviewers:  reviewers,
			Labels:     labels,
			Metadata: types.Metadata{
				"delivery_id":   deliveryID,
				"object_kind":   payload.ObjectKind,
				"merge_request": payload.ObjectAttributes.IID,
				"action":        payload.ObjectAttributes.Action,
				"state":         payload.ObjectAttributes.State,
				"target_branch": payload.ObjectAttributes.TargetBranch,
				"url":           payload.ObjectAttributes.URL,
				"merge_status":  payload.ObjectAttributes.MergeStatus,
				"labels":        labels,
			},
		},
	}, nil
}

func parseGitLabTagPushWebhook(deliveryID string, body []byte) (SCMWebhookResult, error) {
	var payload struct {
		Ref     string `json:"ref"`
		Project struct {
			PathWithNS string `json:"path_with_namespace"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SCMWebhookResult{}, err
	}
	tag := strings.TrimPrefix(payload.Ref, "refs/tags/")
	return SCMWebhookResult{
		Operation:     "gitlab.webhook.tag_push",
		Summary:       fmt.Sprintf("processed gitlab tag push webhook for %s", payload.Project.PathWithNS),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("tag=%s", tag)},
		ResourceCount: 1,
	}, nil
}

func parseGitLabReleaseWebhook(deliveryID string, body []byte) (SCMWebhookResult, error) {
	var payload struct {
		Project struct {
			PathWithNS string `json:"path_with_namespace"`
		} `json:"project"`
		Release struct {
			Tag  string `json:"tag"`
			Name string `json:"name"`
		} `json:"release"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SCMWebhookResult{}, err
	}
	return SCMWebhookResult{
		Operation:     "gitlab.webhook.release",
		Summary:       fmt.Sprintf("processed gitlab release webhook for %s", payload.Project.PathWithNS),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("tag=%s", payload.Release.Tag), fmt.Sprintf("release=%s", payload.Release.Name)},
		ResourceCount: 1,
	}, nil
}

func (c GitLabClient) doJSON(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("gitlab request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func (c GitLabClient) readRepositoryFile(ctx context.Context, projectID, filePath, ref string) (string, string, bool, error) {
	path := fmt.Sprintf(
		"/projects/%s/repository/files/%s?ref=%s",
		url.PathEscape(strings.TrimSpace(projectID)),
		url.PathEscape(strings.Trim(strings.TrimSpace(filePath), "/")),
		url.QueryEscape(strings.TrimSpace(ref)),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return "", "", false, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", false, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", "", false, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", "", false, fmt.Errorf("gitlab request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		BlobID   string `json:"blob_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", false, err
	}
	content := payload.Content
	if strings.EqualFold(strings.TrimSpace(payload.Encoding), "base64") {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		if err != nil {
			return "", "", false, err
		}
		content = string(decoded)
	}
	return content, strings.TrimSpace(payload.BlobID), true, nil
}
