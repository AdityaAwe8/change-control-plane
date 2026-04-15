package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/client"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type cliSession struct {
	Token          string `json:"token"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		usage(stdout)
		return 1
	}

	cfg := common.LoadConfig()
	c := client.New(cfg.APIBaseURL)
	session, _ := loadSession()
	if token := os.Getenv("CCP_API_TOKEN"); token != "" {
		session.Token = token
	}
	if orgID := os.Getenv("CCP_ORGANIZATION_ID"); orgID != "" {
		session.OrganizationID = orgID
	}
	c.SetToken(session.Token)
	c.SetOrganizationID(session.OrganizationID)

	switch args[0] {
	case "auth":
		return handleAuth(ctx, c, args[1:], stdout, stderr)
	case "org":
		return handleOrg(ctx, c, session, args[1:], stdout, stderr)
	case "project":
		return handleProject(ctx, c, session, args[1:], stdout, stderr)
	case "service":
		return handleService(ctx, c, session, args[1:], stdout, stderr)
	case "env":
		return handleEnvironment(ctx, c, session, args[1:], stdout, stderr)
	case "service-account":
		return handleServiceAccount(ctx, c, session, args[1:], stdout, stderr)
	case "token":
		return handleToken(ctx, c, session, args[1:], stdout, stderr)
	case "change":
		return handleChange(ctx, c, session, args[1:], stdout, stderr)
	case "rollout":
		return handleRollout(ctx, c, session, args[1:], stdout, stderr)
	case "verification":
		return handleVerification(ctx, c, session, args[1:], stdout, stderr)
	case "integrations":
		return handleIntegrations(ctx, c, args[1:], stdout, stderr)
	case "audit":
		return handleAudit(ctx, c, args[1:], stdout, stderr)
	case "incident":
		return handleIncident(ctx, c, args[1:], stdout, stderr)
	case "policy", "graph", "bootstrap":
		fmt.Fprintf(stdout, "%s commands are scaffolded for the next phase\n", args[0])
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleAuth(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}

	switch args[0] {
	case "login":
		fs := flag.NewFlagSet("auth login", flag.ExitOnError)
		email := fs.String("email", "", "user email")
		displayName := fs.String("display-name", "", "display name")
		orgName := fs.String("org-name", "", "organization name for bootstrap")
		orgSlug := fs.String("org-slug", "", "organization slug for bootstrap")
		_ = fs.Parse(args[1:])

		result, err := c.DevLogin(ctx, types.DevLoginRequest{
			Email:            *email,
			DisplayName:      *displayName,
			OrganizationName: *orgName,
			OrganizationSlug: *orgSlug,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}

		c.SetToken(result.Token)
		c.SetOrganizationID(result.Session.ActiveOrganizationID)
		if err := saveSession(cliSession{
			Token:          result.Token,
			OrganizationID: result.Session.ActiveOrganizationID,
			Email:          result.Session.Email,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "session":
		result, err := c.Session(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleOrg(ctx context.Context, c *client.Client, _ cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("org create", flag.ExitOnError)
		name := fs.String("name", "", "organization name")
		slug := fs.String("slug", "", "organization slug")
		tier := fs.String("tier", "growth", "organization tier")
		_ = fs.Parse(args[1:])
		result, err := c.CreateOrganization(ctx, types.CreateOrganizationRequest{Name: *name, Slug: *slug, Tier: *tier})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		result, err := c.ListOrganizations(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleProject(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("project create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		name := fs.String("name", "", "project name")
		slug := fs.String("slug", "", "project slug")
		mode := fs.String("mode", "advisory", "adoption mode")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateProject(ctx, types.CreateProjectRequest{OrganizationID: *orgID, Name: *name, Slug: *slug, AdoptionMode: *mode})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		c.SetOrganizationID(session.OrganizationID)
		result, err := c.ListProjects(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleService(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "register":
		fs := flag.NewFlagSet("service register", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		teamID := fs.String("team", "", "team id")
		name := fs.String("name", "", "service name")
		slug := fs.String("slug", "", "service slug")
		criticality := fs.String("criticality", "medium", "criticality")
		customerFacing := fs.Bool("customer-facing", false, "customer-facing service")
		hasSLO := fs.Bool("has-slo", true, "service has slo")
		hasObservability := fs.Bool("has-observability", true, "service has observability")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateService(ctx, types.CreateServiceRequest{
			OrganizationID:   *orgID,
			ProjectID:        *projectID,
			TeamID:           *teamID,
			Name:             *name,
			Slug:             *slug,
			Criticality:      *criticality,
			CustomerFacing:   *customerFacing,
			HasSLO:           *hasSLO,
			HasObservability: *hasObservability,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("service update", flag.ExitOnError)
		id := fs.String("id", "", "service id")
		name := fs.String("name", "", "service name")
		description := fs.String("description", "", "service description")
		criticality := fs.String("criticality", "", "service criticality")
		_ = fs.Parse(args[1:])
		req := types.UpdateServiceRequest{}
		if *name != "" {
			req.Name = name
		}
		if *description != "" {
			req.Description = description
		}
		if *criticality != "" {
			req.Criticality = criticality
		}
		result, err := c.UpdateService(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "archive":
		fs := flag.NewFlagSet("service archive", flag.ExitOnError)
		id := fs.String("id", "", "service id")
		_ = fs.Parse(args[1:])
		result, err := c.ArchiveService(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		c.SetOrganizationID(session.OrganizationID)
		result, err := c.ListServices(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleEnvironment(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("env create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		name := fs.String("name", "", "environment name")
		slug := fs.String("slug", "", "environment slug")
		envType := fs.String("type", "staging", "environment type")
		production := fs.Bool("production", false, "production environment")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateEnvironment(ctx, types.CreateEnvironmentRequest{
			OrganizationID: *orgID,
			ProjectID:      *projectID,
			Name:           *name,
			Slug:           *slug,
			Type:           *envType,
			Production:     *production,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("env update", flag.ExitOnError)
		id := fs.String("id", "", "environment id")
		name := fs.String("name", "", "environment name")
		region := fs.String("region", "", "region")
		_ = fs.Parse(args[1:])
		req := types.UpdateEnvironmentRequest{}
		if *name != "" {
			req.Name = name
		}
		if *region != "" {
			req.Region = region
		}
		result, err := c.UpdateEnvironment(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "archive":
		fs := flag.NewFlagSet("env archive", flag.ExitOnError)
		id := fs.String("id", "", "environment id")
		_ = fs.Parse(args[1:])
		result, err := c.ArchiveEnvironment(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		c.SetOrganizationID(session.OrganizationID)
		result, err := c.ListEnvironments(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleChange(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "analyze" {
		usage(stdout)
		return 1
	}
	fs := flag.NewFlagSet("change analyze", flag.ExitOnError)
	orgID := fs.String("org", session.OrganizationID, "organization id")
	projectID := fs.String("project", "", "project id")
	serviceID := fs.String("service", "", "service id")
	environmentID := fs.String("env", "", "environment id")
	summary := fs.String("summary", "", "change summary")
	fileCount := fs.Int("files", 1, "number of changed files")
	resourceCount := fs.Int("resources", 0, "number of affected resources")
	changeType := fs.String("type", "code", "change type")
	touchesInfra := fs.Bool("infra", false, "touches infrastructure")
	touchesIAM := fs.Bool("iam", false, "touches IAM")
	touchesSecrets := fs.Bool("secrets", false, "touches secrets")
	touchesSchema := fs.Bool("schema", false, "touches schema")
	deps := fs.Bool("dependencies", false, "dependency changes")
	history := fs.Int("historical-incidents", 0, "historical incident count")
	rollback := fs.Bool("poor-rollback-history", false, "poor rollback history")
	_ = fs.Parse(args[1:])

	c.SetOrganizationID(*orgID)
	change, err := c.CreateChangeSet(ctx, types.CreateChangeSetRequest{
		OrganizationID:          *orgID,
		ProjectID:               *projectID,
		ServiceID:               *serviceID,
		EnvironmentID:           *environmentID,
		Summary:                 *summary,
		ChangeTypes:             []string{*changeType},
		FileCount:               *fileCount,
		ResourceCount:           *resourceCount,
		TouchesInfrastructure:   *touchesInfra,
		TouchesIAM:              *touchesIAM,
		TouchesSecrets:          *touchesSecrets,
		TouchesSchema:           *touchesSchema,
		DependencyChanges:       *deps,
		HistoricalIncidentCount: *history,
		PoorRollbackHistory:     *rollback,
	})
	if !exitOnErr(stderr, err) {
		return 1
	}

	assessment, err := c.AssessRisk(ctx, types.CreateRiskAssessmentRequest{ChangeSetID: change.ID})
	if !exitOnErr(stderr, err) {
		return 1
	}

	printJSON(stdout, map[string]any{
		"change":      change,
		"assessment":  assessment.Assessment,
		"policies":    assessment.PolicyDecisions,
		"recommended": assessment.Assessment.RecommendedRolloutStrategy,
	})
	return 0
}

func handleRollout(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "plan":
		fs := flag.NewFlagSet("rollout plan", flag.ExitOnError)
		changeID := fs.String("change", "", "change set id")
		_ = fs.Parse(args[1:])
		result, err := c.CreateRolloutPlan(ctx, types.CreateRolloutPlanRequest{ChangeSetID: *changeID})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "execute":
		fs := flag.NewFlagSet("rollout execute", flag.ExitOnError)
		planID := fs.String("plan", "", "rollout plan id")
		_ = fs.Parse(args[1:])
		result, err := c.CreateRolloutExecution(ctx, types.CreateRolloutExecutionRequest{RolloutPlanID: *planID})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		result, err := c.ListRolloutExecutions(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("rollout show", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		_ = fs.Parse(args[1:])
		result, err := c.GetRolloutExecution(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "advance":
		fs := flag.NewFlagSet("rollout advance", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		action := fs.String("action", "", "transition action")
		reason := fs.String("reason", "", "transition reason")
		_ = fs.Parse(args[1:])
		result, err := c.AdvanceRolloutExecution(ctx, *id, types.AdvanceRolloutExecutionRequest{Action: *action, Reason: *reason})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleVerification(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "record" {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	fs := flag.NewFlagSet("verification record", flag.ExitOnError)
	executionID := fs.String("rollout", "", "rollout execution id")
	outcome := fs.String("outcome", "", "verification outcome")
	decision := fs.String("decision", "", "control decision")
	summary := fs.String("summary", "", "verification summary")
	_ = fs.Parse(args[1:])
	result, err := c.RecordVerificationResult(ctx, *executionID, types.RecordVerificationResultRequest{
		Outcome:  *outcome,
		Decision: *decision,
		Summary:  *summary,
	})
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func handleServiceAccount(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("service-account create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		name := fs.String("name", "", "service account name")
		description := fs.String("description", "", "description")
		role := fs.String("role", "viewer", "organization role")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateServiceAccount(ctx, types.CreateServiceAccountRequest{
			OrganizationID: *orgID,
			Name:           *name,
			Description:    *description,
			Role:           *role,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		result, err := c.ListServiceAccounts(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "deactivate":
		fs := flag.NewFlagSet("service-account deactivate", flag.ExitOnError)
		id := fs.String("id", "", "service account id")
		_ = fs.Parse(args[1:])
		result, err := c.DeactivateServiceAccount(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleToken(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "issue":
		fs := flag.NewFlagSet("token issue", flag.ExitOnError)
		serviceAccountID := fs.String("service-account", "", "service account id")
		name := fs.String("name", "", "token name")
		expiresInHours := fs.Int("expires-in-hours", 0, "token expiry in hours")
		_ = fs.Parse(args[1:])
		result, err := c.IssueServiceAccountToken(ctx, *serviceAccountID, types.IssueAPITokenRequest{Name: *name, ExpiresInHours: *expiresInHours})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		fs := flag.NewFlagSet("token list", flag.ExitOnError)
		serviceAccountID := fs.String("service-account", "", "service account id")
		_ = fs.Parse(args[1:])
		result, err := c.ListServiceAccountTokens(ctx, *serviceAccountID)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "revoke":
		fs := flag.NewFlagSet("token revoke", flag.ExitOnError)
		serviceAccountID := fs.String("service-account", "", "service account id")
		tokenID := fs.String("id", "", "token id")
		_ = fs.Parse(args[1:])
		result, err := c.RevokeServiceAccountToken(ctx, *serviceAccountID, *tokenID)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "rotate":
		fs := flag.NewFlagSet("token rotate", flag.ExitOnError)
		serviceAccountID := fs.String("service-account", "", "service account id")
		tokenID := fs.String("id", "", "token id")
		name := fs.String("name", "", "new token name")
		expiresInHours := fs.Int("expires-in-hours", 0, "token expiry in hours")
		_ = fs.Parse(args[1:])
		result, err := c.RotateServiceAccountToken(ctx, *serviceAccountID, *tokenID, types.RotateAPITokenRequest{Name: *name, ExpiresInHours: *expiresInHours})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	default:
		usage(stdout)
		return 1
	}
}

func handleIntegrations(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		usage(stdout)
		return 1
	}
	result, err := c.ListIntegrations(ctx)
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func handleAudit(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		usage(stdout)
		return 1
	}
	result, err := c.ListAuditEvents(ctx)
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func handleIncident(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		usage(stdout)
		return 1
	}
	result, err := c.ListIncidents(ctx)
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func printJSON(w io.Writer, v any) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(v)
}

func exitOnErr(stderr io.Writer, err error) bool {
	if err == nil {
		return true
	}
	fmt.Fprintln(stderr, err)
	return false
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `ccp commands:
  ccp auth login --email owner@acme.local --display-name "Acme Owner" --org-name Acme --org-slug acme
  ccp auth session
  ccp org list
  ccp org create --name acme --slug acme
  ccp project list
  ccp project create --org <org_id> --name platform --slug platform
  ccp service list
  ccp service register --org <org_id> --project <project_id> --team <team_id> --name api --slug api
  ccp service update --id <service_id> --name api-v2 --description "..."
  ccp service archive --id <service_id>
  ccp env list
  ccp env create --org <org_id> --project <project_id> --name production --slug prod --type production --production
  ccp env update --id <env_id> --name "Production"
  ccp env archive --id <env_id>
  ccp service-account create --org <org_id> --name deployer --role org_member
  ccp service-account list
  ccp token issue --service-account <service_account_id> --name primary
  ccp token revoke --service-account <service_account_id> --id <token_id>
  ccp change analyze --org <org_id> --project <project_id> --service <service_id> --env <environment_id> --summary "..." --type code
  ccp rollout plan --change <change_id>
  ccp rollout execute --plan <rollout_plan_id>
  ccp rollout list
  ccp rollout show --id <rollout_execution_id>
  ccp rollout advance --id <rollout_execution_id> --action approve --reason "approved"
  ccp verification record --rollout <rollout_execution_id> --outcome pass --decision continue --summary "healthy"
  ccp integrations list
  ccp audit list
  ccp incident list`)
}

func sessionPath() string {
	if override := os.Getenv("CCP_CLI_SESSION_PATH"); override != "" {
		return override
	}
	return filepath.Join(".local", "ccp", "session.json")
}

func loadSession() (cliSession, error) {
	payload, err := os.ReadFile(sessionPath())
	if err != nil {
		return cliSession{}, err
	}
	var session cliSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return cliSession{}, err
	}
	return session, nil
}

func saveSession(session cliSession) error {
	if err := os.MkdirAll(filepath.Dir(sessionPath()), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath(), payload, 0o600)
}
