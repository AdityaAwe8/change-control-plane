package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Client struct {
	enabled    bool
	executable string
	scriptPath string
}

type RiskAugmentation struct {
	NormalizedFactors        map[string]float64 `json:"normalized_factors"`
	ConfidenceAdjustment     float64            `json:"confidence_adjustment"`
	SupplementalExplanations []string           `json:"supplemental_explanations"`
	RecommendedGuardrails    []string           `json:"recommended_guardrails"`
	ChangeCluster            string             `json:"change_cluster"`
	HistoricalPattern        types.Metadata     `json:"historical_pattern"`
}

type RolloutSimulation struct {
	RecommendedNextAction string         `json:"recommended_next_action"`
	RiskHotspots          []string       `json:"risk_hotspots"`
	TimelineNotes         []string       `json:"timeline_notes"`
	VerificationFocus     []string       `json:"verification_focus"`
	Metadata              types.Metadata `json:"metadata"`
}

func NewClient(cfg common.Config) *Client {
	scriptPath := filepath.Join(cfg.PythonWorkspace, "intelligence_cli.py")
	if abs, err := filepath.Abs(scriptPath); err == nil {
		scriptPath = abs
	}
	return &Client{
		enabled:    cfg.EnablePythonIntelligence,
		executable: cfg.PythonExecutable,
		scriptPath: scriptPath,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

func (c *Client) AugmentRisk(ctx context.Context, change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment) (RiskAugmentation, error) {
	var result RiskAugmentation
	if !c.Enabled() {
		return result, nil
	}
	payload := map[string]any{
		"change":      change,
		"service":     service,
		"environment": environment,
		"assessment":  assessment,
	}
	if err := c.run(ctx, "risk-augment", payload, &result); err != nil {
		return RiskAugmentation{}, err
	}
	return result, nil
}

func (c *Client) SimulateRollout(ctx context.Context, change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment, plan types.RolloutPlan) (RolloutSimulation, error) {
	var result RolloutSimulation
	if !c.Enabled() {
		return result, nil
	}
	payload := map[string]any{
		"change":      change,
		"service":     service,
		"environment": environment,
		"assessment":  assessment,
		"plan":        plan,
	}
	if err := c.run(ctx, "rollout-simulate", payload, &result); err != nil {
		return RolloutSimulation{}, err
	}
	return result, nil
}

func (c *Client) run(ctx context.Context, command string, payload any, target any) error {
	if c.executable == "" {
		return fmt.Errorf("python intelligence executable is not configured")
	}
	if _, err := os.Stat(c.scriptPath); err != nil {
		return fmt.Errorf("python intelligence script unavailable at %s: %w", c.scriptPath, err)
	}

	stdin := &bytes.Buffer{}
	if err := json.NewEncoder(stdin).Encode(payload); err != nil {
		return err
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := exec.CommandContext(ctx, c.executable, c.scriptPath, command)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if workspace := filepath.Dir(c.scriptPath); workspace != "" {
		cmd.Dir = workspace
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python intelligence %s failed: %w: %s", command, err, stderr.String())
	}
	if err := json.NewDecoder(stdout).Decode(target); err != nil {
		return fmt.Errorf("python intelligence %s returned invalid json: %w", command, err)
	}
	return nil
}
