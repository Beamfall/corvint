package extevidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Beamfall/corvint/internal/contextindex"
)

// Provider states (EEP-V0-005).
const (
	StateLoaded      = "loaded"
	StateUnavailable = "unavailable"
	StateInvalid     = "invalid"
)

// UntrustedTextFields names the provider-authored free text in the section (EEP-V0-013).
var UntrustedTextFields = []string{
	"external.results[].summary", "external.results[].relation.rule", "external.results[].relation.reference",
	"external.downstream[].summary", "external.downstream[].relation.rule", "external.downstream[].relation.reference",
	"external.verification[].summary", "external.verification[].relation.rule", "external.verification[].relation.reference",
}

type provider struct {
	source, sha256, state, reason, freshness string
	record                                   Record
}

// Section reads every selected record and returns the `external` member of
// the impact receipt. It never fails: every problem is a structured entry.
func Section(ctx context.Context, index *contextindex.Index, sources, changedPaths []string, limit int) map[string]any {
	providers := make([]provider, 0, len(sources))
	for _, source := range sources {
		providers = append(providers, load(ctx, index, source))
	}
	repository := repositoryTree(ctx, index, providers)
	changed := make(map[string]struct{}, len(changedPaths))
	for _, path := range changedPaths {
		changed[path] = struct{}{}
	}
	var merged composition
	for _, entry := range providers {
		if entry.state != StateLoaded {
			continue
		}
		part := compose(entry.record, changed, repository)
		merged.results = append(merged.results, part.results...)
		merged.downstream = append(merged.downstream, part.downstream...)
		merged.verification = append(merged.verification, part.verification...)
		merged.unknowns = append(merged.unknowns, part.unknowns...)
	}
	return assemble(providers, merged, limit)
}

func load(ctx context.Context, index *contextindex.Index, source string) provider {
	entry := provider{source: source, state: StateUnavailable}
	resolved := source
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(index.Root, filepath.FromSlash(source))
	}
	info, err := os.Stat(resolved)
	if err != nil {
		entry.reason = "cannot read record: " + describe(err)
		return entry
	}
	if info.Size() > MaxRecordBytes {
		entry.state, entry.reason = StateInvalid, fmt.Sprintf("record exceeds %d bytes", MaxRecordBytes)
		return entry
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		entry.reason = "cannot read record: " + describe(err)
		return entry
	}
	digest := sha256.Sum256(data)
	entry.sha256 = hex.EncodeToString(digest[:])
	record, err := Decode(data)
	if err != nil {
		entry.state, entry.reason = StateInvalid, err.Error()
		return entry
	}
	entry.record, entry.state = record, StateLoaded
	entry.freshness = Freshness(ctx, index.Root, index.CommitRevision, record.Repository.Revision)
	entry.reason = "record decoded; freshness by Git ancestry against " + index.CommitRevision
	return entry
}

// describe strips the file path from an OS error so the reason stays bounded
// and repeats only what the caller already passed.
func describe(err error) string {
	if pathError, ok := err.(*os.PathError); ok {
		return pathError.Err.Error()
	}
	return err.Error()
}

// repositoryTree answers tracked and blob questions for every pinned path the
// loaded records name, resolving blobs the index did not read in one Git call.
func repositoryTree(ctx context.Context, index *contextindex.Index, providers []provider) tree {
	blobs := make(map[string]string)
	var pending []string
	for _, entry := range providers {
		if entry.state != StateLoaded {
			continue
		}
		for _, relation := range entry.record.Relations {
			for _, raw := range []string{relation.From, relation.To} {
				path, isPath := strings.CutPrefix(raw, "path:")
				if !isPath || relation.Blob == "" || checkPath(path) != "" {
					continue
				}
				if _, tracked := index.Tracked[path]; !tracked {
					continue
				}
				if source, read := index.Sources[path]; read {
					blobs[path] = source.BlobHash
					continue
				}
				if _, queued := blobs[path]; !queued {
					blobs[path] = ""
					pending = append(pending, path)
				}
			}
		}
	}
	sort.Strings(pending)
	for path, oid := range batchBlobs(ctx, index.Root, index.CommitRevision, pending) {
		blobs[path] = oid
	}
	return tree{
		tracked: func(path string) bool { _, ok := index.Tracked[path]; return ok },
		blob: func(path string) (string, bool) {
			oid, known := blobs[path]
			return oid, known && oid != ""
		},
	}
}

func batchBlobs(ctx context.Context, root, revision string, paths []string) map[string]string {
	resolved := make(map[string]string, len(paths))
	if len(paths) == 0 {
		return resolved
	}
	var request strings.Builder
	for _, path := range paths {
		request.WriteString(revision + ":" + path + "\n")
	}
	output, err := runGit(ctx, root, []byte(request.String()), "cat-file", "--batch-check")
	if err != nil {
		return resolved
	}
	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	for position, line := range lines {
		fields := strings.Fields(line)
		if position < len(paths) && len(fields) == 3 && fields[1] == "blob" {
			resolved[paths[position]] = fields[0]
		}
	}
	return resolved
}

func assemble(providers []provider, merged composition, limit int) map[string]any {
	rows := make([]any, 0, len(providers))
	for _, entry := range providers {
		rows = append(rows, map[string]any{
			"source": entry.source, "sha256": entry.sha256, "state": entry.state, "reason": entry.reason,
			"id": entry.record.Provider.ID, "revision": entry.record.Provider.Revision,
			"repository_revision": entry.record.Repository.Revision, "freshness": entry.freshness,
			"entities": len(entry.record.Entities), "relations": len(entry.record.Relations),
		})
	}
	sortItems(merged.results)
	sortItems(merged.downstream)
	sortItems(merged.verification)
	sort.SliceStable(merged.unknowns, func(i, j int) bool {
		if merged.unknowns[i].provider != merged.unknowns[j].provider {
			return merged.unknowns[i].provider < merged.unknowns[j].provider
		}
		return lessRelation(merged.unknowns[i].relation, merged.unknowns[j].relation)
	})
	results, omittedResults := boundItems(merged.results, limit)
	downstream, omittedDownstream := boundItems(merged.downstream, limit)
	verification, omittedVerification := boundItems(merged.verification, limit)
	unknowns := make([]any, 0, min(len(merged.unknowns), limit))
	for _, entry := range merged.unknowns[:min(len(merged.unknowns), limit)] {
		unknowns = append(unknowns, entry.toMap())
	}
	fields := make([]any, 0, len(UntrustedTextFields))
	for _, field := range UntrustedTextFields {
		fields = append(fields, field)
	}
	return map[string]any{
		"schema_version":        1,
		"authority":             Authority,
		"providers":             rows,
		"results":               results,
		"downstream":            downstream,
		"verification":          verification,
		"unknowns":              unknowns,
		"omitted":               map[string]any{"results": omittedResults, "downstream": omittedDownstream, "verification": omittedVerification, "unknowns": max(len(merged.unknowns)-limit, 0)},
		"untrusted_text_fields": fields,
	}
}

func boundItems(items []item, limit int) ([]any, int) {
	kept := min(len(items), max(limit, 0))
	out := make([]any, 0, kept)
	for _, entry := range items[:kept] {
		out = append(out, entry.toMap())
	}
	return out, len(items) - kept
}
