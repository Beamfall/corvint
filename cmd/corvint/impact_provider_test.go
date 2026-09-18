package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Beamfall/corvint/internal/contextindex"
)

func providerRecord(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "extevidence", "testdata", "mock-provider.json"))
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(affectedGit(t, root, "rev-parse", "HEAD"))
	data = bytes.Replace(data, []byte(strings.Repeat("0", 40)), []byte(head), 1)
	path := filepath.Join(t.TempDir(), "mock-provider.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImpactProviderFlagParsing(t *testing.T) {
	t.Parallel()
	parsed, err := parseImpactArgumentsForPlatform(options{impactLimit: 10}, []string{"--provider", "a.json", "--provider=b.json", "pkg/main.go"}, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(parsed.impactProviders, ",") != "a.json,b.json" {
		t.Fatalf("providers = %v", parsed.impactProviders)
	}
	refusals := map[string][]string{
		"missing value":        {"pkg/main.go", "--provider"},
		"empty inline value":   {"--provider=", "pkg/main.go"},
		"fifth provider":       {"--provider", "1", "--provider", "2", "--provider", "3", "--provider", "4", "--provider", "5", "pkg/main.go"},
		"with --base":          {"--provider", "a.json", "--base", strings.Repeat("a", 40)},
		"with working tree":    {"--provider", "a.json", "--working-tree-untracked", "pkg/main.go"},
		"still requires paths": {"--provider", "a.json"},
	}
	for name, arguments := range refusals {
		if _, err := parseImpactArgumentsForPlatform(options{impactLimit: 10}, arguments, "darwin"); err == nil {
			t.Errorf("%s: expected an argument error", name)
		}
	}
}

func TestImpactProviderSectionSeparation(t *testing.T) {
	t.Parallel()
	root := impactCLIRepository(t)
	record := providerRecord(t, root)
	code, plain, stderr := runCLI(t, "--root", root, "impact", "pkg/main.go")
	if code != 0 || stderr != "" {
		t.Fatalf("plain impact: exit %d stderr %q", code, stderr)
	}
	code, withProvider, stderr := runCLI(t, "--root", root, "impact", "--provider", record, "pkg/main.go")
	if code != 0 || stderr != "" {
		t.Fatalf("impact --provider: exit %d stderr %q", code, stderr)
	}
	var plainPayload, providerPayload map[string]any
	if err := json.Unmarshal([]byte(plain), &plainPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(withProvider), &providerPayload); err != nil {
		t.Fatal(err)
	}
	plainContext := plainPayload["context"].(map[string]any)
	providerContext := providerPayload["context"].(map[string]any)
	if _, present := plainContext["external"]; present {
		t.Fatal("external must be absent without --provider")
	}
	external, ok := providerContext["external"].(map[string]any)
	if !ok {
		t.Fatalf("external section missing: %s", withProvider)
	}
	delete(providerContext, "external")
	left, _ := contextindex.CanonicalJSON(plainPayload)
	right, _ := contextindex.CanonicalJSON(providerPayload)
	if !bytes.Equal(left, right) {
		t.Fatalf("core receipt must be byte-identical:\n%s\n%s", left, right)
	}
	provider := external["providers"].([]any)[0].(map[string]any)
	if provider["state"] != "loaded" || provider["freshness"] != "equal" || provider["id"] != "mockdocs" {
		t.Fatalf("provider row = %v", provider)
	}
	results := external["results"].([]any)
	if len(results) != 1 || results[0].(map[string]any)["entity"] != "mockdocs:cap-stable-value" {
		t.Fatalf("results = %v", results)
	}
	if results[0].(map[string]any)["authority"] != "external-provider" {
		t.Fatalf("Core must assign the external-provider authority: %v", results[0])
	}
	if got := len(external["verification"].([]any)); got != 1 {
		t.Fatalf("expected one verification relation from pkg/main_test.go, got %d", got)
	}
	for _, entry := range plainContext["results"].([]any) {
		if strings.HasPrefix(entry.(map[string]any)["id"].(string), "mockdocs:") {
			t.Fatal("external entities must never enter core results")
		}
	}
	// A missing record is structured, not fatal (EEP-V0-005).
	code, degraded, stderr := runCLI(t, "--root", root, "impact", "--provider", filepath.Join(t.TempDir(), "absent.json"), "pkg/main.go")
	if code != 0 || stderr != "" || !strings.Contains(degraded, `"state":"unavailable"`) {
		t.Fatalf("missing record: exit %d stderr %q stdout %s", code, stderr, degraded)
	}
}

func TestImpactProviderReadOnly(t *testing.T) {
	t.Parallel()
	root := impactCLIRepository(t)
	record := providerRecord(t, root)
	before, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	status := affectedGit(t, root, "status", "--porcelain")
	code, stdout, stderr := runCLI(t, "--root", root, "impact", "--provider", record, "pkg/main.go", "pkg/main_test.go")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d stderr %q", code, stderr)
	}
	if !strings.Contains(stdout, `"mutates":false`) {
		t.Fatalf("receipt must report mutates false: %s", stdout)
	}
	if after := affectedGit(t, root, "status", "--porcelain"); after != status {
		t.Fatalf("worktree status changed:\n%s\n%s", status, after)
	}
	after, err := os.ReadFile(record)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("provider record bytes must be untouched")
	}
}
