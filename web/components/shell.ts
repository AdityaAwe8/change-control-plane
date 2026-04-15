import { RouteDefinition, routes } from "../app/router";
import { ControlPlaneState } from "../lib/api";

export function renderShell(state: ControlPlaneState, route: RouteDefinition): string {
  if (!state.session.authenticated) {
    return renderSignIn(state);
  }

  const organizations = state.session.organizations || [];
  const activeOrganizationID = state.session.active_organization_id || "";

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
        <div class="status-card">
          <span class="status-pill ${state.connected ? "status-live" : "status-offline"}">
            ${state.connected ? "API Connected" : state.tokenPresent ? "Session Cached" : "Awaiting Sign-In"}
          </span>
          <p>Runtime: ${state.apiBaseUrl}</p>
          <p>Session: ${state.session.actor}</p>
          <p>Org Scope: ${organizations.find((organization) => organization.organization_id === activeOrganizationID)?.organization || "Select an organization"}</p>
        </div>
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

        <section class="hero surface">
          <div>
            <p class="eyebrow">Mission</p>
            <h3>Treat change as a governed business event.</h3>
            <p class="hero-copy">
              Assess risk, select the safest rollout path, enforce policy, verify production behavior, and preserve a clean audit trail.
            </p>
          </div>
          <div class="hero-grid">
            ${metricCard("Organizations", String(state.metrics.organizations), "Tenant boundaries and governance scope")}
            ${metricCard("Services", String(state.metrics.services), "Cataloged systems under control")}
            ${metricCard("Changes", String(state.metrics.changes), "Observed change sets in the plane")}
            ${metricCard("Risk Assessments", String(state.metrics.risk_assessments), "Explainable decisions recorded")}
          </div>
        </section>

        ${renderPage(state, route.key)}
      </main>
    </div>
  `;
}

function renderSignIn(state: ControlPlaneState): string {
  return `
    <div class="auth-shell">
      <section class="auth-panel surface">
        <p class="eyebrow">ChangeControlPlane</p>
        <h1>Authenticate the control plane.</h1>
        <p class="lede">
          Sign in through the development bootstrap flow to get a real persisted user, organization membership, and active tenant scope.
        </p>
        <form id="login-form" class="login-form">
          <label>
            <span>Email</span>
            <input id="login-email" name="email" type="email" placeholder="owner@acme.local" required />
          </label>
          <label>
            <span>Display Name</span>
            <input id="login-display-name" name="display_name" type="text" placeholder="Acme Owner" required />
          </label>
          <label>
            <span>Organization Name</span>
            <input id="login-organization-name" name="organization_name" type="text" placeholder="Acme" />
          </label>
          <label>
            <span>Organization Slug</span>
            <input id="login-organization-slug" name="organization_slug" type="text" placeholder="acme" />
          </label>
          <button class="action" type="submit">Bootstrap and Sign In</button>
        </form>
        <div class="status-card">
          <span class="status-pill ${state.connected ? "status-live" : state.tokenPresent ? "status-offline" : "status-offline"}">
            ${state.connected ? "API Reachable" : state.tokenPresent ? "Session Needs Refresh" : "No Active Session"}
          </span>
          <p>Runtime: ${state.apiBaseUrl}</p>
          <p>Mode: ${state.session.mode}</p>
        </div>
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
  const primaryService = state.catalog.services[0];
  const primaryEnvironment = state.catalog.environments[0];
  const latestChange = state.changes[0];
  const latestRisk = state.riskAssessments[0];
  const latestRollout = state.rolloutPlans[0];
  const latestExecution = state.rolloutExecutions[0];
  const latestIncident = state.incidents[0];
  const serviceAccounts = state.serviceAccounts || [];
  const canAdmin = activeOrganizationRole(state) === "org_admin";
  const canOperate = ["org_admin", "org_member"].includes(activeOrganizationRole(state));

  switch (routeKey) {
    case "catalog":
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Service Catalog</h3>
              <p>Ownership, criticality, and operational coverage in one place.</p>
            </div>
            ${table(
              ["Service", "Criticality", "Customer", "SLO", "Observability", "Dependencies"],
              state.catalog.services.map((service) => [
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
              state.catalog.environments.map((environment) => [
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
            </div>
            ${state.catalog.services.length > 0 ? table(["Service", "Criticality", "Status", "Customer"], state.catalog.services.map((service) => [service.name, service.criticality, service.status || "active", service.customer_facing ? "Yes" : "No"])) : emptyState("No services yet", "Create a service to unlock change review, rollout execution, and graph enrichment.")}
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
            ${canAdmin && primaryService ? `<button class="action ghost" id="archive-service-button" data-service-id="${primaryService.id}">Archive Current Service</button>` : ""}
          </article>
        </section>
      `;
    case "environment":
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
            </div>
            ${state.catalog.environments.length > 0 ? table(["Environment", "Type", "Region", "Status"], state.catalog.environments.map((environment) => [environment.name, environment.type, environment.region, environment.status || "active"])) : emptyState("No environments yet", "Create environments to drive rollout targeting and runtime verification context.")}
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
            ${canAdmin && primaryEnvironment ? `<button class="action ghost" id="archive-environment-button" data-environment-id="${primaryEnvironment.id}">Archive Current Environment</button>` : ""}
          </article>
        </section>
      `;
    case "change-review":
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
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Rollout Plan and Execution</h3>
              <p>${latestRollout ? `${latestRollout.strategy} rollout with window ${latestRollout.deployment_window}.` : "Generate a rollout plan to see approvals, verification signals, and rollback posture."}</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Approval Required", latestRollout?.approval_required ? "Yes" : "No")}
              ${infoCard("Verification", latestRollout?.verification_signals.join(", ") || "No signals yet")}
              ${infoCard("Guardrails", latestRollout?.guardrails.join(", ") || "No plan generated")}
              ${infoCard("Latest Execution", latestExecution ? `${latestExecution.status} at step ${latestExecution.current_step || "pending"}` : "No execution started")}
            </div>
            ${state.rolloutExecutions.length > 0 ? table(["Execution", "Status", "Decision", "Step"], state.rolloutExecutions.map((execution) => [execution.id, execution.status, execution.last_decision || "n/a", execution.current_step || "n/a"])) : emptyState("No rollout executions", "Create a rollout execution from the latest plan to begin the control loop.")}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Operate Rollout</h3>
              <p>Advance lifecycle state and record verification outcomes.</p>
            </div>
            ${canOperate && latestRollout ? `
              <form id="create-rollout-execution-form" class="stack-form">
                <input type="hidden" name="rollout_plan_id" value="${latestRollout.id}" />
                <button class="action" type="submit">Create Execution From Latest Plan</button>
              </form>
              ${latestExecution ? `
                <form id="advance-rollout-form" class="stack-form">
                  <input type="hidden" name="execution_id" value="${latestExecution.id}" />
                  <label><span>Action</span>
                    <select name="action">
                      <option value="approve">Approve</option>
                      <option value="start">Start</option>
                      <option value="pause">Pause</option>
                      <option value="continue">Continue</option>
                      <option value="complete">Complete</option>
                      <option value="rollback">Rollback</option>
                    </select>
                  </label>
                  <label><span>Reason</span><input name="reason" type="text" placeholder="Operator note" /></label>
                  <button class="action ghost" type="submit">Advance Rollout</button>
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
                      <option value="pause">Pause</option>
                      <option value="rollback">Rollback</option>
                      <option value="manual_review_required">Manual Review Required</option>
                    </select>
                  </label>
                  <label><span>Summary</span><textarea name="summary" placeholder="Signals look healthy or explain why the rollout should pause."></textarea></label>
                  <button class="action ghost" type="submit">Record Verification</button>
                </form>
              ` : ""}
            ` : emptyState("Execution access unavailable", "An org member or org admin can create executions and control rollout state from this surface.")}
          </article>
        </section>
      `;
    case "deployments":
      return detailLayout(
        "Deployment History",
        "Release, rollout, and verification events will aggregate here as the deployment model deepens.",
        [
          infoCard("Current State", "Phase 1 focuses on change, risk, and planning before full deployment orchestration."),
          infoCard("Future Expansion", "Temporal-backed workflows, staged cohorts, and rollback automation."),
          infoCard("Operator Benefit", "One surface for promotion history, verification outcomes, and audit context.")
        ]
      );
    case "incidents":
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Incident Feed</h3>
              <p>Reliability signals tied back to change history and ownership.</p>
            </div>
            ${
              state.incidents.length === 0
                ? emptyState("No incidents yet", "Phase 1 keeps the model and UI path ready while delivery governance is built out.")
                : table(
                    ["Title", "Severity", "Status"],
                    state.incidents.map((incident) => [incident.title, incident.severity, incident.status])
                  )
            }
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Change Correlation</h3>
              <p>Deployment-to-incident linking lands here next.</p>
            </div>
            ${infoCard("Latest Incident", latestIncident ? latestIncident.title : "No linked incident")}
          </article>
        </section>
      `;
    case "incident-detail":
      return detailLayout(
        "Incident Detail",
        latestIncident
          ? `${latestIncident.title} is ${latestIncident.status} at severity ${latestIncident.severity}.`
          : "Incident detail will connect timelines, ownership, runbooks, and change correlation.",
        [
          infoCard("Timeline", "Generated timeline and event correlation are staged for the next phase."),
          infoCard("Runbooks", "Runbook suggestions and execution hooks will surface here."),
          infoCard("Blast Radius", "Dependency and customer journey impact views belong on this page.")
        ]
      );
    case "policies":
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Policy Center</h3>
              <p>OPA-style evaluation patterns with explainable outcomes.</p>
            </div>
            ${table(
              ["Policy", "Scope", "Mode", "Enabled", "Description"],
              state.policies.map((policy) => [
                policy.name,
                policy.scope,
                policy.mode,
                policy.enabled ? "Yes" : "No",
                policy.description
              ])
            )}
          </article>
          <article class="surface panel">
            <div class="panel-header">
              <h3>Governance Direction</h3>
              <p>Reserved for premium packs and enterprise control mapping.</p>
            </div>
            ${infoCard("Next Layer", "Compliance packs, approval workflows, and evidence collection.")}
          </article>
        </section>
      `;
    case "audit":
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Audit Trail</h3>
              <p>Critical control-plane actions recorded with tenant and resource context.</p>
            </div>
            ${table(
              ["Action", "Resource", "ID", "Outcome", "Details"],
              state.auditEvents.map((event) => [
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
      return `
        <section class="page-grid">
          ${state.integrations
            .map(
              (integration) => `
                <article class="surface panel">
                  <div class="panel-header">
                    <h3>${integration.name}</h3>
                    <p>${integration.description}</p>
                  </div>
                  <p class="panel-stat">${integration.mode}</p>
                  <p class="panel-muted">Capabilities: ${integration.capabilities.join(", ")}</p>
                </article>
              `
            )
            .join("")}
        </section>
      `;
    case "bootstrap":
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Startup Bootstrap</h3>
              <p>Create the first project and establish the core governance shape of the workspace.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Step 1", "Create the first project for the active organization")}
              ${infoCard("Step 2", "Register teams, services, and environments")}
              ${infoCard("Step 3", "Move from planning into rollout execution")}
            </div>
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
        </section>
      `;
    case "enterprise":
      return detailLayout(
        "Enterprise Integration Mode",
        "Adopt progressively through read-only, advisory, policy, governance, and full orchestration modes.",
        [
          infoCard("Read-Only", "Ingest topology, deployments, and change metadata."),
          infoCard("Advisory", "Compute risk and recommend rollouts without blocking."),
          infoCard("Governed", "Layer in approvals, policy, and verification control.")
        ]
      );
    case "settings":
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Service Accounts and Tokens</h3>
              <p>Machine actors can authenticate with revoked-or-rotated tokens while tenant boundaries stay enforced.</p>
            </div>
            ${serviceAccounts.length > 0 ? table(["Account", "Role", "Status", "Tokens"], serviceAccounts.map((serviceAccount) => [serviceAccount.name, serviceAccount.role, serviceAccount.status, String((state.serviceAccountTokens[serviceAccount.id] || []).length)])) : emptyState("No service accounts yet", "Create a machine actor for rollout automation, graph ingestion, or controlled change execution.")}
            <div class="page-grid">
              ${serviceAccounts.map((serviceAccount) => `
                <article class="surface panel">
                  <div class="panel-header">
                    <h3>${serviceAccount.name}</h3>
                    <p>${serviceAccount.description || "Machine actor ready for scoped automation."}</p>
                  </div>
                  ${(state.serviceAccountTokens[serviceAccount.id] || []).length > 0 ? table(["Prefix", "Status", "Expires"], (state.serviceAccountTokens[serviceAccount.id] || []).map((token) => [token.token_prefix, token.status, token.expires_at || "No expiry"])) : emptyState("No tokens", "Issue a token to authenticate this service account.")}
                  ${canAdmin ? `
                    <form class="stack-form issue-token-form" data-service-account-id="${serviceAccount.id}">
                      <label><span>Token Name</span><input name="name" type="text" placeholder="${serviceAccount.name} primary" required /></label>
                      <button class="action ghost" type="submit">Issue Token</button>
                    </form>
                    ${(state.serviceAccountTokens[serviceAccount.id] || [])[0] ? `<button class="action ghost revoke-token-button" data-service-account-id="${serviceAccount.id}" data-token-id="${(state.serviceAccountTokens[serviceAccount.id] || [])[0].id}">Revoke Latest Token</button>` : ""}
                  ` : ""}
                </article>
              `).join("")}
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
      return detailLayout(
        "System Graph",
        "The digital twin of services, environments, dependencies, policies, and runtime context will expand here.",
        [
          infoCard("Topology", "Who owns what, what depends on what, and what sits in scope."),
          infoCard("Impact", "Blast radius pathing and critical path analysis."),
          infoCard("Maturity", "Missing observability, security, and ownership coverage.")
        ]
      );
    case "costs":
      return detailLayout(
        "Cost Overview",
        "Cost intelligence is staged with service-level attribution, rollout efficiency, and waste detection in mind.",
        [
          infoCard("Foundation", "Cost baselines and environment-level estimates."),
          infoCard("Future", "Idle previews, expensive rollouts, and rightsizing recommendations."),
          infoCard("Commercial Fit", "A natural premium expansion path for platform customers.")
        ]
      );
    case "simulation":
      return detailLayout(
        "Simulation Lab",
        "Dry runs, blast-radius simulation, policy violation previews, and rollout rehearsal live here next.",
        [
          infoCard("Scenario Inputs", "Change sets, topology, and policy packs."),
          infoCard("Expected Outputs", "Rollback readiness, failure domains, and business guardrails."),
          infoCard("Why It Matters", "Simulation is a key long-term differentiator for monetizable change governance.")
        ]
      );
    default:
      return `
        <section class="page-grid">
          <article class="surface panel wide">
            <div class="panel-header">
              <h3>Organization Dashboard</h3>
              <p>Cross-cutting control-plane posture across delivery, governance, and operational readiness.</p>
            </div>
            <div class="highlight-grid">
              ${infoCard("Service Coverage", `${state.metrics.services} cataloged services with governance metadata.`)}
              ${infoCard("Policy Surface", `${state.metrics.policies} active policies ready for evaluation.`)}
              ${infoCard("Integration Readiness", `${state.metrics.integrations} starter adapters available.`)}
              ${infoCard("Audit Depth", `${state.metrics.audit_events} recorded control-plane events.`)}
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

function activeOrganizationRole(state: ControlPlaneState): string {
  const activeOrganizationID = state.session.active_organization_id || "";
  const organization = (state.session.organizations || []).find((entry) => entry.organization_id === activeOrganizationID);
  return organization?.role || "";
}

function projectOptions(state: ControlPlaneState): string {
  if (state.projects.length === 0) {
    return `<option value="">Create a project first</option>`;
  }
  return state.projects.map((project) => `<option value="${project.id}">${project.name}</option>`).join("");
}

function teamOptions(state: ControlPlaneState): string {
  if (state.teams.length === 0) {
    return `<option value="">Create a team through the API or CLI first</option>`;
  }
  return state.teams.map((team) => `<option value="${team.id}">${team.name}</option>`).join("");
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
