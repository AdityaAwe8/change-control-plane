package integrations

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type GitHubRepository = SCMRepository
type GitHubChangedFile = SCMChangedFile
type GitHubWebhookChange = SCMWebhookChange
type GitHubWebhookResult = SCMWebhookResult

type GitHubClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type gitHubInstallationTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

func NewGitHubClient(baseURL, token string) GitHubClient {
	return GitHubClient{
		baseURL: strings.TrimRight(valueOrDefault(strings.TrimSpace(baseURL), "https://api.github.com"), "/"),
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func CreateGitHubAppInstallationToken(ctx context.Context, baseURL, appID, installationID, privateKeyPEM string) (string, *time.Time, error) {
	trimmedAppID := strings.TrimSpace(appID)
	trimmedInstallationID := strings.TrimSpace(installationID)
	if trimmedAppID == "" || trimmedInstallationID == "" {
		return "", nil, fmt.Errorf("github app_id and installation_id are required")
	}
	jwt, err := signGitHubAppJWT(trimmedAppID, privateKeyPEM, time.Now().UTC())
	if err != nil {
		return "", nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	endpoint := strings.TrimRight(valueOrDefault(strings.TrimSpace(baseURL), "https://api.github.com"), "/") + "/app/installations/" + trimmedInstallationID + "/access_tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(`{}`))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", nil, fmt.Errorf("github app installation token request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload gitHubInstallationTokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(payload.Token) == "" {
		return "", nil, fmt.Errorf("github app installation token response did not include a token")
	}
	var expiresAt *time.Time
	if trimmed := strings.TrimSpace(payload.ExpiresAt); trimmed != "" {
		if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
			expiresAt = &parsed
		}
	}
	return strings.TrimSpace(payload.Token), expiresAt, nil
}

func ValidateGitHubWebhookSignature(secret string, body []byte, signature string) bool {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(signature) == "" {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	_, _ = expectedMAC.Write(body)
	expected := prefix + hex.EncodeToString(expectedMAC.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (c GitHubClient) TestConnection(ctx context.Context, owner string) ([]string, error) {
	path := "/user"
	if trimmedOwner := strings.TrimSpace(owner); trimmedOwner != "" {
		path = "/orgs/" + trimmedOwner
	}
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	name := stringValue(payload["login"])
	if name == "" {
		name = stringValue(payload["name"])
	}
	details := []string{fmt.Sprintf("connected to %s", c.baseURL)}
	if name != "" {
		details = append(details, fmt.Sprintf("resolved github principal %s", name))
	}
	return details, nil
}

func (c GitHubClient) DiscoverRepositories(ctx context.Context, owner string) ([]GitHubRepository, error) {
	path := "/user/repos?per_page=100"
	if trimmedOwner := strings.TrimSpace(owner); trimmedOwner != "" {
		path = "/orgs/" + trimmedOwner + "/repos?per_page=100"
	}
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var payload []struct {
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		HTMLURL       string `json:"html_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
		Archived      bool   `json:"archived"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	items := make([]GitHubRepository, 0, len(payload))
	for _, item := range payload {
		items = append(items, GitHubRepository{
			Provider:      "github",
			Name:          item.Name,
			FullName:      item.FullName,
			HTMLURL:       item.HTMLURL,
			DefaultBranch: valueOrDefault(item.DefaultBranch, "main"),
			Owner:         item.Owner.Login,
			Private:       item.Private,
			Archived:      item.Archived,
		})
	}
	return items, nil
}

func (c GitHubClient) LoadCODEOWNERS(ctx context.Context, repository SCMRepository) (RepositoryOwnershipImport, error) {
	owner := strings.TrimSpace(repository.Owner)
	name := strings.TrimSpace(repository.Name)
	ref := valueOrDefault(repository.DefaultBranch, "main")
	if owner == "" || name == "" {
		return RepositoryOwnershipImport{
			Provider: "github",
			Source:   "codeowners",
			Status:   "unavailable",
			Ref:      ref,
			Error:    "github repository owner and name are required for CODEOWNERS import",
		}, nil
	}
	for _, candidate := range codeownersCandidatePaths {
		content, revision, found, err := c.readRepositoryContent(ctx, owner, name, candidate, ref)
		if err != nil {
			return RepositoryOwnershipImport{}, err
		}
		if !found {
			continue
		}
		rules := ParseCODEOWNERS(content)
		return RepositoryOwnershipImport{
			Provider: "github",
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
		Provider: "github",
		Source:   "codeowners",
		Status:   "not_found",
		Ref:      ref,
	}, nil
}

func (c GitHubClient) EnsureOrganizationWebhook(ctx context.Context, owner, callbackURL, secret string) (SCMWebhookRegistration, error) {
	trimmedOwner := strings.TrimSpace(owner)
	if trimmedOwner == "" {
		return SCMWebhookRegistration{}, fmt.Errorf("github webhook registration requires owner or organization scope")
	}
	if strings.TrimSpace(secret) == "" {
		return SCMWebhookRegistration{}, fmt.Errorf("github webhook registration requires a webhook secret")
	}
	body, err := c.doJSON(ctx, http.MethodGet, "/orgs/"+trimmedOwner+"/hooks?per_page=100", nil)
	if err != nil {
		return SCMWebhookRegistration{}, err
	}
	var hooks []struct {
		ID     int  `json:"id"`
		Active bool `json:"active"`
		Config struct {
			URL string `json:"url"`
		} `json:"config"`
	}
	if err := json.Unmarshal(body, &hooks); err != nil {
		return SCMWebhookRegistration{}, err
	}
	request := map[string]any{
		"name":   "web",
		"active": true,
		"events": []string{"push", "pull_request", "release", "workflow_run"},
		"config": map[string]any{
			"url":          callbackURL,
			"content_type": "json",
			"secret":       secret,
			"insecure_ssl": "0",
		},
	}
	registration := SCMWebhookRegistration{
		Provider:        "github",
		ScopeIdentifier: trimmedOwner,
		CallbackURL:     callbackURL,
		Status:          "registered",
		DeliveryHealth:  "unknown",
	}
	for _, hook := range hooks {
		if strings.EqualFold(strings.TrimSpace(hook.Config.URL), strings.TrimSpace(callbackURL)) {
			if _, err := c.doJSON(ctx, http.MethodPatch, fmt.Sprintf("/orgs/%s/hooks/%d", trimmedOwner, hook.ID), request); err != nil {
				return SCMWebhookRegistration{}, err
			}
			registration.ExternalHookID = fmt.Sprintf("%d", hook.ID)
			registration.Details = []string{"existing github organization webhook updated"}
			return registration, nil
		}
	}
	createdBody, err := c.doJSON(ctx, http.MethodPost, "/orgs/"+trimmedOwner+"/hooks", request)
	if err != nil {
		return SCMWebhookRegistration{}, err
	}
	var created struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(createdBody, &created); err == nil && created.ID > 0 {
		registration.ExternalHookID = fmt.Sprintf("%d", created.ID)
	}
	registration.Details = []string{"github organization webhook registered automatically"}
	return registration, nil
}

func (c GitHubClient) PullRequestFiles(ctx context.Context, owner, repo string, number int) ([]GitHubChangedFile, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/files?per_page=100", owner, repo, number)
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var payload []GitHubChangedFile
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func ParseGitHubWebhook(event, deliveryID string, body []byte, fetchPullRequestFiles func(owner, repo string, number int) ([]GitHubChangedFile, error)) (GitHubWebhookResult, error) {
	switch strings.TrimSpace(event) {
	case "push":
		return parseGitHubPushWebhook(deliveryID, body)
	case "pull_request":
		return parseGitHubPullRequestWebhook(deliveryID, body, fetchPullRequestFiles)
	case "release":
		return parseGitHubReleaseWebhook(deliveryID, body)
	case "workflow_run":
		return parseGitHubWorkflowRunWebhook(deliveryID, body)
	default:
		return GitHubWebhookResult{
			Operation: "github.webhook." + strings.TrimSpace(event),
			Summary:   fmt.Sprintf("ignored unsupported github event %s", strings.TrimSpace(event)),
			Details:   []string{"event payload recorded for audit only"},
		}, nil
	}
}

func parseGitHubPushWebhook(deliveryID string, body []byte) (GitHubWebhookResult, error) {
	var payload struct {
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Compare    string `json:"compare"`
		Created    bool   `json:"created"`
		Deleted    bool   `json:"deleted"`
		Repository struct {
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			HTMLURL       string `json:"html_url"`
			DefaultBranch string `json:"default_branch"`
			Private       bool   `json:"private"`
			Archived      bool   `json:"archived"`
			Owner         struct {
				Name  string `json:"name"`
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
		HeadCommit struct {
			ID       string   `json:"id"`
			Message  string   `json:"message"`
			Added    []string `json:"added"`
			Removed  []string `json:"removed"`
			Modified []string `json:"modified"`
		} `json:"head_commit"`
		Commits []struct {
			Added    []string `json:"added"`
			Removed  []string `json:"removed"`
			Modified []string `json:"modified"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GitHubWebhookResult{}, err
	}
	files := uniqueSCMFiles(flattenCommitFiles(payload.HeadCommit.Added, payload.HeadCommit.Removed, payload.HeadCommit.Modified))
	if len(files) == 0 {
		for _, commit := range payload.Commits {
			files = append(files, flattenCommitFiles(commit.Added, commit.Removed, commit.Modified)...)
		}
		files = uniqueSCMFiles(files)
	}
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	tag := strings.TrimPrefix(payload.Ref, "refs/tags/")
	summary := strings.TrimSpace(payload.HeadCommit.Message)
	if summary == "" {
		summary = fmt.Sprintf("Push to %s", valueOrDefault(branch, payload.Ref))
	}
	issueKeys := extractSCMIssueKeys(strings.Join([]string{summary, payload.Compare}, " "))
	return GitHubWebhookResult{
		Operation:     "github.webhook.push",
		Summary:       fmt.Sprintf("processed github push webhook for %s", payload.Repository.FullName),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("files=%d", len(files)), fmt.Sprintf("ref=%s", payload.Ref)},
		ResourceCount: len(files),
		Change: &GitHubWebhookChange{
			Repository: GitHubRepository{
				Provider:      "github",
				Name:          payload.Repository.Name,
				FullName:      payload.Repository.FullName,
				HTMLURL:       payload.Repository.HTMLURL,
				DefaultBranch: valueOrDefault(payload.Repository.DefaultBranch, "main"),
				Owner:         valueOrDefault(payload.Repository.Owner.Login, payload.Repository.Owner.Name),
				Private:       payload.Repository.Private,
				Archived:      payload.Repository.Archived,
			},
			Summary:    summary,
			Branch:     branch,
			Tag:        tag,
			CommitSHA:  valueOrDefault(payload.After, payload.HeadCommit.ID),
			ChangeType: "push",
			FileCount:  len(files),
			Files:      files,
			IssueKeys:  issueKeys,
			Metadata: types.Metadata{
				"delivery_id": deliveryID,
				"compare_url": payload.Compare,
				"created":     payload.Created,
				"deleted":     payload.Deleted,
			},
		},
	}, nil
}

func parseGitHubPullRequestWebhook(deliveryID string, body []byte, fetchPullRequestFiles func(owner, repo string, number int) ([]GitHubChangedFile, error)) (GitHubWebhookResult, error) {
	var payload struct {
		Action     string `json:"action"`
		Number     int    `json:"number"`
		Repository struct {
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			HTMLURL       string `json:"html_url"`
			DefaultBranch string `json:"default_branch"`
			Private       bool   `json:"private"`
			Archived      bool   `json:"archived"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
		PullRequest struct {
			Title        string `json:"title"`
			Body         string `json:"body"`
			HTMLURL      string `json:"html_url"`
			Merged       bool   `json:"merged"`
			ChangedFiles int    `json:"changed_files"`
			Additions    int    `json:"additions"`
			Deletions    int    `json:"deletions"`
			MergedAt     string `json:"merged_at"`
			Head         struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
			} `json:"base"`
			RequestedReviewers []struct {
				Login string `json:"login"`
			} `json:"requested_reviewers"`
			RequestedTeams []struct {
				Slug string `json:"slug"`
			} `json:"requested_teams"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			MergedBy struct {
				Login string `json:"login"`
			} `json:"merged_by"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GitHubWebhookResult{}, err
	}
	files := make([]GitHubChangedFile, 0)
	if fetchPullRequestFiles != nil {
		owner := strings.TrimSpace(payload.Repository.Owner.Login)
		if owner != "" && payload.Repository.Name != "" && payload.Number > 0 {
			fetched, err := fetchPullRequestFiles(owner, payload.Repository.Name, payload.Number)
			if err == nil {
				files = fetched
			}
		}
	}
	reviewers := make([]string, 0, len(payload.PullRequest.RequestedReviewers)+len(payload.PullRequest.RequestedTeams))
	for _, reviewer := range payload.PullRequest.RequestedReviewers {
		reviewers = append(reviewers, reviewer.Login)
	}
	for _, team := range payload.PullRequest.RequestedTeams {
		reviewers = append(reviewers, team.Slug)
	}
	approvers := []string{}
	if payload.PullRequest.Merged && strings.TrimSpace(payload.PullRequest.MergedBy.Login) != "" {
		approvers = append(approvers, payload.PullRequest.MergedBy.Login)
	}
	issueKeys := extractSCMIssueKeys(strings.Join([]string{payload.PullRequest.Title, payload.PullRequest.Body, payload.PullRequest.Head.Ref}, "\n"))
	labels := make([]string, 0, len(payload.PullRequest.Labels))
	for _, label := range payload.PullRequest.Labels {
		if strings.TrimSpace(label.Name) != "" {
			labels = append(labels, label.Name)
		}
	}
	return GitHubWebhookResult{
		Operation:     "github.webhook.pull_request",
		Summary:       fmt.Sprintf("processed github pull request webhook for %s", payload.Repository.FullName),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("action=%s", payload.Action), fmt.Sprintf("files=%d", maxInt(len(files), payload.PullRequest.ChangedFiles))},
		ResourceCount: maxInt(len(files), payload.PullRequest.ChangedFiles),
		Change: &GitHubWebhookChange{
			Repository: GitHubRepository{
				Provider:      "github",
				Name:          payload.Repository.Name,
				FullName:      payload.Repository.FullName,
				HTMLURL:       payload.Repository.HTMLURL,
				DefaultBranch: valueOrDefault(payload.Repository.DefaultBranch, "main"),
				Owner:         payload.Repository.Owner.Login,
				Private:       payload.Repository.Private,
				Archived:      payload.Repository.Archived,
			},
			Summary:    fmt.Sprintf("PR #%d %s", payload.Number, strings.TrimSpace(payload.PullRequest.Title)),
			Branch:     payload.PullRequest.Head.Ref,
			CommitSHA:  payload.PullRequest.Head.SHA,
			ChangeType: "pull_request",
			FileCount:  maxInt(len(files), payload.PullRequest.ChangedFiles),
			Files:      files,
			IssueKeys:  issueKeys,
			Approvers:  approvers,
			Reviewers:  reviewers,
			Labels:     labels,
			Metadata: types.Metadata{
				"delivery_id":         deliveryID,
				"action":              payload.Action,
				"pull_request":        payload.Number,
				"html_url":            payload.PullRequest.HTMLURL,
				"base_ref":            payload.PullRequest.Base.Ref,
				"labels":              labels,
				"merged":              payload.PullRequest.Merged,
				"opened_by":           payload.PullRequest.User.Login,
				"merged_at":           payload.PullRequest.MergedAt,
				"requested_reviewers": reviewers,
			},
		},
	}, nil
}

func parseGitHubReleaseWebhook(deliveryID string, body []byte) (GitHubWebhookResult, error) {
	var payload struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Release struct {
			TagName    string `json:"tag_name"`
			Name       string `json:"name"`
			Target     string `json:"target_commitish"`
			Prerelease bool   `json:"prerelease"`
		} `json:"release"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GitHubWebhookResult{}, err
	}
	return GitHubWebhookResult{
		Operation:     "github.webhook.release",
		Summary:       fmt.Sprintf("processed github release webhook for %s", payload.Repository.FullName),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("tag=%s", payload.Release.TagName), fmt.Sprintf("action=%s", payload.Action)},
		ResourceCount: 1,
	}, nil
}

func parseGitHubWorkflowRunWebhook(deliveryID string, body []byte) (GitHubWebhookResult, error) {
	var payload struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		WorkflowRun struct {
			Name       string `json:"name"`
			HeadBranch string `json:"head_branch"`
			HeadSHA    string `json:"head_sha"`
			Conclusion string `json:"conclusion"`
			Status     string `json:"status"`
		} `json:"workflow_run"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GitHubWebhookResult{}, err
	}
	return GitHubWebhookResult{
		Operation:     "github.webhook.workflow_run",
		Summary:       fmt.Sprintf("processed github workflow webhook for %s", payload.Repository.FullName),
		Details:       []string{fmt.Sprintf("delivery=%s", deliveryID), fmt.Sprintf("workflow=%s", payload.WorkflowRun.Name), fmt.Sprintf("status=%s", payload.WorkflowRun.Status), fmt.Sprintf("conclusion=%s", payload.WorkflowRun.Conclusion)},
		ResourceCount: 1,
	}, nil
}

func (c GitHubClient) doJSON(ctx context.Context, method, path string, payload any) ([]byte, error) {
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
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
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
		return nil, fmt.Errorf("github api request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func (c GitHubClient) readRepositoryContent(ctx context.Context, owner, repo, filePath, ref string) (string, string, bool, error) {
	path := fmt.Sprintf(
		"/repos/%s/%s/contents/%s?ref=%s",
		url.PathEscape(strings.TrimSpace(owner)),
		url.PathEscape(strings.TrimSpace(repo)),
		escapePathSegments(filePath),
		url.QueryEscape(strings.TrimSpace(ref)),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return "", "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
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
		return "", "", false, fmt.Errorf("github api request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		SHA      string `json:"sha"`
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
	return content, strings.TrimSpace(payload.SHA), true, nil
}

func escapePathSegments(path string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(path), "/"), "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(strings.TrimSpace(part))
	}
	return strings.Join(parts, "/")
}

func signGitHubAppJWT(appID, privateKeyPEM string, now time.Time) (string, error) {
	key, err := parseGitHubAppPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	headerRaw, err := json.Marshal(map[string]any{
		"alg": "RS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	claimsRaw, err := json.Marshal(map[string]any{
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": strings.TrimSpace(appID),
	})
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerRaw) + "." + base64.RawURLEncoding.EncodeToString(claimsRaw)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parseGitHubAppPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(privateKeyPEM)))
	if block == nil {
		return nil, fmt.Errorf("github app private key PEM is invalid")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("github app private key must be RSA")
	}
	return key, nil
}
