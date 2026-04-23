import { RouteDefinition, routes } from "../app/router";
import {
  APIToken,
  AuditEvent,
  BrowserSessionInfo,
  BasicMetrics,
  CatalogSummary,
  ChangeSet,
  ControlPlaneState,
  CoverageSummary,
  DiscoveredResource,
  GraphRelationship,
  IdentityProvider,
  Incident,
  IncidentDetail,
  Integration,
  IntegrationSyncRun,
  OutboxEvent,
  Policy,
  PolicyDecision,
  Project,
  Repository,
  RiskAssessment,
  RollbackPolicy,
  RolloutExecution,
  RolloutExecutionDetail,
  RolloutExecutionRuntimeSummary,
  RolloutPlan,
  Service,
  ServiceAccount,
  StatusEvent,
  StatusEventQueryResult,
  Team,
  VerificationResult,
  WebhookRegistration
} from "../lib/api";

export type AuthMode = "login" | "signup";

const EMPTY_METRICS: BasicMetrics = {
  organizations: 0,
  projects: 0,
  teams: 0,
  services: 0,
  environments: 0,
  changes: 0,
  risk_assessments: 0,
  rollout_plans: 0,
  audit_events: 0,
  policies: 0,
  integrations: 0
};

const EMPTY_CATALOG: CatalogSummary = {
  services: [],
  environments: []
};

const EMPTY_STATUS_DASHBOARD: StatusEventQueryResult = {
  events: [],
  summary: {
    total: 0,
    returned: 0,
    limit: 50,
    offset: 0,
    rollback_events: 0,
    automated_events: 0
  }
};

const EMPTY_COVERAGE_SUMMARY: CoverageSummary = {
  enabled_integrations: 0,
  stale_integrations: 0,
  healthy_integrations: 0,
  repositories: 0,
  unmapped_repositories: 0,
  discovered_resources: 0,
  unmapped_discovered_resources: 0,
  workload_coverage_environments: 0,
  signal_coverage_services: 0
};

const EMPTY_PROJECTS: Project[] = [];
const EMPTY_TEAMS: Team[] = [];
const EMPTY_POLICIES: Policy[] = [];
const EMPTY_POLICY_DECISIONS: PolicyDecision[] = [];
const EMPTY_INTEGRATIONS: Integration[] = [];
const EMPTY_INCIDENTS: Incident[] = [];
const EMPTY_CHANGES: ChangeSet[] = [];
const EMPTY_RISK_ASSESSMENTS: RiskAssessment[] = [];
const EMPTY_ROLLOUT_PLANS: RolloutPlan[] = [];
const EMPTY_ROLLOUT_EXECUTIONS: RolloutExecution[] = [];
const EMPTY_AUDIT_EVENTS: AuditEvent[] = [];
const EMPTY_ROLLBACK_POLICIES: RollbackPolicy[] = [];
const EMPTY_STATUS_EVENTS: StatusEvent[] = [];
const EMPTY_GRAPH_RELATIONSHIPS: GraphRelationship[] = [];
const EMPTY_REPOSITORIES: Repository[] = [];
const EMPTY_DISCOVERED_RESOURCES: DiscoveredResource[] = [];
const EMPTY_SYNC_RUNS: Record<string, IntegrationSyncRun[]> = {};
const EMPTY_WEBHOOK_REGISTRATIONS: Record<string, WebhookRegistration | null> = {};
const EMPTY_IDENTITY_PROVIDERS: IdentityProvider[] = [];
const EMPTY_OUTBOX_EVENTS: OutboxEvent[] = [];
const EMPTY_BROWSER_SESSIONS: BrowserSessionInfo[] = [];
const EMPTY_SERVICE_ACCOUNTS: ServiceAccount[] = [];
const EMPTY_SERVICE_ACCOUNT_TOKENS: Record<string, APIToken[]> = {};

export function renderShell(state: ControlPlaneState, route: RouteDefinition, authMode: AuthMode = "login"): string {
  if (!state.session.authenticated) {
    return renderSignIn(state, authMode);
  }

  const organizations = state.session.organizations || [];
  const activeOrganizationID = state.session.active_organization_id || "";
  if (!activeOrganizationID) {
    return renderAwaitingAccess(state);
  }

  return `
    <div class="app-shell">
      <aside class="sidebar surface">
        <div class="brand-block">
          <p class="eyebrow">ChangeControlPlane</p>
          <h1>Governed software change, end to end.</h1>
          <p class="lede">
            Autonomous change control for delivery, infrastructure, reliability, compliance, and cost-aware DevOps.
          </p>
        </div>
        <nav class="nav-group">
          <p class="nav-label">Control Plane</p>
          ${renderNav("primary", route.key)}
        </nav>
        <nav class="nav-group">
          <p class="nav-label">Governance</p>
          ${renderNav("secondary", route.key)}
        </nav>
      </aside>

      <main class="content">
        <header class="topbar surface">
          <div>
            <p class="eyebrow">Operational View</p>
            <h2>${route.label}</h2>
            <p class="subtitle">${route.subtitle}</p>
          </div>
          <div class="topbar-actions">
            <label class="scope-picker">
              <span>Organization</span>
              <select id="organization-switcher">
                ${organizations
                  .map(
                    (organization) => `
                      <option value="${organization.organization_id}" ${organization.organization_id === activeOrganizationID ? "selected" : ""}>
                        ${organization.organization} (${organization.role})
                      </option>
                    `
                  )
                  .join("")}
              </select>
            </label>
            <button class="action ghost" id="refresh-button">Refresh Data</button>
            <button class="action ghost" id="logout-button">Sign Out</button>
            <a class="action" href="#/bootstrap">Bootstrap Workspace</a>
          </div>
        </header>
        <div id="app-feedback" class="app-feedback" hidden></div>

        ${route.key === "dashboard" ? renderDashboardHero(state) : ""}
        ${renderPage(state, route.key)}
      </main>
    </div>
  `;
}

function renderSignIn(state: ControlPlaneState, authMode: AuthMode): string {
  const loginMode = authMode === "login";
  const providers = state.publicIdentityProviders || [];
  return `
    <div class="auth-shell">
      <section class="auth-panel surface">
        <p class="eyebrow">ChangeControlPlane</p>
        <h1>${loginMode ? "Log in to the control plane." : "Create your control-plane account."}</h1>
        <div class="auth-mode-switch" role="tablist" aria-label="Authentication Mode">
          <button
            id="auth-mode-login"
            class="${loginMode ? "auth-mode-tab active" : "auth-mode-tab"}"
            type="button"
            data-auth-mode="login"
            aria-pressed="${loginMode ? "true" : "false"}"
          >
            Log In
          </button>
          <button
            id="auth-mode-signup"
            class="${!loginMode ? "auth-mode-tab active" : "auth-mode-tab"}"
            type="button"
            data-auth-mode="signup"
            aria-pressed="${!loginMode ? "true" : "false"}"
          >
            Sign Up
          </button>
        </div>
        ${loginMode
          ? `
            <form id="login-form" class="login-form auth-form auth-form-login" novalidate>
              <label>
                <span>Email</span>
                <input id="login-email" name="email" type="email" placeholder="owner@acme.local" autocomplete="email" />
              </label>
              <label>
                <span>Password</span>
                <div class="password-input">
                  <input id="login-password" name="password" type="password" placeholder="Enter your password" autocomplete="current-password" />
                  ${passwordToggle("login-password")}
                </div>
              </label>
              <button id="login-submit" class="action" type="submit">Log In</button>
            </form>
          `
          : `
            <form id="signup-form" class="login-form auth-form auth-form-signup" novalidate>
              <label>
                <span>Email</span>
                <input id="signup-email" name="email" type="email" placeholder="owner@acme.local" autocomplete="email" />
              </label>
              <label>
                <span>Display Name</span>
                <input id="signup-display-name" name="display_name" type="text" placeholder="Acme Owner" autocomplete="name" />
              </label>
              <label>
                <span>Password</span>
                <div class="password-input">
                  <input id="signup-password" name="password" type="password" placeholder="Create a password" autocomplete="new-password" />
                  ${passwordToggle("signup-password")}
                </div>
              </label>
              <label>
                <span>Confirm Password</span>
                <div class="password-input">
                  <input id="signup-password-confirmation" name="password_confirmation" type="password" placeholder="Re-enter your password" autocomplete="new-password" />
                  ${passwordToggle("signup-password-confirmation")}
                </div>
              </label>
              <button id="signup-submit" class="action" type="submit">Create Account</button>
            </form>
          `}
        ${providers.length > 0 ? renderEnterpriseSignInOptions(providers, loginMode) : ""}
        <div id="app-feedback" class="app-feedback" hidden></div>
      </section>
    </div>
  `;
}

function passwordToggle(target: string): string {
  return `
    <button
      class="password-toggle"
      type="button"
      data-password-toggle="${target}"
      aria-label="Show password"
      aria-pressed="false"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path d="M1.5 12s3.8-6 10.5-6 10.5 6 10.5 6-3.8 6-10.5 6S1.5 12 1.5 12Z" />
        <circle cx="12" cy="12" r="3.25" />
      </svg>
    </button>
  `;
}

function renderAwaitingAccess(state: ControlPlaneState): string {
  return `
    <div class="auth-shell">
      <section class="auth-panel surface">
        <p class="eyebrow">ChangeControlPlane</p>
        <h1>Waiting for organization access.</h1>
        <p class="lede">
          You are signed in as ${state.session.email || state.session.actor}. Once an administrator grants this email access, your organization will appear here.
        </p>
        <p class="auth-note">
          Use refresh after access is granted, or sign out and return later.
        </p>
        <div class="auth-actions">
          <button class="action" id="refresh-button" type="button">Refresh Access</button>
          <button class="action ghost" id="logout-button" type="button">Sign Out</button>
        </div>
        <div id="app-feedback" class="app-feedback" hidden></div>
      </section>
    </div>
  `;
}

function renderNav(group: "primary" | "secondary", currentRoute: string): string {
  return routes
    .filter((route) => route.nav === group)
    .map((route) => {
      const active = route.key === currentRoute ? "nav-link active" : "nav-link";
      return `<a class="${active}" href="#/${route.key}">${route.label}</a>`;
    })
    .join("");
}

function renderPage(state: ControlPlaneState, routeKey: string): string {
  const catalog = catalogForState(state);
  const changes = changesForState(state);
  const riskAssessments = riskAssessmentsForState(state);
  const rolloutPlans = rolloutPlansForState(state);
  const rolloutExecutions = rolloutExecutionsForState(state);
  const rolloutExecutionDetail = rolloutExecutionDetailForState(state);
  const integrations = integrationsForState(state);
  const incidents = incidentsForState(state);
  const incidentDetail = incidentDetailForState(state);
  const incidentDetailStatus = incidentDetailStatusForState(state);
  const selectedIncidentID = selectedIncidentIDForState(state);
  const policies = policiesForState(state);
  const policyDecisions = policyDecisionsForState(state);
  const auditEvents = auditEventsForState(state);
  const rollbackPolicies = rollbackPoliciesForState(state);
  const statusEvents = statusEventsForState(state);
  const statusDashboard = statusDashboardForState(state);
  const graphRelationships = graphRelationshipsForState(state);
  const repositories = repositoriesForState(state);
  const discoveredResources = discoveredResourcesForState(state);
  const coverageSummary = coverageSummaryForState(state);
  const serviceAccounts = serviceAccountsForState(state);
  const serviceAccountTokens = serviceAccountTokensForState(state);
  const identityProviders = identityProvidersForState(state);
  const outboxEvents = outboxEventsForState(state);
  const browserSessions = browserSessionsForState(state);
  const integrationSyncRuns = integrationSyncRunsForState(state);
  const webhookRegistrations = webhookRegistrationsForState(state);
  const metrics = metricsForState(state);
  const projects = projectsForState(state);
  const teams = teamsForState(state);
  const primaryTeam = teams[0];
  const primaryService = catalog.services[0];
  const primaryEnvironment = catalog.environments[0];
  const latestChange = changes[0];
  const latestRisk = riskAssessments[0];
  const latestRollout = rolloutPlans[0];
  const latestExecution = rolloutExecutions[0];
  const latestExecutionDetail = rolloutExecutionDetail;
  const latestRuntimeSummary = latestExecutionDetail?.runtime_summary;
  const latestBackendIntegration = latestExecution?.backend_integration_id
    ? integrations.find((integration) => integration.id === latestExecution.backend_integration_id)
    : undefined;
  const latestExecutionAdvisoryOnly = Boolean(latestRuntimeSummary?.advisory_only);
  const latestIncident = incidents[0];
  const selectedIncident = incidentDetail?.incident;
  const selectedIncidentTimeline = incidentDetail?.status_timeline || [];
  const canAdmin = activeOrganizationRole(state) === "org_admin";
  const canOperate = ["org_admin", "org_member"].includes(activeOrganizationRole(state));
  const activeServiceExecutions = primaryService ? rolloutExecutions.filter((execution) => execution.service_id === primaryService.id) : [];
  const activeEnvironmentExecutions = primaryEnvironment ? rolloutExecutions.filter((execution) => execution.environment_id === primaryEnvironment.id) : [];
  const rollbackEvents = statusEvents.filter((event) => event.event_type.includes("rollback") || event.new_state === "rolled_back");
  const productionEnvironments = catalog.environments.filter((environment) => environment.production);

  switch (routeKey) {
    case "catalog":
      if (state.catalogPage.status === "loading" || state.catalogPage.status === "idle") {
        return routeStatusLayout(
          "Service Catalog",
          "Loading route-local catalog data for the active organization.",
          emptyState("Loading service catalog", "Fetching services and environments for the catalog surface.")
        );
      }
      if (state.catalogPage.status === "error") {
        return routeStatusLayout(
          "Service Catalog",
          "The route-local catalog read failed.",
          emptyState("Catalog data unavailable", state.catalogPage.error || "Refresh and retry the catalog surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Service Catalog</h3>
              <p>Ownership, criticality, and operational coverage in one place.</p>
            </div>
            ${table(
              ["Service", "Criticality", "Customer", "SLO", "Observability", "Dependencies"],
              catalog.services.map((service) => [
                service.name,
                service.criticality,
                service.customer_facing ? "Yes" : "No",
                service.has_slo ? "Yes" : "No",
                service.has_observability ? "Yes" : "No",
                String(service.dependent_services_count)
              ])
            )}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Environment Matrix</h3>
              <p>Deployment and compliance surface by environment.</p>
            </div>
            ${table(
              ["Environment", "Type", "Region", "Production", "Compliance"],
              catalog.environments.map((environment) => [
                environment.name,
                environment.type,
                environment.region,
                environment.production ? "Yes" : "No",
                environment.compliance_zone || "Standard"
              ])
            )}
          </article>
        </section>
      `;
    case "service":
      if (state.servicePage.status === "loading" || state.servicePage.status === "idle") {
        return routeStatusLayout(
          "Service Detail",
          "Loading route-local service data for the active organization.",
          emptyState("Loading service detail", "Fetching project ownership, catalog services, and rollout execution posture.")
        );
      }
      if (state.servicePage.status === "error") {
        return routeStatusLayout(
          "Service Detail",
          "The route-local service read failed.",
          emptyState("Service data unavailable", state.servicePage.error || "Refresh and retry the service surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Service Detail</h3>
              <p>${primaryService ? `${primaryService.name} is classified ${primaryService.criticality} with status ${primaryService.status || "active"}.` : "Register a service to see criticality, dependencies, and rollout posture here."}</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Current Focus", primaryService?.description || "No service selected yet.")}
              ${infoCard("Dependency Reach", primaryService ? `${primaryService.dependent_services_count} downstream service references` : "Awaiting service registration.")}
              ${infoCard("Governance Flags", primaryService?.regulated_zone ? "Regulated controls in scope." : "Standard controls.")}
              ${infoCard("Execution Readiness", primaryService?.has_observability ? "Verification-ready with observability coverage." : "Needs stronger verification coverage.")}
              ${infoCard("Active Executions", activeServiceExecutions.length > 0 ? activeServiceExecutions.map((execution) => `${execution.status} (${execution.backend_status || "pending"})`).join(", ") : "No active rollout executions for this service.")}
            </div>
            ${catalog.services.length > 0 ? table(["Service", "Criticality", "Status", "Customer"], catalog.services.map((service) => [service.name, service.criticality, service.status || "active", service.customer_facing ? "Yes" : "No"])) : emptyState("No services yet", "Create a service to unlock change review, rollout execution, and graph enrichment.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Create Service</h3>
              <p>Register a governed service in the active organization scope.</p>
            </div>
            ${canAdmin ? `
              <form id="create-service-form" class="stack-form">
                <label><span>Name</span><input name="name" type="text" placeholder="Checkout API" required /></label>
                <label><span>Slug</span><input name="slug" type="text" placeholder="checkout-api" required /></label>
                <label><span>Project</span><select name="project_id">${projectOptions(state)}</select></label>
                <label><span>Team</span><select name="team_id">${teamOptions(state)}</select></label>
                <label><span>Criticality</span><input name="criticality" type="text" placeholder="mission_critical" /></label>
                <label><span>Description</span><textarea name="description" placeholder="Customer-facing payments API"></textarea></label>
                <button class="action" type="submit">Create Service</button>
              </form>
            ` : emptyState("Read-only view", "Only organization administrators can register or archive services in this workspace.")}
            ${canAdmin && primaryService
              ? `
                <form id="update-service-form" class="stack-form compact-top-form" data-service-id="${primaryService.id}">
                  <label><span>Name</span><input name="name" type="text" value="${primaryService.name}" required /></label>
                  <label><span>Slug</span><input name="slug" type="text" value="${primaryService.slug}" required /></label>
                  <label><span>Criticality</span><input name="criticality" type="text" value="${primaryService.criticality}" /></label>
                  <label><span>Description</span><textarea name="description" placeholder="Customer-facing payments API">${primaryService.description || ""}</textarea></label>
                  <button class="action ghost" type="submit">Save Current Service</button>
                </form>
                <button class="action ghost" id="archive-service-button" data-service-id="${primaryService.id}">Archive Current Service</button>
              `
              : ""}
          </article>
        </section>
      `;
    case "environment":
      if (state.environmentPage.status === "loading" || state.environmentPage.status === "idle") {
        return routeStatusLayout(
          "Environment",
          "Loading route-local environment data for the active organization.",
          emptyState("Loading environment detail", "Fetching project context, environment targets, and rollout execution posture.")
        );
      }
      if (state.environmentPage.status === "error") {
        return routeStatusLayout(
          "Environment",
          "The route-local environment read failed.",
          emptyState("Environment data unavailable", state.environmentPage.error || "Refresh and retry the environment surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Environment Detail</h3>
              <p>${primaryEnvironment ? `${primaryEnvironment.name} is ${primaryEnvironment.production ? "production" : primaryEnvironment.type} in ${primaryEnvironment.region}.` : "Create environments to model rollout paths, freeze windows, and compliance context."}</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Promotion Posture", primaryEnvironment?.production ? "Production promotions require governed rollout." : "Pre-production environment.")}
              ${infoCard("Compliance Context", primaryEnvironment?.compliance_zone || "Default operating zone")}
              ${infoCard("Operational View", "Drift detection, provisioning policy, and verification signals fit here next.")}
              ${infoCard("Status", primaryEnvironment?.status || "Awaiting environment registration")}
              ${infoCard("Active Executions", activeEnvironmentExecutions.length > 0 ? activeEnvironmentExecutions.map((execution) => `${execution.status} (${execution.backend_status || "pending"})`).join(", ") : "No active rollout executions for this environment.")}
            </div>
            ${catalog.environments.length > 0 ? table(["Environment", "Type", "Region", "Status"], catalog.environments.map((environment) => [environment.name, environment.type, environment.region, environment.status || "active"])) : emptyState("No environments yet", "Create environments to drive rollout targeting and runtime verification context.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Create Environment</h3>
              <p>Model a rollout target with compliance and production posture.</p>
            </div>
            ${canAdmin ? `
              <form id="create-environment-form" class="stack-form">
                <label><span>Name</span><input name="name" type="text" placeholder="Production" required /></label>
                <label><span>Slug</span><input name="slug" type="text" placeholder="prod" required /></label>
                <label><span>Project</span><select name="project_id">${projectOptions(state)}</select></label>
                <label><span>Type</span><input name="type" type="text" placeholder="production" required /></label>
                <label><span>Region</span><input name="region" type="text" placeholder="us-central1" /></label>
                <label class="checkbox-row"><input name="production" type="checkbox" /> <span>Production environment</span></label>
                <button class="action" type="submit">Create Environment</button>
              </form>
            ` : emptyState("Read-only view", "Only organization administrators can manage environment definitions.")}
            ${canAdmin && primaryEnvironment
              ? `
                <form id="update-environment-form" class="stack-form compact-top-form" data-environment-id="${primaryEnvironment.id}">
                  <label><span>Name</span><input name="name" type="text" value="${primaryEnvironment.name}" required /></label>
                  <label><span>Slug</span><input name="slug" type="text" value="${primaryEnvironment.slug}" required /></label>
                  <label><span>Type</span><input name="type" type="text" value="${primaryEnvironment.type}" required /></label>
                  <label><span>Region</span><input name="region" type="text" value="${primaryEnvironment.region || ""}" /></label>
                  <label><span>Compliance Zone</span><input name="compliance_zone" type="text" value="${primaryEnvironment.compliance_zone || ""}" placeholder="regulated" /></label>
                  <label class="checkbox-row"><input name="production" type="checkbox" ${primaryEnvironment.production ? "checked" : ""} /> <span>Production environment</span></label>
                  <button class="action ghost" type="submit">Save Current Environment</button>
                </form>
                <button class="action ghost" id="archive-environment-button" data-environment-id="${primaryEnvironment.id}">Archive Current Environment</button>
              `
              : ""}
          </article>
        </section>
      `;
    case "change-review":
      if (state.changeReviewPage.status === "loading" || state.changeReviewPage.status === "idle") {
        return routeStatusLayout(
          "Change Review",
          "Loading route-local change context for the active organization.",
          emptyState("Loading change review", "Fetching the latest ingested changes and their review posture.")
        );
      }
      if (state.changeReviewPage.status === "error") {
        return routeStatusLayout(
          "Change Review",
          "The route-local change-review read failed.",
          emptyState("Change review unavailable", state.changeReviewPage.error || "Refresh and retry the change-review surface.")
        );
      }
      return detailLayout(
        "Change Review",
        latestChange
          ? `${latestChange.summary} touches ${latestChange.file_count} files and ${latestChange.resource_count} resources.`
          : "Ingest change sets to review scope, blast radius, and delivery context.",
        [
          infoCard("Change Types", latestChange ? latestChange.change_types.join(", ") : "No change ingested"),
          infoCard("Review Posture", latestChange ? latestChange.status : "Awaiting ingest"),
          infoCard("Operator Note", "Future versions will fold in repository diff intelligence and artifact provenance.")
        ]
      );
    case "risk":
      if (state.riskPage.status === "loading" || state.riskPage.status === "idle") {
        return routeStatusLayout(
          "Risk Assessment",
          "Loading route-local risk analysis for the active organization.",
          emptyState("Loading risk assessment", "Fetching the latest deterministic risk assessments and rollout guidance.")
        );
      }
      if (state.riskPage.status === "error") {
        return routeStatusLayout(
          "Risk Assessment",
          "The route-local risk read failed.",
          emptyState("Risk data unavailable", state.riskPage.error || "Refresh and retry the risk surface.")
        );
      }
      return detailLayout(
        "Risk Assessment",
        latestRisk
          ? `Latest score ${latestRisk.score} (${latestRisk.level}) with rollout ${latestRisk.recommended_rollout_strategy}.`
          : "Run a change assessment to see explainable risk factors here.",
        [
          infoCard("Approval", latestRisk?.recommended_approval_level || "Pending"),
          infoCard("Guardrails", latestRisk?.recommended_guardrails.join(", ") || "No guardrails generated yet"),
          infoCard("Explainability", latestRisk?.explanation.join(" ") || "Deterministic weighted rules will appear here.")
        ]
      );
    case "rollout":
      if (state.rolloutPage.status === "loading" || state.rolloutPage.status === "idle") {
        return routeStatusLayout(
          "Rollout Plan and Execution",
          "Loading rollout-specific runtime data for the active organization.",
          emptyState("Loading rollout data", "Fetching rollout plans, executions, and runtime detail.")
        );
      }
      if (state.rolloutPage.status === "error") {
        return routeStatusLayout(
          "Rollout Plan and Execution",
          "The route-local rollout read failed.",
          emptyState("Rollout data unavailable", state.rolloutPage.error || "Refresh and retry the rollout surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Rollout Plan and Execution</h3>
              <p>${latestRollout ? `${latestRollout.strategy} rollout with window ${latestRollout.deployment_window}.` : "Generate a rollout plan to see approvals, verification signals, and rollback posture."}</p>
            </div>
            ${latestExecutionAdvisoryOnly
              ? advisoryBanner(
                  "Advisory Mode: observe and recommend only",
                  latestRuntimeSummary?.control_rationale || "This live backend integration is in advisory mode. Reconcile can observe provider state and record recommendations, but it will not execute submit, pause, resume, or rollback against the external deployment target."
                )
              : ""}
            <div class="highlight-grid">
              ${infoCard("Approval Required", latestRollout?.approval_required ? "Yes" : "No")}
              ${infoCard("Verification", latestRollout?.verification_signals.join(", ") || "No signals yet")}
              ${infoCard("Guardrails", latestRollout?.guardrails.join(", ") || "No plan generated")}
              ${infoCard("Latest Execution", latestExecution ? `${latestExecution.status} at step ${latestExecution.current_step || "pending"}` : "No execution started")}
              ${infoCard("Backend", latestRuntimeSummary ? `${latestRuntimeSummary.backend_type} / ${latestRuntimeSummary.backend_status}` : "No backend synced yet")}
              ${infoCard("Progress", latestRuntimeSummary ? `${latestRuntimeSummary.progress_percent}%` : "Awaiting reconcile")}
              ${infoCard("Control Mode", latestRuntimeSummary ? rolloutControlModeLabel(latestRuntimeSummary) : latestBackendIntegration ? integrationModeSummary(latestBackendIntegration) : "No backend integration attached")}
              ${infoCard("Latest Signal", latestRuntimeSummary?.latest_signal_health ? `${latestRuntimeSummary.latest_signal_health}: ${latestRuntimeSummary.latest_signal_summary || "signal snapshot available"}` : "No signal snapshot ingested")}
              ${infoCard("Latest Decision", latestRuntimeSummary?.latest_decision ? `${rolloutDecisionLabel(latestRuntimeSummary.latest_decision)} (${latestRuntimeSummary.latest_decision_mode || "manual"})` : "No verification decision recorded")}
              ${infoCard("Latest Provider Action", latestRuntimeSummary?.last_provider_action ? rolloutProviderActionLabel(latestRuntimeSummary) : "No provider action recorded yet")}
              ${infoCard("Rollback Policy", latestExecutionDetail?.effective_rollback_policy ? `${latestExecutionDetail.effective_rollback_policy.name} (${latestExecutionDetail.effective_rollback_policy.rollback_on_critical_signals ? "auto-rollback on critical" : "manual review on critical"})` : "Built-in fallback policy")}
            </div>
            ${rolloutExecutions.length > 0 ? table(["Execution", "Status", "Decision", "Backend", "Progress", "Step"], rolloutExecutions.map((execution) => [execution.id, execution.status, execution.last_decision ? rolloutDecisionLabel(execution.last_decision) : "n/a", execution.backend_status || execution.backend_type || "n/a", `${execution.progress_percent ?? 0}%`, execution.current_step || "n/a"])) : emptyState("No rollout executions", "Create a rollout execution from the latest plan to begin the control loop.")}
            ${latestExecutionDetail ? `
              <div class="page-grid">
                <article class="surface panel">
                  <div class="panel-header">
                    <h3>Verification Timeline</h3>
                    <p>Recommendations and control-capable decisions are recorded separately so operators can see what was merely suggested versus what could drive provider mutation.</p>
                  </div>
                  ${latestExecutionDetail.verification_results.length > 0 ? table(["Decision", "Effect", "Mode", "Summary"], latestExecutionDetail.verification_results.map((result) => [rolloutDecisionLabel(result.decision), verificationEffectLabel(result), result.automated ? "automated" : result.decision_source || "manual", result.summary])) : emptyState("No verification results", "Runtime verification decisions appear here after reconcile reaches a verification gate.")}
                </article>
                <article class="surface panel">
                  <div class="panel-header">
                    <h3>Signal Snapshots</h3>
                    <p>Normalized runtime health bound to the rollout execution, including source window and signal thresholds.</p>
                  </div>
                  ${latestExecutionDetail.signal_snapshots.length > 0 ? table(["Provider", "Health", "Summary", "Signals"], latestExecutionDetail.signal_snapshots.map((snapshot) => [snapshot.provider_type, snapshot.health, snapshot.summary, snapshot.signals.map((signal) => `${signal.name}:${signal.status}`).join(", ")])) : emptyState("No signal snapshots", "Ingest a simulated snapshot or connect a runtime signal provider.")} 
                </article>
                <article class="surface panel wide">
                  <div class="panel-header">
                    <h3>Execution Status Timeline</h3>
                    <p>Canonical status events for observation, recommendation, suppression, execution, and operator actions.</p>
                  </div>
                  ${latestExecutionDetail.status_timeline.length > 0 ? statusEventFeed(latestExecutionDetail.status_timeline) : emptyState("No status events", "Reconcile and rollout control actions will build a durable status timeline here.")}
                </article>
              </div>
            ` : ""}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Operate Rollout</h3>
              <p>${latestExecutionAdvisoryOnly ? "This execution is attached to an advisory live backend. Reconcile and verification can record recommendations, but manual pause, resume, and rollback are disabled." : "Set desired state, ingest runtime signals, and trigger reconcile safely."}</p>
            </div>
            ${canOperate && latestRollout ? `
              <form id="create-rollout-execution-form" class="stack-form">
                <input type="hidden" name="rollout_plan_id" value="${latestRollout.id}" />
                <label><span>Backend</span>
                  <select name="backend_type">
                    <option value="simulated">simulated</option>
                    <option value="kubernetes">kubernetes</option>
                  </select>
                </label>
                <label><span>Signal Provider</span>
                  <select name="signal_provider_type">
                    <option value="simulated">simulated</option>
                    <option value="prometheus">prometheus</option>
                  </select>
                </label>
                <button class="action" type="submit">Create Execution From Latest Plan</button>
              </form>
              ${latestExecution ? `
                <form id="advance-rollout-form" class="stack-form">
                  <input type="hidden" name="execution_id" value="${latestExecution.id}" />
                  <label><span>Action</span>
                    <select name="action">
                      <option value="approve">Approve</option>
                      <option value="start">Start</option>
                      <option value="pause" ${latestExecutionAdvisoryOnly ? "disabled" : ""}>Pause</option>
                      <option value="resume" ${latestExecutionAdvisoryOnly ? "disabled" : ""}>Resume</option>
                      <option value="complete">Complete</option>
                      <option value="rollback" ${latestExecutionAdvisoryOnly ? "disabled" : ""}>Rollback</option>
                    </select>
                  </label>
                  <label><span>Reason</span><input name="reason" type="text" placeholder="Operator note" /></label>
                  ${latestExecutionAdvisoryOnly ? `<p class="panel-muted">Pause, resume, and rollback are disabled here because the live backend is in advisory mode.</p>` : ""}
                  <button class="action ghost" type="submit">Submit Rollout Action</button>
                </form>
                <form id="reconcile-rollout-form" class="stack-form">
                  <input type="hidden" name="execution_id" value="${latestExecution.id}" />
                  <button class="action ghost" type="submit">${latestExecutionAdvisoryOnly ? "Reconcile and Record Recommendation" : "Reconcile Execution"}</button>
                </form>
                <form id="create-signal-snapshot-form" class="stack-form">
                  <input type="hidden" name="execution_id" value="${latestExecution.id}" />
                  <label><span>Health</span>
                    <select name="health">
                      <option value="healthy">healthy</option>
                      <option value="warning">warning</option>
                      <option value="critical">critical</option>
                    </select>
                  </label>
                  <label><span>Summary</span><textarea name="summary" placeholder="Describe the latest runtime posture."></textarea></label>
                  <label><span>Latency P95 (ms)</span><input name="latency_value" type="number" min="0" step="1" value="145" /></label>
                  <label><span>Error Rate (%)</span><input name="error_rate_value" type="number" min="0" step="0.1" value="0.2" /></label>
                  <label><span>Business KPI</span><input name="business_value" type="number" min="0" step="0.1" value="0" /></label>
                  <button class="action ghost" type="submit">Ingest Signal Snapshot</button>
                </form>
                <form id="record-verification-form" class="stack-form">
                  <input type="hidden" name="execution_id" value="${latestExecution.id}" />
                  <label><span>Outcome</span>
                    <select name="outcome">
                      <option value="pass">Pass</option>
                      <option value="fail">Fail</option>
                      <option value="inconclusive">Inconclusive</option>
                    </select>
                  </label>
                  <label><span>Decision</span>
                    <select name="decision">
                      <option value="continue">Continue</option>
                      <option value="verified">Verified</option>
                      <option value="pause">Pause</option>
                      <option value="rollback">Rollback</option>
                      <option value="failed">Failed</option>
                      <option value="manual_review_required">Manual Review Required</option>
                    </select>
                  </label>
                  <label><span>Summary</span><textarea name="summary" placeholder="Signals look healthy or explain why the rollout should pause."></textarea></label>
                  ${latestExecutionAdvisoryOnly ? `<p class="panel-muted">Verification results will be recorded as advisory recommendations only for this live backend.</p>` : ""}
                  <button class="action ghost" type="submit">${latestExecutionAdvisoryOnly ? "Record Advisory Recommendation" : "Record Verification"}</button>
                </form>
              ` : ""}
            ` : emptyState("Execution access unavailable", "An org member or org admin can create executions and control rollout state from this surface.")}
          </article>
        </section>
      `;
    case "deployments":
      if (state.deploymentsPage.status === "loading" || state.deploymentsPage.status === "idle") {
        return routeStatusLayout(
          "Operational Status Dashboard",
          "Loading server-backed deployment history and filters.",
          emptyState("Loading deployment history", "Fetching status search results, rollback policies, and route-local filter context.")
        );
      }
      if (state.deploymentsPage.status === "error") {
        return routeStatusLayout(
          "Operational Status Dashboard",
          "The route-local deployment read failed.",
          emptyState("Deployment history unavailable", state.deploymentsPage.error || "Refresh and retry the deployment dashboard.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Operational Status Dashboard</h3>
              <p>Server-backed search, pagination, and operational scoping across rollout executions, services, environments, and rollback activity.</p>
            </div>
            ${coverageSummary.stale_integrations > 0
              ? advisoryBanner(
                  "Stale integration warning",
                  `${coverageSummary.stale_integrations} integration${coverageSummary.stale_integrations === 1 ? "" : "s"} currently look stale. Status history may be current for control-plane events while connected provider state lags behind.`
                )
              : ""}
            <div class="highlight-grid">
              ${infoCard("Returned Events", `${statusDashboard.summary.returned} event(s) in the current server-backed window.`)}
              ${infoCard("Search Scope", statusWindowLabel(state))}
              ${infoCard("Rollback Events", `${statusDashboard.summary.rollback_events} rollback-related event(s) in this result set.`)}
              ${infoCard("Automated Events", `${statusDashboard.summary.automated_events} automated event(s) in this result set.`)}
            </div>
            <form id="status-search-form" class="stack-form">
              <div class="form-grid">
                <label>
                  <span>Search Status History</span>
                  <input id="status-search-input" name="search" type="text" placeholder="rollback, signal, rollout, checkout" value="${statusFilterValue(state, "search")}" />
                </label>
                <label>
                  <span>Event Type</span>
                  <select name="event_type" data-status-auto-submit="true">
                    ${statusEventTypeOptions(state, statusFilterValue(state, "event_type"))}
                  </select>
                </label>
                <label>
                  <span>Service</span>
                  <select name="service_id" data-status-auto-submit="true">
                    <option value="">All services</option>
                    ${catalog.services
                      .map((service) => `<option value="${service.id}" ${service.id === statusFilterValue(state, "service_id") ? "selected" : ""}>${service.name}</option>`)
                      .join("")}
                  </select>
                </label>
                <label>
                  <span>Environment</span>
                  <select name="environment_id" data-status-auto-submit="true">
                    <option value="">All environments</option>
                    ${catalog.environments
                      .map((environment) => `<option value="${environment.id}" ${environment.id === statusFilterValue(state, "environment_id") ? "selected" : ""}>${environment.name}</option>`)
                      .join("")}
                  </select>
                </label>
                <label>
                  <span>Source</span>
                  <select name="source" data-status-auto-submit="true">
                    ${statusSourceOptions(state, statusFilterValue(state, "source"))}
                  </select>
                </label>
                <label>
                  <span>Automation</span>
                  <select name="automated" data-status-auto-submit="true">
                    <option value="" ${statusFilterValue(state, "automated") === "" ? "selected" : ""}>All events</option>
                    <option value="true" ${statusFilterValue(state, "automated") === "true" ? "selected" : ""}>Automated only</option>
                    <option value="false" ${statusFilterValue(state, "automated") === "false" ? "selected" : ""}>Manual only</option>
                  </select>
                </label>
                <label>
                  <span>Page Size</span>
                  <select name="limit" data-status-auto-submit="true">
                    ${[25, 50, 100]
                      .map((limit) => `<option value="${limit}" ${statusDashboard.summary.limit === limit ? "selected" : ""}>${limit}</option>`)
                      .join("")}
                  </select>
                </label>
                <label class="checkbox-row">
                  <input id="status-rollback-only" name="rollback_only" type="checkbox" ${statusFilterValue(state, "rollback_only") === "true" ? "checked" : ""} data-status-auto-submit="true" />
                  <span>Show rollback-related events only</span>
                </label>
              </div>
              <div class="split-actions">
                <button class="action ghost" type="submit">Search History</button>
                <button class="action ghost" id="status-search-reset" type="button">Reset Filters</button>
              </div>
            </form>
            ${statusDashboard.events.length > 0 ? statusEventFeed(statusDashboard.events) : emptyState("No status events", "No status events matched the current server-backed query. Adjust the filters or wait for the next reconcile, sync, or rollout action.")}
            <div class="pagination-bar">
              <p class="panel-muted">${statusWindowLabel(state)}</p>
              <div class="split-actions">
                <button class="action ghost" type="button" data-status-offset="${Math.max(0, statusDashboard.summary.offset - statusDashboard.summary.limit)}" ${statusDashboard.summary.offset <= 0 ? "disabled" : ""}>Previous Page</button>
                <button class="action ghost" type="button" data-status-offset="${statusDashboard.summary.offset + statusDashboard.summary.limit}" ${(statusDashboard.summary.offset + statusDashboard.summary.returned) >= statusDashboard.summary.total ? "disabled" : ""}>Next Page</button>
              </div>
            </div>
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Rollback Policies</h3>
              <p>The active guardrails that drive automated rollback and manual review.</p>
            </div>
            ${rollbackPolicies.length > 0 ? table(["Name", "Scope", "Error Rate", "Latency", "Critical"], rollbackPolicies.map((policy) => [
              policy.name,
              [policy.project_id ? "project" : "", policy.service_id ? "service" : "", policy.environment_id ? "environment" : ""].filter(Boolean).join(" / ") || "organization",
              policy.max_error_rate ? String(policy.max_error_rate) : "n/a",
              policy.max_latency_ms ? `${policy.max_latency_ms}ms` : "n/a",
              policy.rollback_on_critical_signals ? "Yes" : "No"
            ])) : emptyState("No rollback policies", "Create a rollback policy through the API or CLI to override the built-in fallback guardrails.")}
          </article>
        </section>
      `;
    case "incidents":
      if (state.incidentsPage.status === "loading" || state.incidentsPage.status === "idle") {
        return routeStatusLayout(
          "Incident Feed",
          "Loading route-local incident data for the active organization.",
          emptyState("Loading incidents", "Fetching incident summaries correlated from rollout and reliability state.")
        );
      }
      if (state.incidentsPage.status === "error") {
        return routeStatusLayout(
          "Incident Feed",
          "The route-local incident read failed.",
          emptyState("Incident feed unavailable", state.incidentsPage.error || "Refresh and retry the incident feed.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Incident Feed</h3>
              <p>Reliability signals tied back to change history and ownership.</p>
            </div>
            ${
              incidents.length === 0
                ? emptyState("No incidents yet", "As rollout issues, rollback signals, or failed executions occur, they will show up here with linked change context.")
                : table(
                    ["Title", "Severity", "Status", "Change", "Impacted Scope"],
                    incidents.map((incident) => [
                      `<a class="inline-link" data-incident-link="${incident.id}" href="#/incident-detail?id=${encodeURIComponent(incident.id)}">${incident.title}</a>`,
                      incident.severity,
                      incident.status,
                      incident.related_change || "n/a",
                      (incident.impacted_paths || []).slice(0, 3).join(", ") || "n/a"
                    ])
                  )
            }
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Change Correlation</h3>
              <p>Operational signals mapped back to rollout history.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Latest Incident", latestIncident ? latestIncident.title : "No linked incident")}
              ${infoCard("Related Change", latestIncident?.related_change || "No linked change")}
              ${infoCard("Impacted Scope", (latestIncident?.impacted_paths || []).join(", ") || "Awaiting incident data")}
              ${infoCard("Rollback Signals", rollbackEvents.length > 0 ? `${rollbackEvents.length} rollback-related status events recorded.` : "No rollback events recorded")}
            </div>
          </article>
        </section>
      `;
    case "incident-detail":
      if (state.incidentDetailPage.status === "loading") {
        return routeStatusLayout(
          "Incident Detail",
          "Loading route-local incident detail.",
          emptyState("Loading incident detail", "Fetching rollout-correlated incident detail from the dedicated route.")
        );
      }
      if (state.incidentDetailPage.status === "error") {
        return `
          <section class="page-grid">
            <article class="surface panel wide">
              <div class="panel-header">
                <h3>Incident Detail</h3>
                <p>The dedicated incident read path failed.</p>
              </div>
              ${emptyState("Incident detail unavailable", state.incidentDetailPage.error || "The incident detail route returned an error. Refresh and retry, or inspect the API for the underlying failure.")}
              <p><a class="inline-link" href="#/incidents">Return to incident feed</a></p>
            </article>
          </section>
        `;
      }
      if (!selectedIncidentID) {
        return `
          <section class="page-grid">
            <article class="surface panel wide">
              <div class="panel-header">
                <h3>Incident Detail</h3>
                <p>Select an incident from the feed to inspect the rollout-specific timeline behind it.</p>
              </div>
              ${emptyState("No incident selected", "Open the incident feed and choose a specific incident to load its dedicated detail route.")}
              <p><a class="inline-link" href="#/incidents">Go to incident feed</a></p>
            </article>
          </section>
        `;
      }
      if (incidentDetailStatus === "not_found") {
        return `
          <section class="page-grid">
            <article class="surface panel wide">
              <div class="panel-header">
                <h3>Incident Detail</h3>
                <p>The requested incident is not available in the active organization.</p>
              </div>
              ${emptyState("Incident not found", "This incident either no longer maps to an active rollout-derived incident or it belongs to a different organization scope.")}
              <p><a class="inline-link" href="#/incidents">Return to incident feed</a></p>
            </article>
          </section>
        `;
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Incident Detail</h3>
              <p>${selectedIncident ? `${selectedIncident.title} is ${selectedIncident.status} at severity ${selectedIncident.severity}.` : "Loading incident detail from the dedicated incident route."}</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Linked Change", selectedIncident?.related_change || "No linked change")}
              ${infoCard("Primary Service", selectedIncident?.service_id ? resourceLabel(state, "service", selectedIncident.service_id) : "No impacted service")}
              ${infoCard("Environment", selectedIncident?.environment_id ? resourceLabel(state, "environment", selectedIncident.environment_id) : "No impacted environment")}
              ${infoCard("Rollout Execution", incidentDetail?.rollout_execution_id || "No linked rollout execution")}
              ${infoCard("Impacted Scope", (selectedIncident?.impacted_paths || []).join(", ") || "Awaiting incident correlation")}
            </div>
          </article>
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Operational Timeline</h3>
              <p>Rollout-scoped status events correlated to this incident.</p>
            </div>
            ${selectedIncidentTimeline.length > 0 ? statusEventFeed(selectedIncidentTimeline.slice(0, 12)) : emptyState("No incident timeline yet", "This incident currently has no correlated rollout status events beyond the derived incident summary.")}
          </article>
        </section>
      `;
    case "policies":
      if (state.policiesPage.status === "loading" || state.policiesPage.status === "idle") {
        return routeStatusLayout(
          "Policy Center",
          "Loading route-local governance policies for the active organization.",
          emptyState("Loading policies", "Fetching policy definitions and control modes.")
        );
      }
      if (state.policiesPage.status === "error") {
        return routeStatusLayout(
          "Policy Center",
          "The route-local policy read failed.",
          emptyState("Policy data unavailable", state.policiesPage.error || "Refresh and retry the policy surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Policy Center</h3>
              <p>Deterministic governance policies with persisted scope, explainable outcomes, and operator-visible recent decisions.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Active Policies", `${policies.filter((policy) => policy.enabled).length} enabled / ${policies.length} total`)}
              ${infoCard("Recent Decisions", `${policyDecisions.length} persisted policy decision(s) in the latest route-local window.`)}
              ${infoCard("Governed Surfaces", `${new Set(policies.map((policy) => policy.applies_to)).size || 0} workflow surface(s)`)}
              ${infoCard("Write Access", canAdmin ? "Org admins can author, update, and enable or disable policies here." : "Read-only visibility for non-admin members." )}
            </div>
            ${policies.length > 0 ? table(
              ["Policy", "Scope", "Workflow", "Mode", "Priority", "Enabled", "Conditions"],
              policies.map((policy) => [
                `${policy.name}<br /><span class="panel-muted">${policy.code}</span>`,
                policyScopeSummary(state, policy),
                policy.applies_to,
                policy.mode,
                String(policy.priority ?? 0),
                policy.enabled ? "Yes" : "No",
                policyConditionSummary(policy)
              ])
            ) : emptyState("No policies yet", "Create a policy definition to govern risk assessment and rollout planning decisions.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Create Policy</h3>
              <p>Persist a deterministic policy definition with scope, workflow, mode, and explicit match conditions.</p>
            </div>
            ${canAdmin ? `
              <form id="create-policy-form" class="stack-form">
                <label><span>Name</span><input name="name" type="text" placeholder="Production Schema Freeze" required /></label>
                <label><span>Code</span><input name="code" type="text" placeholder="prod-schema-freeze" /></label>
                <label><span>Workflow</span>
                  <select name="applies_to">
                    <option value="risk_assessment">risk_assessment</option>
                    <option value="rollout_plan">rollout_plan</option>
                  </select>
                </label>
                <label><span>Mode</span>
                  <select name="mode">
                    <option value="advisory">advisory</option>
                    <option value="require_manual_review">require_manual_review</option>
                    <option value="block">block</option>
                  </select>
                </label>
                <label><span>Project Scope</span><select name="project_id">${policyProjectOptions(state)}</select></label>
                <label><span>Service Scope</span><select name="service_id">${policyServiceOptions(state)}</select></label>
                <label><span>Environment Scope</span><select name="environment_id">${policyEnvironmentOptions(state)}</select></label>
                <label><span>Priority</span><input name="priority" type="number" value="50" /></label>
                <label><span>Description</span><textarea name="description" placeholder="Explain what this policy protects and why."></textarea></label>
                <label><span>Minimum Risk</span><input name="min_risk_level" type="text" placeholder="high" /></label>
                <label><span>Required Change Types</span><input name="required_change_types" type="text" placeholder="schema,iam" /></label>
                <label><span>Required Touches</span><input name="required_touches" type="text" placeholder="schema,infrastructure" /></label>
                <label><span>Missing Capabilities</span><input name="missing_capabilities" type="text" placeholder="observability,slo" /></label>
                <label class="checkbox-row"><input name="production_only" type="checkbox" /> <span>Production only</span></label>
                <label class="checkbox-row"><input name="regulated_only" type="checkbox" /> <span>Regulated only</span></label>
                <label class="checkbox-row"><input name="enabled" type="checkbox" checked /> <span>Enabled</span></label>
                <button class="action" type="submit">Create Policy</button>
              </form>
            ` : emptyState("Read-only workspace", "Only organization administrators can author or update governance policies.")}
          </article>
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Manage Policies</h3>
              <p>Enable, disable, and refine persisted policy definitions without leaving the route-local governance surface.</p>
            </div>
            ${policies.length > 0
              ? policies.map((policy) => renderPolicyManagementCard(state, policy, canAdmin)).join("")
              : emptyState("No policies to manage", "Create a policy to unlock inline editing and enable or disable controls.")}
          </article>
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Recent Policy Decisions</h3>
              <p>Persisted evaluation results from risk assessment and rollout planning, including the scope and explanation that matched.</p>
            </div>
            ${policyDecisions.length > 0 ? table(
              ["When", "Policy", "Workflow", "Outcome", "Target", "Summary"],
              policyDecisions.map((decision) => [
                formatTimestamp(decision.created_at),
                `${decision.policy_name}<br /><span class="panel-muted">${decision.policy_code}</span>`,
                decision.applies_to,
                decision.outcome,
                policyDecisionTargetLabel(state, decision),
                `${decision.summary}${decision.reasons.length > 0 ? `<br /><span class="panel-muted">${decision.reasons.join("; ")}</span>` : ""}`
              ])
            ) : emptyState("No policy decisions yet", "Evaluate risk or create a rollout plan to persist explainable governance outcomes here.")}
          </article>
        </section>
      `;
    case "audit":
      if (state.auditPage.status === "loading" || state.auditPage.status === "idle") {
        return routeStatusLayout(
          "Audit Trail",
          "Loading route-local audit events for the active organization.",
          emptyState("Loading audit trail", "Fetching recorded control-plane actions and outcomes.")
        );
      }
      if (state.auditPage.status === "error") {
        return routeStatusLayout(
          "Audit Trail",
          "The route-local audit read failed.",
          emptyState("Audit data unavailable", state.auditPage.error || "Refresh and retry the audit surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Audit Trail</h3>
              <p>Critical control-plane actions recorded with tenant and resource context.</p>
            </div>
            ${table(
              ["Action", "Resource", "ID", "Outcome", "Details"],
              auditEvents.map((event) => [
                event.action,
                event.resource_type,
                event.resource_id,
                event.outcome,
                (event.details || []).join(", ")
              ])
            )}
          </article>
        </section>
      `;
    case "integrations":
      if (state.integrationsPage.status === "loading" || state.integrationsPage.status === "idle") {
        return routeStatusLayout(
          "Business Environment Onboarding",
          "Loading integration health, discovery, and mapping context.",
          emptyState("Loading integration surfaces", "Fetching integrations, sync history, webhook health, and discovery mappings.")
        );
      }
      if (state.integrationsPage.status === "error") {
        return routeStatusLayout(
          "Business Environment Onboarding",
          "The route-local integrations read failed.",
          emptyState("Integration data unavailable", state.integrationsPage.error || "Refresh and retry the integrations surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Business Environment Onboarding</h3>
              <p>Connect source control, deployment/runtime, and metrics in advisory mode first. Move to active control only after mappings, sync health, and recommendation history look trustworthy.</p>
            </div>
            ${advisoryBanner(
              "Pilot guidance",
              "Advisory mode is the safe default. In this mode the control plane can discover resources, ingest change and runtime evidence, and recommend rollback or pause, but it will not issue external deployment mutations."
            )}
            <div class="highlight-grid">
              ${infoCard("Connected", `${coverageSummary.enabled_integrations} of ${integrations.length} integrations enabled`)}
              ${infoCard("Stale Integrations", `${coverageSummary.stale_integrations} integration${coverageSummary.stale_integrations === 1 ? "" : "s"} currently stale`)}
              ${infoCard("Unmapped Discovery", `${coverageSummary.unmapped_repositories + coverageSummary.unmapped_discovered_resources} repository or runtime resource mapping gap(s)`)}
              ${infoCard("Coverage Surface", `${coverageSummary.workload_coverage_environments} environment(s) with workload coverage and ${coverageSummary.signal_coverage_services} service(s) with signal coverage`)}
            </div>
            ${canAdmin ? `
              <form id="create-integration-form" class="stack-form compact-top-form">
                <div class="form-grid">
                  <label>
                    <span>Kind</span>
                    <select name="kind">
                      <option value="github">GitHub</option>
                      <option value="gitlab">GitLab</option>
                      <option value="kubernetes">Kubernetes</option>
                      <option value="prometheus">Prometheus</option>
                    </select>
                  </label>
                  <label><span>Instance Name</span><input name="name" type="text" placeholder="SCM or Runtime Instance" required /></label>
                  <label><span>Instance Key</span><input name="instance_key" type="text" placeholder="gitlab-prod" /></label>
                  <label>
                    <span>Scope Type</span>
                    <select name="scope_type">
                      <option value="organization">organization</option>
                      <option value="environment">environment</option>
                      <option value="service">service</option>
                      <option value="repository_group">repository_group</option>
                    </select>
                  </label>
                  <label><span>Scope Name</span><input name="scope_name" type="text" placeholder="Production" /></label>
                  <label>
                    <span>Auth Strategy</span>
                    <select name="auth_strategy">
                      <option value="">auto</option>
                      <option value="github_app">github_app</option>
                      <option value="personal_access_token">personal_access_token</option>
                    </select>
                  </label>
                </div>
                <button class="action" type="submit">Add Integration Instance</button>
              </form>
            ` : ""}
          </article>
          ${integrations
            .map((integration) => {
              const metadata = integration.metadata || {};
              const runs = integrationSyncRuns[integration.id] || [];
              const integrationRepositories = repositories.filter((repository) => repository.source_integration_id === integration.id);
              const integrationDiscoveredResources = discoveredResources.filter((resource) => resource.integration_id === integration.id);
              const webhookRegistration = webhookRegistrations[integration.id] || null;
              const services = catalog.services;
              const environments = catalog.environments;
              return `
                <article class="surface panel wide integration-panel">
                  <div class="panel-header">
                    <h3>${integration.name}</h3>
                    <p>${integration.description}</p>
                  </div>
                  <div class="highlight-grid">
                    ${infoCard("Instance", integrationInstanceSummary(integration))}
                    ${infoCard("Scope", integration.scope_name || integration.scope_type || "organization")}
                    ${infoCard("Auth", integrationAuthStrategySummary(integration))}
                    ${infoCard("Mode", integrationModeSummary(integration))}
                    ${infoCard("Control Surface", integrationControlSummary(integration))}
                    ${infoCard("Freshness", integrationFreshnessSummary(integration))}
                    ${infoCard("Connection Health", integration.connection_health || "unconfigured")}
                    ${infoCard("Onboarding", integration.onboarding_status || "not_started")}
                    ${infoCard("Next Scheduled Sync", formatTimestamp(integration.next_scheduled_sync_at))}
                    ${infoCard("Last Successful Sync", formatTimestamp(integration.last_sync_succeeded_at || integration.last_synced_at))}
                    ${infoCard("Last Failure", formatTimestamp(integration.last_sync_failed_at))}
                    ${infoCard("Coverage", integrationCoverageSummaryLabel(integration, integrationRepositories, integrationDiscoveredResources))}
                    ${isSCMProviderKind(integration.kind) ? infoCard("Webhook", webhookRegistrationSummary(webhookRegistration)) : ""}
                  </div>
                  ${integration.last_error ? `<p class="panel-muted">Latest error: ${integration.last_error}</p>` : ""}
                  <p class="panel-muted">Capabilities: ${integration.capabilities.join(", ")}</p>
                  <p class="panel-muted">${integration.mode === "active_control" && integration.control_enabled ? "This integration can execute provider actions when rollout logic permits it." : "This integration is currently read-only and recommendation-focused. External provider mutations remain disabled."} ${integration.schedule_enabled ? `Scheduled every ${secondsLabel(integration.schedule_interval_seconds)} with stale detection after ${secondsLabel(integration.sync_stale_after_seconds)}.` : "Scheduling is currently disabled, so freshness will only advance when an operator runs a sync or a webhook arrives."}</p>
                  ${canAdmin
                    ? `
                      <form class="stack-form integration-config-form" data-integration-id="${integration.id}" data-integration-kind="${integration.kind}">
                        <div class="form-grid">
                          <label><span>Instance Name</span><input name="name" type="text" value="${integration.name}" /></label>
                          <label><span>Instance Key</span><input name="instance_key" type="text" value="${integration.instance_key || ""}" /></label>
                          <label>
                            <span>Scope Type</span>
                            <select name="scope_type">
                              <option value="organization" ${integration.scope_type === "organization" || !integration.scope_type ? "selected" : ""}>organization</option>
                              <option value="environment" ${integration.scope_type === "environment" ? "selected" : ""}>environment</option>
                              <option value="service" ${integration.scope_type === "service" ? "selected" : ""}>service</option>
                              <option value="repository_group" ${integration.scope_type === "repository_group" ? "selected" : ""}>repository_group</option>
                            </select>
                          </label>
                          <label><span>Scope Name</span><input name="scope_name" type="text" value="${integration.scope_name || ""}" placeholder="Production" /></label>
                        </div>
                        <label>
                          <span>Mode</span>
                          <select name="mode">
                            <option value="advisory" ${integration.mode === "advisory" ? "selected" : ""}>advisory</option>
                            <option value="active_control" ${integration.mode === "active_control" ? "selected" : ""}>active_control</option>
                          </select>
                        </label>
                        <label class="checkbox-field">
                          <input name="enabled" type="checkbox" ${integration.enabled ? "checked" : ""} />
                          <span>Enable integration for this organization</span>
                        </label>
                        <label class="checkbox-field">
                          <input name="control_enabled" type="checkbox" ${integration.control_enabled ? "checked" : ""} />
                          <span>Allow active deployment control</span>
                        </label>
                        <label class="checkbox-field">
                          <input name="schedule_enabled" type="checkbox" ${integration.schedule_enabled ? "checked" : ""} />
                          <span>Run continuous scheduled sync</span>
                        </label>
                        <div class="form-grid">
                          <label><span>Schedule Interval (seconds)</span><input name="schedule_interval_seconds" type="number" min="60" step="30" value="${integration.schedule_interval_seconds || ""}" placeholder="300" /></label>
                          <label><span>Stale After (seconds)</span><input name="sync_stale_after_seconds" type="number" min="120" step="30" value="${integration.sync_stale_after_seconds || ""}" placeholder="600" /></label>
                        </div>
                        ${integration.mode !== "active_control" || !integration.control_enabled ? `<p class="panel-muted">Advisory mode is safer for pilots. The control plane will observe, sync, and recommend without mutating the connected runtime.</p>` : `<p class="panel-muted">Active control is enabled. External deployment mutations can be issued when rollout logic or operators request them.</p>`}
                        ${integration.kind === "github" ? `
                          <label>
                            <span>Auth Strategy</span>
                            <select name="auth_strategy">
                              <option value="personal_access_token" ${integration.auth_strategy === "personal_access_token" || !integration.auth_strategy ? "selected" : ""}>personal_access_token</option>
                              <option value="github_app" ${integration.auth_strategy === "github_app" ? "selected" : ""}>github_app</option>
                            </select>
                          </label>
                          <label><span>GitHub API Base URL</span><input name="api_base_url" type="text" value="${String(metadata.api_base_url || "https://api.github.com")}" /></label>
                          <label><span>GitHub Web Base URL</span><input name="web_base_url" type="text" value="${String(metadata.web_base_url || "https://github.com")}" /></label>
                          <label><span>GitHub Owner or Org</span><input name="owner" type="text" value="${String(metadata.owner || "")}" placeholder="acme" /></label>
                          <label><span>Access Token Env</span><input name="access_token_env" type="text" value="${String(metadata.access_token_env || "")}" placeholder="CCP_GITHUB_TOKEN" /></label>
                          <label><span>GitHub App ID</span><input name="app_id" type="text" value="${String(metadata.app_id || "")}" placeholder="123456" /></label>
                          <label><span>GitHub App Slug</span><input name="app_slug" type="text" value="${String(metadata.app_slug || "")}" placeholder="change-control-plane" /></label>
                          <label><span>GitHub App Private Key Env</span><input name="private_key_env" type="text" value="${String(metadata.private_key_env || "")}" placeholder="CCP_GITHUB_APP_PRIVATE_KEY" /></label>
                          <label><span>Installation ID</span><input name="installation_id" type="text" value="${String(metadata.installation_id || "")}" placeholder="987654321" /></label>
                          <label><span>Webhook Secret Env</span><input name="webhook_secret_env" type="text" value="${String(metadata.webhook_secret_env || "")}" placeholder="CCP_GITHUB_WEBHOOK_SECRET" /></label>
                        ` : ""}
                        ${integration.kind === "gitlab" ? `
                          <label>
                            <span>Auth Strategy</span>
                            <select name="auth_strategy">
                              <option value="personal_access_token" ${integration.auth_strategy === "personal_access_token" || !integration.auth_strategy ? "selected" : ""}>personal_access_token</option>
                            </select>
                          </label>
                          <label><span>GitLab API Base URL</span><input name="api_base_url" type="text" value="${String(metadata.api_base_url || "https://gitlab.com/api/v4")}" /></label>
                          <label><span>GitLab Group or Namespace</span><input name="group" type="text" value="${String(metadata.group || metadata.namespace || "")}" placeholder="acme/platform" /></label>
                          <label><span>Access Token Env</span><input name="access_token_env" type="text" value="${String(metadata.access_token_env || "")}" placeholder="CCP_GITLAB_TOKEN" /></label>
                          <label><span>Webhook Secret Env</span><input name="webhook_secret_env" type="text" value="${String(metadata.webhook_secret_env || "")}" placeholder="CCP_GITLAB_WEBHOOK_SECRET" /></label>
                        ` : ""}
                        ${integration.kind === "kubernetes" ? `
                          <label><span>Kubernetes API Base URL</span><input name="api_base_url" type="text" value="${String(metadata.api_base_url || "")}" placeholder="https://cluster.example.com" /></label>
                          <label><span>Namespace</span><input name="namespace" type="text" value="${String(metadata.namespace || "")}" placeholder="prod" /></label>
                          <label><span>Deployment Name</span><input name="deployment_name" type="text" value="${String(metadata.deployment_name || "")}" placeholder="checkout" /></label>
                          <label><span>Inventory Path</span><input name="inventory_path" type="text" value="${String(metadata.inventory_path || "")}" placeholder="/apis/apps/v1/namespaces/prod/deployments" /></label>
                          <label><span>Status Path</span><input name="status_path" type="text" value="${String(metadata.status_path || "")}" placeholder="/apis/apps/v1/namespaces/prod/deployments/checkout" /></label>
                          <label><span>Bearer Token Env</span><input name="bearer_token_env" type="text" value="${String(metadata.bearer_token_env || "")}" placeholder="CCP_KUBE_TOKEN" /></label>
                        ` : ""}
                        ${integration.kind === "prometheus" ? `
                          <label><span>Prometheus API Base URL</span><input name="api_base_url" type="text" value="${String(metadata.api_base_url || "")}" placeholder="https://prometheus.example.com" /></label>
                          <label><span>Query Path</span><input name="query_path" type="text" value="${String(metadata.query_path || "/api/v1/query_range")}" placeholder="/api/v1/query_range" /></label>
                          <label><span>Window Seconds</span><input name="window_seconds" type="text" value="${String(metadata.window_seconds || "300")}" placeholder="300" /></label>
                          <label><span>Step Seconds</span><input name="step_seconds" type="text" value="${String(metadata.step_seconds || "60")}" placeholder="60" /></label>
                          <label><span>Bearer Token Env</span><input name="bearer_token_env" type="text" value="${String(metadata.bearer_token_env || "")}" placeholder="CCP_PROM_TOKEN" /></label>
                          <label><span>Queries JSON</span><textarea name="queries" placeholder='[{"name":"error_rate","query":"sum(rate(http_requests_total{status=~\"5..\"}[5m]))"}]'>${metadata.queries ? JSON.stringify(metadata.queries, null, 2) : ""}</textarea></label>
                        ` : ""}
                        <div class="split-actions">
                          <button class="action" type="submit">Save Integration</button>
                          ${integration.kind === "github" && integration.auth_strategy === "github_app" ? `<button class="action ghost github-onboarding-button" type="button" data-integration-id="${integration.id}">Start GitHub App Install</button>` : ""}
                          <button class="action ghost integration-test-button" type="button" data-integration-id="${integration.id}">${integration.mode === "active_control" && integration.control_enabled ? "Test Connection" : "Test Read-Only Connection"}</button>
                          <button class="action ghost integration-sync-button" type="button" data-integration-id="${integration.id}">${integration.mode === "active_control" && integration.control_enabled ? "Run Sync" : "Run Read-Only Sync"}</button>
                        </div>
                      </form>
                    `
                    : emptyState("Read-only workspace", "Only organization administrators can configure integrations, mappings, and advisory mode settings.")}
                  ${isSCMProviderKind(integration.kind) ? renderWebhookRegistrationPanel(webhookRegistration, integration, canAdmin) : ""}
                  ${integrationDiscoveredResources.length > 0 ? `
                    <div class="panel-header">
                      <h3>${integration.kind === "kubernetes" ? "Discovered Workloads" : integration.kind === "prometheus" ? "Discovered Signal Targets" : "Discovered Runtime Resources"}</h3>
                      <p>Resources continuously seen from this integration, including what is still unmapped, partially mapped, or stale.</p>
                    </div>
                    <div class="repository-grid">
                      ${integrationDiscoveredResources
                        .map((resource) => renderDiscoveredResourceCard(state, resource, repositories, canAdmin))
                        .join("")}
                    </div>
                  ` : emptyState("No runtime resources discovered yet", "Run a sync or wait for the scheduled collector to populate first-class workloads or signal targets for this integration.")}
                  ${isSCMProviderKind(integration.kind) ? `
                    <div class="panel-header">
                      <h3>Repository Discovery and Mapping</h3>
                      <p>Map discovered source-control repositories or projects to services and environments before provider webhooks can create linked change sets. Coverage gaps here will also show up in the pilot coverage summary.</p>
                    </div>
                    ${integrationRepositories.length > 0
                      ? `
                        <div class="repository-grid">
                          ${integrationRepositories
                            .map((repository) => `
                              <form class="surface repository-card repository-map-form" data-repository-id="${repository.id}">
                                <div class="panel-header">
                                  <h3>${repository.name}</h3>
                                  <p>${repository.url}</p>
                                </div>
                                <div class="mini-list">
                                  <span><strong>Status:</strong> ${repository.status}${repository.last_synced_at ? `, last synced ${repository.last_synced_at}` : ""}</span>
                                  <span><strong>Ownership:</strong> ${ownershipSummary(state, repository.metadata)}</span>
                                  <span><strong>Mapping Provenance:</strong> ${mappingProvenanceSummary(repository.metadata)}</span>
                                </div>
                                <label>
                                  <span>Service</span>
                                  <select name="service_id">
                                    <option value="">Not mapped</option>
                                    ${services.map((service) => `<option value="${service.id}" ${service.id === repository.service_id ? "selected" : ""}>${service.name}</option>`).join("")}
                                  </select>
                                </label>
                                <label>
                                  <span>Environment</span>
                                  <select name="environment_id">
                                    <option value="">Not mapped</option>
                                    ${environments.map((environment) => `<option value="${environment.id}" ${environment.id === repository.environment_id ? "selected" : ""}>${environment.name}</option>`).join("")}
                                  </select>
                                </label>
                                <button class="action ghost" type="submit">Save Mapping</button>
                              </form>
                            `)
                            .join("")}
                        </div>
                      `
                      : emptyState("No repositories discovered yet", `Save the ${integration.kind === "gitlab" ? "GitLab" : "GitHub"} integration settings and run sync to discover repositories from the configured scope.`)}
                  ` : ""}
                  <div class="panel-header">
                    <h3>Recent Activity</h3>
                    <p>Connection tests, scheduled syncs, retries, and webhook deliveries for this integration, with recorded evidence about what was checked and when the next sync is due.</p>
                  </div>
                  ${runs.length > 0
                    ? table(
                        ["Operation", "Trigger", "Status", "Summary", "Details", "Started"],
                        runs.slice(0, 8).map((run) => [run.operation, run.trigger || "manual", run.status, run.summary, integrationRunDetails(run), run.started_at])
                      )
                    : emptyState("No sync history yet", "Run a connection test or sync to create the first integration activity record.")}
                </article>
              `;
            })
            .join("")}
        </section>
      `;
    case "bootstrap":
      if (state.bootstrapPage.status === "loading" || state.bootstrapPage.status === "idle") {
        return routeStatusLayout(
          "Startup Bootstrap",
          "Loading route-local workspace bootstrap context.",
          emptyState("Loading bootstrap workspace", "Fetching project, team, service, and environment progress for the active organization.")
        );
      }
      if (state.bootstrapPage.status === "error") {
        return routeStatusLayout(
          "Startup Bootstrap",
          "The route-local bootstrap read failed.",
          emptyState("Bootstrap workspace unavailable", state.bootstrapPage.error || "Refresh and retry the bootstrap surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Startup Bootstrap</h3>
              <p>${projects.length > 0
                ? `${projects.length} project(s), ${teams.length} team(s), ${catalog.services.length} service(s), and ${catalog.environments.length} environment(s) are already in scope for this workspace.`
                : "Create the first project and establish the core governance shape of the workspace."}</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Project Setup", projects.length > 0 ? readinessLabel(true) : "Create the first project for the active organization")}
              ${infoCard("Ownership Shape", teams.length > 0 ? `${teams.length} team(s) linked` : "Register teams, services, and environments")}
              ${infoCard("Catalog Depth", catalog.services.length > 0 || catalog.environments.length > 0 ? `${catalog.services.length} services and ${catalog.environments.length} environments modeled` : "Move from planning into rollout execution")}
              ${infoCard("Workspace Scope", `${metrics.projects} projects, ${metrics.services} services, ${metrics.environments} environments`)}
            </div>
            ${projects.length > 0
              ? table(
                  ["Project", "Slug", "Description"],
                  projects.map((project) => [project.name, project.slug, project.description || "No description"])
                )
              : emptyState("No projects yet", "Create the first project here, then add teams, services, and environments to build out the governed delivery surface.")}
            <div class="panel-header compact-top">
              <h3>Teams in Scope</h3>
              <p>${teams.length > 0 ? `${teams.length} team(s) currently anchor service ownership for this workspace.` : "Create the first team to connect ownership, services, and rollout governance."}</p>
            </div>
            ${teams.length > 0
              ? table(
                  ["Team", "Slug", "Project", "Status", "Owners"],
                  teams.map((team) => [
                    team.name,
                    team.slug,
                    projectNameForID(projects, team.project_id),
                    team.status || "active",
                    String((team.owner_user_ids || []).length)
                  ])
                )
              : emptyState("No teams yet", "Create a team after the first project so service ownership and machine actors have a clear home.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Create Project</h3>
              <p>Bootstrap a new governed delivery surface.</p>
            </div>
            ${canAdmin ? `
              <form id="create-project-form" class="stack-form">
                <label><span>Name</span><input name="name" type="text" placeholder="Core Platform" required /></label>
                <label><span>Slug</span><input name="slug" type="text" placeholder="core-platform" required /></label>
                <label><span>Description</span><textarea name="description" placeholder="Shared delivery and governance foundation"></textarea></label>
                <button class="action" type="submit">Create Project</button>
              </form>
            ` : emptyState("Read-only workspace", "Only organization administrators can create new projects.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Manage Teams</h3>
              <p>Create, refine, and archive the current ownership anchor for this workspace.</p>
            </div>
            ${canAdmin
              ? `
                ${projects.length > 0
                  ? `
                    <form id="create-team-form" class="stack-form compact-top-form">
                      <label><span>Project</span><select name="project_id">${projectOptions(state)}</select></label>
                      <label><span>Name</span><input name="name" type="text" placeholder="Core Platform" required /></label>
                      <label><span>Slug</span><input name="slug" type="text" placeholder="core-platform" required /></label>
                      <button class="action" type="submit">Create Team</button>
                    </form>
                  `
                  : emptyState("Create a project first", "Teams are scoped to a project. Create the first project above, then return here to establish ownership.")}
                ${primaryTeam
                  ? `
                    <form id="update-team-form" class="stack-form compact-top-form" data-team-id="${primaryTeam.id}">
                      <label><span>Name</span><input name="name" type="text" value="${primaryTeam.name}" required /></label>
                      <label><span>Slug</span><input name="slug" type="text" value="${primaryTeam.slug}" required /></label>
                      <label><span>Project</span><input name="project_name" type="text" value="${projectNameForID(projects, primaryTeam.project_id)}" readonly /></label>
                      <button class="action ghost" type="submit">Save Current Team</button>
                    </form>
                    <button class="action ghost" id="archive-team-button" data-team-id="${primaryTeam.id}">Archive Current Team</button>
                  `
                  : emptyState("No current team", "Once a team exists, this surface will let you refine its name and slug or archive it when ownership changes.")}
              `
              : emptyState("Read-only workspace", "Only organization administrators can create or change teams in this workspace.")}
          </article>
        </section>
      `;
    case "enterprise":
      if (state.enterprisePage.status === "loading" || state.enterprisePage.status === "idle") {
        return routeStatusLayout(
          "Enterprise Auth and Runtime Trust",
          "Loading route-local enterprise diagnostics.",
          emptyState("Loading enterprise diagnostics", "Fetching identity providers, webhook trust signals, and durable event diagnostics.")
        );
      }
      if (state.enterprisePage.status === "error") {
        return routeStatusLayout(
          "Enterprise Auth and Runtime Trust",
          "The route-local enterprise read failed.",
          emptyState("Enterprise diagnostics unavailable", state.enterprisePage.error || "Refresh and retry the enterprise surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Enterprise Auth and Runtime Trust</h3>
              <p>Configure SSO, verify which auth mode the current operator used, and inspect durable runtime diagnostics before calling this workspace enterprise-ready.</p>
            </div>
            ${advisoryBanner(
              "First-pass enterprise foundation",
              "OIDC sign-in, durable outbox dispatch, and automatic SCM webhook registration are now real foundations in this workspace. They are suitable for careful pilot use, but they are not yet a full enterprise IAM, SCIM, or distributed event-bus story."
            )}
            <div class="highlight-grid">
              ${infoCard("Current Session", enterpriseSessionSummary(state))}
              ${infoCard("Identity Providers", `${identityProviders.length} configured, ${state.publicIdentityProviders.length} public sign-in option${state.publicIdentityProviders.length === 1 ? "" : "s"}`)}
              ${infoCard("Browser Sessions", `${activeBrowserSessionCount(state)} active, ${revokedBrowserSessionCount(state)} revoked or expired in the current diagnostic window`)}
              ${infoCard("Webhook Diagnostics", `${registeredWebhookCount(state)} registered, ${failingWebhookCount(state)} with delivery or registration issues`)}
              ${infoCard("Durable Events", `${pendingOutboxCount(state)} pending, ${processedOutboxCount(state)} processed in the current diagnostic window`)}
            </div>
          </article>
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Enterprise Identity Providers</h3>
              <p>Each provider is organization-scoped and currently uses an OIDC-style browser sign-in flow with domain and default-role guardrails.</p>
            </div>
            <p class="panel-muted">OIDC callback URL: ${state.apiBaseUrl}/api/v1/auth/providers/callback</p>
            ${canAdmin ? `
              <form id="create-identity-provider-form" class="stack-form compact-top-form">
                <div class="form-grid">
                  <label><span>Name</span><input name="name" type="text" placeholder="Acme Okta" required /></label>
                  <label>
                    <span>Kind</span>
                    <select name="kind">
                      <option value="oidc">oidc</option>
                    </select>
                  </label>
                  <label><span>Issuer URL</span><input name="issuer_url" type="text" placeholder="https://acme.okta.com/oauth2/default" required /></label>
                  <label><span>Client ID</span><input name="client_id" type="text" placeholder="0oa123example" required /></label>
                  <label><span>Client Secret Env</span><input name="client_secret_env" type="text" placeholder="CCP_OKTA_CLIENT_SECRET" required /></label>
                  <label><span>Allowed Domains</span><input name="allowed_domains" type="text" placeholder="acme.com, contractors.acme.com" /></label>
                  <label><span>Default Role</span><input name="default_role" type="text" placeholder="org_member" /></label>
                  <label class="checkbox-field">
                    <input name="enabled" type="checkbox" checked />
                    <span>Enable provider for sign-in</span>
                  </label>
                </div>
                <button class="action" type="submit">Add Identity Provider</button>
              </form>
            ` : emptyState("Read-only workspace", "Only organization administrators can add or change enterprise identity providers.")}
            ${identityProviders.length > 0
              ? `
                <div class="repository-grid provider-grid">
                  ${identityProviders
                    .map((provider) => `
                      <form class="surface repository-card identity-provider-config-form" data-provider-id="${provider.id}">
                        <div class="panel-header">
                          <h3>${provider.name}</h3>
                          <p>${provider.kind} • ${provider.enabled ? "enabled" : "disabled"} • ${provider.connection_health || "untested"}</p>
                        </div>
                        <div class="mini-list">
                          <span><strong>Status:</strong> ${provider.status || "configured"}</span>
                          <span><strong>Last Tested:</strong> ${formatTimestamp(provider.last_tested_at)}</span>
                          <span><strong>Last Authenticated:</strong> ${formatTimestamp(provider.last_authenticated_at)}</span>
                          <span><strong>Allowed Domains:</strong> ${(provider.allowed_domains || []).join(", ") || "none"}</span>
                          <span><strong>Default Role:</strong> ${provider.default_role || "org_member"}</span>
                          <span><strong>Client Secret Env:</strong> ${provider.client_secret_env || "not set"}</span>
                        </div>
                        <label><span>Name</span><input name="name" type="text" value="${provider.name}" /></label>
                        <label><span>Issuer URL</span><input name="issuer_url" type="text" value="${provider.issuer_url || ""}" placeholder="https://issuer.example.com" /></label>
                        <label><span>Authorization Endpoint</span><input name="authorization_endpoint" type="text" value="${provider.authorization_endpoint || ""}" placeholder="Optional manual override" /></label>
                        <label><span>Token Endpoint</span><input name="token_endpoint" type="text" value="${provider.token_endpoint || ""}" placeholder="Optional manual override" /></label>
                        <label><span>UserInfo Endpoint</span><input name="userinfo_endpoint" type="text" value="${provider.userinfo_endpoint || ""}" placeholder="Optional manual override" /></label>
                        <label><span>Client ID</span><input name="client_id" type="text" value="${provider.client_id || ""}" /></label>
                        <label><span>Client Secret Env</span><input name="client_secret_env" type="text" value="${provider.client_secret_env || ""}" /></label>
                        <label><span>Allowed Domains</span><input name="allowed_domains" type="text" value="${(provider.allowed_domains || []).join(", ")}" placeholder="acme.com" /></label>
                        <label><span>Default Role</span><input name="default_role" type="text" value="${provider.default_role || ""}" placeholder="org_member" /></label>
                        <label class="checkbox-field">
                          <input name="enabled" type="checkbox" ${provider.enabled ? "checked" : ""} />
                          <span>Provider enabled for browser sign-in</span>
                        </label>
                        ${canAdmin
                          ? `
                            <div class="split-actions">
                              <button class="action" type="submit">Save Provider</button>
                              <button class="action ghost identity-provider-test-button" type="button" data-provider-id="${provider.id}">Test Provider</button>
                            </div>
                          `
                          : `<p class="panel-muted">Read-only access. An organization administrator can test or change this provider.</p>`}
                      </form>
                    `)
                    .join("")}
                </div>
              `
              : emptyState("No identity providers configured", "Create an OIDC provider to enable enterprise sign-in alongside the existing password or development flows.")}
          </article>
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Browser Session Administration</h3>
              <p>Organization administrators can inspect active browser sessions, confirm how each user authenticated, and revoke stale or risky sessions without touching machine tokens.</p>
            </div>
            ${browserSessions.length > 0
              ? table(
                  ["User", "Method", "Provider", "Status", "Last Seen", "Expires", "Current", "Action"],
                  browserSessions.map((session) => [
                    session.user_display_name || session.user_email,
                    session.auth_method || "session",
                    session.auth_provider || session.auth_provider_id || "direct",
                    session.status,
                    formatTimestamp(session.last_seen_at),
                    formatTimestamp(session.expires_at),
                    session.current ? "Yes" : "No",
                    browserSessionActionCell(session, canAdmin)
                  ])
                )
              : emptyState("No browser sessions yet", "Once users sign in through password, dev bootstrap, or enterprise OIDC, their persisted browser sessions will show up here for admin review.")}
          </article>
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Durable Event Diagnostics</h3>
              <p>Important runtime events now land in a persistent outbox before dispatch. This view shows the latest claim, retry, and processing state recorded for the active organization.</p>
            </div>
            ${outboxEvents.length > 0
              ? table(
                  ["Event Type", "Resource", "Status", "Attempts", "Failure Class", "Next Attempt", "Processed", "Last Error", "Action"],
                  outboxEvents.map((event) => [
                    event.event_type,
                    `${event.resource_type}:${event.resource_id}`,
                    event.status,
                    String(event.attempts || 0),
                    String(event.metadata?.last_error_class || "none"),
                    formatTimestamp(event.next_attempt_at),
                    formatTimestamp(event.processed_at),
                    event.last_error || "none",
                    outboxRecoveryActionCell(event, canAdmin)
                  ])
                )
              : emptyState("No durable event records yet", "Once webhook deliveries, sync requests, status events, or rollout updates are published through the outbox, they will show up here for operators to inspect.")} 
          </article>
        </section>
      `;
    case "settings":
      if (state.settingsPage.status === "loading" || state.settingsPage.status === "idle") {
        return routeStatusLayout(
          "Service Accounts and Tokens",
          "Loading route-local machine-actor administration data.",
          emptyState("Loading service accounts", "Fetching machine actors and issued token summaries for the active organization.")
        );
      }
      if (state.settingsPage.status === "error") {
        return routeStatusLayout(
          "Service Accounts and Tokens",
          "The route-local settings read failed.",
          emptyState("Service-account data unavailable", state.settingsPage.error || "Refresh and retry the settings surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Service Accounts and Tokens</h3>
              <p>Machine actors can authenticate with revoked-or-rotated tokens while tenant boundaries stay enforced.</p>
            </div>
            ${serviceAccounts.length > 0 ? table(["Account", "Role", "Status", "Tokens"], serviceAccounts.map((serviceAccount) => [serviceAccount.name, serviceAccount.role, serviceAccount.status, String((serviceAccountTokens[serviceAccount.id] || []).length)])) : emptyState("No service accounts yet", "Create a machine actor for rollout automation, graph ingestion, or controlled change execution.")}
            <div class="page-grid">
              ${serviceAccounts.map((serviceAccount) => {
                const tokens = serviceAccountTokens[serviceAccount.id] || [];
                const activeToken = tokens.find((token) => token.status === "active");
                return `
                  <article class="surface panel">
                    <div class="panel-header">
                      <h3>${serviceAccount.name}</h3>
                      <p>${serviceAccount.description || "Machine actor ready for scoped automation."}</p>
                    </div>
                    ${tokens.length > 0 ? table(["Prefix", "Status", "Expires"], tokens.map((token) => [token.token_prefix, token.status, token.expires_at || "No expiry"])) : emptyState("No tokens", "Issue a token to authenticate this service account.")}
                    ${canAdmin
                      ? `
                        ${serviceAccount.status === "active"
                          ? `
                            <form class="stack-form issue-token-form" data-service-account-id="${serviceAccount.id}">
                              <label><span>Token Name</span><input name="name" type="text" placeholder="${serviceAccount.name} primary" required /></label>
                              <button class="action ghost" type="submit">Issue Token</button>
                            </form>
                            ${activeToken
                              ? `
                                <form class="stack-form rotate-token-form compact-top-form" data-service-account-id="${serviceAccount.id}" data-token-id="${activeToken.id}">
                                  <label><span>Rotated Token Name</span><input name="name" type="text" placeholder="${serviceAccount.name} rotated" /></label>
                                  <label><span>Expires In Hours</span><input name="expires_in_hours" type="number" min="1" step="1" placeholder="24" /></label>
                                  <button class="action ghost" type="submit">Rotate Latest Token</button>
                                </form>
                              `
                              : ""}
                            ${activeToken ? `<button class="action ghost revoke-token-button" data-service-account-id="${serviceAccount.id}" data-token-id="${activeToken.id}">Revoke Latest Token</button>` : ""}
                            <button class="action ghost deactivate-service-account-button" data-service-account-id="${serviceAccount.id}">Deactivate Service Account</button>
                          `
                          : `<p class="panel-muted">This machine actor is inactive. Issue, rotate, and revoke controls stay disabled to match the persisted lifecycle state.</p>`}
                      `
                      : ""}
                  </article>
                `;
              }).join("")}
            </div>
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Create Service Account</h3>
              <p>Provision a machine actor with scoped organization role.</p>
            </div>
            ${canAdmin ? `
              <form id="create-service-account-form" class="stack-form">
                <label><span>Name</span><input name="name" type="text" placeholder="deployer" required /></label>
                <label><span>Description</span><textarea name="description" placeholder="Rollout automation agent"></textarea></label>
                <label><span>Role</span>
                  <select name="role">
                    <option value="viewer">viewer</option>
                    <option value="org_member">org_member</option>
                    <option value="org_admin">org_admin</option>
                  </select>
                </label>
                <button class="action" type="submit">Create Service Account</button>
              </form>
            ` : emptyState("Read-only workspace", "Only organization administrators can issue machine credentials.")}
          </article>
        </section>
      `;
    case "graph":
      if (state.graphPage.status === "loading" || state.graphPage.status === "idle") {
        return routeStatusLayout(
          "System Graph",
          "Loading route-local topology data for the active organization.",
          emptyState("Loading system graph", "Fetching graph relationships and label context for connected resources.")
        );
      }
      if (state.graphPage.status === "error") {
        return routeStatusLayout(
          "System Graph",
          "The route-local graph read failed.",
          emptyState("Graph data unavailable", state.graphPage.error || "Refresh and retry the graph surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>System Graph</h3>
              <p>Topology, dependencies, repositories, and environment bindings for the active organization.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Relationships", `${graphRelationships.length} graph edges tracked.`)}
              ${infoCard("Services Connected", `${new Set(graphRelationships.flatMap((relationship) => [relationship.from_resource_type === "service" ? relationship.from_resource_id : "", relationship.to_resource_type === "service" ? relationship.to_resource_id : ""]).filter(Boolean)).size} services represented in the graph.`)}
              ${infoCard("Environment Bindings", `${graphRelationships.filter((relationship) => relationship.to_resource_type === "environment").length} service-to-environment bindings discovered.`)}
              ${infoCard("Integration Sources", `${new Set(graphRelationships.map((relationship) => relationship.source_integration_id).filter(Boolean)).size} integration source(s) contributing graph edges.`)}
              ${infoCard("Ownership Edges", `${graphRelationships.filter((relationship) => relationship.relationship_type === "team_repository_owner" || relationship.relationship_type === "team_discovered_resource_owner").length} ownership edge(s) recorded.`)}
              ${infoCard("Repo Runtime Links", `${graphRelationships.filter((relationship) => relationship.relationship_type === "discovered_resource_repository").length} repository-to-runtime link(s) recorded.`)}
            </div>
            ${graphRelationships.length > 0
              ? table(
                  ["Relationship", "From", "To", "Source", "Provenance", "Evidence", "Observed"],
                  graphRelationships.map((relationship) => [
                    relationship.relationship_type,
                    resourceLabel(state, relationship.from_resource_type, relationship.from_resource_id),
                    resourceLabel(state, relationship.to_resource_type, relationship.to_resource_id),
                    relationship.source_integration_id ? resourceLabel(state, "integration", relationship.source_integration_id) : "manual",
                    graphRelationshipProvenanceSummary(relationship),
                    graphRelationshipEvidenceSummary(relationship),
                    relationship.last_observed_at || "n/a"
                  ])
                )
              : emptyState("No graph relationships yet", "Connect repositories and runtime bindings to light up the digital twin.")}
          </article>
        </section>
      `;
    case "costs":
      if (state.costsPage.status === "loading" || state.costsPage.status === "idle") {
        return routeStatusLayout(
          "Cost Overview",
          "Loading route-local cost inputs for the active organization.",
          emptyState("Loading cost overview", "Fetching catalog scope and rollback-event context for planning-grade estimates.")
        );
      }
      if (state.costsPage.status === "error") {
        return routeStatusLayout(
          "Cost Overview",
          "The route-local cost read failed.",
          emptyState("Cost data unavailable", state.costsPage.error || "Refresh and retry the cost surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Cost Overview</h3>
              <p>Planning-grade operating estimates derived from service criticality, environment count, and rollout complexity.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Estimated Monthly Spend", formatUSD(totalEstimatedMonthlyCost(state)))}
              ${infoCard("Production Footprint", `${productionEnvironments.length} production environment(s) in active scope.`)}
              ${infoCard("Rollback Cost Risk", formatUSD(rollbackEvents.length * 650))}
              ${infoCard("Most Expensive Service", highestEstimatedCostLabel(state))}
            </div>
            ${catalog.services.length > 0
              ? table(
                  ["Service", "Criticality", "Coverage", "Estimated Monthly Cost"],
                  catalog.services.map((service) => [
                    service.name,
                    service.criticality,
                    [service.has_slo ? "slo" : "", service.has_observability ? "observability" : "", service.customer_facing ? "customer-facing" : ""].filter(Boolean).join(", ") || "standard",
                    formatUSD(estimateServiceMonthlyCost(service, productionEnvironments.length || 1))
                  ])
                )
              : emptyState("No cost inputs yet", "Add services and production environments to unlock a first cost model.")}
          </article>
        </section>
      `;
    case "simulation":
      if (state.simulationPage.status === "loading" || state.simulationPage.status === "idle") {
        return routeStatusLayout(
          "Simulation Lab",
          "Loading route-local scenario-planning inputs for the active organization.",
          emptyState("Loading simulation lab", "Fetching changes, risk posture, rollout context, and rollback guardrails.")
        );
      }
      if (state.simulationPage.status === "error") {
        return routeStatusLayout(
          "Simulation Lab",
          "The route-local simulation read failed.",
          emptyState("Simulation data unavailable", state.simulationPage.error || "Refresh and retry the simulation surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Simulation Lab</h3>
              <p>Scenario planning for the latest change, risk posture, and rollout safeguards.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Latest Change", latestChange ? latestChange.summary : "No change ready for rehearsal")}
              ${infoCard("Risk Posture", latestRisk ? `Score ${latestRisk.score} (${latestRisk.level})` : "No risk assessment yet")}
              ${infoCard("Planned Strategy", latestRollout ? latestRollout.strategy : "No rollout plan generated")}
              ${infoCard("Safest Escape Hatch", latestExecutionDetail?.effective_rollback_policy ? latestExecutionDetail.effective_rollback_policy.name : rollbackPolicies[0]?.name || "Built-in fallback policy")}
            </div>
            ${simulationScenarioRows(state).length > 0
              ? table(["Scenario", "Expected Path", "Trigger", "Operator Posture"], simulationScenarioRows(state))
              : emptyState("No scenarios yet", "Generate a rollout plan to simulate approval, verification, and rollback branches.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Risk Drivers</h3>
              <p>Most recent explainability output from deterministic risk analysis.</p>
            </div>
            ${latestRisk ? `<ul class="plain-list">${latestRisk.explanation.map((item) => `<li>${item}</li>`).join("")}</ul>` : emptyState("No risk drivers yet", "Risk explanations will appear here after an assessment runs.")}
          </article>
        </section>
      `;
    case "dashboard":
    default:
      if (state.dashboardPage.status === "loading" || state.dashboardPage.status === "idle") {
        return routeStatusLayout(
          "Organization Dashboard",
          "Loading route-local dashboard posture for the active organization.",
          emptyState("Loading dashboard", "Fetching metrics, coverage summaries, and latest risk posture.")
        );
      }
      if (state.dashboardPage.status === "error") {
        return routeStatusLayout(
          "Organization Dashboard",
          "The route-local dashboard read failed.",
          emptyState("Dashboard data unavailable", state.dashboardPage.error || "Refresh and retry the dashboard surface.")
        );
      }
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Organization Dashboard</h3>
              <p>Cross-cutting control-plane posture across delivery, governance, and operational readiness.</p>
            </div>
            ${coverageSummary.stale_integrations > 0 || coverageSummary.unmapped_discovered_resources > 0
              ? advisoryBanner(
                  "Pilot runtime depth warning",
                  `${coverageSummary.stale_integrations} stale integration(s) and ${coverageSummary.unmapped_discovered_resources} unmapped discovered runtime resource(s) need attention before this pilot can be treated as fully fresh and well-covered.`
                )
              : ""}
            <div class="highlight-grid">
              ${infoCard("Service Coverage", `${metrics.services} cataloged services with governance metadata.`)}
              ${infoCard("Fresh Integrations", `${coverageSummary.healthy_integrations} integration(s) currently healthy, ${coverageSummary.stale_integrations} stale.`)}
              ${infoCard("Runtime Coverage", `${coverageSummary.workload_coverage_environments} environment(s) with workload coverage and ${coverageSummary.signal_coverage_services} service(s) with signal coverage.`)}
              ${infoCard("Mapping Gaps", `${coverageSummary.unmapped_repositories + coverageSummary.unmapped_discovered_resources} unmapped repository/runtime resource(s) remain.`)}
            </div>
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Latest Risk Signal</h3>
              <p>Most recent risk and rollout recommendation.</p>
            </div>
            ${infoCard(
              "Assessment",
              latestRisk
                ? `Score ${latestRisk.score}, level ${latestRisk.level}, rollout ${latestRisk.recommended_rollout_strategy}.`
                : "No assessments yet. Ingest a change and run analysis."
            )}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Adoption Modes</h3>
              <p>Designed for both greenfield and brownfield teams.</p>
            </div>
            <ul class="plain-list">
              <li>Startup mode with premium bootstrap defaults</li>
              <li>Brownfield advisory mode above existing tools</li>
              <li>Enterprise governance mode with policy and audit depth</li>
            </ul>
          </article>
        </section>
      `;
  }
}

function detailLayout(title: string, summary: string, cards: string[]): string {
  return `
    <section class="page-grid">
      <article class="surface panel wide">
        <div class="panel-header">
          <h3>${title}</h3>
          <p>${summary}</p>
        </div>
        <div class="highlight-grid">
          ${cards.join("")}
        </div>
      </article>
    </section>
  `;
}

function routeStatusLayout(title: string, summary: string, body: string): string {
  return `
    <section class="page-grid">
      <article class="surface panel wide">
        <div class="panel-header">
          <h3>${title}</h3>
          <p>${summary}</p>
        </div>
        ${body}
      </article>
    </section>
  `;
}

function advisoryBanner(title: string, body: string): string {
  return `
    <div class="advisory-banner">
      <strong>${title}</strong>
      <span>${body}</span>
    </div>
  `;
}

function renderDashboardHero(state: ControlPlaneState): string {
  if (state.dashboardPage.status !== "ready") {
    return "";
  }
  const metrics = metricsForState(state);
  return `
    <section class="hero surface">
      <div>
        <p class="eyebrow">Mission</p>
        <h3>Treat change as a governed business event.</h3>
        <p class="hero-copy">
          Assess risk, select the safest rollout path, enforce policy, verify production behavior, and preserve a clean audit trail.
        </p>
      </div>
      <div class="hero-grid">
        ${metricCard("Organizations", String(metrics.organizations), "Tenant boundaries and governance scope")}
        ${metricCard("Services", String(metrics.services), "Cataloged systems under control")}
        ${metricCard("Changes", String(metrics.changes), "Observed change sets in the plane")}
        ${metricCard("Risk Assessments", String(metrics.risk_assessments), "Explainable decisions recorded")}
      </div>
    </section>
  `;
}

function metricCard(label: string, value: string, detail: string): string {
  return `
    <article class="metric-card">
      <p>${label}</p>
      <strong>${value}</strong>
      <span>${detail}</span>
    </article>
  `;
}

function infoCard(label: string, body: string): string {
  return `
    <article class="info-card">
      <p>${label}</p>
      <span>${body}</span>
    </article>
  `;
}

function emptyState(title: string, body: string): string {
  return `
    <div class="empty-state">
      <h4>${title}</h4>
      <p>${body}</p>
    </div>
  `;
}

function renderEnterpriseSignInOptions(providers: ControlPlaneState["publicIdentityProviders"], loginMode: boolean): string {
  return `
    <div class="auth-divider">
      <span>Enterprise sign-in</span>
    </div>
    <p class="auth-note">${loginMode ? "Use your organization's identity provider instead of a local password." : "If your company already connected SSO, use it here instead of creating a password account."}</p>
    <div class="identity-provider-actions">
      ${providers
        .map(
          (provider) => `
            <button class="action ghost identity-provider-login-button" type="button" data-provider-id="${provider.id}">
              Continue with ${provider.name}
            </button>
          `
        )
        .join("")}
    </div>
  `;
}

function renderWebhookRegistrationPanel(
  registration: WebhookRegistration | null,
  integration: Integration,
  canAdmin: boolean
): string {
  return `
    <div class="webhook-registration-card">
      <div class="panel-header">
        <h3>Webhook Health</h3>
        <p>Automatic registration, repair, and delivery diagnostics for this ${integration.kind} instance.</p>
      </div>
      <div class="mini-list">
        <span><strong>Status:</strong> ${registration?.status || "not_registered"}</span>
        <span><strong>Delivery Health:</strong> ${registration?.delivery_health || "unknown"}</span>
        <span><strong>Scope:</strong> ${registration?.scope_identifier || integration.scope_name || integration.scope_type || "not recorded"}</span>
        <span><strong>Callback URL:</strong> ${registration?.callback_url || "not recorded"}</span>
        <span><strong>Last Registered:</strong> ${formatTimestamp(registration?.last_registered_at)}</span>
        <span><strong>Last Validated:</strong> ${formatTimestamp(registration?.last_validated_at)}</span>
        <span><strong>Last Delivery:</strong> ${formatTimestamp(registration?.last_delivery_at)}</span>
        <span><strong>Failure Count:</strong> ${String(registration?.failure_count || 0)}</span>
        <span><strong>Latest Error:</strong> ${registration?.last_error || "none"}</span>
      </div>
      ${canAdmin
        ? `
          <button class="action ghost integration-webhook-sync-button" type="button" data-integration-id="${integration.id}">
            ${webhookActionLabel(registration)}
          </button>
        `
        : `<p class="panel-muted">Read-only access. An organization administrator can register or repair this webhook.</p>`}
    </div>
  `;
}

function integrationModeSummary(integration: Integration): string {
  return integration.mode === "active_control" ? "Active control" : "Advisory only";
}

function integrationInstanceSummary(integration: Integration): string {
  if (integration.instance_key) {
    return `${integration.kind}:${integration.instance_key}`;
  }
  return integration.kind;
}

function integrationAuthStrategySummary(integration: Integration): string {
  switch (integration.auth_strategy) {
    case "github_app":
      return "GitHub App";
    case "personal_access_token":
      return integration.kind === "gitlab" ? "GitLab access token" : "Personal access token";
    case "bearer_token_env":
      return "Bearer token env";
    default:
      return integration.auth_strategy || "auto";
  }
}

function integrationControlSummary(integration: Integration): string {
  if (!integration.enabled) {
    return "Disabled";
  }
  if (integration.mode === "active_control" && integration.control_enabled) {
    return "External actions allowed";
  }
  return "Observe and recommend only";
}

function integrationRunDetails(run: IntegrationSyncRun): string {
  const details: string[] = [];
  if (run.error_class) {
    details.push(`error class ${run.error_class}`);
  }
  if (run.scheduled_for) {
    details.push(`scheduled for ${run.scheduled_for}`);
  }
  if (run.details && run.details.length > 0) {
    details.push(...run.details.slice(0, 3));
  }
  if (details.length === 0) {
    return "No recorded detail";
  }
  return details.join(" | ");
}

function integrationFreshnessSummary(integration: Integration): string {
  switch (integration.freshness_state) {
    case "fresh":
      return `Fresh (${secondsLabel(integration.sync_lag_seconds)})`;
    case "scheduled":
      return "Scheduled, awaiting first successful sync";
    case "manual_only":
      return "Manual only";
    case "stale":
      return `Stale (${secondsLabel(integration.sync_lag_seconds)} since success)`;
    case "error":
      return "Recent sync error";
    case "stale_error":
      return "Stale after recent sync error";
    case "stale_pending":
      return "Stale before first successful sync";
    case "disabled":
      return "Disabled";
    default:
      return integration.schedule_enabled ? "Scheduled" : "Manual only";
  }
}

function integrationCoverageSummaryLabel(integration: Integration, repositories: Repository[], resources: DiscoveredResource[]): string {
  if (isSCMProviderKind(integration.kind)) {
    const mapped = repositories.filter((repository) => repository.service_id && repository.environment_id).length;
    return `${mapped}/${repositories.length} repositories mapped`;
  }
  const mappedResources = resources.filter((resource) => resource.service_id || resource.environment_id || resource.repository_id).length;
  return `${mappedResources}/${resources.length} resources mapped`;
}

function webhookRegistrationSummary(registration: WebhookRegistration | null): string {
  if (!registration) {
    return "Not registered";
  }
  if (registration.status === "registered") {
    return registration.delivery_health === "healthy" ? "Registered and healthy" : `Registered (${registration.delivery_health || "delivery unknown"})`;
  }
  if (registration.status === "repair_recommended") {
    return "Registered, but repair is recommended";
  }
  if (registration.status === "manual_required") {
    return "Manual input still required";
  }
  if (registration.status === "disabled") {
    return "Disabled with integration";
  }
  if (registration.status === "error") {
    return "Registration needs repair";
  }
  return registration.status || "Not registered";
}

function webhookActionLabel(registration: WebhookRegistration | null): string {
  if (!registration || registration.status === "not_registered") {
    return "Register Webhook";
  }
  if (registration.status === "registered") {
    return "Validate Webhook";
  }
  if (registration.status === "repair_recommended") {
    return "Repair Webhook";
  }
  return "Retry Webhook Registration";
}

function isSCMProviderKind(kind: string): boolean {
  return kind === "github" || kind === "gitlab";
}

function rolloutControlModeLabel(summary: RolloutExecutionRuntimeSummary): string {
  if (summary.advisory_only) {
    return "Advisory only";
  }
  if (summary.control_mode === "active_control" && summary.control_enabled) {
    return "Active control enabled";
  }
  if (summary.control_mode) {
    return summary.control_mode;
  }
  return "Unspecified";
}

function rolloutDecisionLabel(decision: string): string {
  switch (decision) {
    case "advisory_rollback":
      return "Rollback recommended";
    case "advisory_pause":
      return "Pause recommended";
    case "advisory_failed":
      return "Failure recommended";
    case "advisory_verified":
      return "Verified recommendation";
    case "manual_review_required":
      return "Manual review required";
    default:
      return decision.replaceAll("_", " ");
  }
}

function rolloutProviderActionLabel(summary: RolloutExecutionRuntimeSummary): string {
  const action = summary.last_provider_action || summary.recommended_action || "sync";
  const disposition = summary.last_action_disposition || "recorded";
  if (disposition === "suppressed") {
    return `${action} recommended, not executed`;
  }
  if (disposition === "executed") {
    return `${action} executed`;
  }
  return `${action} observed`;
}

function verificationEffectLabel(result: VerificationResult): string {
  if (result.action_state === "recommended" || result.control_mode === "advisory") {
    return "Recommendation only";
  }
  return "Control decision recorded";
}

function metricsForState(state: ControlPlaneState) {
  return state.dashboardPage.data?.metrics || state.bootstrapPage.data?.metrics || {
    ...EMPTY_METRICS,
    organizations: state.session.organizations?.length || 0
  };
}

function projectsForState(state: ControlPlaneState) {
  return state.bootstrapPage.data?.projects
    || state.servicePage.data?.projects
    || state.environmentPage.data?.projects
    || state.policiesPage.data?.projects
    || state.graphPage.data?.projects
    || EMPTY_PROJECTS;
}

function teamsForState(state: ControlPlaneState) {
  return state.bootstrapPage.data?.teams || state.servicePage.data?.teams || state.integrationsPage.data?.teams || state.graphPage.data?.teams || EMPTY_TEAMS;
}

function catalogForState(state: ControlPlaneState) {
  return state.catalogPage.data?.catalog
    || state.bootstrapPage.data?.catalog
    || state.servicePage.data?.catalog
    || state.environmentPage.data?.catalog
    || state.policiesPage.data?.catalog
    || state.integrationsPage.data?.catalog
    || state.graphPage.data?.catalog
    || state.deploymentsPage.data?.catalog
    || state.costsPage.data?.catalog
    || EMPTY_CATALOG;
}

function policiesForState(state: ControlPlaneState) {
  return state.policiesPage.data?.policies || EMPTY_POLICIES;
}

function policyDecisionsForState(state: ControlPlaneState) {
  return state.policiesPage.data?.policyDecisions || EMPTY_POLICY_DECISIONS;
}

function integrationsForState(state: ControlPlaneState) {
  return state.rolloutPage.data?.integrations
    || state.integrationsPage.data?.integrations
    || state.enterprisePage.data?.integrations
    || state.graphPage.data?.integrations
    || EMPTY_INTEGRATIONS;
}

function incidentsForState(state: ControlPlaneState) {
  return state.incidentsPage.data?.incidents || EMPTY_INCIDENTS;
}

function selectedIncidentIDForState(state: ControlPlaneState) {
  return state.incidentDetailPage.data?.selectedIncidentID || "";
}

function incidentDetailForState(state: ControlPlaneState) {
  return state.incidentDetailPage.data?.incidentDetail || null;
}

function incidentDetailStatusForState(state: ControlPlaneState) {
  return state.incidentDetailPage.data?.incidentDetailStatus || "idle";
}

function changesForState(state: ControlPlaneState) {
  return state.changeReviewPage.data?.changes || state.graphPage.data?.changes || state.simulationPage.data?.changes || EMPTY_CHANGES;
}

function riskAssessmentsForState(state: ControlPlaneState) {
  return state.riskPage.data?.riskAssessments || state.dashboardPage.data?.riskAssessments || state.simulationPage.data?.riskAssessments || EMPTY_RISK_ASSESSMENTS;
}

function rolloutPlansForState(state: ControlPlaneState) {
  return state.rolloutPage.data?.rolloutPlans || state.simulationPage.data?.rolloutPlans || EMPTY_ROLLOUT_PLANS;
}

function rolloutExecutionsForState(state: ControlPlaneState) {
  return state.rolloutPage.data?.rolloutExecutions
    || state.servicePage.data?.rolloutExecutions
    || state.environmentPage.data?.rolloutExecutions
    || state.simulationPage.data?.rolloutExecutions
    || EMPTY_ROLLOUT_EXECUTIONS;
}

function rolloutExecutionDetailForState(state: ControlPlaneState) {
  return state.rolloutPage.data?.rolloutExecutionDetail || state.simulationPage.data?.rolloutExecutionDetail || null;
}

function auditEventsForState(state: ControlPlaneState) {
  return state.auditPage.data?.auditEvents || EMPTY_AUDIT_EVENTS;
}

function rollbackPoliciesForState(state: ControlPlaneState) {
  return state.deploymentsPage.data?.rollbackPolicies || state.simulationPage.data?.rollbackPolicies || EMPTY_ROLLBACK_POLICIES;
}

function statusEventsForState(state: ControlPlaneState) {
  return state.costsPage.data?.statusEvents || state.simulationPage.data?.statusEvents || EMPTY_STATUS_EVENTS;
}

function statusDashboardForState(state: ControlPlaneState) {
  return state.deploymentsPage.data?.statusDashboard || EMPTY_STATUS_DASHBOARD;
}

function graphRelationshipsForState(state: ControlPlaneState) {
  return state.graphPage.data?.graphRelationships || EMPTY_GRAPH_RELATIONSHIPS;
}

function repositoriesForState(state: ControlPlaneState) {
  return state.integrationsPage.data?.repositories || state.graphPage.data?.repositories || EMPTY_REPOSITORIES;
}

function discoveredResourcesForState(state: ControlPlaneState) {
  return state.integrationsPage.data?.discoveredResources || state.graphPage.data?.discoveredResources || EMPTY_DISCOVERED_RESOURCES;
}

function coverageSummaryForState(state: ControlPlaneState) {
  return state.dashboardPage.data?.coverageSummary || state.deploymentsPage.data?.coverageSummary || state.integrationsPage.data?.coverageSummary || EMPTY_COVERAGE_SUMMARY;
}

function integrationSyncRunsForState(state: ControlPlaneState) {
  return state.integrationsPage.data?.integrationSyncRuns || EMPTY_SYNC_RUNS;
}

function webhookRegistrationsForState(state: ControlPlaneState) {
  return state.integrationsPage.data?.webhookRegistrations || state.enterprisePage.data?.webhookRegistrations || EMPTY_WEBHOOK_REGISTRATIONS;
}

function identityProvidersForState(state: ControlPlaneState) {
  return state.enterprisePage.data?.identityProviders || EMPTY_IDENTITY_PROVIDERS;
}

function outboxEventsForState(state: ControlPlaneState) {
  return state.enterprisePage.data?.outboxEvents || EMPTY_OUTBOX_EVENTS;
}

function browserSessionsForState(state: ControlPlaneState) {
  return state.enterprisePage.data?.browserSessions || EMPTY_BROWSER_SESSIONS;
}

function serviceAccountsForState(state: ControlPlaneState) {
  return state.settingsPage.data?.serviceAccounts || EMPTY_SERVICE_ACCOUNTS;
}

function serviceAccountTokensForState(state: ControlPlaneState) {
  return state.settingsPage.data?.serviceAccountTokens || EMPTY_SERVICE_ACCOUNT_TOKENS;
}

function statusFilterValue(state: ControlPlaneState, key: string): string {
  const value = statusDashboardForState(state).filters?.[key];
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (typeof value === "number") {
    return String(value);
  }
  return "";
}

function statusWindowLabel(state: ControlPlaneState): string {
  const summary = statusDashboardForState(state).summary;
  if (summary.total === 0 || summary.returned === 0) {
    return "No events matched the current query.";
  }
  const start = summary.offset + 1;
  const end = summary.offset + summary.returned;
  return `Showing ${start}-${end} of ${summary.total} matching event(s).`;
}

function statusSourceOptions(state: ControlPlaneState, selected: string): string {
  const sources = new Set<string>(["control_plane", "github_webhook", "kubernetes", "prometheus"]);
  [...statusEventsForState(state), ...statusDashboardForState(state).events].forEach((event) => {
    if (event.source) {
      sources.add(event.source);
    }
  });
  return [`<option value="">All sources</option>`]
    .concat(
      [...sources]
        .filter(Boolean)
        .sort()
        .map((source) => `<option value="${source}" ${source === selected ? "selected" : ""}>${source}</option>`)
    )
    .join("");
}

function statusEventTypeOptions(state: ControlPlaneState, selected: string): string {
  const eventTypes = new Set<string>();
  [...statusEventsForState(state), ...statusDashboardForState(state).events].forEach((event) => {
    if (event.event_type) {
      eventTypes.add(event.event_type);
    }
  });
  return [`<option value="">All event types</option>`]
    .concat(
      [...eventTypes]
        .sort()
        .map((eventType) => `<option value="${eventType}" ${eventType === selected ? "selected" : ""}>${eventType}</option>`)
    )
    .join("");
}

function statusEventEffectLabel(event: StatusEvent): string {
  const metadata = event.metadata || {};
  const recommendedAction = String(metadata.recommended_action || "");
  const disposition = String(metadata.action_disposition || "");
  if (event.event_type === "rollout.execution.action_suppressed") {
    return recommendedAction ? `${recommendedAction} suppressed` : "Suppressed in advisory mode";
  }
  if (event.event_type === "rollout.execution.action_executed") {
    return "Executed against provider";
  }
  if (event.event_type === "verification.recorded" && event.summary.toLowerCase().startsWith("advisory recommendation")) {
    return "Recommendation recorded";
  }
  if (disposition === "suppressed") {
    return recommendedAction ? `${recommendedAction} suppressed` : "Suppressed";
  }
  if (disposition === "executed") {
    return "Executed";
  }
  if (event.event_type.includes("signal")) {
    return "Observed";
  }
  return "Recorded";
}

function activeOrganizationRole(state: ControlPlaneState): string {
  const activeOrganizationID = state.session.active_organization_id || "";
  const organization = (state.session.organizations || []).find((entry) => entry.organization_id === activeOrganizationID);
  return organization?.role || "";
}

function projectOptions(state: ControlPlaneState, selectedID = ""): string {
  const projects = projectsForState(state);
  if (projects.length === 0) {
    return `<option value="">Create a project first</option>`;
  }
  return projects.map((project) => `<option value="${project.id}" ${project.id === selectedID ? "selected" : ""}>${project.name}</option>`).join("");
}

function teamOptions(state: ControlPlaneState, selectedID = ""): string {
  const teams = teamsForState(state);
  if (teams.length === 0) {
    return `<option value="">Create a team through the API or CLI first</option>`;
  }
  return teams.map((team) => `<option value="${team.id}" ${team.id === selectedID ? "selected" : ""}>${team.name}</option>`).join("");
}

function projectNameForID(projects: Project[], projectID: string): string {
  return projects.find((project) => project.id === projectID)?.name || "Unknown project";
}

function policyProjectOptions(state: ControlPlaneState, selectedID = ""): string {
  const projects = projectsForState(state);
  if (projects.length === 0) {
    return `<option value="">Organization-wide</option>`;
  }
  return [
    `<option value="">Organization-wide</option>`,
    ...projects.map((project) => `<option value="${project.id}" ${project.id === selectedID ? "selected" : ""}>${project.name}</option>`)
  ].join("");
}

function policyServiceOptions(state: ControlPlaneState, selectedID = ""): string {
  const services = catalogForState(state).services;
  if (services.length === 0) {
    return `<option value="">Any service</option>`;
  }
  return [
    `<option value="">Any service</option>`,
    ...services.map((service) => `<option value="${service.id}" ${service.id === selectedID ? "selected" : ""}>${service.name}</option>`)
  ].join("");
}

function policyEnvironmentOptions(state: ControlPlaneState, selectedID = ""): string {
  const environments = catalogForState(state).environments;
  if (environments.length === 0) {
    return `<option value="">Any environment</option>`;
  }
  return [
    `<option value="">Any environment</option>`,
    ...environments.map((environment) => `<option value="${environment.id}" ${environment.id === selectedID ? "selected" : ""}>${environment.name}</option>`)
  ].join("");
}

function enterpriseModeRows(state: ControlPlaneState): string[][] {
  const metrics = metricsForState(state);
  const integrations = integrationsForState(state);
  const changes = changesForState(state);
  const riskAssessments = riskAssessmentsForState(state);
  const rolloutPlans = rolloutPlansForState(state);
  const rolloutExecutions = rolloutExecutionsForState(state);
  const rollbackPolicies = rollbackPoliciesForState(state);
  const statusEvents = statusEventsForState(state);
  const serviceAccounts = serviceAccountsForState(state);
  return [
    [
      "Read-Only",
      readinessLabel(integrations.length > 0 && metrics.services > 0),
      `${integrations.length} integrations and ${metrics.services} cataloged services`
    ],
    [
      "Advisory",
      readinessLabel(changes.length > 0 && riskAssessments.length > 0),
      `${changes.length} change(s) and ${riskAssessments.length} explainable assessment(s)`
    ],
    [
      "Governed",
      readinessLabel(rolloutPlans.length > 0 && rollbackPolicies.length > 0),
      `${rolloutPlans.length} rollout plan(s) with ${rollbackPolicies.length} rollback policy override(s)`
    ],
    [
      "Automated",
      readinessLabel(rolloutExecutions.length > 0 && serviceAccounts.length > 0),
      `${rolloutExecutions.length} execution(s), ${statusEvents.length} status events, ${serviceAccounts.length} machine actor(s)`
    ]
  ];
}

function simulationScenarioRows(state: ControlPlaneState): string[][] {
  const latestRisk = riskAssessmentsForState(state)[0];
  const latestRollout = rolloutPlansForState(state)[0];
  const latestExecutionDetail = rolloutExecutionDetailForState(state);
  const rollbackPolicies = rollbackPoliciesForState(state);
  const statusEvents = statusEventsForState(state);
  const rows: string[][] = [];

  if (latestRollout) {
    rows.push([
      "Proceed with planned rollout",
      latestRollout.strategy,
      latestRollout.verification_signals.join(", ") || "verification pending",
      latestRollout.approval_required ? `Approval level ${latestRollout.approval_level}` : "No manual approval required"
    ]);
  }
  if (latestRisk) {
    rows.push([
      "Risk spike before deploy",
      latestRisk.recommended_rollout_strategy,
      latestRisk.explanation[0] || "elevated deterministic risk score",
      `Guardrails: ${latestRisk.recommended_guardrails.join(", ") || "none generated"}`
    ]);
  }
  rows.push([
    "Signals degrade during rollout",
    latestExecutionDetail?.effective_rollback_policy?.rollback_on_critical_signals ? "automatic rollback" : "manual review",
    latestExecutionDetail?.runtime_summary.latest_signal_summary || "critical latency and error signals",
    latestExecutionDetail?.effective_rollback_policy?.name || rollbackPolicies[0]?.name || "built-in fallback policy"
  ]);
  rows.push([
    "Operator pause and inspect",
    "pause or hold progression",
    statusEvents[0]?.summary || "verification requires operator review",
    "Use deployment history and incident detail to trace the timeline"
  ]);

  return rows;
}

function readinessLabel(ready: boolean): string {
  return ready ? "Ready" : "In Progress";
}

function enterpriseSessionSummary(state: ControlPlaneState): string {
  if (!state.session.authenticated) {
    return "Anonymous";
  }
  const provider = state.session.auth_provider || state.session.auth_method || "session";
  const expiry = state.session.expires_at ? ` until ${formatTimestamp(state.session.expires_at)}` : "";
  return `${state.session.email || state.session.actor} via ${provider}${expiry}`;
}

function activeBrowserSessionCount(state: ControlPlaneState): number {
  return browserSessionsForState(state).filter((session) => session.status === "active").length;
}

function revokedBrowserSessionCount(state: ControlPlaneState): number {
  return browserSessionsForState(state).filter((session) => session.status !== "active").length;
}

function registeredWebhookCount(state: ControlPlaneState): number {
  return Object.values(webhookRegistrationsForState(state)).filter((registration) => registration?.status === "registered").length;
}

function failingWebhookCount(state: ControlPlaneState): number {
  return Object.values(webhookRegistrationsForState(state)).filter((registration) => registration && registration.status !== "registered").length;
}

function pendingOutboxCount(state: ControlPlaneState): number {
  return outboxEventsForState(state).filter((event) => event.status !== "processed").length;
}

function processedOutboxCount(state: ControlPlaneState): number {
  return outboxEventsForState(state).filter((event) => event.status === "processed").length;
}

function browserSessionActionCell(session: BrowserSessionInfo, canAdmin: boolean): string {
  if (!canAdmin) {
    return "Read-only";
  }
  if (session.status !== "active") {
    return "No action";
  }
  if (session.current) {
    return "Current session";
  }
  return `<button class="action ghost revoke-browser-session-button" data-browser-session-id="${session.id}">Revoke</button>`;
}

function estimateServiceMonthlyCost(service: Service, productionEnvironmentCount: number): number {
  const criticalityBase =
    service.criticality === "mission_critical" ? 4800 :
    service.criticality === "high" ? 3200 :
    service.criticality === "medium" ? 2200 :
    1600;
  const productionFootprint = Math.max(1, productionEnvironmentCount) * 650;
  const observabilityPremium = service.has_observability ? 425 : 0;
  const sloPremium = service.has_slo ? 275 : 0;
  const dependencyPremium = service.dependent_services_count * 140;
  const customerPremium = service.customer_facing ? 550 : 0;
  return criticalityBase + productionFootprint + observabilityPremium + sloPremium + dependencyPremium + customerPremium;
}

function totalEstimatedMonthlyCost(state: ControlPlaneState): number {
  const catalog = catalogForState(state);
  const productionEnvironmentCount = catalog.environments.filter((environment) => environment.production).length || 1;
  return catalog.services.reduce((sum, service) => sum + estimateServiceMonthlyCost(service, productionEnvironmentCount), 0);
}

function highestEstimatedCostLabel(state: ControlPlaneState): string {
  const catalog = catalogForState(state);
  const productionEnvironmentCount = catalog.environments.filter((environment) => environment.production).length || 1;
  const top = [...catalog.services]
    .sort((left, right) => estimateServiceMonthlyCost(right, productionEnvironmentCount) - estimateServiceMonthlyCost(left, productionEnvironmentCount))[0];
  if (!top) {
    return "No service estimate yet";
  }
  return `${top.name} (${formatUSD(estimateServiceMonthlyCost(top, productionEnvironmentCount))})`;
}

function formatUSD(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 0
  }).format(value);
}

function resourceLabel(state: ControlPlaneState, resourceType: string, resourceID: string): string {
  const catalog = catalogForState(state);
  const integrations = integrationsForState(state);
  const changes = changesForState(state);
  const projects = projectsForState(state);
  const repositories = repositoriesForState(state);
  const discoveredResources = discoveredResourcesForState(state);
  const teams = teamsForState(state);
  switch (resourceType) {
    case "service":
      return catalog.services.find((service) => service.id === resourceID)?.name || resourceID;
    case "environment":
      return catalog.environments.find((environment) => environment.id === resourceID)?.name || resourceID;
    case "integration":
      return integrations.find((integration) => integration.id === resourceID)?.name || resourceID;
    case "change_set":
      return changes.find((change) => change.id === resourceID)?.summary || resourceID;
    case "project":
      return projects.find((project) => project.id === resourceID)?.name || resourceID;
    case "repository":
      return repositories.find((repository) => repository.id === resourceID)?.name || resourceID;
    case "discovered_resource":
      return discoveredResources.find((resource) => resource.id === resourceID)?.name || resourceID;
    case "team":
      return teams.find((team) => team.id === resourceID)?.name || resourceID;
    default:
      return resourceID;
  }
}

function teamLabel(state: ControlPlaneState, teamID: string): string {
  return teamsForState(state).find((team) => team.id === teamID)?.name || teamID;
}

function repositoryLabel(state: ControlPlaneState, repositoryID: string): string {
  return repositoriesForState(state).find((repository) => repository.id === repositoryID)?.name || repositoryID;
}

function metadataRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}

function metadataString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function metadataStringList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((entry) => metadataString(entry)).filter(Boolean);
  }
  return [];
}

function ownershipSummary(state: ControlPlaneState, metadata?: Record<string, unknown>): string {
  const ownership = metadataRecord(metadata?.ownership);
  const inferredOwner = metadataRecord(metadata?.inferred_owner);
  const importedOwners = metadataStringList(ownership.owners);
  const inferredTeamID = metadataString(inferredOwner.team_id);
  const pieces = [];
  if (importedOwners.length > 0) {
    pieces.push(`CODEOWNERS ${importedOwners.slice(0, 3).join(", ")}${importedOwners.length > 3 ? ` +${importedOwners.length - 3} more` : ""}`);
  } else if (metadataString(ownership.status) === "not_found") {
    pieces.push("CODEOWNERS not found");
  } else if (metadataString(ownership.status) === "unavailable") {
    pieces.push(`CODEOWNERS unavailable${metadataString(ownership.error) ? `: ${metadataString(ownership.error)}` : ""}`);
  }
  if (inferredTeamID) {
    pieces.push(`team ${teamLabel(state, inferredTeamID)} inferred from service mapping`);
  }
  return pieces.join(" | ") || "No ownership evidence recorded";
}

function mappingProvenanceSummary(metadata?: Record<string, unknown>): string {
  const mapping = metadataRecord(metadata?.mapping_provenance);
  const parts = Object.entries(mapping).map(([field, value]) => {
    const entry = metadataRecord(value);
    const source = metadataString(entry.source) || "unknown";
    return `${field}: ${source}`;
  });
  return parts.join(", ") || "No mapping provenance recorded";
}

function graphRelationshipProvenanceSummary(relationship: GraphRelationship): string {
  const metadata = metadataRecord(relationship.metadata);
  return metadataString(metadata.provenance_source) || "unspecified";
}

function policyScopeSummary(state: ControlPlaneState, policy: Policy): string {
  const scopedParts = [];
  if (policy.project_id) {
    scopedParts.push(`project ${resourceLabel(state, "project", policy.project_id)}`);
  }
  if (policy.service_id) {
    scopedParts.push(`service ${resourceLabel(state, "service", policy.service_id)}`);
  }
  if (policy.environment_id) {
    scopedParts.push(`environment ${resourceLabel(state, "environment", policy.environment_id)}`);
  }
  return scopedParts.join(" | ") || "organization-wide";
}

function policyConditionSummary(policy: Policy): string {
  const condition = policy.conditions || {};
  const parts = [];
  if (condition.min_risk_level) {
    parts.push(`risk >= ${condition.min_risk_level}`);
  }
  if (condition.production_only) {
    parts.push("production only");
  }
  if (condition.regulated_only) {
    parts.push("regulated only");
  }
  if ((condition.required_change_types || []).length > 0) {
    parts.push(`change types: ${(condition.required_change_types || []).join(", ")}`);
  }
  if ((condition.required_touches || []).length > 0) {
    parts.push(`touches: ${(condition.required_touches || []).join(", ")}`);
  }
  if ((condition.missing_capabilities || []).length > 0) {
    parts.push(`missing: ${(condition.missing_capabilities || []).join(", ")}`);
  }
  return parts.join(" | ") || "No extra conditions";
}

function policyDecisionTargetLabel(state: ControlPlaneState, decision: PolicyDecision): string {
  if (decision.rollout_execution_id) {
    return `execution ${decision.rollout_execution_id}`;
  }
  if (decision.rollout_plan_id) {
    return `plan ${decision.rollout_plan_id}`;
  }
  if (decision.risk_assessment_id) {
    return `risk ${decision.risk_assessment_id}`;
  }
  if (decision.change_set_id) {
    return resourceLabel(state, "change_set", decision.change_set_id);
  }
  const scopedParts = [];
  if (decision.project_id) {
    scopedParts.push(`project ${resourceLabel(state, "project", decision.project_id)}`);
  }
  if (decision.service_id) {
    scopedParts.push(`service ${resourceLabel(state, "service", decision.service_id)}`);
  }
  if (decision.environment_id) {
    scopedParts.push(`environment ${resourceLabel(state, "environment", decision.environment_id)}`);
  }
  return scopedParts.join(" | ") || "organization-wide";
}

function renderPolicyManagementCard(state: ControlPlaneState, policy: Policy, canAdmin: boolean): string {
  const triggers = (policy.triggers || []).join(", ") || "No computed triggers";
  return `
    <form class="surface repository-card policy-config-form" data-policy-id="${policy.id}">
      <div class="panel-header">
        <h3>${policy.name}</h3>
        <p>${policy.code} · ${policy.applies_to} · ${policy.mode}</p>
      </div>
      <div class="mini-list">
        <span><strong>Scope:</strong> ${policyScopeSummary(state, policy)}</span>
        <span><strong>Enabled:</strong> ${policy.enabled ? "Yes" : "No"}</span>
        <span><strong>Priority:</strong> ${policy.priority ?? 0}</span>
        <span><strong>Conditions:</strong> ${policyConditionSummary(policy)}</span>
        <span><strong>Triggers:</strong> ${triggers}</span>
        <span><strong>Description:</strong> ${policy.description || "No description recorded"}</span>
      </div>
      ${canAdmin ? `
        <label><span>Name</span><input name="name" type="text" value="${policy.name}" /></label>
        <label><span>Code</span><input name="code" type="text" value="${policy.code}" /></label>
        <label><span>Workflow</span>
          <select name="applies_to">
            <option value="risk_assessment" ${policy.applies_to === "risk_assessment" ? "selected" : ""}>risk_assessment</option>
            <option value="rollout_plan" ${policy.applies_to === "rollout_plan" ? "selected" : ""}>rollout_plan</option>
          </select>
        </label>
        <label><span>Mode</span>
          <select name="mode">
            <option value="advisory" ${policy.mode === "advisory" ? "selected" : ""}>advisory</option>
            <option value="require_manual_review" ${policy.mode === "require_manual_review" ? "selected" : ""}>require_manual_review</option>
            <option value="block" ${policy.mode === "block" ? "selected" : ""}>block</option>
          </select>
        </label>
        <label><span>Project Scope</span><select name="project_id">${policyProjectOptions(state, policy.project_id || "")}</select></label>
        <label><span>Service Scope</span><select name="service_id">${policyServiceOptions(state, policy.service_id || "")}</select></label>
        <label><span>Environment Scope</span><select name="environment_id">${policyEnvironmentOptions(state, policy.environment_id || "")}</select></label>
        <label><span>Priority</span><input name="priority" type="number" value="${policy.priority ?? 0}" /></label>
        <label><span>Description</span><textarea name="description">${policy.description || ""}</textarea></label>
        <label><span>Minimum Risk</span><input name="min_risk_level" type="text" value="${policy.conditions?.min_risk_level || ""}" /></label>
        <label><span>Required Change Types</span><input name="required_change_types" type="text" value="${(policy.conditions?.required_change_types || []).join(", ")}" /></label>
        <label><span>Required Touches</span><input name="required_touches" type="text" value="${(policy.conditions?.required_touches || []).join(", ")}" /></label>
        <label><span>Missing Capabilities</span><input name="missing_capabilities" type="text" value="${(policy.conditions?.missing_capabilities || []).join(", ")}" /></label>
        <label class="checkbox-row"><input name="production_only" type="checkbox" ${policy.conditions?.production_only ? "checked" : ""} /> <span>Production only</span></label>
        <label class="checkbox-row"><input name="regulated_only" type="checkbox" ${policy.conditions?.regulated_only ? "checked" : ""} /> <span>Regulated only</span></label>
        <label class="checkbox-row"><input name="enabled" type="checkbox" ${policy.enabled ? "checked" : ""} /> <span>Enabled</span></label>
        <div class="button-row">
          <button class="action ghost" type="submit">Save Policy</button>
          <button class="action ghost policy-toggle-button" type="button" data-policy-id="${policy.id}" data-policy-enabled="${policy.enabled ? "true" : "false"}">${policy.enabled ? "Disable" : "Enable"}</button>
        </div>
      ` : `<p class="panel-muted">Read-only: org admins can edit or enable and disable this policy.</p>`}
    </form>
  `;
}

function graphRelationshipEvidenceSummary(relationship: GraphRelationship): string {
  const metadata = metadataRecord(relationship.metadata);
  const evidence = metadataString(metadata.evidence);
  return evidence || "No evidence summary";
}

function renderDiscoveredResourceCard(
  state: ControlPlaneState,
  resource: DiscoveredResource,
  repositories: Repository[],
  canAdmin: boolean
): string {
  const catalog = catalogForState(state);
  const serviceOptions = catalog.services
    .map((service) => `<option value="${service.id}" ${service.id === resource.service_id ? "selected" : ""}>${service.name}</option>`)
    .join("");
  const environmentOptions = catalog.environments
    .map((environment) => `<option value="${environment.id}" ${environment.id === resource.environment_id ? "selected" : ""}>${environment.name}</option>`)
    .join("");
  const repositoryOptions = repositories
    .map((repository) => `<option value="${repository.id}" ${repository.id === resource.repository_id ? "selected" : ""}>${repository.name}</option>`)
    .join("");

  return `
    <form class="surface repository-card discovered-resource-map-form" data-discovered-resource-id="${resource.id}">
      <div class="panel-header">
        <h3>${resource.name}</h3>
        <p>${resource.provider} / ${resource.resource_type}</p>
      </div>
      <div class="mini-list">
        <span><strong>Namespace:</strong> ${resource.namespace || "n/a"}</span>
        <span><strong>Health:</strong> ${resource.health || "unknown"}</span>
        <span><strong>Status:</strong> ${resource.status || "observed"}</span>
        <span><strong>Last Seen:</strong> ${formatTimestamp(resource.last_seen_at)}</span>
        <span><strong>Mapping:</strong> ${discoveredResourceMappingSummary(state, resource)}</span>
        <span><strong>Ownership:</strong> ${ownershipSummary(state, resource.metadata)}</span>
        <span><strong>Provenance:</strong> ${mappingProvenanceSummary(resource.metadata)}</span>
        <span><strong>Summary:</strong> ${resource.summary || "No provider summary recorded"}</span>
      </div>
      ${canAdmin ? `
        <label>
          <span>Service</span>
          <select name="service_id">
            <option value="">Not mapped</option>
            ${serviceOptions}
          </select>
        </label>
        <label>
          <span>Environment</span>
          <select name="environment_id">
            <option value="">Not mapped</option>
            ${environmentOptions}
          </select>
        </label>
        <label>
          <span>Repository</span>
          <select name="repository_id">
            <option value="">Not mapped</option>
            ${repositoryOptions}
          </select>
        </label>
        <label>
          <span>Review Status</span>
          <select name="status">
            ${["candidate", "discovered", "review_required", "mapped", "ignored"]
              .map((status) => `<option value="${status}" ${resource.status === status ? "selected" : ""}>${status}</option>`)
              .join("")}
          </select>
        </label>
        <button class="action ghost" type="submit">Save Runtime Mapping</button>
      ` : `<p class="panel-muted">Read-only access. An organization administrator can review or map this discovered resource.</p>`}
    </form>
  `;
}

function discoveredResourceMappingSummary(state: ControlPlaneState, resource: DiscoveredResource): string {
  const mappings = [];
  if (resource.service_id) {
    mappings.push(`service ${resourceLabel(state, "service", resource.service_id)}`);
  }
  if (resource.environment_id) {
    mappings.push(`environment ${resourceLabel(state, "environment", resource.environment_id)}`);
  }
  if (resource.repository_id) {
    mappings.push(`repository ${repositoryLabel(state, resource.repository_id)}`);
  }
  if (mappings.length === 0) {
    return "Unmapped";
  }
  return mappings.join(", ");
}

function formatTimestamp(value?: string): string {
  if (!value) {
    return "Never";
  }
  return value;
}

function secondsLabel(value?: number): string {
  if (!value || value <= 0) {
    return "not recorded";
  }
  if (value < 60) {
    return `${value}s`;
  }
  if (value < 3600) {
    return `${Math.round(value / 60)}m`;
  }
  if (value < 86400) {
    return `${Math.round(value / 3600)}h`;
  }
  return `${Math.round(value / 86400)}d`;
}

function table(headers: string[], rows: string[][]): string {
  if (rows.length === 0) {
    return emptyState("No data yet", "Connect the API and ingest records to populate this surface.");
  }

  return `
    <div class="table-wrap">
      <table>
        <thead>
          <tr>${headers.map((header) => `<th>${header}</th>`).join("")}</tr>
        </thead>
        <tbody>
          ${rows
            .map((row) => `<tr>${row.map((value) => `<td>${value}</td>`).join("")}</tr>`)
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function statusEventFeed(events: StatusEvent[]): string {
  return `
    <div class="table-wrap">
      <table id="status-event-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Event</th>
            <th>Summary</th>
            <th>Effect</th>
            <th>Resource</th>
            <th>Source</th>
            <th>Outcome</th>
          </tr>
        </thead>
        <tbody>
          ${events
            .map((event) => {
              const searchable = [
                event.event_type,
                event.summary,
                event.resource_type,
                event.resource_id,
                event.source || "",
                event.actor || "",
                ...(event.explanation || [])
              ]
                .join(" ")
                .toLowerCase();
              const rollbackRelated =
                event.event_type.includes("rollback") || event.new_state === "rolled_back" || event.summary.toLowerCase().includes("rollback");
              return `
                <tr data-status-event-row data-searchable="${searchable}" data-rollback="${rollbackRelated ? "true" : "false"}">
                  <td>${event.created_at || ""}</td>
                  <td>${event.event_type}</td>
                  <td>${event.summary}</td>
                  <td>${statusEventEffectLabel(event)}</td>
                  <td>${event.resource_type}:${event.resource_id}</td>
                  <td>${event.source || event.actor_type || "control_plane"}</td>
                  <td>${event.outcome || event.new_state || "recorded"}</td>
                </tr>
              `;
            })
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function outboxRecoveryActionCell(event: OutboxEvent, canAdmin: boolean): string {
  if (!canAdmin) {
    return "Inspect only";
  }
  if (event.status === "error") {
    return `<button class="action ghost outbox-retry-button" type="button" data-outbox-event-id="${event.id}">Retry Now</button>`;
  }
  if (event.status === "dead_letter") {
    return `<button class="action ghost outbox-requeue-button" type="button" data-outbox-event-id="${event.id}">Requeue</button>`;
  }
  return "Inspect only";
}
