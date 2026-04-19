export type RouteDefinition = {
  key: string;
  label: string;
  subtitle: string;
  nav: "primary" | "secondary";
};

export const routes: RouteDefinition[] = [
  { key: "dashboard", label: "Dashboard", subtitle: "Organization operating view", nav: "primary" },
  { key: "catalog", label: "Service Catalog", subtitle: "Services, ownership, and topology", nav: "primary" },
  { key: "service", label: "Service Detail", subtitle: "Criticality, coverage, and dependencies", nav: "primary" },
  { key: "environment", label: "Environment", subtitle: "Promotion posture and runtime context", nav: "primary" },
  { key: "change-review", label: "Change Review", subtitle: "Change set review and blast radius", nav: "primary" },
  { key: "risk", label: "Risk Assessment", subtitle: "Explainable deterministic risk scoring", nav: "primary" },
  { key: "rollout", label: "Rollout Plan", subtitle: "Progressive delivery guidance", nav: "primary" },
  { key: "deployments", label: "Deployment History", subtitle: "Release and rollout activity", nav: "primary" },
  { key: "incidents", label: "Incidents", subtitle: "Reliability and linked operational issues", nav: "primary" },
  { key: "incident-detail", label: "Incident Detail", subtitle: "Timeline and correlated change context", nav: "primary" },
  { key: "policies", label: "Policy Center", subtitle: "Governance and approval controls", nav: "secondary" },
  { key: "audit", label: "Audit Trail", subtitle: "Immutable control-plane actions", nav: "secondary" },
  { key: "integrations", label: "Integrations", subtitle: "Connected systems and adoption modes", nav: "secondary" },
  { key: "bootstrap", label: "Startup Bootstrap", subtitle: "Zero-to-production onboarding", nav: "secondary" },
  { key: "enterprise", label: "Enterprise Mode", subtitle: "Progressive brownfield adoption", nav: "secondary" },
  { key: "settings", label: "Settings", subtitle: "Administration and tenant controls", nav: "secondary" },
  { key: "graph", label: "System Graph", subtitle: "Digital twin and dependency view", nav: "secondary" },
  { key: "costs", label: "Cost Overview", subtitle: "Efficiency and cost-aware delivery", nav: "secondary" },
  { key: "simulation", label: "Simulation Lab", subtitle: "Dry runs and future scenario planning", nav: "secondary" }
];

export const defaultRoute = "dashboard";

export function getCurrentRoute(): RouteDefinition {
  const routeKey = getCurrentRouteKey();
  return routes.find((route) => route.key === routeKey) ?? routes[0];
}

export function getCurrentRouteQuery(): URLSearchParams {
  const trimmed = window.location.hash.replace(/^#\/?/, "");
  const separator = trimmed.indexOf("?");
  return new URLSearchParams(separator >= 0 ? trimmed.slice(separator + 1) : "");
}

function getCurrentRouteKey(): string {
  const trimmed = window.location.hash.replace(/^#\/?/, "");
  if (!trimmed) {
    return defaultRoute;
  }
  return trimmed.split("?")[0] || defaultRoute;
}
