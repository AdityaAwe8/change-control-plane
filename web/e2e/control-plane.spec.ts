import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const apiBaseURL = "http://127.0.0.1:18085";
const browserOrigin = "http://127.0.0.1:4173";

type BrowserSession = {
  cookieHeader: string;
  organizationID: string;
};

test("login shows invalid-credential feedback and seeded admin can sign in", async ({ page }) => {
  await page.goto("/");
  await page.locator("#login-email").fill("wrong@changecontrolplane.local");
  await page.locator("#login-password").fill("WrongPass123!");
  await page.locator("#login-submit").click();

  await expect(page.locator("#app-feedback")).toContainText("Invalid email or password.");
  await expect(page.locator("#app-feedback")).not.toContainText("session expired");
  await expect(page.locator("#login-submit")).toBeVisible();

  await logInThroughUI(page, "admin@changecontrolplane.local", "ChangeMe123!");
});

test("bootstrap login, refresh, project creation, and sign-out work from the browser", async ({ page, request }) => {
  const seed = uniqueSeed();
  const email = `owner-${seed}@acme.local`;
  const password = "ChangeMe123!";
  await signUpThroughUI(page, email, `Owner ${seed}`, password);
  await expect(page.getByText("Waiting for organization access.")).toBeVisible();
  await bootstrapOrganizationForSignedInUser(page, request, `Acme ${seed}`, `acme-${seed}`);

  await page.getByRole("button", { name: "Refresh Data" }).click();
  await expect(page.locator("#app-feedback")).toContainText("refreshed");
  await expect(page.locator("#app-feedback")).toBeHidden({ timeout: 6000 });

  await page.getByRole("link", { name: "Startup Bootstrap" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Startup Bootstrap");
  await expect(page.locator("body")).toContainText("No projects yet");

  await page.locator('#create-project-form input[name="name"]').fill(`Platform ${seed}`);
  await page.locator('#create-project-form input[name="slug"]').fill(`platform-${seed}`);
  await page.locator('#create-project-form textarea[name="description"]').fill("Browser-created project");
  await page.locator('#create-project-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Project created");
  await expect(page.locator("body")).toContainText(`Platform ${seed}`);

  const session = await currentSession(page);
  const projects = await apiGetList<any>(request, session, "/api/v1/projects");
  expect(projects.some((project) => project.slug === `platform-${seed}`)).toBeTruthy();

  await page.getByRole("button", { name: "Sign Out" }).click();
  await expect(page.locator("#login-submit")).toBeVisible();

  await logInThroughUI(page, email, password);
});

test("browser cookie session survives page reload on protected routes", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Reload ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Reload ${seed}`, `reload-${seed}`);

  await page.reload();
  await expect(page.locator(".topbar h2")).toHaveText("Dashboard");

  await page.goto("/#/settings");
  await expect(page.locator(".topbar h2")).toHaveText("Settings");
});

test("missing browser session cookie forces a truthful sign-out on the next reload", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Expiry ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Expiry ${seed}`, `expiry-${seed}`);

  await page.context().clearCookies();
  await page.reload();

  await expect(page.locator("#login-submit")).toBeVisible();
  await expect(page.locator("#app-feedback")).toContainText("session expired");
});

test("admin browser flows can manage teams, services, environments, and machine-auth lifecycle", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Ops ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Ops ${seed}`, `ops-${seed}`);
  const session = await currentSession(page);

  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Platform ${seed}`,
    slug: `platform-${seed}`
  });

  await page.getByRole("link", { name: "Startup Bootstrap" }).click();
  await page.locator('#create-team-form select[name="project_id"]').selectOption(project.id);
  await page.locator('#create-team-form input[name="name"]').fill(`Core ${seed}`);
  await page.locator('#create-team-form input[name="slug"]').fill(`core-${seed}`);
  await page.locator('#create-team-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Team created");
  await expect(page.locator("body")).toContainText(`Core ${seed}`);

  await page.locator('#update-team-form input[name="name"]').fill(`Core Platform ${seed}`);
  await page.locator('#update-team-form input[name="slug"]').fill(`core-platform-${seed}`);
  await page.locator('#update-team-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Team updated");
  await expect(page.locator("body")).toContainText(`Core Platform ${seed}`);

  let teams = await apiGetList<any>(request, session, "/api/v1/teams");
  const team = teams.find((entry) => entry.slug === `core-platform-${seed}`);
  expect(team).toBeTruthy();
  expect(team.status).toBe("active");

  await page.getByRole("link", { name: "Service Detail" }).click();
  await page.locator('#create-service-form input[name="name"]').fill(`Checkout ${seed}`);
  await page.locator('#create-service-form input[name="slug"]').fill(`checkout-${seed}`);
  await page.locator('#create-service-form select[name="project_id"]').selectOption(project.id);
  await page.locator('#create-service-form select[name="team_id"]').selectOption(team.id);
  await page.locator('#create-service-form input[name="criticality"]').fill("mission_critical");
  await page.locator('#create-service-form textarea[name="description"]').fill("Customer-facing checkout service");
  await page.locator('#create-service-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Service created");
  await expect(page.locator("body")).toContainText(`Checkout ${seed}`);
  let services = await apiGetList<any>(request, session, "/api/v1/services");
  expect(services.some((service) => service.slug === `checkout-${seed}` && service.status === "active")).toBeTruthy();

  await page.locator('#update-service-form input[name="name"]').fill(`Checkout API ${seed}`);
  await page.locator('#update-service-form input[name="slug"]').fill(`checkout-api-${seed}`);
  await page.locator('#update-service-form input[name="criticality"]').fill("business_critical");
  await page.locator('#update-service-form textarea[name="description"]').fill("Updated browser-managed checkout service");
  await page.locator('#update-service-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Service updated");
  await expect(page.locator("body")).toContainText(`Checkout API ${seed}`);
  services = await apiGetList<any>(request, session, "/api/v1/services");
  expect(services.some((service) =>
    service.slug === `checkout-api-${seed}` &&
    service.name === `Checkout API ${seed}` &&
    service.criticality === "business_critical" &&
    service.description === "Updated browser-managed checkout service"
  )).toBeTruthy();

  await page.getByRole("button", { name: "Archive Current Service" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Service archived");
  await expect(page.locator("body")).toContainText("archived");
  services = await apiGetList<any>(request, session, "/api/v1/services");
  expect(services.some((service) => service.slug === `checkout-api-${seed}` && service.status === "archived")).toBeTruthy();

  await page.getByRole("link", { name: "Environment" }).click();
  await page.locator('#create-environment-form input[name="name"]').fill(`Production ${seed}`);
  await page.locator('#create-environment-form input[name="slug"]').fill(`prod-${seed}`);
  await page.locator('#create-environment-form select[name="project_id"]').selectOption(project.id);
  await page.locator('#create-environment-form input[name="type"]').fill("production");
  await page.locator('#create-environment-form input[name="region"]').fill("us-central1");
  await page.locator('#create-environment-form input[name="production"]').check();
  await page.locator('#create-environment-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Environment created");
  await expect(page.locator("body")).toContainText(`Production ${seed}`);
  let environments = await apiGetList<any>(request, session, "/api/v1/environments");
  expect(environments.some((environment) => environment.slug === `prod-${seed}` && environment.status === "active")).toBeTruthy();

  await page.locator('#update-environment-form input[name="name"]').fill(`Production Primary ${seed}`);
  await page.locator('#update-environment-form input[name="slug"]').fill(`prod-primary-${seed}`);
  await page.locator('#update-environment-form input[name="type"]').fill("production");
  await page.locator('#update-environment-form input[name="region"]').fill("us-east1");
  await page.locator('#update-environment-form input[name="compliance_zone"]').fill("regulated");
  await page.locator('#update-environment-form input[name="production"]').check();
  await page.locator('#update-environment-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Environment updated");
  await expect(page.locator("body")).toContainText(`Production Primary ${seed}`);
  environments = await apiGetList<any>(request, session, "/api/v1/environments");
  expect(environments.some((environment) =>
    environment.slug === `prod-primary-${seed}` &&
    environment.name === `Production Primary ${seed}` &&
    environment.region === "us-east1" &&
    environment.compliance_zone === "regulated"
  )).toBeTruthy();

  await page.getByRole("button", { name: "Archive Current Environment" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Environment archived");
  await expect(page.locator("body")).toContainText("archived");
  environments = await apiGetList<any>(request, session, "/api/v1/environments");
  expect(environments.some((environment) => environment.slug === `prod-primary-${seed}` && environment.status === "archived")).toBeTruthy();

  await page.getByRole("link", { name: "Settings" }).click();
  await page.locator('#create-service-account-form input[name="name"]').fill(`deployer-${seed}`);
  await page.locator('#create-service-account-form textarea[name="description"]').fill("Browser e2e machine actor");
  await page.locator('#create-service-account-form select[name="role"]').selectOption("org_member");
  await page.locator('#create-service-account-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Service account created");
  await expect(page.locator("table")).toContainText(`deployer-${seed}`);

  await page.locator('.issue-token-form input[name="name"]').first().fill(`primary-${seed}`);
  const issuedDialog = page.waitForEvent("dialog");
  await page.getByRole("button", { name: "Issue Token" }).first().click();
  const issuedTokenDialog = await issuedDialog;
  const issuedToken = issuedTokenDialog.message();
  await issuedTokenDialog.accept();
  expect(issuedToken).toContain("ccpt_");
  await expect(page.locator("#app-feedback")).toContainText("token issued");

  await page.locator('.rotate-token-form input[name="name"]').first().fill(`rotated-${seed}`);
  await page.locator('.rotate-token-form input[name="expires_in_hours"]').first().fill("24");
  const rotatedDialog = page.waitForEvent("dialog");
  await page.getByRole("button", { name: "Rotate Latest Token" }).first().click();
  const rotatedTokenDialog = await rotatedDialog;
  const rotatedToken = rotatedTokenDialog.message();
  await rotatedTokenDialog.accept();
  expect(rotatedToken).toContain("ccpt_");
  await expect(page.locator("#app-feedback")).toContainText("rotated");

  await page.getByRole("button", { name: "Revoke Latest Token" }).first().click();
  await expect(page.locator("#app-feedback")).toContainText("Token revoked");
  const serviceAccounts = await apiGetList<any>(request, session, "/api/v1/service-accounts");
  const createdAccount = serviceAccounts.find((serviceAccount) => serviceAccount.name === `deployer-${seed}`);
  expect(createdAccount).toBeTruthy();
  let tokens = await apiGetList<any>(request, session, `/api/v1/service-accounts/${createdAccount.id}/tokens`);
  expect(tokens.some((token) => token.name === `rotated-${seed}` && token.status === "revoked")).toBeTruthy();

  await page.getByRole("button", { name: "Deactivate Service Account" }).first().click();
  await expect(page.locator("#app-feedback")).toContainText("Service account deactivated");
  await expect(page.locator("table").first()).toContainText("inactive");

  const updatedAccounts = await apiGetList<any>(request, session, "/api/v1/service-accounts");
  const deactivatedAccount = updatedAccounts.find((serviceAccount) => serviceAccount.id === createdAccount.id);
  expect(deactivatedAccount?.status).toBe("inactive");
  tokens = await apiGetList<any>(request, session, `/api/v1/service-accounts/${createdAccount.id}/tokens`);
  expect(tokens.every((token) => token.status !== "active")).toBeTruthy();

  await page.getByRole("link", { name: "Startup Bootstrap" }).click();
  await page.getByRole("button", { name: "Archive Current Team" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Team archived");
  teams = await apiGetList<any>(request, session, "/api/v1/teams");
  expect(teams.some((entry) => entry.slug === `core-platform-${seed}` && entry.status === "archived")).toBeTruthy();
});

test("bootstrap route shows route-local loading and error states when bootstrap data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Bootstrap ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Bootstrap ${seed}`, `bootstrap-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/projects`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated bootstrap route failure"
        }
      })
    });
  });

  await page.goto("/#/bootstrap");
  await expect(page.locator("body")).toContainText("Loading bootstrap workspace");
  await expect(page.locator("body")).toContainText("Bootstrap workspace unavailable");
  await expect(page.locator("body")).toContainText("simulated bootstrap route failure");
});

test("service route shows route-local loading and error states when service data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Service ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Service ${seed}`, `service-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/catalog`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated service route failure"
        }
      })
    });
  });

  await page.goto("/#/service");
  await expect(page.locator("body")).toContainText("Loading service detail");
  await expect(page.locator("body")).toContainText("Service data unavailable");
  await expect(page.locator("body")).toContainText("simulated service route failure");
});

test("environment route shows route-local loading and error states when environment data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Environment ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Environment ${seed}`, `environment-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/projects`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated environment route failure"
        }
      })
    });
  });

  await page.goto("/#/environment");
  await expect(page.locator("body")).toContainText("Loading environment detail");
  await expect(page.locator("body")).toContainText("Environment data unavailable");
  await expect(page.locator("body")).toContainText("simulated environment route failure");
});

test("dashboard route shows route-local loading and error states when dashboard data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Dashboard",
    routeHash: "dashboard",
    routeMatcher: `${apiBaseURL}/api/v1/metrics/basics`,
    loadingText: "Loading dashboard",
    unavailableText: "Dashboard data unavailable",
    failureMessage: "simulated dashboard route failure"
  });
});

test("catalog route shows route-local loading and error states when catalog data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Catalog",
    routeHash: "catalog",
    routeMatcher: `${apiBaseURL}/api/v1/catalog`,
    loadingText: "Loading service catalog",
    unavailableText: "Catalog data unavailable",
    failureMessage: "simulated catalog route failure"
  });
});

test("change-review route shows route-local loading and error states when change data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Change Review",
    routeHash: "change-review",
    routeMatcher: `${apiBaseURL}/api/v1/changes`,
    loadingText: "Loading change review",
    unavailableText: "Change review unavailable",
    failureMessage: "simulated change-review route failure"
  });
});

test("risk route shows route-local loading and error states when risk data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Risk",
    routeHash: "risk",
    routeMatcher: `${apiBaseURL}/api/v1/risk-assessments`,
    loadingText: "Loading risk assessment",
    unavailableText: "Risk data unavailable",
    failureMessage: "simulated risk route failure"
  });
});

test("policies route shows route-local loading and error states when policy data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Policies",
    routeHash: "policies",
    routeMatcher: `${apiBaseURL}/api/v1/policies`,
    loadingText: "Loading policies",
    unavailableText: "Policy data unavailable",
    failureMessage: "simulated policies route failure"
  });
});

test("audit route shows route-local loading and error states when audit data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Audit",
    routeHash: "audit",
    routeMatcher: `${apiBaseURL}/api/v1/audit-events`,
    loadingText: "Loading audit trail",
    unavailableText: "Audit data unavailable",
    failureMessage: "simulated audit route failure"
  });
});

test("graph route shows route-local loading and error states when graph data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Graph",
    routeHash: "graph",
    routeMatcher: `${apiBaseURL}/api/v1/page-state/graph`,
    loadingText: "Loading system graph",
    unavailableText: "Graph data unavailable",
    failureMessage: "simulated graph route failure"
  });
});

test("costs route shows route-local loading and error states when cost data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Costs",
    routeHash: "costs",
    routeMatcher: `${apiBaseURL}/api/v1/status-events?limit=200`,
    loadingText: "Loading cost overview",
    unavailableText: "Cost data unavailable",
    failureMessage: "simulated costs route failure"
  });
});

test("simulation route shows route-local loading and error states when simulation data fails", async ({ page, request }) => {
  await expectRouteLocalFailureState(page, request, {
    label: "Simulation",
    routeHash: "simulation",
    routeMatcher: `${apiBaseURL}/api/v1/page-state/simulation`,
    loadingText: "Loading simulation lab",
    unavailableText: "Simulation data unavailable",
    failureMessage: "simulated simulation route failure"
  });
});

test("settings route shows route-local loading and error states when service-account data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Settings ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Settings ${seed}`, `settings-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/service-accounts`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated settings route failure"
        }
      })
    });
  });

  await page.goto("/#/settings");
  await expect(page.locator("body")).toContainText("Loading service accounts");
  await expect(page.locator("body")).toContainText("Service-account data unavailable");
  await expect(page.locator("body")).toContainText("simulated settings route failure");
});

test("rollout automation and operational status history are visible from the browser", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Runtime ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Runtime ${seed}`, `runtime-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  await page.getByRole("link", { name: "Rollout Plan" }).click();
  await page.locator('#create-rollout-execution-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Rollout execution created");

  await page.locator('#advance-rollout-form select[name="action"]').selectOption("approve");
  await page.locator('#advance-rollout-form input[name="reason"]').fill("approval for browser verification");
  await page.locator('#advance-rollout-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("approved");

  await page.locator('#advance-rollout-form select[name="action"]').selectOption("start");
  await page.locator('#advance-rollout-form input[name="reason"]').fill("start browser rollout");
  await page.locator('#advance-rollout-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("started");

  await page.locator('#reconcile-rollout-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("reconciled");

  await page.locator('#create-signal-snapshot-form select[name="health"]').selectOption("critical");
  await page.locator('#create-signal-snapshot-form textarea[name="summary"]').fill("Rollback required from browser test");
  await page.locator('#create-signal-snapshot-form input[name="latency_value"]').fill("710");
  await page.locator('#create-signal-snapshot-form input[name="error_rate_value"]').fill("4.8");
  await page.locator('#create-signal-snapshot-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Signal snapshot ingested");

  await page.locator('#reconcile-rollout-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("reconciled");
  await expect(page.locator("body")).toContainText("rolled_back");
  await expect(page.locator("body")).toContainText("rollback");

  const executions = await apiGetList<any>(request, session, "/api/v1/rollout-executions");
  const latestExecution = executions.find((execution) => execution.rollout_plan_id === rollout.plan.id);
  expect(latestExecution?.status).toBe("rolled_back");

  await page.getByRole("link", { name: "Deployment History" }).click();
  await page.locator("#status-search-input").fill("rollback");
  await page.locator("#status-rollback-only").check();
  await page.getByRole("button", { name: "Search History" }).click();
  await expect(page.locator("body")).toContainText("Showing 1-");
  const visibleRows = page.locator('[data-status-event-row]:visible');
  await expect.poll(async () => await visibleRows.count()).toBeGreaterThan(0);
  await expect(page.locator("#status-event-table")).toContainText("rollback");
});

test("rollout control form uses dedicated pause, resume, and rollback routes", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Runtime Controls ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Runtime Controls ${seed}`, `runtime-controls-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  const execution = await apiPostItem<any>(request, session, "/api/v1/rollout-executions", {
    rollout_plan_id: rollout.plan.id
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "approve",
    reason: "approve for dedicated control-route proof"
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "start",
    reason: "start for dedicated control-route proof"
  });

  await page.getByRole("link", { name: "Rollout Plan" }).click();

  const pauseRequest = page.waitForRequest((req) =>
    req.method() === "POST" && req.url() === `${apiBaseURL}/api/v1/rollout-executions/${execution.id}/pause`
  );
  await page.locator('#advance-rollout-form select[name="action"]').selectOption("pause");
  await page.locator('#advance-rollout-form input[name="reason"]').fill("pause from browser route proof");
  await Promise.all([pauseRequest, page.locator('#advance-rollout-form button[type="submit"]').click()]);
  await expect(page.locator("#app-feedback")).toContainText("paused");
  await expect(page.locator("body")).toContainText("paused");

  const resumeRequest = page.waitForRequest((req) =>
    req.method() === "POST" && req.url() === `${apiBaseURL}/api/v1/rollout-executions/${execution.id}/resume`
  );
  await page.locator('#advance-rollout-form select[name="action"]').selectOption("resume");
  await page.locator('#advance-rollout-form input[name="reason"]').fill("resume from browser route proof");
  await Promise.all([resumeRequest, page.locator('#advance-rollout-form button[type="submit"]').click()]);
  await expect(page.locator("#app-feedback")).toContainText("resumed");
  await expect(page.locator("body")).toContainText("in_progress");

  const rollbackRequest = page.waitForRequest((req) =>
    req.method() === "POST" && req.url() === `${apiBaseURL}/api/v1/rollout-executions/${execution.id}/rollback`
  );
  await page.locator('#advance-rollout-form select[name="action"]').selectOption("rollback");
  await page.locator('#advance-rollout-form input[name="reason"]').fill("rollback from browser route proof");
  await Promise.all([rollbackRequest, page.locator('#advance-rollout-form button[type="submit"]').click()]);
  await expect(page.locator("#app-feedback")).toContainText("rolled back");
  await expect(page.locator("body")).toContainText("rolled_back");

  const executions = await apiGetList<any>(request, session, "/api/v1/rollout-executions");
  const updated = executions.find((item) => item.id === execution.id);
  expect(updated?.status).toBe("rolled_back");
});

test("verification form records active rollout decisions through the dedicated verification route", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Runtime Verification ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Runtime Verification ${seed}`, `runtime-verification-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  const execution = await apiPostItem<any>(request, session, "/api/v1/rollout-executions", {
    rollout_plan_id: rollout.plan.id
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "approve",
    reason: "approve for browser verification form proof"
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "start",
    reason: "start for browser verification form proof"
  });

  await page.getByRole("link", { name: "Rollout Plan" }).click();

  const verificationRequest = page.waitForRequest((req) =>
    req.method() === "POST" && req.url() === `${apiBaseURL}/api/v1/rollout-executions/${execution.id}/verification`
  );
  await page.locator('#record-verification-form select[name="outcome"]').selectOption("fail");
  await page.locator('#record-verification-form select[name="decision"]').selectOption("pause");
  await page.locator('#record-verification-form textarea[name="summary"]').fill("manual verification caught a latency regression");
  await Promise.all([verificationRequest, page.locator('#record-verification-form button[type="submit"]').click()]);

  await expect(page.locator("#app-feedback")).toContainText("Verification result recorded");
  await expect(page.locator("body")).toContainText("manual verification caught a latency regression");
  await expect(page.locator("body")).toContainText("paused");

  const detail = await apiGetItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}`);
  expect(detail.execution.status).toBe("paused");
  expect(detail.verification_results.some((result: any) => result.summary === "manual verification caught a latency regression" && result.decision === "pause")).toBeTruthy();
});

test("rollout route shows route-local loading and error states when rollout data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Route Local Rollout ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Route Local Rollout ${seed}`, `route-local-rollout-${seed}`);

  await page.route("**/api/v1/page-state/rollout*", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated rollout route failure"
        }
      })
    });
  });

  await page.goto("/#/rollout");
  await expect(page.locator("body")).toContainText("Loading rollout data");
  await expect(page.locator("body")).toContainText("Rollout data unavailable");
  await expect(page.locator("body")).toContainText("simulated rollout route failure");
});

test("deployments route shows route-local loading and error states when status search fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Route Local Deployments ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Route Local Deployments ${seed}`, `route-local-deployments-${seed}`);

  await page.route(new RegExp(`${apiBaseURL}/api/v1/page-state/deployments(\\?|$)`), async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated deployments route failure"
        }
      })
    });
  });

  await page.goto("/#/deployments");
  await expect(page.locator("body")).toContainText("Loading deployment history");
  await expect(page.locator("body")).toContainText("Deployment history unavailable");
  await expect(page.locator("body")).toContainText("simulated deployments route failure");
});

test("integrations route shows route-local loading and error states when integration data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Route Local Integrations ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Route Local Integrations ${seed}`, `route-local-integrations-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/page-state/integrations`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated integrations route failure"
        }
      })
    });
  });

  await page.goto("/#/integrations");
  await expect(page.locator("body")).toContainText("Loading integration surfaces");
  await expect(page.locator("body")).toContainText("Integration data unavailable");
  await expect(page.locator("body")).toContainText("simulated integrations route failure");
});

test("enterprise route shows route-local loading and error states when enterprise data fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Route Local Enterprise ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Route Local Enterprise ${seed}`, `route-local-enterprise-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/page-state/enterprise`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated enterprise route failure"
        }
      })
    });
  });

  await page.goto("/#/enterprise");
  await expect(page.locator("body")).toContainText("Loading enterprise diagnostics");
  await expect(page.locator("body")).toContainText("Enterprise diagnostics unavailable");
  await expect(page.locator("body")).toContainText("simulated enterprise route failure");
});

test("all authenticated routes use route-local loads and the chattiest pages use bundled page-state requests", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Route Isolation ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Route Isolation ${seed}`, `route-isolation-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  const incidentExecution = await apiPostItem<any>(request, session, "/api/v1/rollout-executions", {
    rollout_plan_id: rollout.plan.id
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${incidentExecution.id}/advance`, {
    action: "approve",
    reason: "approve for isolation proof"
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${incidentExecution.id}/advance`, {
    action: "start",
    reason: "start for isolation proof"
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${incidentExecution.id}/pause`, {
    reason: "pause for isolation proof"
  });
  const incidentID = `incident_${incidentExecution.id}`;

  const requests: string[] = [];
  page.on("request", (req) => {
    if (req.url().startsWith(`${apiBaseURL}/api/v1/`)) {
      requests.push(req.url().replace(apiBaseURL, ""));
    }
  });

  const expectRouteRequests = async (
    hash: string,
    heading: string,
    expected: string[],
    forbidden: string[]
  ) => {
    requests.length = 0;
    await page.goto(`/#/${hash}`);
    await expect(page.locator(".topbar h2")).toHaveText(heading);
    await expect.poll(() => expected.every((fragment) => requests.some((url) => url.includes(fragment)))).toBeTruthy();
    forbidden.forEach((fragment) => {
      expect(requests.some((url) => url.includes(fragment))).toBeFalsy();
    });
  };

  await expectRouteRequests("dashboard", "Dashboard", [
    "/api/v1/metrics/basics",
    "/api/v1/integrations/coverage",
    "/api/v1/risk-assessments"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/graph/relationships"
  ]);

  await expectRouteRequests("bootstrap", "Startup Bootstrap", [
    "/api/v1/metrics/basics",
    "/api/v1/projects",
    "/api/v1/teams",
    "/api/v1/catalog"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/rollout-plans"
  ]);

  await expectRouteRequests("catalog", "Service Catalog", [
    "/api/v1/catalog"
  ], [
    "/api/v1/changes",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events"
  ]);

  await expectRouteRequests("change-review", "Change Review", [
    "/api/v1/changes"
  ], [
    "/api/v1/catalog",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events"
  ]);

  await expectRouteRequests("risk", "Risk Assessment", [
    "/api/v1/risk-assessments"
  ], [
    "/api/v1/catalog",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events"
  ]);

  await expectRouteRequests("service", "Service Detail", [
    "/api/v1/projects",
    "/api/v1/teams",
    "/api/v1/catalog",
    "/api/v1/rollout-executions"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/identity-providers"
  ]);

  await expectRouteRequests("environment", "Environment", [
    "/api/v1/projects",
    "/api/v1/catalog",
    "/api/v1/rollout-executions"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/identity-providers"
  ]);

  await expectRouteRequests("policies", "Policy Center", [
    "/api/v1/policies",
    "/api/v1/policy-decisions",
    "/api/v1/projects",
    "/api/v1/catalog"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/graph/relationships"
  ]);

  await expectRouteRequests("audit", "Audit Trail", [
    "/api/v1/audit-events"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/graph/relationships"
  ]);

  await expectRouteRequests("rollout", "Rollout Plan", [
    "/api/v1/page-state/rollout"
  ], [
    "/api/v1/rollout-plans",
    "/api/v1/rollout-executions",
    "/api/v1/integrations",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/repositories"
  ]);

  await expectRouteRequests("deployments", "Deployment History", [
    "/api/v1/page-state/deployments"
  ], [
    "/api/v1/status-events/search",
    "/api/v1/rollback-policies",
    "/api/v1/catalog",
    "/api/v1/integrations/coverage",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/identity-providers"
  ]);

  await expectRouteRequests("graph", "System Graph", [
    "/api/v1/page-state/graph"
  ], [
    "/api/v1/graph/relationships",
    "/api/v1/integrations",
    "/api/v1/projects",
    "/api/v1/changes",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/rollback-policies"
  ]);

  await expectRouteRequests("costs", "Cost Overview", [
    "/api/v1/catalog",
    "/api/v1/status-events?limit=200"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/graph/relationships"
  ]);

  await expectRouteRequests("integrations", "Integrations", [
    "/api/v1/page-state/integrations"
  ], [
    "/api/v1/integrations/",
    "/api/v1/repositories",
    "/api/v1/discovered-resources",
    "/api/v1/integrations/coverage",
    "/api/v1/sync-runs",
    "/api/v1/webhook-registration",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/rollout-plans"
  ]);

  await expectRouteRequests("simulation", "Simulation Lab", [
    "/api/v1/page-state/simulation"
  ], [
    "/api/v1/changes",
    "/api/v1/risk-assessments",
    "/api/v1/rollout-plans",
    "/api/v1/rollout-executions",
    "/api/v1/rollback-policies",
    "/api/v1/status-events?limit=200",
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/graph/relationships"
  ]);

  await expectRouteRequests("enterprise", "Enterprise Mode", [
    "/api/v1/page-state/enterprise"
  ], [
    "/api/v1/identity-providers",
    "/api/v1/outbox-events",
    "/api/v1/integrations/",
    "/api/v1/webhook-registration",
    "/api/v1/repositories",
    "/api/v1/discovered-resources",
    "/api/v1/rollout-plans"
  ]);

  await expectRouteRequests("settings", "Settings", [
    "/api/v1/service-accounts"
  ], [
    "/api/v1/incidents",
    "/api/v1/outbox-events",
    "/api/v1/rollout-plans"
  ]);

  await expectRouteRequests("incidents", "Incidents", [
    "/api/v1/incidents"
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/rollout-plans"
  ]);

  await expectRouteRequests(`incident-detail?id=${incidentID}`, "Incident Detail", [
    `/api/v1/incidents/${incidentID}`
  ], [
    "/api/v1/service-accounts",
    "/api/v1/outbox-events",
    "/api/v1/rollout-plans"
  ]);
});

test("organization switching reloads route-local data for the selected tenant", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Tenant Switch ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Primary ${seed}`, `primary-${seed}`);
  const primarySession = await currentSession(page);

  const primaryProject = await apiPostItem<any>(request, primarySession, "/api/v1/projects", {
    organization_id: primarySession.organizationID,
    name: `Primary Platform ${seed}`,
    slug: `primary-platform-${seed}`
  });
  const primaryTeam = await apiPostItem<any>(request, primarySession, "/api/v1/teams", {
    organization_id: primarySession.organizationID,
    project_id: primaryProject.id,
    name: `Primary Core ${seed}`,
    slug: `primary-core-${seed}`
  });
  await apiPostItem<any>(request, primarySession, "/api/v1/services", {
    organization_id: primarySession.organizationID,
    project_id: primaryProject.id,
    team_id: primaryTeam.id,
    name: `Primary Checkout ${seed}`,
    slug: `primary-checkout-${seed}`
  });

  const secondaryOrganization = await apiPostItem<any>(request, primarySession, "/api/v1/organizations", {
    name: `Secondary ${seed}`,
    slug: `secondary-${seed}`
  });
  const secondarySession = {
    ...primarySession,
    organizationID: secondaryOrganization.id
  };
  const secondaryProject = await apiPostItem<any>(request, secondarySession, "/api/v1/projects", {
    organization_id: secondaryOrganization.id,
    name: `Secondary Platform ${seed}`,
    slug: `secondary-platform-${seed}`
  });
  const secondaryTeam = await apiPostItem<any>(request, secondarySession, "/api/v1/teams", {
    organization_id: secondaryOrganization.id,
    project_id: secondaryProject.id,
    name: `Secondary Core ${seed}`,
    slug: `secondary-core-${seed}`
  });
  await apiPostItem<any>(request, secondarySession, "/api/v1/services", {
    organization_id: secondaryOrganization.id,
    project_id: secondaryProject.id,
    team_id: secondaryTeam.id,
    name: `Secondary Billing ${seed}`,
    slug: `secondary-billing-${seed}`
  });

  await page.getByRole("button", { name: "Refresh Data" }).click();
  await expect(page.locator("#app-feedback")).toContainText("refreshed");

  await page.getByRole("link", { name: "Service Catalog" }).click();
  await expect(page.locator("body")).toContainText(`Primary Checkout ${seed}`);
  await expect(page.locator("body")).not.toContainText(`Secondary Billing ${seed}`);

  await page.locator("#organization-switcher").selectOption(secondaryOrganization.id);
  await expect(page.locator("body")).toContainText(`Secondary Billing ${seed}`);
  await expect(page.locator("body")).not.toContainText(`Primary Checkout ${seed}`);
  await expect.poll(async () => (await currentSession(page)).organizationID).toBe(secondaryOrganization.id);
});

test("rapid route changes ignore slower stale route responses", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Stale Guard ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Stale Guard ${seed}`, `stale-guard-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/page-state/graph`, async (route) => {
    const response = await route.fetch();
    const body = await response.body();
    await new Promise((resolve) => setTimeout(resolve, 700));
    await route.fulfill({ response, body });
  });

  await page.goto("/#/graph");
  await expect(page.locator("body")).toContainText("Loading system graph");

  await page.goto("/#/costs");
  await expect(page.locator(".topbar h2")).toHaveText("Cost Overview");
  await expect(page.locator("body")).toContainText("Estimated Monthly Spend");

  await page.waitForTimeout(900);
  await expect(page.locator(".topbar h2")).toHaveText("Cost Overview");
  await expect(page.locator("body")).toContainText("Estimated Monthly Spend");
});

test("settings route stays truthful across consecutive route-local mutations", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Consecutive ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Consecutive ${seed}`, `consecutive-${seed}`);

  await page.getByRole("link", { name: "Settings" }).click();

  await page.locator('#create-service-account-form input[name="name"]').fill(`alpha-${seed}`);
  await page.locator('#create-service-account-form textarea[name="description"]').fill("first route-local mutation");
  await page.locator('#create-service-account-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Service account created");

  await page.locator('#create-service-account-form input[name="name"]').fill(`beta-${seed}`);
  await page.locator('#create-service-account-form textarea[name="description"]').fill("second route-local mutation");
  await page.locator('#create-service-account-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Service account created");
  await expect(page.locator("table")).toContainText(`alpha-${seed}`);
  await expect(page.locator("table")).toContainText(`beta-${seed}`);

  let issuedToken = "";
  page.once("dialog", async (dialog) => {
    issuedToken = dialog.message();
    await dialog.accept();
  });

  const betaIssueForm = page.locator("form.issue-token-form").filter({
    has: page.getByPlaceholder(`beta-${seed} primary`)
  });
  await betaIssueForm.getByPlaceholder(`beta-${seed} primary`).fill(`beta-primary-${seed}`);
  await betaIssueForm.getByRole("button", { name: "Issue Token" }).click();
  await expect.poll(() => issuedToken).toContain("ccpt_");
  await expect(page.locator("#app-feedback")).toContainText("token issued");

  await page.locator(".revoke-token-button").last().click();
  await expect(page.locator("#app-feedback")).toContainText("Token revoked");
  await expect(page.locator("table").first()).toContainText(`alpha-${seed}`);
  await expect(page.locator("table").first()).toContainText(`beta-${seed}`);

  const session = await currentSession(page);
  const serviceAccounts = await apiGetList<any>(request, session, "/api/v1/service-accounts");
  const betaAccount = serviceAccounts.find((serviceAccount) => serviceAccount.name === `beta-${seed}`);
  expect(betaAccount).toBeTruthy();
  const betaTokens = await apiGetList<any>(request, session, `/api/v1/service-accounts/${betaAccount.id}/tokens`);
  expect(betaTokens.some((token) => token.status === "revoked")).toBeTruthy();
});

test("incident feed links into a dedicated incident detail route with rollout-scoped timeline", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Incidents ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Incidents ${seed}`, `incidents-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  const execution = await apiPostItem<any>(request, session, "/api/v1/rollout-executions", {
    rollout_plan_id: rollout.plan.id
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "approve",
    reason: "approve for incident detail browser test"
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "start",
    reason: "start for incident detail browser test"
  });
  await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/pause`, {
    reason: "pause for incident detail browser test"
  });

  await page.getByRole("link", { name: "Incidents" }).click();
  const incidentLink = page.locator(`[data-incident-link="incident_${execution.id}"]`);
  await expect(incidentLink).toBeVisible();
  await incidentLink.click();

  await expect(page).toHaveURL(new RegExp(`#\\/incident-detail\\?id=incident_${execution.id}$`));
  await expect(page.locator(".topbar h2")).toHaveText("Incident Detail");
  await expect(page.locator("body")).toContainText(execution.id);
  await expect(page.locator("#status-event-table")).toContainText("pause");
});

test("incident detail route shows empty selection and not-found states truthfully", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Incidents ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Incidents ${seed}`, `incidents-empty-${seed}`);

  await page.getByRole("link", { name: "Incident Detail" }).click();
  await expect(page.locator("body")).toContainText("No incident selected");

  await page.goto(`/#/incident-detail?id=incident-missing-${seed}`);
  await expect(page.locator("body")).toContainText("Incident not found");
});

test("incident detail route shows API failure feedback when the detail fetch fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Incidents ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Incidents ${seed}`, `incidents-error-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/incidents`, async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ data: [] }) });
  });

  const incidentID = `incident-error-${seed}`;
  await page.route(`${apiBaseURL}/api/v1/incidents/${incidentID}`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated incident detail failure"
        }
      })
    });
  });

  await page.goto(`/#/incident-detail?id=${incidentID}`);
  await expect(page.locator("body")).toContainText("Loading incident detail");
  await expect(page.locator("body")).toContainText("Incident detail unavailable");
  await expect(page.locator("body")).toContainText("simulated incident detail failure");
});

test("incidents route shows route-local loading and error states when the incident feed fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Incidents Feed ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Incidents Feed ${seed}`, `incidents-feed-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/incidents`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated incidents route failure"
        }
      })
    });
  });

  await page.goto("/#/incidents");
  await expect(page.locator("body")).toContainText("Loading incidents");
  await expect(page.locator("body")).toContainText("Incident feed unavailable");
  await expect(page.locator("body")).toContainText("simulated incidents route failure");
});

test("dashboard and remaining read-heavy routes render truthful seeded data through route-local loads", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Read Models ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Read Models ${seed}`, `read-models-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  await apiPostItem<any>(request, session, "/api/v1/rollback-policies", {
    organization_id: session.organizationID,
    project_id: rollout.project.id,
    service_id: rollout.service.id,
    environment_id: rollout.environment.id,
    name: `Browser guardrail ${seed}`,
    priority: 70,
    max_error_rate: 1.5,
    max_latency_ms: 325,
    rollback_on_critical_signals: true
  });

  const integrations = await apiGetList<any>(request, session, "/api/v1/integrations");
  const github = integrations.find((integration) => integration.kind === "github");
  expect(github).toBeTruthy();

  const repositoryURL = `https://github.com/acme/checkout-${seed}`;
  await apiPostItem<any>(request, session, `/api/v1/integrations/${github.id}/graph-ingest`, {
    repositories: [
      {
        project_id: rollout.project.id,
        service_id: rollout.service.id,
        name: `checkout-${seed}`,
        provider: "github",
        url: repositoryURL,
        default_branch: "main"
      }
    ],
    service_environments: [
      {
        service_id: rollout.service.id,
        environment_id: rollout.environment.id
      }
    ],
    change_repositories: [
      {
        change_set_id: rollout.change.id,
        repository_url: repositoryURL
      }
    ]
  });

  await page.getByRole("link", { name: "Dashboard" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Dashboard");
  await expect(page.locator("body")).toContainText("Fresh Integrations");
  await expect(page.locator("body")).toContainText("Latest Risk Signal");
  await expect(page.locator("body")).toContainText("Score");

  await page.getByRole("link", { name: "Service Catalog" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Service Catalog");
  await expect(page.locator("body")).toContainText(`Checkout ${seed}`);
  await expect(page.locator("body")).toContainText(`Production ${seed}`);

  await page.getByRole("link", { name: "Change Review" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Change Review");
  await expect(page.locator("body")).toContainText(`Browser rollout ${seed}`);
  await expect(page.locator("body")).toContainText("Change Types");

  await page.getByRole("link", { name: "Risk Assessment" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Risk Assessment");
  await expect(page.locator("body")).toContainText("Latest score");
  await expect(page.locator("body")).toContainText("Approval");

  await page.getByRole("link", { name: "Policy Center" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Policy Center");
  await expect(page.locator("body")).toContainText("Production High Risk Approval");
  await expect(page.locator("body")).toContainText("Recent Policy Decisions");

  await page.getByRole("link", { name: "Audit Trail" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Audit Trail");
  await expect(page.locator("body")).toContainText("rollback_policy.created");
  await expect(page.locator("body")).toContainText("rollout.planned");

  await page.getByRole("link", { name: "System Graph" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("System Graph");
  await expect(page.locator("body")).toContainText("service_environment");
  await expect(page.locator("body")).toContainText("integration_graph_ingest");
  await expect(page.locator("body")).toContainText(`Checkout ${seed}`);
  await expect(page.locator("body")).toContainText(`Production ${seed}`);

  await page.getByRole("link", { name: "Cost Overview" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Cost Overview");
  await expect(page.locator("body")).toContainText("Estimated Monthly Spend");
  await expect(page.locator("body")).toContainText(`Checkout ${seed}`);

  await page.getByRole("link", { name: "Simulation Lab" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Simulation Lab");
  await expect(page.locator("body")).toContainText("Proceed with planned rollout");
  await expect(page.locator("body")).toContainText(`Browser guardrail ${seed}`);
});

test("org members see read-only controls for admin-only UI surfaces", async ({ page, request }) => {
  const seed = uniqueSeed();
  const slug = `readonly-${seed}`;
  const ownerEmail = `owner-${seed}@acme.local`;

  await signUpThroughUI(page, ownerEmail, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Owner ${seed}`, slug);
  let session = await currentSession(page);
  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Platform ${seed}`,
    slug: `platform-${seed}`
  });
  const team = await apiPostItem<any>(request, session, "/api/v1/teams", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Core ${seed}`,
    slug: `core-${seed}`
  });
  await apiPostItem<any>(request, session, "/api/v1/services", {
    organization_id: session.organizationID,
    project_id: project.id,
    team_id: team.id,
    name: `Checkout ${seed}`,
    slug: `checkout-${seed}`
  });
  await apiPostItem<any>(request, session, "/api/v1/environments", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Production ${seed}`,
    slug: `prod-${seed}`,
    type: "production",
    production: true
  });
  const serviceAccount = await apiPostItem<any>(request, session, "/api/v1/service-accounts", {
    organization_id: session.organizationID,
    name: `readonly-bot-${seed}`,
    role: "org_member"
  });
  await apiPostItem<any>(request, session, `/api/v1/service-accounts/${serviceAccount.id}/tokens`, {
    name: `readonly-primary-${seed}`
  });
  await page.getByRole("button", { name: "Sign Out" }).click();
  await expect(page.locator("#login-submit")).toBeVisible();

  const memberEmail = `member-${seed}@acme.local`;
  await grantOrganizationAccess(request, memberEmail, `Member ${seed}`, slug);
  await signUpThroughUI(page, memberEmail, `Member ${seed}`, "ChangeMe123!", "dashboard");
  session = await currentSession(page);

  await page.getByRole("link", { name: "Startup Bootstrap" }).click();
  await expect(page.getByText("Read-only workspace").first()).toBeVisible();
  await expect(page.locator("#create-project-form")).toHaveCount(0);
  await expect(page.locator("#create-team-form")).toHaveCount(0);
  await expect(page.locator("#update-team-form")).toHaveCount(0);
  await expect(page.locator("#archive-team-button")).toHaveCount(0);

  await page.getByRole("link", { name: "Service Detail" }).click();
  await expect(page.getByText("Read-only view")).toBeVisible();
  await expect(page.locator("#create-service-form")).toHaveCount(0);
  await expect(page.locator("#update-service-form")).toHaveCount(0);
  await expect(page.locator("#archive-service-button")).toHaveCount(0);

  await page.getByRole("link", { name: "Environment" }).click();
  await expect(page.getByText("Read-only view")).toBeVisible();
  await expect(page.locator("#create-environment-form")).toHaveCount(0);
  await expect(page.locator("#update-environment-form")).toHaveCount(0);
  await expect(page.locator("#archive-environment-button")).toHaveCount(0);

  await page.getByRole("link", { name: "Settings" }).click();
  await expect(page.locator("#create-service-account-form")).toHaveCount(0);
  await expect(page.locator(".issue-token-form")).toHaveCount(0);
  await expect(page.locator(".rotate-token-form")).toHaveCount(0);
  await expect(page.locator(".revoke-token-button")).toHaveCount(0);
  await expect(page.locator(".deactivate-service-account-button")).toHaveCount(0);

  await page.getByRole("link", { name: "Policy Center" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Policy Center");
  await expect(page.locator("body")).toContainText("Production High Risk Approval");
  await expect(page.getByText("Read-only workspace")).toBeVisible();
  await expect(page.locator("#create-policy-form")).toHaveCount(0);
  await expect(page.locator(".policy-toggle-button")).toHaveCount(0);

  await page.route(`${apiBaseURL}/api/v1/page-state/enterprise`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          identity_providers: [],
          integrations: [],
          webhook_registrations: {},
          outbox_events: [
            {
              id: "evt_retry_readonly",
              event_type: "integration.sync.requested",
              organization_id: session.organizationID,
              resource_type: "integration",
              resource_id: "integration_readonly_retry",
              status: "error",
              attempts: 2,
              last_error: "temporary dispatch failure",
              metadata: { last_error_class: "temporary" }
            },
            {
              id: "evt_requeue_readonly",
              event_type: "webhook.received",
              organization_id: session.organizationID,
              resource_type: "webhook",
              resource_id: "delivery_readonly_dead",
              status: "dead_letter",
              attempts: 5,
              last_error: "permanent payload failure",
              metadata: { last_error_class: "permanent" }
            }
          ]
        }
      })
    });
  });

  await page.getByRole("link", { name: "Enterprise Mode" }).click();
  await expect(page.getByText("Read-only workspace")).toBeVisible();
  await expect(page.getByRole("button", { name: "Retry Now" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Requeue" })).toHaveCount(0);
});

test("policy center supports policy authoring, updates, disablement, and recent-decision visibility", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Policy ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Policy ${seed}`, `policy-${seed}`);
  const session = await currentSession(page);

  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Governance ${seed}`,
    slug: `governance-${seed}`
  });
  const team = await apiPostItem<any>(request, session, "/api/v1/teams", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Operators ${seed}`,
    slug: `operators-${seed}`
  });
  const service = await apiPostItem<any>(request, session, "/api/v1/services", {
    organization_id: session.organizationID,
    project_id: project.id,
    team_id: team.id,
    name: `Policy Service ${seed}`,
    slug: `policy-service-${seed}`,
    criticality: "low",
    has_slo: true,
    has_observability: true
  });
  const environment = await apiPostItem<any>(request, session, "/api/v1/environments", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Production ${seed}`,
    slug: `policy-prod-${seed}`,
    type: "production",
    production: true
  });

  const policyName = `Production Review ${seed}`;
  const policyCode = `production-review-${seed}`;
  await page.goto("/#/policies");
  await page.locator('#create-policy-form input[name="name"]').fill(policyName);
  await page.locator('#create-policy-form input[name="code"]').fill(policyCode);
  await page.locator('#create-policy-form select[name="applies_to"]').selectOption("rollout_plan");
  await page.locator('#create-policy-form select[name="mode"]').selectOption("require_manual_review");
  await page.locator('#create-policy-form select[name="project_id"]').selectOption(project.id);
  await page.locator('#create-policy-form select[name="service_id"]').selectOption(service.id);
  await page.locator('#create-policy-form select[name="environment_id"]').selectOption(environment.id);
  await page.locator('#create-policy-form textarea[name="description"]').fill("Require a policy review before low-risk production rollout planning.");
  await page.locator('#create-policy-form input[name="production_only"]').check();
  await page.locator('#create-policy-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("Policy created.");
  await expect(page.locator("body")).toContainText(policyName);

  const policyCard = page.locator(".policy-config-form").filter({ hasText: policyName });
  await policyCard.locator('textarea[name="description"]').fill("Updated from the browser to verify inline governance edits.");
  await policyCard.getByRole("button", { name: "Save Policy" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Policy updated.");
  await expect(policyCard).toContainText("Updated from the browser to verify inline governance edits.");

  const change = await apiPostItem<any>(request, session, "/api/v1/changes", {
    organization_id: session.organizationID,
    project_id: project.id,
    service_id: service.id,
    environment_id: environment.id,
    summary: `Browser policy rollout ${seed}`,
    change_types: [],
    file_count: 0,
    resource_count: 0
  });
  const rollout = await apiPostItem<any>(request, session, "/api/v1/rollout-plans", {
    change_set_id: change.id
  });
  expect(rollout.plan.approval_level).toBe("policy-review");

  await page.getByRole("button", { name: "Refresh Data" }).click();
  await expect(page.locator("body")).toContainText(policyCode);
  await expect(page.locator("body")).toContainText("require_manual_review");
  await expect(page.locator("body")).toContainText(`plan ${rollout.plan.id}`);

  await policyCard.getByRole("button", { name: "Disable" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Policy disabled.");
  await expect(policyCard).toContainText("Enabled: No");
  await expect(policyCard.getByRole("button", { name: "Enable" })).toBeVisible();
});

test("auth validation errors clear while editing and return on the next submit", async ({ page }) => {
  await page.goto("/");
  await page.locator('[data-password-toggle="login-password"]').click();
  await expect(page.locator("#login-password")).toHaveAttribute("type", "text");
  await page.locator('[data-password-toggle="login-password"]').click();
  await expect(page.locator("#login-password")).toHaveAttribute("type", "password");

  await page.locator("#auth-mode-signup").click();
  await page.locator('[data-password-toggle="signup-password"]').click();
  await expect(page.locator("#signup-password")).toHaveAttribute("type", "text");
  await page.locator('[data-password-toggle="signup-password"]').click();
  await expect(page.locator("#signup-password")).toHaveAttribute("type", "password");
  await page.locator("#signup-email").fill("owner@acme.local");
  await page.locator("#signup-display-name").fill("Owner");
  await page.locator("#signup-password").fill("ChangeMe123!");
  await page.locator("#signup-password-confirmation").fill("Mismatch123!");
  await page.locator("#signup-submit").click();
  await expect(page.locator("#app-feedback")).toContainText("Passwords must match");

  await page.locator("#signup-password-confirmation").fill("AlmostRight123!");
  await expect(page.locator("#app-feedback")).toBeHidden();

  await page.locator("#signup-submit").click();
  await expect(page.locator("#app-feedback")).toContainText("Passwords must match");
  await expect(page.locator("#app-feedback")).toBeHidden({ timeout: 3500 });
});

test("enterprise admin surfaces show identity-provider setup and public SSO entry points", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Enterprise ${seed}`, `enterprise-${seed}`);
  const session = await currentSession(page);

  await apiPostItem<any>(request, session, "/api/v1/identity-providers", {
    organization_id: session.organizationID,
    name: `Acme Okta ${seed}`,
    kind: "oidc",
    issuer_url: "https://issuer.example.com",
    client_id: `client-${seed}`,
    client_secret_env: "CCP_ENTERPRISE_TEST_SECRET",
    allowed_domains: ["acme.com"],
    default_role: "org_member",
    enabled: true
  });

  await page.getByRole("link", { name: "Enterprise Mode" }).click();
  await expect(page.getByRole("heading", { name: "Enterprise Identity Providers" })).toBeVisible();
  await expect(page.getByText(`Acme Okta ${seed}`)).toBeVisible();
  await expect(page.getByRole("button", { name: "Test Provider" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Durable Event Diagnostics" })).toBeVisible();
  await expect(page.getByText("Current Session")).toBeVisible();
  await page.locator('#create-identity-provider-form input[name="name"]').fill(`Browser Okta ${seed}`);
  await page.locator('#create-identity-provider-form input[name="issuer_url"]').fill("https://browser.example.com/oauth2/default");
  await page.locator('#create-identity-provider-form input[name="client_id"]').fill(`browser-client-${seed}`);
  await page.locator('#create-identity-provider-form input[name="client_secret_env"]').fill("CCP_BROWSER_OKTA_SECRET");
  await page.locator('#create-identity-provider-form input[name="allowed_domains"]').fill("browser.example.com");
  await page.locator('#create-identity-provider-form input[name="default_role"]').fill("org_member");
  await page.locator('#create-identity-provider-form button[type="submit"]').click();
  await expect(page.locator("#app-feedback")).toContainText("identity provider created");
  await expect(page.getByText(`Browser Okta ${seed}`)).toBeVisible();

  await page.getByRole("button", { name: "Sign Out" }).click();
  await expect(page.getByRole("button", { name: `Continue with Acme Okta ${seed}` })).toBeVisible();
  await expect(page.getByRole("button", { name: `Continue with Browser Okta ${seed}` })).toBeVisible();
});

test("enterprise outbox diagnostics allow retry and requeue with route-local refresh", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Outbox ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Outbox ${seed}`, `outbox-${seed}`);

  let outboxEvents: any[] = [
    {
      id: "evt_retry_browser",
      event_type: "integration.sync.requested",
      organization_id: "org_browser",
      resource_type: "integration",
      resource_id: "integration_retry_browser",
      status: "error",
      attempts: 2,
      next_attempt_at: "2026-04-18T12:00:00Z",
      last_error: "temporary dispatch failure",
      metadata: { last_error_class: "temporary" }
    },
    {
      id: "evt_requeue_browser",
      event_type: "webhook.received",
      organization_id: "org_browser",
      resource_type: "webhook",
      resource_id: "delivery_requeue_browser",
      status: "dead_letter",
      attempts: 5,
      last_error: "permanent payload failure",
      metadata: { last_error_class: "permanent" }
    },
    {
      id: "evt_processed_browser",
      event_type: "status.created",
      organization_id: "org_browser",
      resource_type: "status_event",
      resource_id: "status_processed_browser",
      status: "processed",
      attempts: 1,
      processed_at: "2026-04-18T12:01:00Z",
      last_error: "",
      metadata: {}
    }
  ];

  await page.route(`${apiBaseURL}/api/v1/page-state/enterprise`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          identity_providers: [],
          integrations: [],
          webhook_registrations: {},
          outbox_events: outboxEvents
        }
      })
    });
  });

  await page.route(new RegExp(`${apiBaseURL}/api/v1/outbox-events/[^/]+/retry$`), async (route) => {
    const match = route.request().url().match(/\/api\/v1\/outbox-events\/([^/]+)\/retry$/);
    const outboxEventID = decodeURIComponent(match?.[1] || "");
    outboxEvents = outboxEvents.map((event) => event.id === outboxEventID
      ? {
          ...event,
          status: "pending",
          next_attempt_at: undefined,
          metadata: {
            ...event.metadata,
            manual_recovery_last_action: "retry"
          }
        }
      : event);
    const updated = outboxEvents.find((event) => event.id === outboxEventID);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: updated })
    });
  });

  await page.route(new RegExp(`${apiBaseURL}/api/v1/outbox-events/[^/]+/requeue$`), async (route) => {
    const match = route.request().url().match(/\/api\/v1\/outbox-events\/([^/]+)\/requeue$/);
    const outboxEventID = decodeURIComponent(match?.[1] || "");
    outboxEvents = outboxEvents.map((event) => event.id === outboxEventID
      ? {
          ...event,
          status: "pending",
          next_attempt_at: undefined,
          metadata: {
            ...event.metadata,
            manual_recovery_last_action: "requeue"
          }
        }
      : event);
    const updated = outboxEvents.find((event) => event.id === outboxEventID);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: updated })
    });
  });

  await page.goto("/#/enterprise");

  const retryRow = page.locator("table tbody tr").filter({ hasText: "integration:integration_retry_browser" });
  const requeueRow = page.locator("table tbody tr").filter({ hasText: "webhook:delivery_requeue_browser" });
  const processedRow = page.locator("table tbody tr").filter({ hasText: "status_event:status_processed_browser" });

  await expect(retryRow.getByRole("button", { name: "Retry Now" })).toBeVisible();
  await expect(requeueRow.getByRole("button", { name: "Requeue" })).toBeVisible();
  await expect(processedRow.getByRole("button")).toHaveCount(0);

  await retryRow.getByRole("button", { name: "Retry Now" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Outbox event marked pending for immediate retry");
  await expect(retryRow).toContainText("pending");
  await expect(retryRow.getByRole("button", { name: "Retry Now" })).toHaveCount(0);

  await requeueRow.getByRole("button", { name: "Requeue" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Outbox event requeued for another dispatch attempt");
  await expect(requeueRow).toContainText("pending");
  await expect(requeueRow.getByRole("button", { name: "Requeue" })).toHaveCount(0);
});

test("enterprise outbox recovery shows failure feedback when a recovery action fails", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Outbox Failure ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Outbox Failure ${seed}`, `outbox-failure-${seed}`);

  await page.route(`${apiBaseURL}/api/v1/page-state/enterprise`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          identity_providers: [],
          integrations: [],
          webhook_registrations: {},
          outbox_events: [
            {
              id: "evt_retry_failure",
              event_type: "integration.sync.requested",
              organization_id: "org_browser",
              resource_type: "integration",
              resource_id: "integration_retry_failure",
              status: "error",
              attempts: 3,
              last_error: "temporary dispatch failure",
              metadata: { last_error_class: "temporary" }
            }
          ]
        }
      })
    });
  });

  await page.route(new RegExp(`${apiBaseURL}/api/v1/outbox-events/[^/]+/retry$`), async (route) => {
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: "simulated outbox retry failure"
        }
      })
    });
  });

  await page.goto("/#/enterprise");
  const retryRow = page.locator("table tbody tr").filter({ hasText: "integration:integration_retry_failure" });
  await retryRow.getByRole("button", { name: "Retry Now" }).click();
  await expect(page.locator("#app-feedback")).toContainText("simulated outbox retry failure");
  await expect(retryRow).toContainText("error");
  await expect(retryRow.getByRole("button", { name: "Retry Now" })).toBeVisible();
});

test("integration onboarding surfaces show advisory configuration and repository mapping", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Integrations ${seed}`, `integrations-${seed}`);
  const session = await currentSession(page);

  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Platform ${seed}`,
    slug: `platform-${seed}`
  });
  const team = await apiPostItem<any>(request, session, "/api/v1/teams", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Core ${seed}`,
    slug: `core-${seed}`
  });
  const service = await apiPostItem<any>(request, session, "/api/v1/services", {
    organization_id: session.organizationID,
    project_id: project.id,
    team_id: team.id,
    name: `Checkout ${seed}`,
    slug: `checkout-${seed}`
  });
  const environment = await apiPostItem<any>(request, session, "/api/v1/environments", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Production ${seed}`,
    slug: `prod-${seed}`,
    type: "production",
    production: true
  });
  const integrations = await apiGetList<any>(request, session, "/api/v1/integrations");
  const github = integrations.find((integration) => integration.kind === "github");
  expect(github).toBeTruthy();

  await apiPatchItem<any>(request, session, `/api/v1/integrations/${github.id}`, {
    enabled: true,
    mode: "advisory",
    metadata: {
      access_token_env: "CCP_GITHUB_TOKEN",
      webhook_secret_env: "CCP_GITHUB_WEBHOOK_SECRET"
    }
  });
  await apiPostItem<any>(request, session, `/api/v1/integrations/${github.id}/graph-ingest`, {
    repositories: [
      {
        project_id: project.id,
        name: `checkout-${seed}`,
        provider: "github",
        url: `https://github.com/acme/checkout-${seed}`,
        default_branch: "main"
      }
    ]
  });

  await page.getByRole("link", { name: "Integrations" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("Integrations");
  await expect(page.locator('select[name="mode"]').first()).toHaveValue("advisory");
  await expect(page.getByText("Stale Integrations")).toBeVisible();
  await page.locator('.integration-config-form input[name="schedule_enabled"]').first().check();
  await page.locator('.integration-config-form input[name="schedule_interval_seconds"]').first().fill("300");
  await page.locator('.integration-config-form input[name="sync_stale_after_seconds"]').first().fill("900");
  await page.locator('.integration-config-form button[type="submit"]').first().click();
  await expect(page.locator("#app-feedback")).toContainText("Integration settings saved");
  await expect(page.getByRole("heading", { name: "Repository Discovery and Mapping" }).first()).toBeVisible();
  await expect(page.getByText(`https://github.com/acme/checkout-${seed}`)).toBeVisible();

  await page.locator('.repository-map-form select[name="service_id"]').first().selectOption(service.id);
  await page.locator('.repository-map-form select[name="environment_id"]').first().selectOption(environment.id);
  await page.locator('.repository-map-form button[type="submit"]').first().click();
  await expect(page.locator("#app-feedback")).toContainText("Repository mapping saved");

  const repositoryCard = page.locator(".repository-card").filter({ hasText: `https://github.com/acme/checkout-${seed}` }).first();
  await expect(repositoryCard).toContainText(`team Core ${seed} inferred from service mapping`);
  await expect(repositoryCard).toContainText("service: manual");
  await expect(repositoryCard).toContainText("environment: manual");

  await page.getByRole("link", { name: "System Graph" }).click();
  await expect(page.locator(".topbar h2")).toHaveText("System Graph");
  await expect(page.locator("body")).toContainText("team_repository_owner");
  await expect(page.locator("body")).toContainText("inferred_owner");
  await expect(page.locator("body")).toContainText(`Core ${seed}`);
  await expect(page.locator("body")).toContainText(`checkout-${seed}`);
});

test("integration operator controls exercise connection test, sync, and discovered-resource mapping with route-local refresh", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Runtime ${seed}`, `runtime-${seed}`);
  const session = await currentSession(page);

  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Platform ${seed}`,
    slug: `platform-${seed}`
  });
  const team = await apiPostItem<any>(request, session, "/api/v1/teams", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Core ${seed}`,
    slug: `core-${seed}`
  });
  const service = await apiPostItem<any>(request, session, "/api/v1/services", {
    organization_id: session.organizationID,
    project_id: project.id,
    team_id: team.id,
    name: `Checkout ${seed}`,
    slug: `checkout-${seed}`
  });
  const environment = await apiPostItem<any>(request, session, "/api/v1/environments", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Production ${seed}`,
    slug: `prod-${seed}`,
    type: "production",
    production: true
  });
  const integration = await apiPostItem<any>(request, session, "/api/v1/integrations", {
    organization_id: session.organizationID,
    kind: "kubernetes",
    name: `Kubernetes ${seed}`,
    instance_key: `k8s-${seed}`,
    scope_type: "environment",
    scope_name: `Production ${seed}`,
    mode: "advisory"
  });
  await apiPatchItem<any>(request, session, `/api/v1/integrations/${integration.id}`, {
    enabled: true,
    schedule_enabled: true,
    schedule_interval_seconds: 300,
    sync_stale_after_seconds: 900,
    metadata: {
      api_base_url: "https://cluster.example.com",
      namespace: `prod-${seed}`,
      deployment_name: `checkout-${seed}`,
      bearer_token_env: "CCP_KUBE_TOKEN"
    }
  });

  let syntheticRuns: any[] = [];
  let syntheticResource: any = {
    id: `dr_${seed}`,
    organization_id: session.organizationID,
    integration_id: integration.id,
    project_id: project.id,
    service_id: "",
    environment_id: "",
    repository_id: "",
    resource_type: "kubernetes_workload",
    provider: "kubernetes",
    external_id: `prod-${seed}/checkout-${seed}`,
    name: `checkout-workload-${seed}`,
    namespace: `prod-${seed}`,
    health: "healthy",
    status: "candidate",
    summary: "Observed checkout workload",
    last_seen_at: "2026-04-19T12:00:00Z",
    metadata: {
      ownership: { status: "not_found" },
      mapping_provenance: {
        service: { source: "manual" },
        environment: { source: "manual" }
      }
    }
  };

  await page.route(`${apiBaseURL}/api/v1/page-state/integrations`, async (route) => {
    const response = await route.fetch();
    const payload = await response.json();
    payload.data.discovered_resources = [
      ...(payload.data.discovered_resources || []).filter((resource: any) => resource.id !== syntheticResource.id),
      syntheticResource
    ];
    payload.data.integration_sync_runs = {
      ...(payload.data.integration_sync_runs || {}),
      [integration.id]: syntheticRuns
    };
    await route.fulfill({ response, json: payload });
  });

  await page.route(new RegExp(`${apiBaseURL}/api/v1/integrations/${integration.id}/test$`), async (route) => {
    syntheticRuns = [
      {
        id: `run_test_${seed}`,
        integration_id: integration.id,
        operation: "test",
        trigger: "manual",
        status: "succeeded",
        summary: "connection ok",
        started_at: "2026-04-19T12:05:00Z"
      },
      ...syntheticRuns
    ];
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          integration: {
            id: integration.id,
            organization_id: session.organizationID,
            name: `Kubernetes ${seed}`,
            kind: "kubernetes",
            instance_key: `k8s-${seed}`,
            mode: "advisory",
            enabled: true
          },
          status: "healthy",
          summary: "connection ok",
          details: ["provider reachable"]
        }
      })
    });
  });

  await page.route(new RegExp(`${apiBaseURL}/api/v1/integrations/${integration.id}/sync$`), async (route) => {
    syntheticRuns = [
      {
        id: `run_sync_${seed}`,
        integration_id: integration.id,
        operation: "sync",
        trigger: "manual",
        status: "succeeded",
        summary: "runtime inventory refreshed",
        started_at: "2026-04-19T12:06:00Z"
      },
      ...syntheticRuns
    ];
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          integration: {
            id: integration.id,
            organization_id: session.organizationID,
            name: `Kubernetes ${seed}`,
            kind: "kubernetes",
            instance_key: `k8s-${seed}`,
            mode: "advisory",
            enabled: true
          },
          sync_run: {
            id: `run_sync_${seed}`,
            integration_id: integration.id,
            operation: "sync",
            trigger: "manual",
            status: "succeeded",
            summary: "runtime inventory refreshed",
            started_at: "2026-04-19T12:06:00Z"
          }
        }
      })
    });
  });

  await page.route(new RegExp(`${apiBaseURL}/api/v1/discovered-resources/${syntheticResource.id}$`), async (route) => {
    const body = route.request().postDataJSON() as Record<string, string>;
    syntheticResource = {
      ...syntheticResource,
      service_id: body.service_id || "",
      environment_id: body.environment_id || "",
      repository_id: body.repository_id || "",
      status: body.status || syntheticResource.status,
      metadata: {
        ...syntheticResource.metadata,
        mapping_provenance: {
          service: { source: body.service_id ? "manual" : "unmapped" },
          environment: { source: body.environment_id ? "manual" : "unmapped" },
          repository: { source: body.repository_id ? "manual" : "unmapped" }
        }
      }
    };
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: syntheticResource })
    });
  });

  await page.getByRole("link", { name: "Integrations" }).click();
  const panel = page.locator(".integration-panel").filter({ hasText: `Kubernetes ${seed}` });
  await expect(panel).toContainText(`checkout-workload-${seed}`);

  await panel.getByRole("button", { name: "Test Read-Only Connection" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Connection test completed");
  await expect(panel).toContainText("connection ok");

  await panel.getByRole("button", { name: "Run Read-Only Sync" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Integration sync completed");
  await expect(panel).toContainText("runtime inventory refreshed");

  const resourceCard = panel.locator(".repository-card").filter({ hasText: `checkout-workload-${seed}` }).first();
  await resourceCard.locator('select[name="service_id"]').selectOption(service.id);
  await resourceCard.locator('select[name="environment_id"]').selectOption(environment.id);
  await resourceCard.locator('select[name="status"]').selectOption("mapped");
  await resourceCard.getByRole("button", { name: "Save Runtime Mapping" }).click();
  await expect(page.locator("#app-feedback")).toContainText("Discovered resource mapping saved");
  await expect(resourceCard).toContainText(`service ${service.name}`);
  await expect(resourceCard).toContainText(`environment ${environment.name}`);
  await expect(resourceCard).toContainText("mapped");
});

test("integration panels show webhook diagnostics for SCM providers", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Webhook ${seed}`, `webhook-${seed}`);

  await page.getByRole("link", { name: "Integrations" }).click();
  await expect(page.getByRole("heading", { name: "Webhook Health" }).first()).toBeVisible();
  await expect(page.getByText("Automatic registration, repair, and delivery diagnostics").first()).toBeVisible();
  await expect(page.getByRole("button", { name: /Webhook/ }).first()).toBeVisible();
});

test("multi-instance integration surfaces show instance scope and github app onboarding affordance", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Instances ${seed}`, `instances-${seed}`);

  await page.getByRole("link", { name: "Integrations" }).click();
  await page.locator('#create-integration-form select[name="kind"]').selectOption("github");
  await page.locator('#create-integration-form input[name="name"]').fill(`GitHub App ${seed}`);
  await page.locator('#create-integration-form input[name="instance_key"]').fill(`github-app-${seed}`);
  await page.locator('#create-integration-form input[name="scope_name"]').fill(`Sandbox ${seed}`);
  await page.locator('#create-integration-form select[name="auth_strategy"]').selectOption("github_app");
  await page.locator('#create-integration-form button[type="submit"]').click();

  await expect(page.locator("#app-feedback")).toContainText("Integration instance created");
  await expect(page.getByText(`GitHub App ${seed}`)).toBeVisible();
  await expect(page.getByText(`github:github-app-${seed}`)).toBeVisible();
  await expect(page.getByRole("button", { name: "Start GitHub App Install" })).toBeVisible();
});

test("gitlab integration surfaces are provider-aware and show discovered repository mapping", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `GitLab ${seed}`, `gitlab-${seed}`);
  const session = await currentSession(page);

  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Platform ${seed}`,
    slug: `platform-${seed}`
  });
  const team = await apiPostItem<any>(request, session, "/api/v1/teams", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Core ${seed}`,
    slug: `core-${seed}`
  });
  const service = await apiPostItem<any>(request, session, "/api/v1/services", {
    organization_id: session.organizationID,
    project_id: project.id,
    team_id: team.id,
    name: `Checkout ${seed}`,
    slug: `checkout-${seed}`
  });
  const environment = await apiPostItem<any>(request, session, "/api/v1/environments", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Production ${seed}`,
    slug: `prod-${seed}`,
    type: "production",
    production: true
  });

  await page.getByRole("link", { name: "Integrations" }).click();
  await page.locator('#create-integration-form select[name="kind"]').selectOption("gitlab");
  await page.locator('#create-integration-form input[name="name"]').fill(`GitLab ${seed}`);
  await page.locator('#create-integration-form input[name="instance_key"]').fill(`gitlab-${seed}`);
  await page.locator('#create-integration-form input[name="scope_name"]').fill(`Acme GitLab ${seed}`);
  await page.locator('#create-integration-form select[name="auth_strategy"]').selectOption("personal_access_token");
  await page.locator('#create-integration-form button[type="submit"]').click();

  await expect(page.locator("#app-feedback")).toContainText("Integration instance created");
  await expect(page.getByText(`gitlab:gitlab-${seed}`)).toBeVisible();

  const integrations = await apiGetList<any>(request, session, "/api/v1/integrations?kind=gitlab&instance_key=" + encodeURIComponent(`gitlab-${seed}`));
  const gitlab = integrations.find((integration) => integration.instance_key === `gitlab-${seed}`);
  expect(gitlab).toBeTruthy();

  await apiPatchItem<any>(request, session, `/api/v1/integrations/${gitlab.id}`, {
    enabled: true,
    mode: "advisory",
    auth_strategy: "personal_access_token",
    metadata: {
      api_base_url: "https://gitlab.com/api/v4",
      group: "acme",
      access_token_env: "CCP_GITLAB_TOKEN",
      webhook_secret_env: "CCP_GITLAB_WEBHOOK_SECRET"
    }
  });
  await apiPostItem<any>(request, session, `/api/v1/integrations/${gitlab.id}/graph-ingest`, {
    repositories: [
      {
        project_id: project.id,
        name: `checkout-${seed}`,
        provider: "gitlab",
        url: `https://gitlab.com/acme/checkout-${seed}`,
        default_branch: "main"
      }
    ]
  });

  await page.getByRole("button", { name: "Refresh Data" }).click();
  await expect(page.locator("#app-feedback")).toContainText("refreshed");
  const gitlabPanel = page.locator(".integration-panel").filter({ hasText: `GitLab ${seed}` });
  await expect(gitlabPanel.getByRole("heading", { name: "Repository Discovery and Mapping" })).toBeVisible();
  await expect(gitlabPanel.getByText(`https://gitlab.com/acme/checkout-${seed}`)).toBeVisible();
  await expect(gitlabPanel.getByRole("button", { name: "Start GitHub App Install" })).toHaveCount(0);

  await gitlabPanel.locator('.repository-map-form select[name="service_id"]').first().selectOption(service.id);
  await gitlabPanel.locator('.repository-map-form select[name="environment_id"]').first().selectOption(environment.id);
  await gitlabPanel.locator('.repository-map-form button[type="submit"]').first().click();
  await expect(page.locator("#app-feedback")).toContainText("Repository mapping saved");
});

test("advisory rollout surfaces clearly show recommendation-only mode", async ({ page, request }) => {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `Owner ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `Advisory ${seed}`, `advisory-${seed}`);
  const session = await currentSession(page);
  const rollout = await seedRolloutScenario(request, session, seed);

  const integrations = await apiGetList<any>(request, session, "/api/v1/integrations");
  const kubernetes = integrations.find((integration) => integration.kind === "kubernetes");
  expect(kubernetes).toBeTruthy();

  await apiPatchItem<any>(request, session, `/api/v1/integrations/${kubernetes.id}`, {
    enabled: true,
    mode: "advisory",
    control_enabled: false,
    metadata: {
      api_base_url: "https://cluster.example.com",
      namespace: "prod",
      deployment_name: "checkout"
    }
  });

  let execution = await apiPostItem<any>(request, session, "/api/v1/rollout-executions", {
    rollout_plan_id: rollout.plan.id,
    backend_type: "kubernetes",
    backend_integration_id: kubernetes.id,
    signal_provider_type: "simulated"
  });
  execution = await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "approve",
    reason: "approve advisory rollout"
  });
  execution = await apiPostItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}/advance`, {
    action: "start",
    reason: "start advisory rollout"
  });

  await page.getByRole("link", { name: "Rollout Plan" }).click();
  await expect(page.getByText("Advisory Mode: observe and recommend only")).toBeVisible();
  await expect(page.getByText("Pause, resume, and rollback are disabled here because the live backend is in advisory mode.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Reconcile and Record Recommendation" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Record Advisory Recommendation" })).toBeVisible();
  await expect(page.locator('#advance-rollout-form option[value="pause"]')).toHaveAttribute("disabled", "");
  await expect(page.locator('#advance-rollout-form option[value="resume"]')).toHaveAttribute("disabled", "");
  await expect(page.locator('#advance-rollout-form option[value="rollback"]')).toHaveAttribute("disabled", "");

  const verificationRequest = page.waitForRequest((req) =>
    req.method() === "POST" && req.url() === `${apiBaseURL}/api/v1/rollout-executions/${execution.id}/verification`
  );
  await page.locator('#record-verification-form select[name="outcome"]').selectOption("fail");
  await page.locator('#record-verification-form select[name="decision"]').selectOption("rollback");
  await page.locator('#record-verification-form textarea[name="summary"]').fill("manual advisory verification recommends rollback");
  await Promise.all([verificationRequest, page.getByRole("button", { name: "Record Advisory Recommendation" }).click()]);

  await expect(page.locator("#app-feedback")).toContainText("Advisory recommendation recorded");
  await expect(page.locator("body")).toContainText("Rollback recommended");
  await expect(page.locator("body")).toContainText("manual advisory verification recommends rollback");

  const detail = await apiGetItem<any>(request, session, `/api/v1/rollout-executions/${execution.id}`);
  expect(detail.execution.status).toBe("in_progress");
  expect(detail.verification_results.some((result: any) =>
    result.summary === "Advisory recommendation: manual advisory verification recommends rollback" &&
    result.decision === "advisory_rollback" &&
    Array.isArray(result.explanation) &&
    result.explanation.some((entry: string) => entry.includes("external deployment control is disabled"))
  )).toBeTruthy();
});

async function signUpThroughUI(page: Page, email: string, displayName: string, password: string, landing: "awaiting_access" | "dashboard" = "awaiting_access") {
  await page.goto("/");
  await page.locator("#auth-mode-signup").click();
  await page.locator("#signup-email").fill(email);
  await page.locator("#signup-display-name").fill(displayName);
  await page.locator("#signup-password").fill(password);
  await page.locator("#signup-password-confirmation").fill(password);
  await page.locator("#signup-submit").click();
  if (landing === "dashboard") {
    await expect(page.locator(".topbar h2")).toHaveText("Dashboard");
    return;
  }
  await expect(page.getByText("Waiting for organization access.")).toBeVisible();
}

async function logInThroughUI(page: Page, email: string, password: string, landing: "awaiting_access" | "dashboard" = "dashboard") {
  await page.goto("/");
  await page.locator("#login-email").fill(email);
  await page.locator("#login-password").fill(password);
  await page.locator("#login-submit").click();
  if (landing === "dashboard") {
    await expect(page.locator(".topbar h2")).toHaveText("Dashboard");
    return;
  }
  await expect(page.getByText("Waiting for organization access.")).toBeVisible();
}

async function bootstrapOrganizationForSignedInUser(page: Page, request: APIRequestContext, name: string, slug: string) {
  const session = await currentSession(page);
  await apiPostItem<any>(request, session, "/api/v1/organizations", {
    name,
    slug
  });
  await page.reload();
  await expect(page.locator(".topbar h2")).toHaveText("Dashboard");
}

async function grantOrganizationAccess(request: APIRequestContext, email: string, displayName: string, organizationSlug: string) {
  const response = await request.post(apiBaseURL + "/api/v1/auth/dev/login", {
    data: {
      email,
      display_name: displayName,
      organization_slug: organizationSlug
    }
  });
  expect(response.ok()).toBeTruthy();
}

async function currentSession(page: Page): Promise<BrowserSession> {
  const cookies = await page.context().cookies(apiBaseURL);
  const cookieHeader = cookies.map((cookie) => `${cookie.name}=${cookie.value}`).join("; ");
  return page.evaluate(() => ({
    organizationID: window.sessionStorage.getItem("ccp.organization") || window.localStorage.getItem("ccp.organization") || ""
  })).then((session) => ({
    ...session,
    cookieHeader
  }));
}

async function apiPostItem<T>(request: APIRequestContext, session: BrowserSession, path: string, body: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (session.cookieHeader) {
    headers.Cookie = session.cookieHeader;
    headers.Origin = browserOrigin;
    headers.Referer = `${browserOrigin}/`;
  }
  if (session.organizationID) {
    headers["X-CCP-Organization-ID"] = session.organizationID;
  }
  const response = await request.post(apiBaseURL + path, { data: body, headers });
  expect(response.ok(), `${path} should succeed`).toBeTruthy();
  return (await response.json()).data as T;
}

async function apiGetList<T>(request: APIRequestContext, session: BrowserSession, path: string): Promise<T[]> {
  const headers: Record<string, string> = {};
  if (session.cookieHeader) {
    headers.Cookie = session.cookieHeader;
  }
  if (session.organizationID) {
    headers["X-CCP-Organization-ID"] = session.organizationID;
  }
  const response = await request.get(apiBaseURL + path, { headers });
  expect(response.ok(), `${path} should succeed`).toBeTruthy();
  return (await response.json()).data as T[];
}

async function apiGetItem<T>(request: APIRequestContext, session: BrowserSession, path: string): Promise<T> {
  const headers: Record<string, string> = {};
  if (session.cookieHeader) {
    headers.Cookie = session.cookieHeader;
  }
  if (session.organizationID) {
    headers["X-CCP-Organization-ID"] = session.organizationID;
  }
  const response = await request.get(apiBaseURL + path, { headers });
  expect(response.ok(), `${path} should succeed`).toBeTruthy();
  return (await response.json()).data as T;
}

async function apiPatchItem<T>(request: APIRequestContext, session: BrowserSession, path: string, body: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (session.cookieHeader) {
    headers.Cookie = session.cookieHeader;
    headers.Origin = browserOrigin;
    headers.Referer = `${browserOrigin}/`;
  }
  if (session.organizationID) {
    headers["X-CCP-Organization-ID"] = session.organizationID;
  }
  const response = await request.patch(apiBaseURL + path, { data: body, headers });
  expect(response.ok(), `${path} should succeed`).toBeTruthy();
  return (await response.json()).data as T;
}

async function seedRolloutScenario(request: APIRequestContext, session: BrowserSession, seed: string) {
  const project = await apiPostItem<any>(request, session, "/api/v1/projects", {
    organization_id: session.organizationID,
    name: `Runtime ${seed}`,
    slug: `runtime-${seed}`
  });
  const team = await apiPostItem<any>(request, session, "/api/v1/teams", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Core ${seed}`,
    slug: `core-${seed}`
  });
  const service = await apiPostItem<any>(request, session, "/api/v1/services", {
    organization_id: session.organizationID,
    project_id: project.id,
    team_id: team.id,
    name: `Checkout ${seed}`,
    slug: `checkout-${seed}`,
    criticality: "mission_critical",
    customer_facing: true,
    has_slo: true,
    has_observability: true
  });
  const environment = await apiPostItem<any>(request, session, "/api/v1/environments", {
    organization_id: session.organizationID,
    project_id: project.id,
    name: `Production ${seed}`,
    slug: `prod-${seed}`,
    type: "production",
    region: "us-central1",
    production: true
  });
  const change = await apiPostItem<any>(request, session, "/api/v1/changes", {
    organization_id: session.organizationID,
    project_id: project.id,
    service_id: service.id,
    environment_id: environment.id,
    summary: `Browser rollout ${seed}`,
    change_types: ["code"],
    file_count: 4
  });
  await apiPostItem<any>(request, session, "/api/v1/risk-assessments", {
    change_set_id: change.id
  });
  const rollout = await apiPostItem<any>(request, session, "/api/v1/rollout-plans", {
    change_set_id: change.id
  });
  return { project, team, service, environment, change, plan: rollout.plan };
}

async function expectRouteLocalFailureState(
  page: Page,
  request: APIRequestContext,
  options: {
    label: string;
    routeHash: string;
    routeMatcher: Parameters<Page["route"]>[0];
    loadingText: string;
    unavailableText: string;
    failureMessage: string;
  }
) {
  const seed = uniqueSeed();
  await signUpThroughUI(page, `owner-${seed}@acme.local`, `${options.label} ${seed}`, "ChangeMe123!");
  await bootstrapOrganizationForSignedInUser(page, request, `${options.label} ${seed}`, `${options.label.toLowerCase().replaceAll(/\s+/g, "-")}-${seed}`);

  await page.route(options.routeMatcher, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "internal_error",
          message: options.failureMessage
        }
      })
    });
  });

  await page.goto(`/#/${options.routeHash}`);
  await expect(page.locator("body")).toContainText(options.loadingText);
  await expect(page.locator("body")).toContainText(options.unavailableText);
  await expect(page.locator("body")).toContainText(options.failureMessage);
}

function uniqueSeed() {
  return `${Date.now()}-${Math.floor(Math.random() * 1000)}`;
}
