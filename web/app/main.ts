import "../styles/app.css";

import { getCurrentRoute } from "./router";
import { renderShell } from "../components/shell";
import {
  advanceRolloutExecution,
  archiveEnvironment,
  archiveService,
  clearSession,
  createEnvironment,
  createProject,
  createRolloutExecution,
  createService,
  createServiceAccount,
  getStoredOrganizationID,
  issueServiceAccountToken,
  loadControlPlaneState,
  loginDev,
  recordVerificationResult,
  revokeServiceAccountToken,
  setActiveOrganization
} from "../lib/api";

const app = document.querySelector<HTMLDivElement>("#app");

if (!app) {
  throw new Error("Application root not found");
}

const root = app;

async function render() {
  const state = await loadControlPlaneState();
  const route = getCurrentRoute();
  root.innerHTML = renderShell(state, route);

  const loginForm = document.querySelector<HTMLFormElement>("#login-form");
  loginForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const email = (document.querySelector<HTMLInputElement>("#login-email")?.value || "").trim();
    const displayName = (document.querySelector<HTMLInputElement>("#login-display-name")?.value || "").trim();
    const organizationName = (document.querySelector<HTMLInputElement>("#login-organization-name")?.value || "").trim();
    const organizationSlug = (document.querySelector<HTMLInputElement>("#login-organization-slug")?.value || "").trim();

    try {
      await loginDev({
        email,
        display_name: displayName,
        organization_name: organizationName,
        organization_slug: organizationSlug
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to sign in");
    }
  });

  const organizationSwitcher = document.querySelector<HTMLSelectElement>("#organization-switcher");
  organizationSwitcher?.addEventListener("change", async (event) => {
    const target = event.target as HTMLSelectElement;
    setActiveOrganization(target.value);
    await render();
  });

  const refreshButton = document.querySelector<HTMLButtonElement>("#refresh-button");
  refreshButton?.addEventListener("click", async () => {
    if (!getStoredOrganizationID() && state.session.active_organization_id) {
      setActiveOrganization(state.session.active_organization_id);
    }
    await render();
  });

  const logoutButton = document.querySelector<HTMLButtonElement>("#logout-button");
  logoutButton?.addEventListener("click", async () => {
    clearSession();
    await render();
  });

  const createProjectForm = document.querySelector<HTMLFormElement>("#create-project-form");
  createProjectForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await createProject({
        organization_id: state.session.active_organization_id || "",
        name: readInputValue(createProjectForm, "name"),
        slug: readInputValue(createProjectForm, "slug"),
        description: readInputValue(createProjectForm, "description")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to create project");
    }
  });

  const createServiceForm = document.querySelector<HTMLFormElement>("#create-service-form");
  createServiceForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await createService({
        organization_id: state.session.active_organization_id || "",
        project_id: readInputValue(createServiceForm, "project_id"),
        team_id: readInputValue(createServiceForm, "team_id"),
        name: readInputValue(createServiceForm, "name"),
        slug: readInputValue(createServiceForm, "slug"),
        description: readInputValue(createServiceForm, "description"),
        criticality: readInputValue(createServiceForm, "criticality")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to create service");
    }
  });

  const archiveServiceButton = document.querySelector<HTMLButtonElement>("#archive-service-button");
  archiveServiceButton?.addEventListener("click", async () => {
    const serviceID = archiveServiceButton.dataset.serviceId || "";
    if (!serviceID) {
      return;
    }
    try {
      await archiveService(serviceID);
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to archive service");
    }
  });

  const createEnvironmentForm = document.querySelector<HTMLFormElement>("#create-environment-form");
  createEnvironmentForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await createEnvironment({
        organization_id: state.session.active_organization_id || "",
        project_id: readInputValue(createEnvironmentForm, "project_id"),
        name: readInputValue(createEnvironmentForm, "name"),
        slug: readInputValue(createEnvironmentForm, "slug"),
        type: readInputValue(createEnvironmentForm, "type"),
        region: readInputValue(createEnvironmentForm, "region"),
        production: readCheckboxValue(createEnvironmentForm, "production")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to create environment");
    }
  });

  const archiveEnvironmentButton = document.querySelector<HTMLButtonElement>("#archive-environment-button");
  archiveEnvironmentButton?.addEventListener("click", async () => {
    const environmentID = archiveEnvironmentButton.dataset.environmentId || "";
    if (!environmentID) {
      return;
    }
    try {
      await archiveEnvironment(environmentID);
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to archive environment");
    }
  });

  const createServiceAccountForm = document.querySelector<HTMLFormElement>("#create-service-account-form");
  createServiceAccountForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await createServiceAccount({
        organization_id: state.session.active_organization_id || "",
        name: readInputValue(createServiceAccountForm, "name"),
        description: readInputValue(createServiceAccountForm, "description"),
        role: readInputValue(createServiceAccountForm, "role")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to create service account");
    }
  });

  document.querySelectorAll<HTMLFormElement>(".issue-token-form").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const serviceAccountID = form.dataset.serviceAccountId || "";
      try {
        const result = await issueServiceAccountToken(serviceAccountID, {
          name: readInputValue(form, "name")
        });
        window.alert(`Copy this token now:\n\n${result.token}`);
        await render();
      } catch (error) {
        window.alert(error instanceof Error ? error.message : "Unable to issue token");
      }
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".revoke-token-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const serviceAccountID = button.dataset.serviceAccountId || "";
      const tokenID = button.dataset.tokenId || "";
      try {
        await revokeServiceAccountToken(serviceAccountID, tokenID);
        await render();
      } catch (error) {
        window.alert(error instanceof Error ? error.message : "Unable to revoke token");
      }
    });
  });

  const createRolloutExecutionForm = document.querySelector<HTMLFormElement>("#create-rollout-execution-form");
  createRolloutExecutionForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await createRolloutExecution({
        rollout_plan_id: readInputValue(createRolloutExecutionForm, "rollout_plan_id")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to create rollout execution");
    }
  });

  const advanceRolloutForm = document.querySelector<HTMLFormElement>("#advance-rollout-form");
  advanceRolloutForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await advanceRolloutExecution(readInputValue(advanceRolloutForm, "execution_id"), {
        action: readInputValue(advanceRolloutForm, "action"),
        reason: readInputValue(advanceRolloutForm, "reason")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to advance rollout");
    }
  });

  const recordVerificationForm = document.querySelector<HTMLFormElement>("#record-verification-form");
  recordVerificationForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await recordVerificationResult(readInputValue(recordVerificationForm, "execution_id"), {
        outcome: readInputValue(recordVerificationForm, "outcome"),
        decision: readInputValue(recordVerificationForm, "decision"),
        summary: readInputValue(recordVerificationForm, "summary")
      });
      await render();
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Unable to record verification");
    }
  });
}

window.addEventListener("hashchange", () => {
  void render();
});

void render();

function readInputValue(form: HTMLFormElement, name: string): string {
  const field = form.elements.namedItem(name);
  if (field instanceof HTMLInputElement || field instanceof HTMLSelectElement || field instanceof HTMLTextAreaElement) {
    return field.value.trim();
  }
  return "";
}

function readCheckboxValue(form: HTMLFormElement, name: string): boolean {
  const field = form.elements.namedItem(name);
  return field instanceof HTMLInputElement && field.checked;
}
