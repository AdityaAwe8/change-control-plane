package risk

import (
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Assess(change types.ChangeSet, service types.Service, environment types.Environment) types.RiskAssessment {
	now := time.Now().UTC()
	score := 5
	explanations := make([]string, 0, 12)

	add := func(points int, explanation string) {
		score += points
		explanations = append(explanations, fmt.Sprintf("+%d %s", points, explanation))
	}

	for _, changeType := range change.ChangeTypes {
		switch strings.ToLower(changeType) {
		case "code":
			add(6, "application code changes introduce runtime behavior changes")
		case "config":
			add(8, "configuration changes can alter environment behavior quickly")
		case "infra":
			add(15, "infrastructure changes expand operational impact")
		case "schema":
			add(16, "schema changes can introduce compatibility and rollback risk")
		case "iam":
			add(18, "IAM changes can affect access and privilege boundaries")
		case "secret":
			add(14, "secret changes can disrupt connectivity and runtime access")
		case "dependency":
			add(9, "dependency changes can introduce compatibility regressions")
		}
	}

	if environment.Production {
		add(20, "production environment increases blast radius and customer exposure")
	} else {
		switch strings.ToLower(environment.Type) {
		case "staging":
			add(8, "staging change still matters because it gates production confidence")
		case "preview", "ephemeral":
			add(3, "ephemeral environments have contained but non-zero operational cost")
		}
	}

	switch strings.ToLower(service.Criticality) {
	case "high":
		add(12, "service is marked high criticality")
	case "critical", "mission_critical":
		add(18, "service is mission critical")
	case "medium":
		add(6, "service has moderate criticality")
	}

	if change.FileCount > 0 {
		points := min(12, change.FileCount/5+1)
		add(points, fmt.Sprintf("change touches %d files", change.FileCount))
	}

	if change.ResourceCount > 0 {
		points := min(10, change.ResourceCount*2)
		add(points, fmt.Sprintf("change touches %d infrastructure or runtime resources", change.ResourceCount))
	}

	if change.TouchesInfrastructure {
		add(10, "change affects infrastructure state")
	}
	if change.TouchesIAM {
		add(16, "change affects IAM or permissions")
	}
	if change.TouchesSecrets {
		add(14, "change affects secrets or secret references")
	}
	if change.TouchesSchema {
		add(14, "change affects database schema or data compatibility")
	}
	if change.DependencyChanges {
		add(7, "change includes dependency updates")
	}
	if service.CustomerFacing {
		add(8, "service is customer-facing")
	}
	if change.HistoricalIncidentCount > 0 {
		points := min(12, change.HistoricalIncidentCount*2)
		add(points, fmt.Sprintf("service has %d historical incidents", change.HistoricalIncidentCount))
	}
	if change.PoorRollbackHistory {
		add(10, "service has poor rollback history")
	}
	if !service.HasObservability {
		add(9, "service lacks observability coverage")
	}
	if !service.HasSLO {
		add(7, "service lacks SLO coverage")
	}
	if service.RegulatedZone || environment.ComplianceZone != "" {
		add(12, "regulated zone handling requires additional control")
	}

	level := classify(score)
	approval := recommendedApproval(level, environment.Production, service.RegulatedZone || environment.ComplianceZone != "")
	strategy := recommendedStrategy(level, environment, service)
	window := recommendedWindow(level, environment)
	guardrails := recommendedGuardrails(level, change, service, environment)
	blastRadius := buildBlastRadius(change, service, environment)

	confidence := 0.9
	if !service.HasObservability || !service.HasSLO {
		confidence -= 0.15
	}
	if change.HistoricalIncidentCount > 2 {
		confidence -= 0.05
	}
	if confidence < 0.45 {
		confidence = 0.45
	}

	return types.RiskAssessment{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("risk"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:              change.OrganizationID,
		ProjectID:                   change.ProjectID,
		ChangeSetID:                 change.ID,
		ServiceID:                   change.ServiceID,
		EnvironmentID:               change.EnvironmentID,
		Score:                       score,
		Level:                       level,
		ConfidenceScore:             confidence,
		Explanation:                 explanations,
		BlastRadius:                 blastRadius,
		RecommendedApprovalLevel:    approval,
		RecommendedRolloutStrategy:  strategy,
		RecommendedDeploymentWindow: window,
		RecommendedGuardrails:       guardrails,
	}
}

func classify(score int) types.RiskLevel {
	switch {
	case score >= 80:
		return types.RiskLevelCritical
	case score >= 55:
		return types.RiskLevelHigh
	case score >= 30:
		return types.RiskLevelMedium
	default:
		return types.RiskLevelLow
	}
}

func recommendedApproval(level types.RiskLevel, production, regulated bool) string {
	switch {
	case regulated || level == types.RiskLevelCritical:
		return "change-advisory-board"
	case production && level == types.RiskLevelHigh:
		return "platform-owner"
	case level == types.RiskLevelMedium:
		return "team-lead"
	default:
		return "self-serve"
	}
}

func recommendedStrategy(level types.RiskLevel, environment types.Environment, service types.Service) string {
	if !environment.Production && level == types.RiskLevelLow {
		return "direct-deploy"
	}
	switch level {
	case types.RiskLevelCritical:
		return "phased-rollout"
	case types.RiskLevelHigh:
		if service.CustomerFacing {
			return "canary"
		}
		return "phased-rollout"
	case types.RiskLevelMedium:
		return "canary"
	default:
		return "direct-deploy"
	}
}

func recommendedWindow(level types.RiskLevel, environment types.Environment) string {
	if !environment.Production {
		return "team-defined"
	}
	switch level {
	case types.RiskLevelCritical:
		return "off-hours-required"
	case types.RiskLevelHigh:
		return "off-hours-preferred"
	default:
		return "business-hours-allowed"
	}
}

func recommendedGuardrails(level types.RiskLevel, change types.ChangeSet, service types.Service, environment types.Environment) []string {
	guards := []string{"health-check-gates", "error-budget-monitoring"}
	if level == types.RiskLevelHigh || level == types.RiskLevelCritical {
		guards = append(guards, "manual-rollback-ready", "canary-metric-checks")
	}
	if change.TouchesSchema {
		guards = append(guards, "schema-compatibility-check", "rollback-database-plan")
	}
	if change.TouchesIAM || change.TouchesSecrets {
		guards = append(guards, "security-review", "access-regression-check")
	}
	if service.RegulatedZone || environment.ComplianceZone != "" {
		guards = append(guards, "evidence-capture", "regulated-signal-review")
	}
	if !service.HasObservability || !service.HasSLO {
		guards = append(guards, "operator-observability-watch")
	}
	return guards
}

func buildBlastRadius(change types.ChangeSet, service types.Service, environment types.Environment) types.BlastRadius {
	servicesImpacted := 1 + max(0, service.DependentServicesCount)
	resourcesImpacted := max(1, change.ResourceCount)
	journeys := []string{}
	if service.CustomerFacing {
		journeys = append(journeys, "primary-customer-path")
	}
	if environment.Production {
		journeys = append(journeys, "production-traffic")
	}
	scope := "contained"
	switch {
	case environment.Production && service.CustomerFacing && service.DependentServicesCount >= 2:
		scope = "broad"
	case environment.Production || service.DependentServicesCount > 0:
		scope = "moderate"
	}

	summaryParts := []string{
		fmt.Sprintf("%d service(s) impacted", servicesImpacted),
		fmt.Sprintf("%d resource(s) touched", resourcesImpacted),
	}
	if environment.Production {
		summaryParts = append(summaryParts, "production exposure present")
	}
	if service.CustomerFacing {
		summaryParts = append(summaryParts, "customer-facing surface included")
	}
	if service.RegulatedZone || environment.ComplianceZone != "" {
		summaryParts = append(summaryParts, "regulated controls in scope")
	}

	return types.BlastRadius{
		Scope:                scope,
		ServicesImpacted:     servicesImpacted,
		ResourcesImpacted:    resourcesImpacted,
		CustomerJourneys:     journeys,
		RegulatedSystems:     service.RegulatedZone || environment.ComplianceZone != "",
		ProductionImpact:     environment.Production,
		CustomerFacingImpact: service.CustomerFacing,
		Summary:              strings.Join(summaryParts, ", "),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
