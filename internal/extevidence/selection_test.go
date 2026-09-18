package extevidence

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Beamfall/corvint/internal/contextindex"
)

type selectionCase struct {
	Name         string   `json:"name"`
	Record       string   `json:"record"`
	Profile      string   `json:"profile"`
	Changed      []string `json:"changed"`
	Worktree     []string `json:"worktree"`
	Incomplete   []string `json:"incomplete"`
	Bind         []string `json:"bind"`
	State        string   `json:"state"`
	Codes        []string `json:"codes"`
	Selected     []string `json:"selected"`
	Relevant     []string `json:"relevant"`
	SafeToNarrow bool     `json:"safe_to_narrow"`
}

func selectionCases(t *testing.T) []selectionCase {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "conformance-selection", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Cases []selectionCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Cases
}

// selectionRecord reads one fixture: "v0" is the V0 mock record, "" names a
// source that does not exist, anything else is a V1 selection fixture.
func selectionRecord(t *testing.T, p pair, name string) string {
	t.Helper()
	dir := t.TempDir()
	switch name {
	case "":
		return filepath.Join(dir, "absent.json")
	case "v0":
		return writeRecord(t, dir, "provider.json", fixture(t, p.app.head))
	}
	data, err := os.ReadFile(filepath.Join("testdata", "conformance-selection", name))
	if err != nil {
		t.Fatal(err)
	}
	values := with(with(p.values(), "APP_FIRST", p.app.first), "E2E_FIRST", p.e2e.first)
	for key, value := range values {
		data = bytes.ReplaceAll(data, []byte("{{"+key+"}}"), []byte(value))
	}
	return writeRecord(t, dir, "provider.json", data)
}

func selectionInput(c selectionCase) SelectionInput {
	return SelectionInput{Changed: c.Changed, Worktree: c.Worktree, Incomplete: c.Incomplete, Profile: c.Profile, Limit: 20}
}

func runSelection(t *testing.T, p pair, source string, checkouts []Checkout, input SelectionInput) (map[string]any, []byte) {
	t.Helper()
	out, err := contextindex.CanonicalJSON(Selection(context.Background(), p.app.root, p.app.head, []string{source}, checkouts, input))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded, out
}

func bindCase(p pair, c selectionCase) []Checkout {
	var checkouts []Checkout
	for _, id := range c.Bind {
		checkouts = append(checkouts, Checkout{ID: id, Source: p.e2e.root})
	}
	return checkouts
}

// selectedTests lists every selected row as repository:path; a V0 row has no
// repository member.
func selectedTests(selection map[string]any) []string {
	var out []string
	for _, raw := range selection["selected"].([]any) {
		test := raw.(map[string]any)["test"].(map[string]any)
		repository, _ := test["repository"].(string)
		out = append(out, repository+":"+test["path"].(string))
	}
	sort.Strings(out)
	return out
}

// selectionCodes is every reason code the advice reports anywhere.
func selectionCodes(selection map[string]any) map[string]bool {
	codes := map[string]bool{}
	for _, list := range []string{"blocking_reasons", "candidates", "unknowns"} {
		for _, raw := range selection[list].([]any) {
			if code, ok := raw.(map[string]any)["code"].(string); ok {
				codes[code] = true
			}
		}
	}
	for _, list := range []string{"uncovered_paths", "uncovered_entities"} {
		for _, raw := range selection[list].([]any) {
			for _, code := range raw.(map[string]any)["reasons"].([]any) {
				codes[code.(string)] = true
			}
		}
	}
	return codes
}

// TestSelectionConformance runs every labelled fixture (ETS-V0-003..006,
// ETS-V0-008, ETS-V0-011).
func TestSelectionConformance(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	for _, c := range selectionCases(t) {
		selection, _ := runSelection(t, p, selectionRecord(t, p, c.Record), bindCase(p, c), selectionInput(c))
		if selection["state"] != c.State {
			t.Errorf("%s: state %v, want %s (reason %v, blocking %v, uncovered %v %v)", c.Name, selection["state"], c.State,
				selection["state_reason"], selection["blocking_reasons"], selection["uncovered_paths"], selection["uncovered_entities"])
		}
		codes := selectionCodes(selection)
		for _, code := range c.Codes {
			if !codes[code] {
				t.Errorf("%s: reason %s missing from %v", c.Name, code, codes)
			}
		}
		if got := selectedTests(selection); strings.Join(got, ",") != strings.Join(c.Selected, ",") {
			t.Errorf("%s: selected %v, want %v", c.Name, got, c.Selected)
		}
		for _, raw := range selection["candidates"].([]any) {
			if row := raw.(map[string]any); row["code"] == "" || row["reason"] == "" {
				t.Errorf("%s: an excluded relation must carry a code and a reason: %v", c.Name, row)
			}
		}
	}
}

// TestSelectionEvaluation reports the ETS-V0-012 measures over the labelled
// fixtures and fails on any unsafe narrowing.
func TestSelectionEvaluation(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	var selected, relevant, unsafe, unsafeDenominator, abstain, abstainCorrect, largest int
	var latencies []time.Duration
	for _, c := range selectionCases(t) {
		source, checkouts := selectionRecord(t, p, c.Record), bindCase(p, c)
		started := time.Now()
		selection, out := runSelection(t, p, source, checkouts, selectionInput(c))
		latencies = append(latencies, time.Since(started))
		largest = max(largest, len(out))
		labelled := map[string]bool{}
		for _, test := range c.Relevant {
			labelled[test] = true
		}
		for _, test := range selectedTests(selection) {
			selected++
			if labelled[test] {
				relevant++
			}
		}
		if !c.SafeToNarrow {
			unsafeDenominator++
			if selection["state"] == SelectionNarrow {
				unsafe++
			}
		}
		if c.State == SelectionUnknown || c.State == SelectionBlocked {
			abstain++
			if selection["state"] == c.State {
				abstainCorrect++
			}
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	precision := float64(relevant) / float64(max(selected, 1))
	t.Logf("cases=%d precision=%.3f (%d/%d) unsafe-narrowing=%d/%d abstention-accuracy=%d/%d latency-p50=%s latency-max=%s receipt-max-bytes=%d",
		len(latencies), precision, relevant, selected, unsafe, unsafeDenominator, abstainCorrect, abstain,
		latencies[len(latencies)/2], latencies[len(latencies)-1], largest)
	if unsafe != 0 || relevant != selected || abstainCorrect != abstain {
		t.Fatalf("unsafe narrowing %d, precision %d/%d, abstention %d/%d", unsafe, relevant, selected, abstainCorrect, abstain)
	}
}

// TestSelectionRowProvenance checks the ETS-V0-007 members of a selected row.
func TestSelectionRowProvenance(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	c := selectionCases(t)[0]
	selection, _ := runSelection(t, p, selectionRecord(t, p, c.Record), bindCase(p, c), selectionInput(c))
	for _, raw := range selection["selected"].([]any) {
		row := raw.(map[string]any)
		for _, member := range []string{"authority", "confidence", "provider", "provider_revision", "entity", "entity_kind", "test", "relation", "relation_type", "evidence", "verification", "limitations", "identity", "binding", "freshness", "test_revision", "source_revision", "relation_state", "crosses_repositories", "reason"} {
			if _, present := row[member]; !present {
				t.Errorf("selected row lacks %s: %v", member, row)
			}
		}
		if row["relation_state"] != RelationFresh || row["confidence"] != confidenceUnscored {
			t.Errorf("a selected row must be fresh and unscored: %v", row)
		}
	}
	crossing := selection["selected"].([]any)[1].(map[string]any)
	if crossing["crosses_repositories"] != true || crossing["binding"] != BindingCheckout || crossing["test_revision"] != p.e2e.head || crossing["source_revision"] != p.app.head {
		t.Fatalf("a cross-repository row must carry each side's own revision and binding: %v", crossing)
	}
}

// TestSelectionMandatoryEchoedUnchanged: evidence can widen the obligations
// but the mandatory set is echoed verbatim whatever the state (ETS-V0-009).
func TestSelectionMandatoryEchoedUnchanged(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	mandatory := []any{map[string]any{"command": "make gate", "kind": "mandatory", "reason": "Makefile gate target", "source": "Makefile"}}
	want, _ := json.Marshal(mandatory)
	for _, c := range selectionCases(t) {
		input := selectionInput(c)
		input.Mandatory = mandatory
		selection, _ := runSelection(t, p, selectionRecord(t, p, c.Record), bindCase(p, c), input)
		if got, _ := json.Marshal(selection["mandatory"]); !bytes.Equal(got, want) {
			t.Fatalf("%s: mandatory %s, want %s", c.Name, got, want)
		}
	}
}

// TestSelectionDeterministic: repeated runs and a reordered record give the
// same bytes (ETS-V0-010).
func TestSelectionDeterministic(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	c := selectionCases(t)[0]
	source := selectionRecord(t, p, c.Record)
	_, first := runSelection(t, p, source, bindCase(p, c), selectionInput(c))
	_, second := runSelection(t, p, source, bindCase(p, c), selectionInput(c))
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	reversed := writeRecord(t, t.TempDir(), "provider.json", mutate(t, data, func(record map[string]any) {
		list := relations(record)
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
	}))
	third, _ := runSelection(t, p, reversed, bindCase(p, c), selectionInput(c))
	delete(third, "provider_evidence")
	thirdBytes, _ := contextindex.CanonicalJSON(third)
	firstMap := map[string]any{}
	_ = json.Unmarshal(first, &firstMap)
	delete(firstMap, "provider_evidence")
	firstBytes, _ := contextindex.CanonicalJSON(firstMap)
	if !bytes.Equal(first, second) || !bytes.Equal(firstBytes, thirdBytes) {
		t.Fatalf("selection must not depend on run or record order:\n%s\n%s", firstBytes, thirdBytes)
	}
}

// TestSelectionOmissionAccounting: every list is bounded and the remainder
// counted (ETS-V0-010).
func TestSelectionOmissionAccounting(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	c := selectionCases(t)[0]
	input := selectionInput(c)
	input.Limit = 1
	selection, _ := runSelection(t, p, selectionRecord(t, p, c.Record), bindCase(p, c), input)
	omitted := selection["omitted"].(map[string]any)
	if len(selection["selected"].([]any)) != 1 || omitted["selected"] != float64(1) {
		t.Fatalf("two selected rows at limit 1 must keep one and count one: %v", omitted)
	}
	if selection["state"] != SelectionNarrow {
		t.Fatalf("omission must not change the state: %v", selection["state"])
	}
}

// TestSelectionPrivate: a checkout is echoed as given, never resolved, no
// file body reaches the advice, and provider free text is named untrusted
// (ETS-V0-013).
func TestSelectionPrivate(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	c := selectionCases(t)[0]
	checkouts := []Checkout{{ID: "e2e", Source: relativeTo(t, p.app.root, p.e2e.root)}}
	selection, out := runSelection(t, p, selectionRecord(t, p, c.Record), checkouts, selectionInput(c))
	canonical, err := filepath.EvalSymlinks(p.e2e.root)
	if err != nil {
		t.Fatal(err)
	}
	if selection["state"] != SelectionNarrow {
		t.Fatalf("a relative checkout must bind as an absolute one does: %v", selection["state_reason"])
	}
	for _, secret := range []string{p.app.root, canonical, "test('account')", "package main"} {
		if bytes.Contains(out, []byte(secret)) {
			t.Fatalf("advice leaks %q", secret)
		}
	}
	if len(selection["untrusted_text_fields"].([]any)) == 0 {
		t.Fatal("rule and reference must be named untrusted text")
	}
}
