package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunArtifactSafetyCheckPassesWithoutLeaks(t *testing.T) {
	t.Setenv("CCP_RELEASE_TEST_TOKEN", "ccpt_test_secret_value")

	reportPath := filepath.Join(t.TempDir(), "release-report.md")
	if err := os.WriteFile(reportPath, []byte("release evidence without secrets\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"--auto-secret-envs=false", "--secret-env", "CCP_RELEASE_TEST_TOKEN", "--path", reportPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected successful scan, got exit=%d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "without leaks") {
		t.Fatalf("expected success output, got stdout=%s", stdout.String())
	}
}

func TestRunArtifactSafetyCheckDetectsLeakWithoutPrintingSecret(t *testing.T) {
	t.Setenv("CCP_RELEASE_TEST_SECRET", "release-proof-secret-value")

	reportPath := filepath.Join(t.TempDir(), "release-report.md")
	if err := os.WriteFile(reportPath, []byte("artifact leaked release-proof-secret-value\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"--auto-secret-envs=false", "--secret-env", "CCP_RELEASE_TEST_SECRET", "--path", reportPath}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("expected leak detection failure, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "CCP_RELEASE_TEST_SECRET") || !strings.Contains(stderr.String(), reportPath) {
		t.Fatalf("expected finding output to include env name and file path, got stderr=%s", stderr.String())
	}
	if strings.Contains(stderr.String(), "release-proof-secret-value") {
		t.Fatalf("expected secret value to stay redacted, got stderr=%s", stderr.String())
	}
}

func TestRunArtifactSafetyCheckIgnoresSecretEnvPointers(t *testing.T) {
	t.Setenv("CCP_OKTA_CLIENT_SECRET_ENV", "CCP_OKTA_SECRET")

	reportPath := filepath.Join(t.TempDir(), "release-report.md")
	if err := os.WriteFile(reportPath, []byte("client secret env pointer: CCP_OKTA_SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"--path", reportPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected env-pointer reference to be ignored, got exit=%d stderr=%s", exitCode, stderr.String())
	}
}
