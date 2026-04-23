package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	case "team":
		return handleTeam(ctx, c, session, args[1:], stdout, stderr)
	case "service":
		return handleService(ctx, c, session, args[1:], stdout, stderr)
	case "env":
		return handleEnvironment(ctx, c, session, args[1:], stdout, stderr)
	case "config-set":
		return handleConfigSet(ctx, c, session, args[1:], stdout, stderr)
	case "release":
		return handleRelease(ctx, c, session, args[1:], stdout, stderr)
	case "db-connection":
		return handleDatabaseConnection(ctx, c, session, args[1:], stdout, stderr)
	case "db-change":
		return handleDatabaseChange(ctx, c, session, args[1:], stdout, stderr)
	case "db-check":
		return handleDatabaseCheck(ctx, c, session, args[1:], stdout, stderr)
	case "db-execution":
		return handleDatabaseExecution(ctx, c, session, args[1:], stdout, stderr)
	case "service-account":
		return handleServiceAccount(ctx, c, session, args[1:], stdout, stderr)
	case "token":
		return handleToken(ctx, c, session, args[1:], stdout, stderr)
	case "change":
		return handleChange(ctx, c, session, args[1:], stdout, stderr)
	case "risk":
		return handleRisk(ctx, c, session, args[1:], stdout, stderr)
	case "rollout":
		return handleRollout(ctx, c, session, args[1:], stdout, stderr)
	case "rollout-plan":
		return handleRolloutPlan(ctx, c, session, args[1:], stdout, stderr)
	case "verification":
		return handleVerification(ctx, c, session, args[1:], stdout, stderr)
	case "signal":
		return handleSignal(ctx, c, session, args[1:], stdout, stderr)
	case "status":
		return handleStatus(ctx, c, args[1:], stdout, stderr)
	case "rollback-policy":
		return handleRollbackPolicy(ctx, c, session, args[1:], stdout, stderr)
	case "integrations":
		return handleIntegrations(ctx, c, args[1:], stdout, stderr)
	case "identity-provider":
		return handleIdentityProviders(ctx, c, args[1:], stdout, stderr)
	case "browser-session":
		return handleBrowserSessions(ctx, c, session, args[1:], stdout, stderr)
	case "repository":
		return handleRepository(ctx, c, args[1:], stdout, stderr)
	case "discovery":
		return handleDiscovery(ctx, c, args[1:], stdout, stderr)
	case "outbox":
		return handleOutbox(ctx, c, args[1:], stdout, stderr)
	case "audit":
		return handleAudit(ctx, c, args[1:], stdout, stderr)
	case "incident":
		return handleIncident(ctx, c, session, args[1:], stdout, stderr)
	case "graph":
		return handleGraph(ctx, c, args[1:], stdout, stderr)
	case "policy":
		return handlePolicy(ctx, c, session, args[1:], stdout, stderr)
	case "policy-decision":
		return handlePolicyDecision(ctx, c, session, args[1:], stdout, stderr)
	case "bootstrap":
		fmt.Fprintln(stdout, "bootstrap commands are scaffolded for the next phase")
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

func handleTeam(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("team create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		name := fs.String("name", "", "team name")
		slug := fs.String("slug", "", "team slug")
		owners := fs.String("owners", "", "comma-separated owner user ids")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateTeam(ctx, types.CreateTeamRequest{
			OrganizationID: *orgID,
			ProjectID:      *projectID,
			Name:           *name,
			Slug:           *slug,
			OwnerUserIDs:   splitCSV(*owners),
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("team show", flag.ExitOnError)
		id := fs.String("id", "", "team id")
		_ = fs.Parse(args[1:])
		result, err := c.GetTeam(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("team update", flag.ExitOnError)
		id := fs.String("id", "", "team id")
		name := fs.String("name", "", "team name")
		slug := fs.String("slug", "", "team slug")
		owners := fs.String("owners", "", "comma-separated owner user ids")
		status := fs.String("status", "", "team status")
		_ = fs.Parse(args[1:])
		req := types.UpdateTeamRequest{}
		if *name != "" {
			req.Name = name
		}
		if *slug != "" {
			req.Slug = slug
		}
		if flagProvided(fs, "owners") {
			ownerIDs := splitCSV(*owners)
			if strings.TrimSpace(*owners) == "" {
				ownerIDs = []string{}
			}
			req.OwnerUserIDs = &ownerIDs
		}
		if *status != "" {
			req.Status = status
		}
		result, err := c.UpdateTeam(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "archive":
		fs := flag.NewFlagSet("team archive", flag.ExitOnError)
		id := fs.String("id", "", "team id")
		_ = fs.Parse(args[1:])
		result, err := c.ArchiveTeam(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		c.SetOrganizationID(session.OrganizationID)
		result, err := c.ListTeams(ctx)
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

func handleConfigSet(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListConfigSets(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("config-set show", flag.ExitOnError)
		id := fs.String("id", "", "config set id")
		_ = fs.Parse(args[1:])
		result, err := c.GetConfigSet(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("config-set create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		environmentID := fs.String("env", "", "environment id")
		serviceID := fs.String("service", "", "service id")
		name := fs.String("name", "", "config set name")
		version := fs.String("version", "", "config set version")
		entriesJSON := fs.String("entries-json", "[]", "config entries as JSON array")
		_ = fs.Parse(args[1:])
		entries, err := parseConfigEntriesJSON(*entriesJSON)
		if !exitOnErr(stderr, err) {
			return 1
		}
		c.SetOrganizationID(*orgID)
		result, err := c.CreateConfigSet(ctx, types.CreateConfigSetRequest{
			OrganizationID: *orgID,
			ProjectID:      *projectID,
			EnvironmentID:  *environmentID,
			ServiceID:      *serviceID,
			Name:           *name,
			Version:        *version,
			Entries:        entries,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("config-set update", flag.ExitOnError)
		id := fs.String("id", "", "config set id")
		name := fs.String("name", "", "config set name")
		version := fs.String("version", "", "config set version")
		status := fs.String("status", "", "config set status")
		entriesJSON := fs.String("entries-json", "", "config entries as JSON array")
		_ = fs.Parse(args[1:])
		req := types.UpdateConfigSetRequest{}
		if *name != "" {
			req.Name = name
		}
		if *version != "" {
			req.Version = version
		}
		if *status != "" {
			req.Status = status
		}
		if strings.TrimSpace(*entriesJSON) != "" {
			entries, err := parseConfigEntriesJSON(*entriesJSON)
			if !exitOnErr(stderr, err) {
				return 1
			}
			req.Entries = &entries
		}
		result, err := c.UpdateConfigSet(ctx, *id, req)
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

func handleRelease(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListReleases(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("release show", flag.ExitOnError)
		id := fs.String("id", "", "release id")
		_ = fs.Parse(args[1:])
		result, err := c.GetRelease(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("release create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		environmentID := fs.String("env", "", "environment id")
		name := fs.String("name", "", "release name")
		summary := fs.String("summary", "", "release summary")
		version := fs.String("version", "", "release version")
		changes := fs.String("changes", "", "comma-separated change set ids")
		configSets := fs.String("config-sets", "", "comma-separated config set ids")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateRelease(ctx, types.CreateReleaseRequest{
			OrganizationID: *orgID,
			ProjectID:      *projectID,
			EnvironmentID:  *environmentID,
			Name:           *name,
			Summary:        *summary,
			ChangeSetIDs:   splitCSV(*changes),
			ConfigSetIDs:   splitCSV(*configSets),
			Version:        *version,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("release update", flag.ExitOnError)
		id := fs.String("id", "", "release id")
		name := fs.String("name", "", "release name")
		summary := fs.String("summary", "", "release summary")
		version := fs.String("version", "", "release version")
		status := fs.String("status", "", "release status")
		changes := fs.String("changes", "", "comma-separated change set ids")
		configSets := fs.String("config-sets", "", "comma-separated config set ids")
		_ = fs.Parse(args[1:])
		req := types.UpdateReleaseRequest{}
		if *name != "" {
			req.Name = name
		}
		if *summary != "" {
			req.Summary = summary
		}
		if *version != "" {
			req.Version = version
		}
		if *status != "" {
			req.Status = status
		}
		if strings.TrimSpace(*changes) != "" {
			changeIDs := splitCSV(*changes)
			req.ChangeSetIDs = &changeIDs
		}
		if strings.TrimSpace(*configSets) != "" {
			configSetIDs := splitCSV(*configSets)
			req.ConfigSetIDs = &configSetIDs
		}
		result, err := c.UpdateRelease(ctx, *id, req)
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

func handleDatabaseChange(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListDatabaseChanges(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("db-change show", flag.ExitOnError)
		id := fs.String("id", "", "database change id")
		_ = fs.Parse(args[1:])
		result, err := c.GetDatabaseChange(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("db-change create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		environmentID := fs.String("env", "", "environment id")
		serviceID := fs.String("service", "", "service id")
		changeSetID := fs.String("change", "", "change set id")
		name := fs.String("name", "", "database change name")
		datastore := fs.String("datastore", "", "target datastore")
		operationType := fs.String("operation", "schema_change", "operation type")
		executionIntent := fs.String("intent", "pre_deploy", "execution intent")
		compatibility := fs.String("compatibility", "backward_compatible", "compatibility posture")
		reversibility := fs.String("reversibility", "reversible", "reversibility posture")
		riskLevel := fs.String("risk", "medium", "database risk level")
		lockRisk := fs.Bool("lock-risk", false, "mark the database change as lock-risk")
		manualApproval := fs.Bool("manual-approval", false, "require manual approval")
		summary := fs.String("summary", "", "database change summary")
		evidence := fs.String("evidence", "", "comma-separated evidence notes")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateDatabaseChange(ctx, types.CreateDatabaseChangeRequest{
			OrganizationID:         *orgID,
			ProjectID:              *projectID,
			EnvironmentID:          *environmentID,
			ServiceID:              *serviceID,
			ChangeSetID:            *changeSetID,
			Name:                   *name,
			Datastore:              *datastore,
			OperationType:          *operationType,
			ExecutionIntent:        *executionIntent,
			Compatibility:          *compatibility,
			Reversibility:          *reversibility,
			RiskLevel:              types.RiskLevel(strings.ToLower(strings.TrimSpace(*riskLevel))),
			LockRisk:               *lockRisk,
			ManualApprovalRequired: *manualApproval,
			Summary:                *summary,
			Evidence:               splitCSV(*evidence),
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("db-change update", flag.ExitOnError)
		id := fs.String("id", "", "database change id")
		name := fs.String("name", "", "database change name")
		datastore := fs.String("datastore", "", "target datastore")
		operationType := fs.String("operation", "", "operation type")
		executionIntent := fs.String("intent", "", "execution intent")
		compatibility := fs.String("compatibility", "", "compatibility posture")
		reversibility := fs.String("reversibility", "", "reversibility posture")
		riskLevel := fs.String("risk", "", "database risk level")
		status := fs.String("status", "", "database change status")
		lockRisk := fs.Bool("lock-risk", false, "mark the database change as lock-risk")
		manualApproval := fs.Bool("manual-approval", false, "require manual approval")
		summary := fs.String("summary", "", "database change summary")
		evidence := fs.String("evidence", "", "comma-separated evidence notes")
		_ = fs.Parse(args[1:])
		req := types.UpdateDatabaseChangeRequest{}
		if *name != "" {
			req.Name = name
		}
		if *datastore != "" {
			req.Datastore = datastore
		}
		if *operationType != "" {
			req.OperationType = operationType
		}
		if *executionIntent != "" {
			req.ExecutionIntent = executionIntent
		}
		if *compatibility != "" {
			req.Compatibility = compatibility
		}
		if *reversibility != "" {
			req.Reversibility = reversibility
		}
		if *riskLevel != "" {
			normalizedRisk := types.RiskLevel(strings.ToLower(strings.TrimSpace(*riskLevel)))
			req.RiskLevel = &normalizedRisk
		}
		if *status != "" {
			req.Status = status
		}
		if flagProvided(fs, "lock-risk") {
			req.LockRisk = lockRisk
		}
		if flagProvided(fs, "manual-approval") {
			req.ManualApprovalRequired = manualApproval
		}
		if *summary != "" {
			req.Summary = summary
		}
		if strings.TrimSpace(*evidence) != "" {
			evidenceItems := splitCSV(*evidence)
			req.Evidence = &evidenceItems
		}
		result, err := c.UpdateDatabaseChange(ctx, *id, req)
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

func handleDatabaseConnection(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListDatabaseConnectionReferences(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("db-connection show", flag.ExitOnError)
		id := fs.String("id", "", "database connection reference id")
		_ = fs.Parse(args[1:])
		result, err := c.GetDatabaseConnectionReference(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("db-connection create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		environmentID := fs.String("env", "", "environment id")
		serviceID := fs.String("service", "", "service id")
		name := fs.String("name", "", "connection reference name")
		datastore := fs.String("datastore", "", "logical datastore name")
		driver := fs.String("driver", "postgres", "database driver")
		sourceType := fs.String("source-type", "env_dsn", "connection source type")
		dsnEnv := fs.String("dsn-env", "CCP_DB_DSN", "env var name holding the database DSN")
		secretRef := fs.String("secret-ref", "", "secret reference for DSN-backed sources")
		secretRefEnv := fs.String("secret-ref-env", "", "env var name that resolves the logical secret reference at runtime")
		readOnlyCapable := fs.Bool("read-only-capable", true, "mark this reference as read-only capable")
		summary := fs.String("summary", "", "connection reference summary")
		_ = fs.Parse(args[1:])
		resolvedDSNEnv := *dsnEnv
		if *sourceType == "secret_ref_dsn" && !flagProvided(fs, "dsn-env") {
			resolvedDSNEnv = ""
		}
		resolvedSecretRefEnv := *secretRefEnv
		c.SetOrganizationID(*orgID)
		result, err := c.CreateDatabaseConnectionReference(ctx, types.CreateDatabaseConnectionReferenceRequest{
			OrganizationID:  *orgID,
			ProjectID:       *projectID,
			EnvironmentID:   *environmentID,
			ServiceID:       *serviceID,
			Name:            *name,
			Datastore:       *datastore,
			Driver:          *driver,
			SourceType:      *sourceType,
			DSNEnv:          resolvedDSNEnv,
			SecretRef:       *secretRef,
			SecretRefEnv:    resolvedSecretRefEnv,
			ReadOnlyCapable: *readOnlyCapable,
			Summary:         *summary,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("db-connection update", flag.ExitOnError)
		id := fs.String("id", "", "database connection reference id")
		name := fs.String("name", "", "connection reference name")
		datastore := fs.String("datastore", "", "logical datastore name")
		driver := fs.String("driver", "", "database driver")
		sourceType := fs.String("source-type", "", "connection source type")
		dsnEnv := fs.String("dsn-env", "", "env var name holding the database DSN")
		secretRef := fs.String("secret-ref", "", "secret reference for DSN-backed sources")
		secretRefEnv := fs.String("secret-ref-env", "", "env var name that resolves the logical secret reference at runtime")
		readOnlyCapable := fs.Bool("read-only-capable", true, "mark this reference as read-only capable")
		summary := fs.String("summary", "", "connection reference summary")
		_ = fs.Parse(args[1:])
		req := types.UpdateDatabaseConnectionReferenceRequest{}
		if *name != "" {
			req.Name = name
		}
		if *datastore != "" {
			req.Datastore = datastore
		}
		if *driver != "" {
			req.Driver = driver
		}
		if *sourceType != "" {
			req.SourceType = sourceType
			if *sourceType == "env_dsn" && !flagProvided(fs, "secret-ref") {
				empty := ""
				req.SecretRef = &empty
			}
			if *sourceType == "env_dsn" && !flagProvided(fs, "secret-ref-env") {
				empty := ""
				req.SecretRefEnv = &empty
			}
			if *sourceType == "secret_ref_dsn" && !flagProvided(fs, "dsn-env") {
				empty := ""
				req.DSNEnv = &empty
			}
		}
		if *dsnEnv != "" {
			req.DSNEnv = dsnEnv
		}
		if *secretRef != "" {
			req.SecretRef = secretRef
		}
		if *secretRefEnv != "" {
			req.SecretRefEnv = secretRefEnv
		}
		if flagProvided(fs, "read-only-capable") {
			req.ReadOnlyCapable = readOnlyCapable
		}
		if *summary != "" {
			req.Summary = summary
		}
		result, err := c.UpdateDatabaseConnectionReference(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "test":
		fs := flag.NewFlagSet("db-connection test", flag.ExitOnError)
		id := fs.String("id", "", "database connection reference id")
		trigger := fs.String("trigger", "manual", "test trigger")
		_ = fs.Parse(args[1:])
		result, err := c.TestDatabaseConnectionReference(ctx, *id, types.TestDatabaseConnectionReferenceRequest{
			Trigger: *trigger,
		})
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

func handleDatabaseCheck(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListDatabaseValidationChecks(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("db-check show", flag.ExitOnError)
		id := fs.String("id", "", "database validation check id")
		_ = fs.Parse(args[1:])
		result, err := c.GetDatabaseValidationCheck(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("db-check create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		environmentID := fs.String("env", "", "environment id")
		serviceID := fs.String("service", "", "service id")
		changeSetID := fs.String("change", "", "change set id")
		databaseChangeID := fs.String("db-change", "", "database change id")
		connectionRefID := fs.String("conn-ref", "", "database connection reference id")
		name := fs.String("name", "", "database check name")
		phase := fs.String("phase", "pre_deploy", "validation phase")
		checkType := fs.String("type", "migration_completion", "validation check type")
		readOnly := fs.Bool("read-only", true, "mark the check as read only")
		required := fs.Bool("required", true, "require this check")
		executionMode := fs.String("mode", "manual_attestation", "execution mode")
		status := fs.String("status", "defined", "validation check status")
		specification := fs.String("spec", "", "operator-facing validation specification")
		summary := fs.String("summary", "", "validation check summary")
		evidence := fs.String("evidence", "", "comma-separated evidence notes")
		_ = fs.Parse(args[1:])
		c.SetOrganizationID(*orgID)
		result, err := c.CreateDatabaseValidationCheck(ctx, types.CreateDatabaseValidationCheckRequest{
			OrganizationID:   *orgID,
			ProjectID:        *projectID,
			EnvironmentID:    *environmentID,
			ServiceID:        *serviceID,
			ChangeSetID:      *changeSetID,
			DatabaseChangeID: *databaseChangeID,
			ConnectionRefID:  *connectionRefID,
			Name:             *name,
			Phase:            *phase,
			CheckType:        *checkType,
			ReadOnly:         *readOnly,
			Required:         *required,
			ExecutionMode:    *executionMode,
			Status:           *status,
			Specification:    *specification,
			Summary:          *summary,
			Evidence:         splitCSV(*evidence),
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("db-check update", flag.ExitOnError)
		id := fs.String("id", "", "database validation check id")
		databaseChangeID := fs.String("db-change", "", "database change id")
		connectionRefID := fs.String("conn-ref", "", "database connection reference id")
		name := fs.String("name", "", "database check name")
		phase := fs.String("phase", "", "validation phase")
		checkType := fs.String("type", "", "validation check type")
		readOnly := fs.Bool("read-only", true, "mark the check as read only")
		required := fs.Bool("required", true, "require this check")
		executionMode := fs.String("mode", "", "execution mode")
		status := fs.String("status", "", "validation check status")
		specification := fs.String("spec", "", "operator-facing validation specification")
		summary := fs.String("summary", "", "validation check summary")
		resultSummary := fs.String("result", "", "latest validation result summary")
		evidence := fs.String("evidence", "", "comma-separated evidence notes")
		_ = fs.Parse(args[1:])
		req := types.UpdateDatabaseValidationCheckRequest{}
		if flagProvided(fs, "db-change") {
			req.DatabaseChangeID = databaseChangeID
		}
		if flagProvided(fs, "conn-ref") {
			req.ConnectionRefID = connectionRefID
		}
		if *name != "" {
			req.Name = name
		}
		if *phase != "" {
			req.Phase = phase
		}
		if *checkType != "" {
			req.CheckType = checkType
		}
		if flagProvided(fs, "read-only") {
			req.ReadOnly = readOnly
		}
		if flagProvided(fs, "required") {
			req.Required = required
		}
		if *executionMode != "" {
			req.ExecutionMode = executionMode
		}
		if *status != "" {
			req.Status = status
		}
		if *specification != "" {
			req.Specification = specification
		}
		if *summary != "" {
			req.Summary = summary
		}
		if *resultSummary != "" {
			req.LastResultSummary = resultSummary
		}
		if strings.TrimSpace(*evidence) != "" {
			evidenceItems := splitCSV(*evidence)
			req.Evidence = &evidenceItems
		}
		result, err := c.UpdateDatabaseValidationCheck(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "execute":
		fs := flag.NewFlagSet("db-check execute", flag.ExitOnError)
		id := fs.String("id", "", "database validation check id")
		trigger := fs.String("trigger", "manual", "execution trigger")
		_ = fs.Parse(args[1:])
		result, err := c.ExecuteDatabaseValidationCheck(ctx, *id, types.ExecuteDatabaseValidationCheckRequest{
			Trigger: *trigger,
		})
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

func handleDatabaseExecution(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("db-execution list", flag.ExitOnError)
		checkID := fs.String("check", "", "database validation check id")
		connectionRefID := fs.String("conn-ref", "", "database connection reference id")
		status := fs.String("status", "", "execution status")
		_ = fs.Parse(args[1:])
		values := url.Values{}
		if strings.TrimSpace(*checkID) != "" {
			values.Set("validation_check_id", *checkID)
		}
		if strings.TrimSpace(*connectionRefID) != "" {
			values.Set("connection_ref_id", *connectionRefID)
		}
		if strings.TrimSpace(*status) != "" {
			values.Set("status", *status)
		}
		result, err := c.ListDatabaseValidationExecutions(ctx, values.Encode())
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("db-execution show", flag.ExitOnError)
		id := fs.String("id", "", "database validation execution id")
		_ = fs.Parse(args[1:])
		result, err := c.GetDatabaseValidationExecution(ctx, *id)
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
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListChangeSets(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("change show", flag.ExitOnError)
		id := fs.String("id", "", "change set id")
		_ = fs.Parse(args[1:])
		result, err := c.GetChangeSet(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "analyze":
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
	default:
		usage(stdout)
		return 1
	}
}

func handleRisk(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	result, err := c.ListRiskAssessments(ctx)
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func handleRolloutPlan(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	result, err := c.ListRolloutPlans(ctx)
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
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
		releaseID := fs.String("release", "", "optional release bundle id")
		backendType := fs.String("backend", "", "backend type")
		signalType := fs.String("signal", "", "signal provider type")
		_ = fs.Parse(args[1:])
		result, err := c.CreateRolloutExecution(ctx, types.CreateRolloutExecutionRequest{
			RolloutPlanID:      *planID,
			ReleaseID:          *releaseID,
			BackendType:        *backendType,
			SignalProviderType: *signalType,
		})
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
	case "status":
		fs := flag.NewFlagSet("rollout status", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		_ = fs.Parse(args[1:])
		result, err := c.GetRolloutExecution(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "evidence":
		fs := flag.NewFlagSet("rollout evidence", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		_ = fs.Parse(args[1:])
		result, err := c.GetRolloutEvidencePack(ctx, *id)
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
	case "pause":
		fs := flag.NewFlagSet("rollout pause", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		reason := fs.String("reason", "operator requested pause", "transition reason")
		_ = fs.Parse(args[1:])
		result, err := c.PauseRolloutExecution(ctx, *id, *reason)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "resume":
		fs := flag.NewFlagSet("rollout resume", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		reason := fs.String("reason", "operator requested resume", "transition reason")
		_ = fs.Parse(args[1:])
		result, err := c.ResumeRolloutExecution(ctx, *id, *reason)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "rollback":
		fs := flag.NewFlagSet("rollout rollback", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		reason := fs.String("reason", "operator requested rollback", "transition reason")
		_ = fs.Parse(args[1:])
		result, err := c.RollbackRolloutExecution(ctx, *id, *reason)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "timeline":
		fs := flag.NewFlagSet("rollout timeline", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		_ = fs.Parse(args[1:])
		result, err := c.ListRolloutExecutionTimeline(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "reconcile":
		fs := flag.NewFlagSet("rollout reconcile", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		_ = fs.Parse(args[1:])
		result, err := c.ReconcileRolloutExecution(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "watch":
		fs := flag.NewFlagSet("rollout watch", flag.ExitOnError)
		id := fs.String("id", "", "rollout execution id")
		iterations := fs.Int("iterations", 5, "number of poll iterations")
		interval := fs.Duration("interval", 2*time.Second, "poll interval")
		_ = fs.Parse(args[1:])
		for index := 0; index < *iterations; index++ {
			result, err := c.GetRolloutExecution(ctx, *id)
			if !exitOnErr(stderr, err) {
				return 1
			}
			printJSON(stdout, result)
			if index+1 < *iterations {
				time.Sleep(*interval)
			}
		}
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

func handleSignal(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "ingest" {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	fs := flag.NewFlagSet("signal ingest", flag.ExitOnError)
	executionID := fs.String("rollout", "", "rollout execution id")
	provider := fs.String("provider", "simulated", "signal provider type")
	health := fs.String("health", "healthy", "normalized health")
	summary := fs.String("summary", "", "signal summary")
	latency := fs.Float64("latency", 0, "latency value")
	errorRate := fs.Float64("error-rate", 0, "error rate value")
	businessMetric := fs.Float64("business-metric", 0, "business metric value")
	_ = fs.Parse(args[1:])

	signals := make([]types.SignalValue, 0, 3)
	if *latency > 0 {
		signals = append(signals, types.SignalValue{Name: "latency_p95_ms", Category: "technical", Value: *latency, Unit: "ms", Status: *health, Threshold: 250, Comparator: ">"})
	}
	if *errorRate > 0 {
		signals = append(signals, types.SignalValue{Name: "error_rate", Category: "technical", Value: *errorRate, Unit: "%", Status: *health, Threshold: 1, Comparator: ">"})
	}
	if *businessMetric > 0 {
		signals = append(signals, types.SignalValue{Name: "business_kpi", Category: "business", Value: *businessMetric, Status: *health})
	}

	result, err := c.CreateSignalSnapshot(ctx, *executionID, types.CreateSignalSnapshotRequest{
		ProviderType: *provider,
		Health:       *health,
		Summary:      *summary,
		Signals:      signals,
	})
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func handleStatus(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("status list", flag.ExitOnError)
		projectID := fs.String("project", "", "project id filter")
		serviceID := fs.String("service", "", "service id filter")
		environmentID := fs.String("env", "", "environment id filter")
		rolloutID := fs.String("rollout", "", "rollout execution id filter")
		source := fs.String("source", "", "event source filter")
		eventType := fs.String("event-type", "", "event type filter")
		automated := fs.String("automated", "", "automated filter (true|false)")
		search := fs.String("search", "", "search text")
		rollbackOnly := fs.Bool("rollback-only", false, "show only rollback-related events")
		limit := fs.Int("limit", 100, "maximum number of events")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		query := make([]string, 0, 10)
		if *projectID != "" {
			query = append(query, "project_id="+url.QueryEscape(*projectID))
		}
		if *serviceID != "" {
			query = append(query, "service_id="+url.QueryEscape(*serviceID))
		}
		if *environmentID != "" {
			query = append(query, "environment_id="+url.QueryEscape(*environmentID))
		}
		if *rolloutID != "" {
			query = append(query, "rollout_execution_id="+url.QueryEscape(*rolloutID))
		}
		if *source != "" {
			query = append(query, "source="+url.QueryEscape(*source))
		}
		if *eventType != "" {
			query = append(query, "event_type="+url.QueryEscape(*eventType))
		}
		if *automated != "" {
			query = append(query, "automated="+url.QueryEscape(*automated))
		}
		if *search != "" {
			query = append(query, "search="+url.QueryEscape(*search))
		}
		if *rollbackOnly {
			query = append(query, "rollback_only=true")
		}
		if *limit > 0 {
			query = append(query, fmt.Sprintf("limit=%d", *limit))
		}
		if *offset > 0 {
			query = append(query, fmt.Sprintf("offset=%d", *offset))
		}
		result, err := c.SearchStatusEvents(ctx, strings.Join(query, "&"))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("status show", flag.ExitOnError)
		id := fs.String("id", "", "status event id")
		_ = fs.Parse(args[1:])
		result, err := c.GetStatusEvent(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "project":
		fs := flag.NewFlagSet("status project", flag.ExitOnError)
		id := fs.String("id", "", "project id")
		rollbackOnly := fs.Bool("rollback-only", false, "show only rollback-related events")
		limit := fs.Int("limit", 100, "maximum number of events")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		result, err := c.ListProjectStatusEvents(ctx, *id, scopedStatusQuery(*rollbackOnly, *limit, *offset))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "service":
		fs := flag.NewFlagSet("status service", flag.ExitOnError)
		id := fs.String("id", "", "service id")
		rollbackOnly := fs.Bool("rollback-only", false, "show only rollback-related events")
		limit := fs.Int("limit", 100, "maximum number of events")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		result, err := c.ListServiceStatusEvents(ctx, *id, scopedStatusQuery(*rollbackOnly, *limit, *offset))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "env":
		fs := flag.NewFlagSet("status env", flag.ExitOnError)
		id := fs.String("id", "", "environment id")
		rollbackOnly := fs.Bool("rollback-only", false, "show only rollback-related events")
		limit := fs.Int("limit", 100, "maximum number of events")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		result, err := c.ListEnvironmentStatusEvents(ctx, *id, scopedStatusQuery(*rollbackOnly, *limit, *offset))
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

func handleRollbackPolicy(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListRollbackPolicies(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("rollback-policy create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		serviceID := fs.String("service", "", "service id")
		environmentID := fs.String("env", "", "environment id")
		name := fs.String("name", "", "policy name")
		maxErrorRate := fs.Float64("max-error-rate", 0, "maximum tolerated error rate")
		maxLatencyMs := fs.Float64("max-latency-ms", 0, "maximum tolerated latency in milliseconds")
		rollbackOnCritical := fs.Bool("rollback-on-critical", true, "rollback when critical signals breach guardrails")
		_ = fs.Parse(args[1:])
		result, err := c.CreateRollbackPolicy(ctx, types.CreateRollbackPolicyRequest{
			OrganizationID:            *orgID,
			ProjectID:                 *projectID,
			ServiceID:                 *serviceID,
			EnvironmentID:             *environmentID,
			Name:                      *name,
			MaxErrorRate:              *maxErrorRate,
			MaxLatencyMs:              *maxLatencyMs,
			RollbackOnCriticalSignals: rollbackOnCritical,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("rollback-policy update", flag.ExitOnError)
		id := fs.String("id", "", "policy id")
		name := fs.String("name", "", "policy name")
		maxErrorRate := fs.Float64("max-error-rate", 0, "maximum tolerated error rate")
		maxLatencyMs := fs.Float64("max-latency-ms", 0, "maximum tolerated latency in milliseconds")
		enabled := fs.String("enabled", "", "policy enabled state (true|false)")
		_ = fs.Parse(args[1:])
		req := types.UpdateRollbackPolicyRequest{}
		if *name != "" {
			req.Name = name
		}
		if fs.Lookup("max-error-rate").Value.String() != "0" {
			req.MaxErrorRate = maxErrorRate
		}
		if fs.Lookup("max-latency-ms").Value.String() != "0" {
			req.MaxLatencyMs = maxLatencyMs
		}
		if *enabled != "" {
			parsed, err := strconv.ParseBool(*enabled)
			if !exitOnErr(stderr, err) {
				return 1
			}
			req.Enabled = &parsed
		}
		result, err := c.UpdateRollbackPolicy(ctx, *id, req)
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

func handlePolicy(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		result, err := c.ListPolicies(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("policy show", flag.ExitOnError)
		id := fs.String("id", "", "policy id")
		_ = fs.Parse(args[1:])
		result, err := c.GetPolicy(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("policy create", flag.ExitOnError)
		orgID := fs.String("org", session.OrganizationID, "organization id")
		projectID := fs.String("project", "", "project id")
		serviceID := fs.String("service", "", "service id")
		environmentID := fs.String("env", "", "environment id")
		name := fs.String("name", "", "policy name")
		code := fs.String("code", "", "policy code")
		appliesTo := fs.String("applies-to", "risk_assessment", "workflow surface (risk_assessment|rollout_plan)")
		mode := fs.String("mode", "advisory", "policy mode (advisory|require_manual_review|block)")
		priority := fs.Int("priority", 0, "policy priority")
		description := fs.String("description", "", "policy description")
		enabled := fs.String("enabled", "", "policy enabled state (true|false)")
		minRiskLevel := fs.String("min-risk-level", "", "minimum risk level")
		productionOnly := fs.Bool("production-only", false, "only match production environments")
		regulatedOnly := fs.Bool("regulated-only", false, "only match regulated services or environments")
		requiredChangeTypes := fs.String("required-change-types", "", "comma-separated change types")
		requiredTouches := fs.String("required-touches", "", "comma-separated touches (infrastructure,secrets,schema,dependencies,poor_rollback_history)")
		missingCapabilities := fs.String("missing-capabilities", "", "comma-separated missing capabilities (observability,slo)")
		_ = fs.Parse(args[1:])
		req := types.CreatePolicyRequest{
			OrganizationID: *orgID,
			ProjectID:      *projectID,
			ServiceID:      *serviceID,
			EnvironmentID:  *environmentID,
			Name:           *name,
			Code:           *code,
			AppliesTo:      *appliesTo,
			Mode:           *mode,
			Priority:       *priority,
			Description:    *description,
			Conditions: types.PolicyCondition{
				MinRiskLevel:        *minRiskLevel,
				ProductionOnly:      *productionOnly,
				RegulatedOnly:       *regulatedOnly,
				RequiredChangeTypes: splitCSV(*requiredChangeTypes),
				RequiredTouches:     splitCSV(*requiredTouches),
				MissingCapabilities: splitCSV(*missingCapabilities),
			},
		}
		if *enabled != "" {
			parsed, err := strconv.ParseBool(*enabled)
			if !exitOnErr(stderr, err) {
				return 1
			}
			req.Enabled = &parsed
		}
		result, err := c.CreatePolicy(ctx, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("policy update", flag.ExitOnError)
		id := fs.String("id", "", "policy id")
		projectID := fs.String("project", "", "project id")
		serviceID := fs.String("service", "", "service id")
		environmentID := fs.String("env", "", "environment id")
		name := fs.String("name", "", "policy name")
		code := fs.String("code", "", "policy code")
		appliesTo := fs.String("applies-to", "", "workflow surface (risk_assessment|rollout_plan)")
		mode := fs.String("mode", "", "policy mode (advisory|require_manual_review|block)")
		priority := fs.Int("priority", 0, "policy priority")
		description := fs.String("description", "", "policy description")
		enabled := fs.String("enabled", "", "policy enabled state (true|false)")
		minRiskLevel := fs.String("min-risk-level", "", "minimum risk level")
		productionOnly := fs.String("production-only", "", "production-only match state (true|false)")
		regulatedOnly := fs.String("regulated-only", "", "regulated-only match state (true|false)")
		requiredChangeTypes := fs.String("required-change-types", "", "comma-separated change types")
		requiredTouches := fs.String("required-touches", "", "comma-separated touches")
		missingCapabilities := fs.String("missing-capabilities", "", "comma-separated missing capabilities")
		_ = fs.Parse(args[1:])

		req := types.UpdatePolicyRequest{}
		if fs.Lookup("project").Value.String() != "" {
			req.ProjectID = projectID
		}
		if fs.Lookup("service").Value.String() != "" {
			req.ServiceID = serviceID
		}
		if fs.Lookup("env").Value.String() != "" {
			req.EnvironmentID = environmentID
		}
		if *name != "" {
			req.Name = name
		}
		if *code != "" {
			req.Code = code
		}
		if *appliesTo != "" {
			req.AppliesTo = appliesTo
		}
		if *mode != "" {
			req.Mode = mode
		}
		if fs.Lookup("priority").Value.String() != "0" {
			req.Priority = priority
		}
		if *description != "" {
			req.Description = description
		}
		if *enabled != "" {
			parsed, err := strconv.ParseBool(*enabled)
			if !exitOnErr(stderr, err) {
				return 1
			}
			req.Enabled = &parsed
		}
		var condition *types.PolicyCondition
		setCondition := func() {
			if condition == nil {
				condition = &types.PolicyCondition{}
			}
		}
		if *minRiskLevel != "" {
			setCondition()
			condition.MinRiskLevel = *minRiskLevel
		}
		if *productionOnly != "" {
			parsed, err := strconv.ParseBool(*productionOnly)
			if !exitOnErr(stderr, err) {
				return 1
			}
			setCondition()
			condition.ProductionOnly = parsed
		}
		if *regulatedOnly != "" {
			parsed, err := strconv.ParseBool(*regulatedOnly)
			if !exitOnErr(stderr, err) {
				return 1
			}
			setCondition()
			condition.RegulatedOnly = parsed
		}
		if fs.Lookup("required-change-types").Value.String() != "" {
			setCondition()
			condition.RequiredChangeTypes = splitCSV(*requiredChangeTypes)
		}
		if fs.Lookup("required-touches").Value.String() != "" {
			setCondition()
			condition.RequiredTouches = splitCSV(*requiredTouches)
		}
		if fs.Lookup("missing-capabilities").Value.String() != "" {
			setCondition()
			condition.MissingCapabilities = splitCSV(*missingCapabilities)
		}
		if condition != nil {
			req.Conditions = condition
		}
		result, err := c.UpdatePolicy(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "enable":
		fs := flag.NewFlagSet("policy enable", flag.ExitOnError)
		id := fs.String("id", "", "policy id")
		_ = fs.Parse(args[1:])
		enabled := true
		result, err := c.UpdatePolicy(ctx, *id, types.UpdatePolicyRequest{Enabled: &enabled})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "disable":
		fs := flag.NewFlagSet("policy disable", flag.ExitOnError)
		id := fs.String("id", "", "policy id")
		_ = fs.Parse(args[1:])
		enabled := false
		result, err := c.UpdatePolicy(ctx, *id, types.UpdatePolicyRequest{Enabled: &enabled})
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

func handlePolicyDecision(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	fs := flag.NewFlagSet("policy-decision list", flag.ExitOnError)
	projectID := fs.String("project", "", "project id filter")
	policyID := fs.String("policy", "", "policy id filter")
	changeID := fs.String("change", "", "change set id filter")
	riskID := fs.String("risk", "", "risk assessment id filter")
	planID := fs.String("plan", "", "rollout plan id filter")
	rolloutID := fs.String("rollout", "", "rollout execution id filter")
	appliesTo := fs.String("applies-to", "", "workflow surface filter")
	limit := fs.Int("limit", 50, "maximum number of policy decisions")
	offset := fs.Int("offset", 0, "pagination offset")
	_ = fs.Parse(args[1:])
	query := make([]string, 0, 8)
	if *projectID != "" {
		query = append(query, "project_id="+url.QueryEscape(*projectID))
	}
	if *policyID != "" {
		query = append(query, "policy_id="+url.QueryEscape(*policyID))
	}
	if *changeID != "" {
		query = append(query, "change_set_id="+url.QueryEscape(*changeID))
	}
	if *riskID != "" {
		query = append(query, "risk_assessment_id="+url.QueryEscape(*riskID))
	}
	if *planID != "" {
		query = append(query, "rollout_plan_id="+url.QueryEscape(*planID))
	}
	if *rolloutID != "" {
		query = append(query, "rollout_execution_id="+url.QueryEscape(*rolloutID))
	}
	if *appliesTo != "" {
		query = append(query, "applies_to="+url.QueryEscape(*appliesTo))
	}
	if *limit > 0 {
		query = append(query, fmt.Sprintf("limit=%d", *limit))
	}
	if *offset > 0 {
		query = append(query, fmt.Sprintf("offset=%d", *offset))
	}
	result, err := c.ListPolicyDecisions(ctx, strings.Join(query, "&"))
	if !exitOnErr(stderr, err) {
		return 1
	}
	printJSON(stdout, result)
	return 0
}

func scopedStatusQuery(rollbackOnly bool, limit, offset int) string {
	query := make([]string, 0, 3)
	if rollbackOnly {
		query = append(query, "rollback_only=true")
	}
	if limit > 0 {
		query = append(query, fmt.Sprintf("limit=%d", limit))
	}
	if offset > 0 {
		query = append(query, fmt.Sprintf("offset=%d", offset))
	}
	return strings.Join(query, "&")
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
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("integrations create", flag.ExitOnError)
		orgID := fs.String("org", c.OrganizationID(), "organization id")
		kind := fs.String("kind", "", "integration kind")
		name := fs.String("name", "", "integration instance name")
		instanceKey := fs.String("instance-key", "", "instance key")
		scopeType := fs.String("scope-type", "organization", "scope type")
		scopeName := fs.String("scope-name", "", "scope name")
		mode := fs.String("mode", "advisory", "integration mode")
		authStrategy := fs.String("auth-strategy", "", "auth strategy")
		_ = fs.Parse(args[1:])
		result, err := c.CreateIntegration(ctx, types.CreateIntegrationRequest{
			OrganizationID: *orgID,
			Kind:           *kind,
			Name:           *name,
			InstanceKey:    *instanceKey,
			ScopeType:      *scopeType,
			ScopeName:      *scopeName,
			Mode:           *mode,
			AuthStrategy:   *authStrategy,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "list":
		fs := flag.NewFlagSet("integrations list", flag.ExitOnError)
		kind := fs.String("kind", "", "integration kind filter")
		instanceKey := fs.String("instance-key", "", "instance key filter")
		scopeType := fs.String("scope-type", "", "scope type filter")
		authStrategy := fs.String("auth-strategy", "", "auth strategy filter")
		enabled := fs.String("enabled", "", "enabled filter")
		search := fs.String("search", "", "search text")
		_ = fs.Parse(args[1:])
		query := make([]string, 0, 6)
		if *kind != "" {
			query = append(query, "kind="+url.QueryEscape(*kind))
		}
		if *instanceKey != "" {
			query = append(query, "instance_key="+url.QueryEscape(*instanceKey))
		}
		if *scopeType != "" {
			query = append(query, "scope_type="+url.QueryEscape(*scopeType))
		}
		if *authStrategy != "" {
			query = append(query, "auth_strategy="+url.QueryEscape(*authStrategy))
		}
		if *enabled != "" {
			query = append(query, "enabled="+url.QueryEscape(*enabled))
		}
		if *search != "" {
			query = append(query, "search="+url.QueryEscape(*search))
		}
		result, err := c.ListIntegrationsWithQuery(ctx, strings.Join(query, "&"))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("integrations show", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		_ = fs.Parse(args[1:])
		result, err := c.ListIntegrations(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		for _, integration := range result {
			if integration.ID == *id {
				printJSON(stdout, integration)
				return 0
			}
		}
		fmt.Fprintf(stderr, "integration %s not found\n", *id)
		return 1
	case "update":
		fs := flag.NewFlagSet("integrations update", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		name := fs.String("name", "", "integration name")
		mode := fs.String("mode", "", "integration mode (advisory|active_control)")
		status := fs.String("status", "", "integration status")
		enabled := fs.String("enabled", "", "set enabled true or false")
		controlEnabled := fs.String("control-enabled", "", "set control-enabled true or false")
		scheduleEnabled := fs.String("schedule-enabled", "", "set schedule-enabled true or false")
		scheduleInterval := fs.String("schedule-interval", "", "scheduled sync interval in seconds")
		staleAfter := fs.String("stale-after", "", "stale threshold in seconds")
		metadataJSON := fs.String("metadata-json", "", "integration metadata as JSON")
		_ = fs.Parse(args[1:])
		req := types.UpdateIntegrationRequest{}
		if strings.TrimSpace(*name) != "" {
			req.Name = name
		}
		if strings.TrimSpace(*mode) != "" {
			req.Mode = mode
		}
		if strings.TrimSpace(*status) != "" {
			req.Status = status
		}
		if strings.TrimSpace(*enabled) != "" {
			value, err := strconv.ParseBool(*enabled)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.Enabled = &value
		}
		if strings.TrimSpace(*controlEnabled) != "" {
			value, err := strconv.ParseBool(*controlEnabled)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.ControlEnabled = &value
		}
		if strings.TrimSpace(*scheduleEnabled) != "" {
			value, err := strconv.ParseBool(*scheduleEnabled)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.ScheduleEnabled = &value
		}
		if strings.TrimSpace(*scheduleInterval) != "" {
			value, err := strconv.Atoi(strings.TrimSpace(*scheduleInterval))
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.ScheduleIntervalSeconds = &value
		}
		if strings.TrimSpace(*staleAfter) != "" {
			value, err := strconv.Atoi(strings.TrimSpace(*staleAfter))
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.SyncStaleAfterSeconds = &value
		}
		if strings.TrimSpace(*metadataJSON) != "" {
			metadata, err := parseMetadata(*metadataJSON)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.Metadata = metadata
		}
		result, err := c.UpdateIntegration(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "coverage":
		result, err := c.IntegrationCoverageSummary(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "test":
		fs := flag.NewFlagSet("integrations test", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		_ = fs.Parse(args[1:])
		result, err := c.TestIntegration(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "sync":
		fs := flag.NewFlagSet("integrations sync", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		_ = fs.Parse(args[1:])
		result, err := c.SyncIntegration(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "runs":
		fs := flag.NewFlagSet("integrations runs", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		_ = fs.Parse(args[1:])
		result, err := c.ListIntegrationSyncRuns(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "github-start":
		fs := flag.NewFlagSet("integrations github-start", flag.ExitOnError)
		id := fs.String("id", "", "github integration id")
		_ = fs.Parse(args[1:])
		result, err := c.StartGitHubOnboarding(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "webhook-show":
		fs := flag.NewFlagSet("integrations webhook-show", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		_ = fs.Parse(args[1:])
		result, err := c.GetWebhookRegistration(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "webhook-sync":
		fs := flag.NewFlagSet("integrations webhook-sync", flag.ExitOnError)
		id := fs.String("id", "", "integration id")
		_ = fs.Parse(args[1:])
		result, err := c.SyncWebhookRegistration(ctx, *id)
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

func handleIdentityProviders(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "list":
		result, err := c.ListIdentityProviders(ctx)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "create":
		fs := flag.NewFlagSet("identity-provider create", flag.ExitOnError)
		orgID := fs.String("org", c.OrganizationID(), "organization id")
		name := fs.String("name", "", "identity provider name")
		kind := fs.String("kind", "oidc", "identity provider kind")
		issuerURL := fs.String("issuer-url", "", "issuer url")
		authorizationEndpoint := fs.String("authorization-endpoint", "", "authorization endpoint override")
		tokenEndpoint := fs.String("token-endpoint", "", "token endpoint override")
		userinfoEndpoint := fs.String("userinfo-endpoint", "", "userinfo endpoint override")
		clientID := fs.String("client-id", "", "oidc client id")
		clientSecretEnv := fs.String("client-secret-env", "", "env var name holding the client secret")
		allowedDomains := fs.String("allowed-domains", "", "comma-separated allowed email domains")
		defaultRole := fs.String("default-role", "org_member", "default organization role")
		enabled := fs.Bool("enabled", true, "enable provider for sign-in")
		_ = fs.Parse(args[1:])

		result, err := c.CreateIdentityProvider(ctx, types.CreateIdentityProviderRequest{
			OrganizationID:        *orgID,
			Name:                  *name,
			Kind:                  *kind,
			IssuerURL:             *issuerURL,
			AuthorizationEndpoint: *authorizationEndpoint,
			TokenEndpoint:         *tokenEndpoint,
			UserInfoEndpoint:      *userinfoEndpoint,
			ClientID:              *clientID,
			ClientSecretEnv:       *clientSecretEnv,
			AllowedDomains:        splitCSV(*allowedDomains),
			DefaultRole:           *defaultRole,
			Enabled:               *enabled,
		})
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "update":
		fs := flag.NewFlagSet("identity-provider update", flag.ExitOnError)
		id := fs.String("id", "", "identity provider id")
		name := fs.String("name", "", "provider name")
		issuerURL := fs.String("issuer-url", "", "issuer url")
		authorizationEndpoint := fs.String("authorization-endpoint", "", "authorization endpoint override")
		tokenEndpoint := fs.String("token-endpoint", "", "token endpoint override")
		userinfoEndpoint := fs.String("userinfo-endpoint", "", "userinfo endpoint override")
		clientID := fs.String("client-id", "", "oidc client id")
		clientSecretEnv := fs.String("client-secret-env", "", "env var name holding the client secret")
		allowedDomains := fs.String("allowed-domains", "", "comma-separated allowed email domains")
		defaultRole := fs.String("default-role", "", "default organization role")
		enabled := fs.String("enabled", "", "set enabled true or false")
		_ = fs.Parse(args[1:])

		req := types.UpdateIdentityProviderRequest{}
		if strings.TrimSpace(*name) != "" {
			req.Name = name
		}
		if strings.TrimSpace(*issuerURL) != "" {
			req.IssuerURL = issuerURL
		}
		if strings.TrimSpace(*authorizationEndpoint) != "" {
			req.AuthorizationEndpoint = authorizationEndpoint
		}
		if strings.TrimSpace(*tokenEndpoint) != "" {
			req.TokenEndpoint = tokenEndpoint
		}
		if strings.TrimSpace(*userinfoEndpoint) != "" {
			req.UserInfoEndpoint = userinfoEndpoint
		}
		if strings.TrimSpace(*clientID) != "" {
			req.ClientID = clientID
		}
		if strings.TrimSpace(*clientSecretEnv) != "" {
			req.ClientSecretEnv = clientSecretEnv
		}
		if strings.TrimSpace(*allowedDomains) != "" {
			domains := splitCSV(*allowedDomains)
			req.AllowedDomains = &domains
		}
		if strings.TrimSpace(*defaultRole) != "" {
			req.DefaultRole = defaultRole
		}
		if strings.TrimSpace(*enabled) != "" {
			value, err := strconv.ParseBool(*enabled)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			req.Enabled = &value
		}
		result, err := c.UpdateIdentityProvider(ctx, *id, req)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "test":
		fs := flag.NewFlagSet("identity-provider test", flag.ExitOnError)
		id := fs.String("id", "", "identity provider id")
		_ = fs.Parse(args[1:])
		result, err := c.TestIdentityProvider(ctx, *id)
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

func handleBrowserSessions(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("browser-session list", flag.ExitOnError)
		userID := fs.String("user", "", "user id filter")
		status := fs.String("status", "", "session status filter")
		limit := fs.Int("limit", 50, "maximum number of sessions")
		offset := fs.Int("offset", 0, "result offset")
		_ = fs.Parse(args[1:])

		params := url.Values{}
		if strings.TrimSpace(*userID) != "" {
			params.Set("user_id", strings.TrimSpace(*userID))
		}
		if strings.TrimSpace(*status) != "" {
			params.Set("status", strings.TrimSpace(*status))
		}
		if *limit > 0 {
			params.Set("limit", strconv.Itoa(*limit))
		}
		if *offset > 0 {
			params.Set("offset", strconv.Itoa(*offset))
		}
		result, err := c.ListBrowserSessions(ctx, params.Encode())
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "revoke":
		fs := flag.NewFlagSet("browser-session revoke", flag.ExitOnError)
		id := fs.String("id", "", "browser session id")
		_ = fs.Parse(args[1:])
		result, err := c.RevokeBrowserSession(ctx, *id)
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

func handleDiscovery(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("discovery list", flag.ExitOnError)
		integrationID := fs.String("integration", "", "integration id filter")
		resourceType := fs.String("type", "", "resource type filter")
		provider := fs.String("provider", "", "provider filter")
		projectID := fs.String("project", "", "project id filter")
		serviceID := fs.String("service", "", "service id filter")
		environmentID := fs.String("env", "", "environment id filter")
		repositoryID := fs.String("repo", "", "repository id filter")
		status := fs.String("status", "", "resource status filter")
		search := fs.String("search", "", "search text")
		unmappedOnly := fs.Bool("unmapped-only", false, "show only unmapped discovered resources")
		limit := fs.Int("limit", 100, "maximum number of resources")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		query := make([]string, 0, 11)
		if *integrationID != "" {
			query = append(query, "integration_id="+url.QueryEscape(*integrationID))
		}
		if *resourceType != "" {
			query = append(query, "resource_type="+url.QueryEscape(*resourceType))
		}
		if *provider != "" {
			query = append(query, "provider="+url.QueryEscape(*provider))
		}
		if *projectID != "" {
			query = append(query, "project_id="+url.QueryEscape(*projectID))
		}
		if *serviceID != "" {
			query = append(query, "service_id="+url.QueryEscape(*serviceID))
		}
		if *environmentID != "" {
			query = append(query, "environment_id="+url.QueryEscape(*environmentID))
		}
		if *repositoryID != "" {
			query = append(query, "repository_id="+url.QueryEscape(*repositoryID))
		}
		if *status != "" {
			query = append(query, "status="+url.QueryEscape(*status))
		}
		if *search != "" {
			query = append(query, "search="+url.QueryEscape(*search))
		}
		if *unmappedOnly {
			query = append(query, "unmapped_only=true")
		}
		if *limit > 0 {
			query = append(query, fmt.Sprintf("limit=%d", *limit))
		}
		if *offset > 0 {
			query = append(query, fmt.Sprintf("offset=%d", *offset))
		}
		result, err := c.ListDiscoveredResources(ctx, strings.Join(query, "&"))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "map":
		fs := flag.NewFlagSet("discovery map", flag.ExitOnError)
		id := fs.String("id", "", "discovered resource id")
		projectID := fs.String("project", "", "project id")
		serviceID := fs.String("service", "", "service id")
		environmentID := fs.String("env", "", "environment id")
		repositoryID := fs.String("repo", "", "repository id")
		status := fs.String("status", "", "discovered resource status")
		_ = fs.Parse(args[1:])
		req := types.UpdateDiscoveredResourceRequest{}
		if strings.TrimSpace(*projectID) != "" {
			req.ProjectID = projectID
		}
		if strings.TrimSpace(*serviceID) != "" {
			req.ServiceID = serviceID
		}
		if strings.TrimSpace(*environmentID) != "" {
			req.EnvironmentID = environmentID
		}
		if strings.TrimSpace(*repositoryID) != "" {
			req.RepositoryID = repositoryID
		}
		if strings.TrimSpace(*status) != "" {
			req.Status = status
		}
		result, err := c.UpdateDiscoveredResource(ctx, *id, req)
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

func handleGraph(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("graph list", flag.ExitOnError)
		sourceIntegrationID := fs.String("source-integration", "", "source integration id filter")
		relationshipType := fs.String("type", "", "relationship type filter")
		fromResourceID := fs.String("from", "", "from resource id filter")
		toResourceID := fs.String("to", "", "to resource id filter")
		limit := fs.Int("limit", 100, "maximum number of graph relationships")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		query := make([]string, 0, 6)
		if strings.TrimSpace(*sourceIntegrationID) != "" {
			query = append(query, "source_integration_id="+url.QueryEscape(*sourceIntegrationID))
		}
		if strings.TrimSpace(*relationshipType) != "" {
			query = append(query, "relationship_type="+url.QueryEscape(*relationshipType))
		}
		if strings.TrimSpace(*fromResourceID) != "" {
			query = append(query, "from_resource_id="+url.QueryEscape(*fromResourceID))
		}
		if strings.TrimSpace(*toResourceID) != "" {
			query = append(query, "to_resource_id="+url.QueryEscape(*toResourceID))
		}
		if *limit > 0 {
			query = append(query, fmt.Sprintf("limit=%d", *limit))
		}
		if *offset > 0 {
			query = append(query, fmt.Sprintf("offset=%d", *offset))
		}
		result, err := c.ListGraphRelationships(ctx, strings.Join(query, "&"))
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

func handleRepository(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("repository list", flag.ExitOnError)
		provider := fs.String("provider", "", "repository provider filter")
		sourceIntegrationID := fs.String("source-integration", "", "source integration id filter")
		_ = fs.Parse(args[1:])
		query := make([]string, 0, 2)
		if strings.TrimSpace(*provider) != "" {
			query = append(query, "provider="+url.QueryEscape(*provider))
		}
		if strings.TrimSpace(*sourceIntegrationID) != "" {
			query = append(query, "source_integration_id="+url.QueryEscape(*sourceIntegrationID))
		}
		result, err := c.ListRepositories(ctx, strings.Join(query, "&"))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "map":
		fs := flag.NewFlagSet("repository map", flag.ExitOnError)
		id := fs.String("id", "", "repository id")
		projectID := fs.String("project", "", "project id")
		serviceID := fs.String("service", "", "service id")
		environmentID := fs.String("env", "", "environment id")
		status := fs.String("status", "mapped", "repository status")
		_ = fs.Parse(args[1:])
		req := types.UpdateRepositoryRequest{Status: status}
		if strings.TrimSpace(*projectID) != "" {
			req.ProjectID = projectID
		}
		if strings.TrimSpace(*serviceID) != "" {
			req.ServiceID = serviceID
		}
		if strings.TrimSpace(*environmentID) != "" {
			req.EnvironmentID = environmentID
		}
		result, err := c.UpdateRepository(ctx, *id, req)
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

func handleOutbox(ctx context.Context, c *client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("outbox list", flag.ExitOnError)
		eventType := fs.String("event-type", "", "event type filter")
		status := fs.String("status", "", "status filter")
		limit := fs.Int("limit", 100, "maximum number of outbox events")
		offset := fs.Int("offset", 0, "pagination offset")
		_ = fs.Parse(args[1:])
		query := make([]string, 0, 4)
		if strings.TrimSpace(*eventType) != "" {
			query = append(query, "event_type="+url.QueryEscape(*eventType))
		}
		if strings.TrimSpace(*status) != "" {
			query = append(query, "status="+url.QueryEscape(*status))
		}
		if *limit > 0 {
			query = append(query, fmt.Sprintf("limit=%d", *limit))
		}
		if *offset > 0 {
			query = append(query, fmt.Sprintf("offset=%d", *offset))
		}
		result, err := c.ListOutboxEvents(ctx, strings.Join(query, "&"))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "retry":
		fs := flag.NewFlagSet("outbox retry", flag.ExitOnError)
		id := fs.String("id", "", "outbox event id")
		_ = fs.Parse(args[1:])
		result, err := c.RetryOutboxEvent(ctx, *id)
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "requeue":
		fs := flag.NewFlagSet("outbox requeue", flag.ExitOnError)
		id := fs.String("id", "", "outbox event id")
		_ = fs.Parse(args[1:])
		result, err := c.RequeueOutboxEvent(ctx, *id)
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

func handleIncident(ctx context.Context, c *client.Client, session cliSession, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	c.SetOrganizationID(session.OrganizationID)
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("incident list", flag.ExitOnError)
		projectID := fs.String("project", "", "project id filter")
		serviceID := fs.String("service", "", "service id filter")
		environmentID := fs.String("env", "", "environment id filter")
		changeSetID := fs.String("change", "", "change set id filter")
		severity := fs.String("severity", "", "severity filter")
		status := fs.String("status", "", "status filter")
		search := fs.String("search", "", "search text")
		limit := fs.Int("limit", 0, "maximum incidents to return")
		_ = fs.Parse(args[1:])

		query := make([]string, 0, 8)
		if strings.TrimSpace(*projectID) != "" {
			query = append(query, "project_id="+url.QueryEscape(*projectID))
		}
		if strings.TrimSpace(*serviceID) != "" {
			query = append(query, "service_id="+url.QueryEscape(*serviceID))
		}
		if strings.TrimSpace(*environmentID) != "" {
			query = append(query, "environment_id="+url.QueryEscape(*environmentID))
		}
		if strings.TrimSpace(*changeSetID) != "" {
			query = append(query, "change_set_id="+url.QueryEscape(*changeSetID))
		}
		if strings.TrimSpace(*severity) != "" {
			query = append(query, "severity="+url.QueryEscape(*severity))
		}
		if strings.TrimSpace(*status) != "" {
			query = append(query, "status="+url.QueryEscape(*status))
		}
		if strings.TrimSpace(*search) != "" {
			query = append(query, "search="+url.QueryEscape(*search))
		}
		if *limit > 0 {
			query = append(query, fmt.Sprintf("limit=%d", *limit))
		}

		result, err := c.ListIncidents(ctx, strings.Join(query, "&"))
		if !exitOnErr(stderr, err) {
			return 1
		}
		printJSON(stdout, result)
		return 0
	case "show":
		fs := flag.NewFlagSet("incident show", flag.ExitOnError)
		id := fs.String("id", "", "incident id")
		_ = fs.Parse(args[1:])

		result, err := c.GetIncidentDetail(ctx, *id)
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

func printJSON(w io.Writer, v any) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(v)
}

func parseMetadata(raw string) (types.Metadata, error) {
	metadata := types.Metadata{}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	items := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func parseConfigEntriesJSON(raw string) ([]types.ConfigEntry, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var entries []types.ConfigEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("invalid entries-json payload: %w", err)
	}
	return entries, nil
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
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
  ccp team list
  ccp team create --org <org_id> --project <project_id> --name core --slug core --owners <user_id_1>,<user_id_2>
  ccp team show --id <team_id>
  ccp team update --id <team_id> --name "Platform Core" --owners <user_id_1>
  ccp team archive --id <team_id>
  ccp service list
  ccp service register --org <org_id> --project <project_id> --team <team_id> --name api --slug api
  ccp service update --id <service_id> --name api-v2 --description "..."
  ccp service archive --id <service_id>
  ccp env list
  ccp env create --org <org_id> --project <project_id> --name production --slug prod --type production --production
  ccp env update --id <env_id> --name "Production"
  ccp env archive --id <env_id>
  ccp config-set list
  ccp config-set show --id <config_set_id>
  ccp config-set create --org <org_id> --project <project_id> --env <env_id> --service <service_id> --name prod-app --version v1 --entries-json '[{"key":"DB_PASSWORD_REF","value":"prod/payments/db/password","value_type":"secret_ref"}]'
  ccp release list
  ccp release show --id <release_id>
  ccp release create --org <org_id> --project <project_id> --env <env_id> --name "April bundle" --version 2026.04.23 --changes <change_id_1>,<change_id_2> --config-sets <config_set_id>
  ccp db-connection list
  ccp db-connection create --org <org_id> --project <project_id> --env <env_id> --name primary --datastore checkout-primary --source-type env_dsn --dsn-env CCP_DB_DSN --summary "Read-only validation reference"
  ccp db-connection create --org <org_id> --project <project_id> --env <env_id> --name primary-secret-ref --datastore checkout-primary --source-type secret_ref_dsn --secret-ref prod/checkout/db/runtime_dsn --secret-ref-env CCP_CHECKOUT_RUNTIME_DSN --summary "Logical secret-ref-backed read-only validation reference"
  ccp db-connection test --id <database_connection_ref_id>
  ccp db-check create --org <org_id> --project <project_id> --env <env_id> --change <change_id> --conn-ref <db_connection_ref_id> --name "Orders table exists" --type existence_assertion --mode runtime_read_only --spec '{"subject":"table","schema":"public","table":"organizations"}' --summary "Confirm the table exists before rollout"
  ccp db-check execute --id <database_check_id>
  ccp db-execution list --check <database_check_id>
  ccp service-account create --org <org_id> --name deployer --role org_member
  ccp service-account list
  ccp token issue --service-account <service_account_id> --name primary
  ccp token revoke --service-account <service_account_id> --id <token_id>
  ccp change list
  ccp change show --id <change_id>
  ccp change analyze --org <org_id> --project <project_id> --service <service_id> --env <environment_id> --summary "..." --type code
  ccp risk list
  ccp rollout-plan list
  ccp rollout plan --change <change_id>
  ccp rollout execute --plan <rollout_plan_id> --release <release_id>
  ccp rollout list
  ccp rollout show --id <rollout_execution_id>
  ccp rollout status --id <rollout_execution_id>
  ccp rollout evidence --id <rollout_execution_id>
  ccp rollout advance --id <rollout_execution_id> --action approve --reason "approved"
  ccp rollout pause --id <rollout_execution_id> --reason "operator pause"
  ccp rollout resume --id <rollout_execution_id> --reason "resume after mitigation"
  ccp rollout rollback --id <rollout_execution_id> --reason "rollback due to verification failure"
  ccp rollout timeline --id <rollout_execution_id>
  ccp rollout reconcile --id <rollout_execution_id>
  ccp rollout watch --id <rollout_execution_id> --iterations 10 --interval 2s
  ccp status list --rollout <rollout_execution_id> --rollback-only --source kubernetes --event-type rollout.execution.action_suppressed
  ccp status show --id <status_event_id>
  ccp status project --id <project_id> --rollback-only
  ccp status service --id <service_id> --rollback-only
  ccp status env --id <environment_id> --rollback-only
  ccp rollback-policy list
  ccp rollback-policy create --org <org_id> --service <service_id> --env <env_id> --name "Prod strict" --max-error-rate 1
  ccp policy list
  ccp policy show --id <policy_id>
  ccp policy create --org <org_id> --project <project_id> --service <service_id> --env <env_id> --name "Prod Review" --applies-to rollout_plan --mode require_manual_review --production-only --min-risk-level high
  ccp policy update --id <policy_id> --description "..." --priority 100 --enabled false
  ccp policy enable --id <policy_id>
  ccp policy disable --id <policy_id>
  ccp policy-decision list --risk <risk_assessment_id> --policy <policy_id>
  ccp signal ingest --rollout <rollout_execution_id> --health healthy --summary "latency stable" --latency 145 --error-rate 0.2
  ccp verification record --rollout <rollout_execution_id> --outcome pass --decision continue --summary "healthy"
  ccp integrations list
  ccp integrations show --id <integration_id>
  ccp integrations update --id <integration_id> --enabled true --mode advisory --schedule-enabled true --schedule-interval 300 --stale-after 900 --metadata-json '{"access_token_env":"CCP_GITHUB_TOKEN"}'
  ccp integrations coverage
  ccp integrations test --id <integration_id>
  ccp integrations sync --id <integration_id>
  ccp integrations runs --id <integration_id>
  ccp integrations webhook-show --id <integration_id>
  ccp integrations webhook-sync --id <integration_id>
  ccp identity-provider list
  ccp identity-provider create --org <org_id> --name "Acme Okta" --issuer-url https://issuer.example.com --client-id abc --client-secret-env CCP_OKTA_SECRET
  ccp identity-provider update --id <provider_id> --allowed-domains acme.com,contractors.acme.com --enabled true
  ccp identity-provider test --id <provider_id>
  ccp browser-session list --status active --limit 25
  ccp browser-session revoke --id <browser_session_id>
  ccp graph list --type team_repository_owner --limit 50
  ccp repository list
  ccp repository map --id <repository_id> --service <service_id> --env <environment_id>
  ccp discovery list --integration <integration_id> --unmapped-only
  ccp discovery map --id <resource_id> --service <service_id> --env <environment_id>
  ccp outbox list --status error --limit 25
  ccp outbox retry --id <outbox_event_id>
  ccp outbox requeue --id <outbox_event_id>
  ccp audit list
  ccp incident list --service <service_id> --severity high --status monitoring --search checkout --limit 10
  ccp incident show --id <incident_id>`)
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
