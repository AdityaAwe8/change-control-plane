package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		*f = append(*f, trimmed)
	}
	return nil
}

type secretValue struct {
	Name  string
	Value string
}

type finding struct {
	Path    string
	EnvName string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("artifact-safety-check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var paths stringListFlag
	var secretEnvNames stringListFlag
	autoSecretEnvs := fs.Bool("auto-secret-envs", true, "scan current environment for secret-like env names")
	fs.Var(&paths, "path", "path to scan (repeatable; files and directories supported)")
	fs.Var(&secretEnvNames, "secret-env", "explicit secret env var to scan for (repeatable)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	paths = append(paths, fs.Args()...)
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "at least one --path or positional path is required")
		return 1
	}

	files, err := expandPaths(paths)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "no files found to scan")
		return 1
	}

	secrets := collectSecrets(secretEnvNames, *autoSecretEnvs)
	findings, err := scanFiles(files, secrets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(findings) > 0 {
		for _, finding := range findings {
			fmt.Fprintf(stderr, "detected secret-backed value from %s in %s\n", finding.EnvName, finding.Path)
		}
		return 1
	}

	fmt.Fprintf(stdout, "scanned %d files and checked %d secret env values without leaks\n", len(files), len(secrets))
	return 0
}

func expandPaths(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	files := make([]string, 0, len(paths))
	for _, rawPath := range paths {
		cleaned := filepath.Clean(rawPath)
		info, err := os.Stat(cleaned)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if _, ok := seen[cleaned]; !ok {
				seen[cleaned] = struct{}{}
				files = append(files, cleaned)
			}
			continue
		}
		err = filepath.WalkDir(cleaned, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			candidate := filepath.Clean(path)
			if _, ok := seen[candidate]; ok {
				return nil
			}
			seen[candidate] = struct{}{}
			files = append(files, candidate)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func collectSecrets(explicit []string, auto bool) []secretValue {
	byName := map[string]string{}
	for _, name := range explicit {
		value := os.Getenv(name)
		if isSecretValueCandidate(value) {
			byName[name] = value
		}
	}
	if auto {
		for _, env := range os.Environ() {
			name, value, ok := strings.Cut(env, "=")
			if !ok || !isSensitiveEnvName(name) || !isSecretValueCandidate(value) {
				continue
			}
			byName[name] = value
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	secrets := make([]secretValue, 0, len(names))
	for _, name := range names {
		secrets = append(secrets, secretValue{Name: name, Value: byName[name]})
	}
	return secrets
}

func isSensitiveEnvName(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" {
		return false
	}
	for _, suffix := range []string{"_PATH", "_FILE", "_DIR", "_URL", "_HOST", "_PORT", "_ID", "_PREFIX", "_ENDPOINT", "_ENV"} {
		if strings.HasSuffix(upper, suffix) {
			return false
		}
	}
	return strings.Contains(upper, "PRIVATE_KEY") ||
		strings.HasSuffix(upper, "_TOKEN") ||
		strings.Contains(upper, "_TOKEN_") ||
		strings.HasSuffix(upper, "_SECRET") ||
		strings.Contains(upper, "_SECRET_") ||
		strings.HasSuffix(upper, "_PASSWORD") ||
		strings.Contains(upper, "_PASSWORD_") ||
		strings.HasSuffix(upper, "_COOKIE") ||
		strings.Contains(upper, "_COOKIE_")
}

func isSecretValueCandidate(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if !strings.ContainsAny(trimmed, "\r\n") && len(trimmed) < 8 {
		return false
	}
	return true
}

func scanFiles(paths []string, secrets []secretValue) ([]finding, error) {
	findings := make([]finding, 0)
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, secret := range secrets {
			if bytes.Contains(body, []byte(secret.Value)) {
				findings = append(findings, finding{Path: path, EnvName: secret.Name})
			}
		}
	}
	return findings, nil
}
