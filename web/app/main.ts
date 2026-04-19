import "../styles/app.css";

import { getCurrentRoute, getCurrentRouteQuery } from "./router";
import { renderShell, type AuthMode } from "../components/shell";
import {
  advanceRolloutExecution,
  archiveTeam,
  archiveEnvironment,
  archiveService,
  consumeAuthRedirectQuery,
  createEnvironment,
  createIntegration,
  createIdentityProvider,
  createPolicy,
  createProject,
  createRolloutExecution,
  createSignalSnapshot,
  createService,
  createTeam,
  createServiceAccount,
  deactivateServiceAccount,
  loadAuditPageState,
  loadBootstrapPageState,
  loadCatalogPageState,
  loadChangeReviewPageState,
  loadCostsPageState,
  loadDashboardPageState,
  loadDeploymentsPageState,
  loadEnvironmentPageState,
  loadEnterprisePageState,
  loadGraphPageState,
  loadIncidentDetailPageState,
  loadIncidentsPageState,
  loadIntegrationsPageState,
  loadPoliciesPageState,
  loadRiskPageState,
  loadRolloutPageState,
  loadServicePageState,
  loadSettingsPageState,
  loadSimulationPageState,
  getStoredOrganizationID,
  issueServiceAccountToken,
  loadControlPlaneState,
  logout,
  pauseRolloutExecution,
  reconcileRolloutExecution,
  recordVerificationResult,
  requeueOutboxEvent,
  resumeRolloutExecution,
  retryOutboxEvent,
  rollbackRolloutExecution,
  revokeServiceAccountToken,
  rotateServiceAccountToken,
  sessionExpiredEventName,
  SignalValue,
  signIn,
  signUp,
  startIdentityProviderSignIn,
  type ControlPlaneState,
  setActiveOrganization,
  startGitHubOnboarding,
  syncWebhookRegistration,
  syncIntegration,
  testIntegration,
  testIdentityProvider,
  updateDiscoveredResource,
  updateEnvironment,
  updateIntegration,
  updateIdentityProvider,
  updatePolicy,
  updateService,
  updateTeam,
  updateRepository
} from "../lib/api";

const app = document.querySelector<HTMLDivElement>("#app");
const AUTH_FEEDBACK_DURATION_MS = 2000;
const APP_FEEDBACK_DURATION_MS = 4000;

if (!app) {
  throw new Error("Application root not found");
}

const root = app;
let feedback: { kind: "info" | "success" | "error"; message: string; durationMs: number | null; expiresAt: number | null } | null = null;
let feedbackTimeoutID: number | null = null;
let feedbackDurationMs = APP_FEEDBACK_DURATION_MS;
let authMode: AuthMode = "login";
type StatusDashboardQueryState = {
  search: string;
  rollbackOnly: boolean;
  serviceID: string;
  environmentID: string;
  source: string;
  eventType: string;
  automated: string;
  limit: number;
  offset: number;
};

const statusDashboardQueryState: StatusDashboardQueryState = {
  search: "",
  rollbackOnly: false,
  serviceID: "",
  environmentID: "",
  source: "",
  eventType: "",
  automated: "",
  limit: 25,
  offset: 0
};

const authRedirect = consumeAuthRedirectQuery();
if (authRedirect?.error) {
  setFeedback("error", authRedirect.error, { durationMs: AUTH_FEEDBACK_DURATION_MS });
} else if (authRedirect?.completed) {
  setFeedback("success", "Enterprise sign-in completed.", { durationMs: AUTH_FEEDBACK_DURATION_MS });
}

window.addEventListener(sessionExpiredEventName, () => {
  authMode = "login";
  setFeedback("error", "Your session expired. Sign in again.", { durationMs: AUTH_FEEDBACK_DURATION_MS });
  void render();
});

let renderRequestID = 0;

type RouteLocalKey =
  | "dashboard"
  | "bootstrap"
  | "catalog"
  | "change-review"
  | "risk"
  | "service"
  | "environment"
  | "policies"
  | "audit"
  | "rollout"
  | "deployments"
  | "graph"
  | "costs"
  | "integrations"
  | "enterprise"
  | "settings"
  | "incidents"
  | "incident-detail"
  | "simulation";

async function render() {
  const requestID = ++renderRequestID;
  const route = getCurrentRoute();
  const routeQuery = getCurrentRouteQuery();
  const incidentID = route.key === "incident-detail" ? (routeQuery.get("id") || "").trim() : "";
  const statusQuery = buildStatusDashboardQuery();

  const shellState = await loadControlPlaneState();
  if (requestID !== renderRequestID) {
    return;
  }
  applyRuntimeState(shellState);
  if (!shouldLoadRouteData(shellState) || !isRouteLocalRoute(route.key)) {
    applyRenderedState(route, shellState);
    return;
  }

  applyRenderedState(route, withRoutePageState(shellState, route.key, { status: "loading", data: null, error: "" }));

  try {
    const state = await loadRouteLocalState(shellState, route.key, {
      statusQuery,
      incidentID
    });
    if (requestID !== renderRequestID) {
      return;
    }
    applyRenderedState(route, state);
  } catch (error) {
    if (requestID !== renderRequestID) {
      return;
    }
    applyRenderedState(
      route,
      withRoutePageState(shellState, route.key, {
        status: "error",
        data: null,
        error: error instanceof Error ? error.message : `${route.label} data failed to load.`
      })
    );
  }
}

function applyRuntimeState(state: ControlPlaneState) {
  feedbackDurationMs = state.session.authenticated ? APP_FEEDBACK_DURATION_MS : AUTH_FEEDBACK_DURATION_MS;
  if (state.session.authenticated) {
    authMode = "login";
  }
}

function shouldLoadRouteData(state: ControlPlaneState): boolean {
  return state.connected && state.session.authenticated && Boolean(state.session.active_organization_id);
}

async function loadRouteLocalState(
  state: ControlPlaneState,
  routeKey: RouteLocalKey,
  options: { statusQuery: string; incidentID: string }
): Promise<ControlPlaneState> {
  switch (routeKey) {
    case "dashboard":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadDashboardPageState(),
        error: ""
      });
    case "bootstrap":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadBootstrapPageState(),
        error: ""
      });
    case "catalog":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadCatalogPageState(),
        error: ""
      });
    case "change-review":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadChangeReviewPageState(),
        error: ""
      });
    case "risk":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadRiskPageState(),
        error: ""
      });
    case "service":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadServicePageState(),
        error: ""
      });
    case "environment":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadEnvironmentPageState(),
        error: ""
      });
    case "policies":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadPoliciesPageState(),
        error: ""
      });
    case "audit":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadAuditPageState(),
        error: ""
      });
    case "rollout":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadRolloutPageState(),
        error: ""
      });
    case "deployments":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadDeploymentsPageState({ statusQuery: options.statusQuery }),
        error: ""
      });
    case "graph":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadGraphPageState(),
        error: ""
      });
    case "costs":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadCostsPageState(),
        error: ""
      });
    case "integrations":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadIntegrationsPageState(),
        error: ""
      });
    case "enterprise":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadEnterprisePageState(),
        error: ""
      });
    case "settings":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadSettingsPageState(),
        error: ""
      });
    case "incidents":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadIncidentsPageState(),
        error: ""
      });
    case "incident-detail":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadIncidentDetailPageState(options.incidentID),
        error: ""
      });
    case "simulation":
      return withRoutePageState(state, routeKey, {
        status: "ready",
        data: await loadSimulationPageState(),
        error: ""
      });
  }
  throw new Error(`Unsupported route-local state request for ${routeKey}.`);
}

function withRoutePageState(state: ControlPlaneState, routeKey: RouteLocalKey, pageState: { status: "idle" | "loading" | "ready" | "error"; data: any; error: string }): ControlPlaneState {
  switch (routeKey) {
    case "dashboard":
      return {
        ...state,
        dashboardPage: pageState
      };
    case "bootstrap":
      return {
        ...state,
        bootstrapPage: pageState
      };
    case "catalog":
      return {
        ...state,
        catalogPage: pageState
      };
    case "change-review":
      return {
        ...state,
        changeReviewPage: pageState
      };
    case "risk":
      return {
        ...state,
        riskPage: pageState
      };
    case "service":
      return {
        ...state,
        servicePage: pageState
      };
    case "environment":
      return {
        ...state,
        environmentPage: pageState
      };
    case "policies":
      return {
        ...state,
        policiesPage: pageState
      };
    case "audit":
      return {
        ...state,
        auditPage: pageState
      };
    case "rollout":
      return {
        ...state,
        rolloutPage: pageState
      };
    case "deployments":
      return {
        ...state,
        deploymentsPage: pageState
      };
    case "graph":
      return {
        ...state,
        graphPage: pageState
      };
    case "costs":
      return {
        ...state,
        costsPage: pageState
      };
    case "integrations":
      return {
        ...state,
        integrationsPage: pageState
      };
    case "enterprise":
      return {
        ...state,
        enterprisePage: pageState
      };
    case "settings":
      return {
        ...state,
        settingsPage: pageState
      };
    case "incidents":
      return {
        ...state,
        incidentsPage: pageState
      };
    case "incident-detail":
      return {
        ...state,
        incidentDetailPage: pageState
      };
    case "simulation":
      return {
        ...state,
        simulationPage: pageState
      };
  }
  return state;
}

function isRouteLocalRoute(routeKey: string): routeKey is RouteLocalKey {
  return routeKey === "dashboard"
    || routeKey === "bootstrap"
    || routeKey === "catalog"
    || routeKey === "change-review"
    || routeKey === "risk"
    || routeKey === "service"
    || routeKey === "environment"
    || routeKey === "policies"
    || routeKey === "audit"
    || routeKey === "rollout"
    || routeKey === "deployments"
    || routeKey === "graph"
    || routeKey === "costs"
    || routeKey === "integrations"
    || routeKey === "enterprise"
    || routeKey === "settings"
    || routeKey === "incidents"
    || routeKey === "incident-detail"
    || routeKey === "simulation";
}

function applyRenderedState(route: ReturnType<typeof getCurrentRoute>, state: ControlPlaneState) {
  root.innerHTML = renderShell(state, route, authMode);
  applyFeedback();

  document.querySelectorAll<HTMLButtonElement>("[data-auth-mode]").forEach((button) => {
    button.addEventListener("click", () => {
      const nextMode = button.dataset.authMode;
      if (!isAuthMode(nextMode) || nextMode === authMode) {
        return;
      }
      authMode = nextMode;
      clearAuthErrorFeedback();
      void render();
    });
  });

  document.querySelectorAll<HTMLButtonElement>("[data-password-toggle]").forEach((button) => {
    button.addEventListener("click", () => {
      const targetID = button.dataset.passwordToggle;
      if (!targetID) {
        return;
      }
      const input = document.getElementById(targetID);
      if (!(input instanceof HTMLInputElement)) {
        return;
      }
      const visible = input.type === "text";
      input.type = visible ? "password" : "text";
      button.setAttribute("aria-pressed", visible ? "false" : "true");
      button.setAttribute("aria-label", visible ? "Show password" : "Hide password");
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".identity-provider-login-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const providerID = button.dataset.providerId || "";
      await runAction(button, "Redirecting to enterprise sign-in...", null, async () => {
        const result = await startIdentityProviderSignIn(providerID, {
          return_to: window.location.origin + window.location.pathname + window.location.hash
        });
        window.location.assign(result.authorize_url);
      });
    });
  });

  const loginForm = document.querySelector<HTMLFormElement>("#login-form");
  clearAuthErrorOnEdit(loginForm);
  loginForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const email = readInputValue(loginForm, "email");
    const password = readRawInputValue(loginForm, "password");
    const validationError = validateLogin({ email, password });
    if (validationError) {
      setFeedback("error", validationError);
      return;
    }
    await runAction(submitButton(loginForm), "Signing in to the control plane...", null, async () => {
      try {
        await signIn({
          email,
          password
        });
      } catch (error) {
        throw normalizeAuthError(error, "login");
      }
    });
  });

  const signupForm = document.querySelector<HTMLFormElement>("#signup-form");
  clearAuthErrorOnEdit(signupForm);
  signupForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const email = readInputValue(signupForm, "email");
    const displayName = readInputValue(signupForm, "display_name");
    const password = readRawInputValue(signupForm, "password");
    const passwordConfirmation = readRawInputValue(signupForm, "password_confirmation");
    const validationError = validateSignup({
      email,
      displayName,
      password,
      passwordConfirmation
    });
    if (validationError) {
      setFeedback("error", validationError);
      return;
    }
    await runAction(submitButton(signupForm), "Creating your control-plane account...", null, async () => {
      try {
        await signUp({
          email,
          display_name: displayName,
          password,
          password_confirmation: passwordConfirmation
        });
      } catch (error) {
        throw normalizeAuthError(error, "signup");
      }
    });
  });

  const organizationSwitcher = document.querySelector<HTMLSelectElement>("#organization-switcher");
  organizationSwitcher?.addEventListener("change", async (event) => {
    const target = event.target as HTMLSelectElement;
    setActiveOrganization(target.value);
    await render();
  });

  const refreshButton = document.querySelector<HTMLButtonElement>("#refresh-button");
  refreshButton?.addEventListener("click", async () => {
    await runAction(refreshButton, "Refreshing control-plane data...", "Control-plane data refreshed.", async () => {
      if (!getStoredOrganizationID() && state.session.active_organization_id) {
        setActiveOrganization(state.session.active_organization_id);
      }
    });
  });

  const logoutButton = document.querySelector<HTMLButtonElement>("#logout-button");
  logoutButton?.addEventListener("click", async () => {
    await runAction(logoutButton, "Clearing session...", "Signed out.", async () => {
      await logout();
      authMode = "login";
    });
  });

  document.querySelectorAll<HTMLFormElement>(".integration-config-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const integrationID = form.dataset.integrationId || "";
      await runAction(submitButton(form), "Saving integration settings...", "Integration settings saved.", async () => {
      await updateIntegration(integrationID, {
        name: readInputValue(form, "name"),
        instance_key: readInputValue(form, "instance_key"),
        scope_type: readInputValue(form, "scope_type"),
        scope_name: readInputValue(form, "scope_name"),
        mode: readInputValue(form, "mode"),
        auth_strategy: readInputValue(form, "auth_strategy"),
        enabled: readCheckboxValue(form, "enabled"),
        control_enabled: readCheckboxValue(form, "control_enabled"),
        schedule_enabled: readCheckboxValue(form, "schedule_enabled"),
        schedule_interval_seconds: readOptionalPositiveNumber(form, "schedule_interval_seconds"),
        sync_stale_after_seconds: readOptionalPositiveNumber(form, "sync_stale_after_seconds"),
        metadata: buildIntegrationMetadata(form)
      });
    });
  });
  });

  document.querySelectorAll<HTMLButtonElement>(".integration-test-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const integrationID = button.dataset.integrationId || "";
      await runAction(button, "Testing integration connection...", "Connection test completed. No deployment action was executed.", async () => {
        await testIntegration(integrationID);
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".integration-sync-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const integrationID = button.dataset.integrationId || "";
      await runAction(button, "Running integration sync...", "Integration sync completed in read-only observation mode.", async () => {
        await syncIntegration(integrationID);
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".integration-webhook-sync-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const integrationID = button.dataset.integrationId || "";
      await runAction(button, "Registering webhook automatically...", "Webhook registration checked and updated.", async () => {
        await syncWebhookRegistration(integrationID);
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".github-onboarding-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const integrationID = button.dataset.integrationId || "";
      await runAction(button, "Preparing GitHub App install...", "GitHub App onboarding URL generated.", async () => {
        const result = await startGitHubOnboarding(integrationID);
        window.open(result.authorize_url, "_blank", "noopener,noreferrer");
      });
    });
  });

  const createIdentityProviderForm = document.querySelector<HTMLFormElement>("#create-identity-provider-form");
  createIdentityProviderForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createIdentityProviderForm), "Creating enterprise identity provider...", "Enterprise identity provider created.", async () => {
      await createIdentityProvider({
        organization_id: state.session.active_organization_id || "",
        name: readInputValue(createIdentityProviderForm, "name"),
        kind: readInputValue(createIdentityProviderForm, "kind"),
        issuer_url: readInputValue(createIdentityProviderForm, "issuer_url") || undefined,
        authorization_endpoint: readInputValue(createIdentityProviderForm, "authorization_endpoint") || undefined,
        token_endpoint: readInputValue(createIdentityProviderForm, "token_endpoint") || undefined,
        userinfo_endpoint: readInputValue(createIdentityProviderForm, "userinfo_endpoint") || undefined,
        client_id: readInputValue(createIdentityProviderForm, "client_id") || undefined,
        client_secret_env: readInputValue(createIdentityProviderForm, "client_secret_env") || undefined,
        allowed_domains: splitCommaValues(readInputValue(createIdentityProviderForm, "allowed_domains")),
        default_role: readInputValue(createIdentityProviderForm, "default_role") || undefined,
        enabled: readCheckboxValue(createIdentityProviderForm, "enabled")
      });
    });
  });

  document.querySelectorAll<HTMLFormElement>(".identity-provider-config-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const providerID = form.dataset.providerId || "";
      await runAction(submitButton(form), "Saving enterprise identity settings...", "Enterprise identity settings saved.", async () => {
        await updateIdentityProvider(providerID, {
          name: readInputValue(form, "name") || undefined,
          issuer_url: readInputValue(form, "issuer_url") || undefined,
          authorization_endpoint: readInputValue(form, "authorization_endpoint") || undefined,
          token_endpoint: readInputValue(form, "token_endpoint") || undefined,
          userinfo_endpoint: readInputValue(form, "userinfo_endpoint") || undefined,
          client_id: readInputValue(form, "client_id") || undefined,
          client_secret_env: readInputValue(form, "client_secret_env") || undefined,
          allowed_domains: splitCommaValues(readInputValue(form, "allowed_domains")),
          default_role: readInputValue(form, "default_role") || undefined,
          enabled: readCheckboxValue(form, "enabled")
        });
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".identity-provider-test-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const providerID = button.dataset.providerId || "";
      await runAction(button, "Testing enterprise identity provider...", "Identity provider test completed.", async () => {
        await testIdentityProvider(providerID);
      });
    });
  });

  const createPolicyForm = document.querySelector<HTMLFormElement>("#create-policy-form");
  createPolicyForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createPolicyForm), "Creating governance policy...", "Policy created.", async () => {
      await createPolicy({
        organization_id: state.session.active_organization_id || "",
        project_id: readInputValue(createPolicyForm, "project_id") || undefined,
        service_id: readInputValue(createPolicyForm, "service_id") || undefined,
        environment_id: readInputValue(createPolicyForm, "environment_id") || undefined,
        name: readInputValue(createPolicyForm, "name"),
        code: readInputValue(createPolicyForm, "code") || undefined,
        applies_to: readInputValue(createPolicyForm, "applies_to"),
        mode: readInputValue(createPolicyForm, "mode"),
        priority: readOptionalInteger(createPolicyForm, "priority") ?? 0,
        description: readInputValue(createPolicyForm, "description") || undefined,
        enabled: readCheckboxValue(createPolicyForm, "enabled"),
        conditions: buildPolicyConditions(createPolicyForm)
      });
    });
  });

  document.querySelectorAll<HTMLFormElement>(".policy-config-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const policyID = form.dataset.policyId || "";
      await runAction(submitButton(form), "Saving governance policy...", "Policy updated.", async () => {
        await updatePolicy(policyID, {
          project_id: readInputValue(form, "project_id") || undefined,
          service_id: readInputValue(form, "service_id") || undefined,
          environment_id: readInputValue(form, "environment_id") || undefined,
          name: readInputValue(form, "name") || undefined,
          code: readInputValue(form, "code") || undefined,
          applies_to: readInputValue(form, "applies_to") || undefined,
          mode: readInputValue(form, "mode") || undefined,
          priority: readOptionalInteger(form, "priority"),
          description: readInputValue(form, "description") || undefined,
          enabled: readCheckboxValue(form, "enabled"),
          conditions: buildPolicyConditions(form)
        });
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".policy-toggle-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const policyID = button.dataset.policyId || "";
      const enabled = button.dataset.policyEnabled !== "true";
      await runAction(
        button,
        `${enabled ? "Enabling" : "Disabling"} governance policy...`,
        `Policy ${enabled ? "enabled" : "disabled"}.`,
        async () => {
          await updatePolicy(policyID, { enabled });
        }
      );
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".outbox-retry-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const outboxEventID = button.dataset.outboxEventId || "";
      await runAction(button, "Scheduling an immediate durable-event retry...", "Outbox event marked pending for immediate retry.", async () => {
        await retryOutboxEvent(outboxEventID);
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".outbox-requeue-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const outboxEventID = button.dataset.outboxEventId || "";
      await runAction(button, "Requeueing the dead-lettered event...", "Outbox event requeued for another dispatch attempt.", async () => {
        await requeueOutboxEvent(outboxEventID);
      });
    });
  });

  document.querySelectorAll<HTMLFormElement>(".repository-map-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const repositoryID = form.dataset.repositoryId || "";
      await runAction(submitButton(form), "Saving repository mapping...", "Repository mapping saved.", async () => {
        await updateRepository(repositoryID, {
          service_id: readInputValue(form, "service_id") || undefined,
          environment_id: readInputValue(form, "environment_id") || undefined,
          status: "mapped"
        });
      });
    });
  });

  document.querySelectorAll<HTMLFormElement>(".discovered-resource-map-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const discoveredResourceID = form.dataset.discoveredResourceId || "";
      await runAction(submitButton(form), "Saving discovered resource mapping...", "Discovered resource mapping saved.", async () => {
        await updateDiscoveredResource(discoveredResourceID, {
          service_id: readInputValue(form, "service_id") || undefined,
          environment_id: readInputValue(form, "environment_id") || undefined,
          repository_id: readInputValue(form, "repository_id") || undefined,
          status: readInputValue(form, "status") || undefined
        });
      });
    });
  });

  const createProjectForm = document.querySelector<HTMLFormElement>("#create-project-form");
  createProjectForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createProjectForm), "Creating project...", "Project created.", async () => {
      await createProject({
        organization_id: state.session.active_organization_id || "",
        name: readInputValue(createProjectForm, "name"),
        slug: readInputValue(createProjectForm, "slug"),
        description: readInputValue(createProjectForm, "description")
      });
    });
  });

  const createTeamForm = document.querySelector<HTMLFormElement>("#create-team-form");
  createTeamForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createTeamForm), "Creating team...", "Team created.", async () => {
      await createTeam({
        organization_id: state.session.active_organization_id || "",
        project_id: readInputValue(createTeamForm, "project_id"),
        name: readInputValue(createTeamForm, "name"),
        slug: readInputValue(createTeamForm, "slug")
      });
    });
  });

  const updateTeamForm = document.querySelector<HTMLFormElement>("#update-team-form");
  updateTeamForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const teamID = updateTeamForm.dataset.teamId || "";
    await runAction(submitButton(updateTeamForm), "Saving current team...", "Team updated.", async () => {
      await updateTeam(teamID, {
        name: readInputValue(updateTeamForm, "name") || undefined,
        slug: readInputValue(updateTeamForm, "slug") || undefined
      });
    });
  });

  const archiveTeamButton = document.querySelector<HTMLButtonElement>("#archive-team-button");
  archiveTeamButton?.addEventListener("click", async () => {
    const teamID = archiveTeamButton.dataset.teamId || "";
    if (!teamID) {
      return;
    }
    await runAction(archiveTeamButton, "Archiving team...", "Team archived.", async () => {
      await archiveTeam(teamID);
    });
  });

  const createIntegrationForm = document.querySelector<HTMLFormElement>("#create-integration-form");
  syncCreateIntegrationForm(createIntegrationForm);
  const createIntegrationKindField = createIntegrationForm?.elements.namedItem("kind");
  if (createIntegrationKindField instanceof HTMLSelectElement) {
    createIntegrationKindField.addEventListener("change", () => {
      syncCreateIntegrationForm(createIntegrationForm);
    });
  }
  createIntegrationForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createIntegrationForm), "Creating integration instance...", "Integration instance created.", async () => {
      await createIntegration({
        organization_id: state.session.active_organization_id || "",
        kind: readInputValue(createIntegrationForm, "kind"),
        name: readInputValue(createIntegrationForm, "name"),
        instance_key: readInputValue(createIntegrationForm, "instance_key") || undefined,
        scope_type: readInputValue(createIntegrationForm, "scope_type") || undefined,
        scope_name: readInputValue(createIntegrationForm, "scope_name") || undefined,
        auth_strategy: readInputValue(createIntegrationForm, "auth_strategy") || undefined
      });
    });
  });

  const createServiceForm = document.querySelector<HTMLFormElement>("#create-service-form");
  createServiceForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createServiceForm), "Registering service...", "Service created.", async () => {
      await createService({
        organization_id: state.session.active_organization_id || "",
        project_id: readInputValue(createServiceForm, "project_id"),
        team_id: readInputValue(createServiceForm, "team_id"),
        name: readInputValue(createServiceForm, "name"),
        slug: readInputValue(createServiceForm, "slug"),
        description: readInputValue(createServiceForm, "description"),
        criticality: readInputValue(createServiceForm, "criticality")
      });
    });
  });

  const archiveServiceButton = document.querySelector<HTMLButtonElement>("#archive-service-button");
  archiveServiceButton?.addEventListener("click", async () => {
    const serviceID = archiveServiceButton.dataset.serviceId || "";
    if (!serviceID) {
      return;
    }
    await runAction(archiveServiceButton, "Archiving service...", "Service archived.", async () => {
      await archiveService(serviceID);
    });
  });

  const updateServiceForm = document.querySelector<HTMLFormElement>("#update-service-form");
  updateServiceForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const serviceID = updateServiceForm.dataset.serviceId || "";
    await runAction(submitButton(updateServiceForm), "Saving current service...", "Service updated.", async () => {
      await updateService(serviceID, {
        name: readInputValue(updateServiceForm, "name") || undefined,
        slug: readInputValue(updateServiceForm, "slug") || undefined,
        criticality: readInputValue(updateServiceForm, "criticality") || undefined,
        description: readInputValue(updateServiceForm, "description") || undefined
      });
    });
  });

  const createEnvironmentForm = document.querySelector<HTMLFormElement>("#create-environment-form");
  createEnvironmentForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createEnvironmentForm), "Creating environment...", "Environment created.", async () => {
      await createEnvironment({
        organization_id: state.session.active_organization_id || "",
        project_id: readInputValue(createEnvironmentForm, "project_id"),
        name: readInputValue(createEnvironmentForm, "name"),
        slug: readInputValue(createEnvironmentForm, "slug"),
        type: readInputValue(createEnvironmentForm, "type"),
        region: readInputValue(createEnvironmentForm, "region"),
        production: readCheckboxValue(createEnvironmentForm, "production")
      });
    });
  });

  const archiveEnvironmentButton = document.querySelector<HTMLButtonElement>("#archive-environment-button");
  archiveEnvironmentButton?.addEventListener("click", async () => {
    const environmentID = archiveEnvironmentButton.dataset.environmentId || "";
    if (!environmentID) {
      return;
    }
    await runAction(archiveEnvironmentButton, "Archiving environment...", "Environment archived.", async () => {
      await archiveEnvironment(environmentID);
    });
  });

  const updateEnvironmentForm = document.querySelector<HTMLFormElement>("#update-environment-form");
  updateEnvironmentForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const environmentID = updateEnvironmentForm.dataset.environmentId || "";
    await runAction(submitButton(updateEnvironmentForm), "Saving current environment...", "Environment updated.", async () => {
      await updateEnvironment(environmentID, {
        name: readInputValue(updateEnvironmentForm, "name") || undefined,
        slug: readInputValue(updateEnvironmentForm, "slug") || undefined,
        type: readInputValue(updateEnvironmentForm, "type") || undefined,
        region: readInputValue(updateEnvironmentForm, "region") || undefined,
        compliance_zone: readInputValue(updateEnvironmentForm, "compliance_zone") || undefined,
        production: readCheckboxValue(updateEnvironmentForm, "production")
      });
    });
  });

  const createServiceAccountForm = document.querySelector<HTMLFormElement>("#create-service-account-form");
  createServiceAccountForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createServiceAccountForm), "Creating service account...", "Service account created.", async () => {
      await createServiceAccount({
        organization_id: state.session.active_organization_id || "",
        name: readInputValue(createServiceAccountForm, "name"),
        description: readInputValue(createServiceAccountForm, "description"),
        role: readInputValue(createServiceAccountForm, "role")
      });
    });
  });

  document.querySelectorAll<HTMLFormElement>(".issue-token-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const serviceAccountID = form.dataset.serviceAccountId || "";
      await runAction(submitButton(form), "Issuing service-account token...", "Service-account token issued.", async () => {
        const result = await issueServiceAccountToken(serviceAccountID, {
          name: readInputValue(form, "name")
        });
        window.alert(`Copy this token now:\n\n${result.token}`);
      });
    });
  });

  document.querySelectorAll<HTMLFormElement>(".rotate-token-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const serviceAccountID = form.dataset.serviceAccountId || "";
      const tokenID = form.dataset.tokenId || "";
      await runAction(submitButton(form), "Rotating service-account token...", "Service-account token rotated.", async () => {
        const result = await rotateServiceAccountToken(serviceAccountID, tokenID, {
          name: readInputValue(form, "name") || undefined,
          expires_in_hours: readOptionalPositiveNumber(form, "expires_in_hours")
        });
        window.alert(`Copy this rotated token now:\n\n${result.token}`);
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".revoke-token-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const serviceAccountID = button.dataset.serviceAccountId || "";
      const tokenID = button.dataset.tokenId || "";
      await runAction(button, "Revoking token...", "Token revoked.", async () => {
        await revokeServiceAccountToken(serviceAccountID, tokenID);
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".deactivate-service-account-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const serviceAccountID = button.dataset.serviceAccountId || "";
      await runAction(button, "Deactivating service account...", "Service account deactivated.", async () => {
        await deactivateServiceAccount(serviceAccountID);
      });
    });
  });

  const createRolloutExecutionForm = document.querySelector<HTMLFormElement>("#create-rollout-execution-form");
  createRolloutExecutionForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createRolloutExecutionForm), "Creating rollout execution...", "Rollout execution created.", async () => {
      await createRolloutExecution({
        rollout_plan_id: readInputValue(createRolloutExecutionForm, "rollout_plan_id"),
        backend_type: readInputValue(createRolloutExecutionForm, "backend_type"),
        signal_provider_type: readInputValue(createRolloutExecutionForm, "signal_provider_type")
      });
    });
  });

  const advanceRolloutForm = document.querySelector<HTMLFormElement>("#advance-rollout-form");
  advanceRolloutForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const executionID = readInputValue(advanceRolloutForm, "execution_id");
    const action = readInputValue(advanceRolloutForm, "action");
    const reason = readInputValue(advanceRolloutForm, "reason");
    if (isManualControlBlocked(state, executionID, action)) {
      setFeedback("error", `Advisory mode blocks manual ${action}. Reconcile can observe the live backend and record recommendations, but it will not execute external deployment actions.`);
      return;
    }
    const actionMessages = rolloutActionMessages(action);
    await runAction(submitButton(advanceRolloutForm), actionMessages.pending, actionMessages.success, async () => {
      switch (action) {
        case "pause":
          await pauseRolloutExecution(executionID, { reason });
          return;
        case "resume":
          await resumeRolloutExecution(executionID, { reason });
          return;
        case "rollback":
          await rollbackRolloutExecution(executionID, { reason });
          return;
        default:
          await advanceRolloutExecution(executionID, { action, reason });
      }
    });
  });

  const reconcileRolloutForm = document.querySelector<HTMLFormElement>("#reconcile-rollout-form");
  reconcileRolloutForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(
      submitButton(reconcileRolloutForm),
      "Reconciling rollout execution...",
      latestExecutionIsAdvisory(state)
        ? "Advisory reconcile completed. Recommendations may have been recorded without executing external deployment actions."
        : "Rollout execution reconciled.",
      async () => {
      await reconcileRolloutExecution(readInputValue(reconcileRolloutForm, "execution_id"));
      }
    );
  });

  const createSignalSnapshotForm = document.querySelector<HTMLFormElement>("#create-signal-snapshot-form");
  createSignalSnapshotForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(submitButton(createSignalSnapshotForm), "Ingesting signal snapshot...", "Signal snapshot ingested.", async () => {
      const health = readInputValue(createSignalSnapshotForm, "health");
      const signals: SignalValue[] = [
        {
          name: "latency_p95_ms",
          category: "technical",
          value: readNumberValue(createSignalSnapshotForm, "latency_value"),
          unit: "ms",
          status: health,
          threshold: 250,
          comparator: ">"
        },
        {
          name: "error_rate",
          category: "technical",
          value: readNumberValue(createSignalSnapshotForm, "error_rate_value"),
          unit: "%",
          status: health,
          threshold: 1,
          comparator: ">"
        }
      ];
      const businessValue = readNumberValue(createSignalSnapshotForm, "business_value");
      if (businessValue > 0) {
        signals.push({
          name: "business_kpi",
          category: "business",
          value: businessValue,
          status: health
        });
      }
      await createSignalSnapshot(readInputValue(createSignalSnapshotForm, "execution_id"), {
        provider_type: "simulated",
        health,
        summary: readInputValue(createSignalSnapshotForm, "summary"),
        signals
      });
    });
  });

  const recordVerificationForm = document.querySelector<HTMLFormElement>("#record-verification-form");
  recordVerificationForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runAction(
      submitButton(recordVerificationForm),
      "Recording verification result...",
      latestExecutionIsAdvisory(state) ? "Advisory recommendation recorded. No deployment action was executed." : "Verification result recorded.",
      async () => {
      await recordVerificationResult(readInputValue(recordVerificationForm, "execution_id"), {
        outcome: readInputValue(recordVerificationForm, "outcome"),
        decision: readInputValue(recordVerificationForm, "decision"),
        summary: readInputValue(recordVerificationForm, "summary")
      });
      }
    );
  });

  const statusSearchForm = document.querySelector<HTMLFormElement>("#status-search-form");
  statusSearchForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    syncStatusDashboardQuery(statusSearchForm, true);
    await render();
  });

  document.querySelectorAll<HTMLButtonElement>("[data-status-offset]").forEach((button) => {
    button.addEventListener("click", async () => {
      const nextOffset = Number(button.dataset.statusOffset || "0");
      statusDashboardQueryState.offset = Math.max(0, nextOffset);
      await render();
    });
  });

  document.querySelectorAll<HTMLInputElement | HTMLSelectElement>("[data-status-auto-submit]").forEach((field) => {
    field.addEventListener("change", async () => {
      const form = field.closest<HTMLFormElement>("#status-search-form");
      if (!form) {
        return;
      }
      syncStatusDashboardQuery(form, true);
      await render();
    });
  });

  const statusSearchReset = document.querySelector<HTMLButtonElement>("#status-search-reset");
  statusSearchReset?.addEventListener("click", async () => {
    resetStatusDashboardQuery();
    await render();
  });
}

window.addEventListener("hashchange", () => {
  void render();
});

void render();

function applyFeedback() {
  const feedbackNode = document.querySelector<HTMLDivElement>("#app-feedback");
  if (!feedbackNode) {
    return;
  }
  if (!feedback) {
    clearFeedbackTimer();
    feedbackNode.hidden = true;
    feedbackNode.textContent = "";
    feedbackNode.removeAttribute("data-kind");
    return;
  }
  if (feedback.durationMs !== null && feedback.expiresAt === null) {
    feedback.expiresAt = Date.now() + feedback.durationMs;
  }
  if (feedback.expiresAt !== null && feedback.expiresAt <= Date.now()) {
    clearFeedback();
    return;
  }
  feedbackNode.hidden = false;
  feedbackNode.dataset.kind = feedback.kind;
  feedbackNode.textContent = feedback.message;
  scheduleFeedbackClear();
}

function setFeedback(
  kind: "info" | "success" | "error",
  message: string,
  options: { durationMs?: number; persistent?: boolean } = {}
) {
  clearFeedbackTimer();
  feedback = {
    kind,
    message,
    durationMs: options.persistent ? null : options.durationMs ?? feedbackDurationMs,
    expiresAt: null
  };
  applyFeedback();
}

function clearFeedback() {
  clearFeedbackTimer();
  feedback = null;
  applyFeedback();
}

function clearAuthErrorFeedback() {
  if (feedback?.kind !== "error") {
    return;
  }
  clearFeedback();
}

async function runAction(control: HTMLButtonElement | null, pendingMessage: string, successMessage: string | null, action: () => Promise<void>) {
  setFeedback("info", pendingMessage, { persistent: true });
  if (control) {
    control.disabled = true;
  }
  try {
    await action();
    await render();
    if (successMessage) {
      setFeedback("success", successMessage);
    } else {
      clearFeedback();
    }
  } catch (error) {
    setFeedback("error", error instanceof Error ? error.message : "Action failed");
  } finally {
    if (control) {
      control.disabled = false;
    }
  }
}

function scheduleFeedbackClear() {
  clearFeedbackTimer();
  if (!feedback || feedback.expiresAt === null) {
    return;
  }
  const remaining = feedback.expiresAt - Date.now();
  if (remaining <= 0) {
    clearFeedback();
    return;
  }
  feedbackTimeoutID = window.setTimeout(() => {
    clearFeedback();
  }, remaining);
}

function clearFeedbackTimer() {
  if (feedbackTimeoutID === null) {
    return;
  }
  window.clearTimeout(feedbackTimeoutID);
  feedbackTimeoutID = null;
}

function submitButton(form: HTMLFormElement | null): HTMLButtonElement | null {
  return form?.querySelector('button[type="submit"]') ?? null;
}

function clearAuthErrorOnEdit(form: HTMLFormElement | null) {
  if (!form) {
    return;
  }
  const clear = () => {
    clearAuthErrorFeedback();
  };
  form.addEventListener("input", clear);
  form.addEventListener("change", clear);
}

function isAuthMode(value: string | undefined): value is AuthMode {
  return value === "login" || value === "signup";
}

function readInputValue(form: HTMLFormElement, name: string): string {
  const field = form.elements.namedItem(name);
  if (field instanceof HTMLInputElement || field instanceof HTMLSelectElement || field instanceof HTMLTextAreaElement) {
    return field.value.trim();
  }
  return "";
}

function readRawInputValue(form: HTMLFormElement, name: string): string {
  const field = form.elements.namedItem(name);
  if (field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement) {
    return field.value;
  }
  return "";
}

function readCheckboxValue(form: HTMLFormElement, name: string): boolean {
  const field = form.elements.namedItem(name);
  return field instanceof HTMLInputElement && field.checked;
}

function readNumberValue(form: HTMLFormElement, name: string): number {
  const field = form.elements.namedItem(name);
  if (field instanceof HTMLInputElement) {
    const value = Number(field.value);
    if (!Number.isNaN(value)) {
      return value;
    }
  }
  return 0;
}

function readOptionalPositiveNumber(form: HTMLFormElement, name: string): number | undefined {
  const field = form.elements.namedItem(name);
  if (!(field instanceof HTMLInputElement || field instanceof HTMLSelectElement)) {
    return undefined;
  }
  const trimmed = field.value.trim();
  if (trimmed === "") {
    return undefined;
  }
  const value = Number(trimmed);
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error(`Enter a valid positive number for ${name.replaceAll("_", " ")}.`);
  }
  return Math.round(value);
}

function readOptionalInteger(form: HTMLFormElement, name: string): number | undefined {
  const field = form.elements.namedItem(name);
  if (!(field instanceof HTMLInputElement || field instanceof HTMLSelectElement)) {
    return undefined;
  }
  const trimmed = field.value.trim();
  if (trimmed === "") {
    return undefined;
  }
  const value = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(value)) {
    throw new Error(`Enter a valid integer for ${name.replaceAll("_", " ")}.`);
  }
  return value;
}

function splitCommaValues(value: string): string[] | undefined {
  const items = value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  return items.length > 0 ? items : undefined;
}

function buildPolicyConditions(form: HTMLFormElement): {
  min_risk_level?: string;
  production_only?: boolean;
  regulated_only?: boolean;
  required_change_types?: string[];
  required_touches?: string[];
  missing_capabilities?: string[];
} {
  const minRiskLevel = readInputValue(form, "min_risk_level") || undefined;
  const productionOnly = readCheckboxValue(form, "production_only");
  const regulatedOnly = readCheckboxValue(form, "regulated_only");
  const requiredChangeTypes = splitCommaValues(readInputValue(form, "required_change_types"));
  const requiredTouches = splitCommaValues(readInputValue(form, "required_touches"));
  const missingCapabilities = splitCommaValues(readInputValue(form, "missing_capabilities"));

  return {
    min_risk_level: minRiskLevel,
    production_only: productionOnly,
    regulated_only: regulatedOnly,
    required_change_types: requiredChangeTypes,
    required_touches: requiredTouches,
    missing_capabilities: missingCapabilities
  };
}

function buildIntegrationMetadata(form: HTMLFormElement): Record<string, unknown> {
  const metadata: Record<string, unknown> = {};
  const assign = (name: string) => {
    const value = readInputValue(form, name);
    if (value) {
      metadata[name] = value;
    }
  };
  assign("api_base_url");
  assign("owner");
  assign("group");
  assign("web_base_url");
  assign("access_token_env");
  assign("app_id");
  assign("app_slug");
  assign("private_key_env");
  assign("installation_id");
  assign("webhook_secret_env");
  assign("inventory_path");
  assign("status_path");
  assign("namespace");
  assign("deployment_name");
  assign("query_path");
  assign("window_seconds");
  assign("step_seconds");
  assign("bearer_token_env");
  const queries = readRawInputValue(form, "queries").trim();
  if (queries) {
    metadata.queries = JSON.parse(queries);
  }
  return metadata;
}

function syncCreateIntegrationForm(form: HTMLFormElement | null) {
  if (!form) {
    return;
  }
  const kindField = form.elements.namedItem("kind");
  const authField = form.elements.namedItem("auth_strategy");
  if (!(kindField instanceof HTMLSelectElement) || !(authField instanceof HTMLSelectElement)) {
    return;
  }
  const kind = kindField.value.trim().toLowerCase();
  const previous = authField.value.trim();
  const options = integrationAuthStrategyOptions(kind);
  authField.innerHTML = options
    .map((option) => `<option value="${option.value}">${option.label}</option>`)
    .join("");
  const nextValue = options.some((option) => option.value === previous) ? previous : options[0]?.value || "";
  authField.value = nextValue;
}

function integrationAuthStrategyOptions(kind: string): Array<{ value: string; label: string }> {
  switch (kind) {
    case "github":
      return [
        { value: "", label: "auto" },
        { value: "github_app", label: "github_app" },
        { value: "personal_access_token", label: "personal_access_token" }
      ];
    case "gitlab":
      return [
        { value: "", label: "auto" },
        { value: "personal_access_token", label: "personal_access_token" }
      ];
    case "kubernetes":
    case "prometheus":
      return [{ value: "", label: "auto" }];
    default:
      return [{ value: "", label: "auto" }];
  }
}

function buildStatusDashboardQuery(): string {
  const params = new URLSearchParams();
  if (statusDashboardQueryState.search) {
    params.set("search", statusDashboardQueryState.search);
  }
  if (statusDashboardQueryState.rollbackOnly) {
    params.set("rollback_only", "true");
  }
  if (statusDashboardQueryState.serviceID) {
    params.set("service_id", statusDashboardQueryState.serviceID);
  }
  if (statusDashboardQueryState.environmentID) {
    params.set("environment_id", statusDashboardQueryState.environmentID);
  }
  if (statusDashboardQueryState.source) {
    params.set("source", statusDashboardQueryState.source);
  }
  if (statusDashboardQueryState.eventType) {
    params.set("event_type", statusDashboardQueryState.eventType);
  }
  if (statusDashboardQueryState.automated) {
    params.set("automated", statusDashboardQueryState.automated);
  }
  params.set("limit", String(statusDashboardQueryState.limit));
  params.set("offset", String(statusDashboardQueryState.offset));
  return params.toString();
}

function syncStatusDashboardQuery(form: HTMLFormElement, resetOffset: boolean) {
  statusDashboardQueryState.search = readInputValue(form, "search");
  statusDashboardQueryState.rollbackOnly = readCheckboxValue(form, "rollback_only");
  statusDashboardQueryState.serviceID = readInputValue(form, "service_id");
  statusDashboardQueryState.environmentID = readInputValue(form, "environment_id");
  statusDashboardQueryState.source = readInputValue(form, "source");
  statusDashboardQueryState.eventType = readInputValue(form, "event_type");
  statusDashboardQueryState.automated = readInputValue(form, "automated");
  statusDashboardQueryState.limit = readOptionalPositiveNumber(form, "limit") || 25;
  if (resetOffset) {
    statusDashboardQueryState.offset = 0;
  }
}

function resetStatusDashboardQuery() {
  statusDashboardQueryState.search = "";
  statusDashboardQueryState.rollbackOnly = false;
  statusDashboardQueryState.serviceID = "";
  statusDashboardQueryState.environmentID = "";
  statusDashboardQueryState.source = "";
  statusDashboardQueryState.eventType = "";
  statusDashboardQueryState.automated = "";
  statusDashboardQueryState.limit = 25;
  statusDashboardQueryState.offset = 0;
}

function validateLogin(values: { email: string; password: string }): string | null {
  const emailError = validateEmail(values.email);
  if (emailError) {
    return emailError;
  }
  if (!values.password) {
    return "Enter your password.";
  }
  return null;
}

function validateSignup(values: { email: string; displayName: string; password: string; passwordConfirmation: string }): string | null {
  const emailError = validateEmail(values.email);
  if (emailError) {
    return emailError;
  }
  if (!values.displayName) {
    return "Enter your display name.";
  }
  if (!values.password) {
    return "Create a password.";
  }
  if (values.password.length < 8) {
    return "Password must be at least 8 characters.";
  }
  if (!values.passwordConfirmation) {
    return "Confirm your password.";
  }
  if (values.password !== values.passwordConfirmation) {
    return "Passwords must match.";
  }
  return null;
}

function validateEmail(email: string): string | null {
  if (!email) {
    return "Enter your email address.";
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return "Enter a valid email address.";
  }
  return null;
}

function normalizeAuthError(error: unknown, mode: AuthMode): Error {
  const message = (error instanceof Error ? error.message : "Authentication failed")
    .replace(/^validation failed:\s*/i, "")
    .replace(/^unauthorized:\s*/i, "")
    .trim();
  if (mode === "login" && message.includes("invalid email or password")) {
    return new Error("Invalid email or password.");
  }
  if (mode === "signup" && message.includes("email already has an account")) {
    return new Error("An account already exists for this email. Log in instead.");
  }
  return new Error(message.replace(/^validation failed:\s*/i, ""));
}

function latestExecutionIsAdvisory(state: ControlPlaneState): boolean {
  const execution = latestRolloutExecution(state);
  if (!execution) {
    return false;
  }
  return isExecutionAdvisory(state, execution.backend_integration_id || "");
}

function isManualControlBlocked(state: ControlPlaneState, executionID: string, action: string): boolean {
  if (!["pause", "resume", "rollback"].includes(action)) {
    return false;
  }
  const rolloutData = state.rolloutPage.data;
  const execution = rolloutData?.rolloutExecutions.find((item) => item.id === executionID) || rolloutData?.rolloutExecutionDetail?.execution;
  if (!execution) {
    return false;
  }
  return isExecutionAdvisory(state, execution.backend_integration_id || "");
}

function isExecutionAdvisory(state: ControlPlaneState, backendIntegrationID: string): boolean {
  if (!backendIntegrationID) {
    return false;
  }
  const integration = state.rolloutPage.data?.integrations.find((item) => item.id === backendIntegrationID);
  if (!integration) {
    return false;
  }
  return integration.enabled && integration.kind !== "simulated" && (!integration.control_enabled || integration.mode !== "active_control");
}

function latestRolloutExecution(state: ControlPlaneState) {
  const rolloutData = state.rolloutPage.data;
  return rolloutData?.rolloutExecutionDetail?.execution || rolloutData?.rolloutExecutions[0];
}

function rolloutActionMessages(action: string): { pending: string; success: string } {
  switch (action) {
    case "approve":
      return { pending: "Approving rollout execution...", success: "Rollout execution approved." };
    case "start":
      return { pending: "Starting rollout execution...", success: "Rollout execution started." };
    case "pause":
      return { pending: "Pausing rollout execution...", success: "Rollout execution paused." };
    case "resume":
      return { pending: "Resuming rollout execution...", success: "Rollout execution resumed." };
    case "complete":
      return { pending: "Completing rollout execution...", success: "Rollout execution completed." };
    case "rollback":
      return { pending: "Rolling back rollout execution...", success: "Rollout execution rolled back." };
    default:
      return { pending: "Updating rollout execution...", success: "Rollout execution updated." };
  }
}
