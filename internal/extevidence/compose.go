package extevidence

import (
	"fmt"
	"sort"
	"strings"
)

// Verification states for a path endpoint (EEP-V0-010).
const (
	VerificationVerified    = "verified"
	VerificationStale       = "stale"
	VerificationMissing     = "missing"
	VerificationUnsupported = "unsupported"
)

// Unknown states (EEP-V0-006, EEP-V0-007).
const (
	unknownUnresolved = "unresolved"
	unknownExcluded   = "excluded"
)

var verificationTypes = map[string]struct{}{"verifies": {}, "covers": {}, "asserts": {}}

var evidenceKinds = map[string]struct{}{EvidenceDeclared: {}, EvidenceObserved: {}, EvidenceInferred: {}}

// tree answers the two repository questions composition asks about a path.
type tree struct {
	tracked func(path string) bool
	blob    func(path string) (string, bool)
}

type endpoint struct {
	path   string
	entity string
}

func (e endpoint) isPath() bool { return e.path != "" }

type link struct {
	relation Relation
	from, to endpoint
}

type item struct {
	provider string
	entity   Entity
	path     string
	link     link
	state    string
	reason   string
}

type unknown struct {
	provider string
	relation Relation
	state    string
	reason   string
}

type composition struct {
	results, downstream, verification []item
	unknowns                          []unknown
}

// compose applies EEP-V0-006, -007, -010, and -011 to one loaded record.
func compose(record Record, changed map[string]struct{}, repository tree) composition {
	entities := make(map[string]Entity, len(record.Entities))
	for _, entity := range record.Entities {
		entities[entity.ID] = entity
	}
	var out composition
	links := make([]link, 0, len(record.Relations))
	for _, relation := range record.Relations {
		resolved, failure := resolve(record, entities, relation)
		if failure != nil {
			out.unknowns = append(out.unknowns, *failure)
			continue
		}
		links = append(links, resolved)
	}
	out.results = directResults(record, entities, links, changed, repository)
	listed := entitySet(out.results)
	out.downstream = downstreamOf(record, entities, links, listed)
	for id := range entitySet(out.downstream) {
		listed[id] = struct{}{}
	}
	out.verification = verificationOf(record, entities, links, changed, listed, repository)
	sortItems(out.results)
	sortItems(out.downstream)
	sortItems(out.verification)
	sort.Slice(out.unknowns, func(i, j int) bool { return lessRelation(out.unknowns[i].relation, out.unknowns[j].relation) })
	return out
}

func resolve(record Record, entities map[string]Entity, relation Relation) (link, *unknown) {
	if _, known := evidenceKinds[relation.Evidence]; !known {
		return link{}, &unknown{
			provider: record.Provider.ID, relation: relation, state: unknownExcluded,
			reason: fmt.Sprintf("evidence kind %q is not declared, observed, or inferred", relation.Evidence),
		}
	}
	from, reason := parseEndpoint(record, entities, relation.From)
	if reason != "" {
		return link{}, &unknown{provider: record.Provider.ID, relation: relation, state: unknownUnresolved, reason: "from: " + reason}
	}
	to, reason := parseEndpoint(record, entities, relation.To)
	if reason != "" {
		return link{}, &unknown{provider: record.Provider.ID, relation: relation, state: unknownUnresolved, reason: "to: " + reason}
	}
	return link{relation: relation, from: from, to: to}, nil
}

func parseEndpoint(record Record, entities map[string]Entity, raw string) (endpoint, string) {
	if path, isPath := strings.CutPrefix(raw, "path:"); isPath {
		return endpoint{path: path}, checkPath(path)
	}
	provider, id, found := strings.Cut(raw, ":")
	if !found {
		return endpoint{}, "endpoint has neither a path: prefix nor a provider prefix"
	}
	if provider != record.Provider.ID {
		return endpoint{}, fmt.Sprintf("endpoint names provider %q; V0 resolves only the record's own provider", provider)
	}
	if _, declared := entities[id]; !declared {
		return endpoint{}, fmt.Sprintf("entity %q is not declared by the record", id)
	}
	return endpoint{entity: id}, ""
}

func checkPath(path string) string {
	if path == "" || len(path) > maxPath {
		return fmt.Sprintf("path must be non-empty and at most %d bytes", maxPath)
	}
	if strings.HasPrefix(path, "/") {
		return "path must be repository-relative"
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return "path must not contain a parent-traversal segment"
		}
	}
	return ""
}

func verify(repository tree, path, pinned string) string {
	if !repository.tracked(path) {
		return VerificationMissing
	}
	if pinned == "" {
		return VerificationVerified
	}
	actual, known := repository.blob(path)
	if known && actual == pinned {
		return VerificationVerified
	}
	return VerificationStale
}

// pathAndEntity splits a link into its path side and entity side when it has
// exactly one of each.
func pathAndEntity(candidate link) (path, entity string, ok bool) {
	if candidate.from.isPath() && !candidate.to.isPath() {
		return candidate.from.path, candidate.to.entity, true
	}
	if !candidate.from.isPath() && candidate.to.isPath() {
		return candidate.to.path, candidate.from.entity, true
	}
	return "", "", false
}

func directResults(record Record, entities map[string]Entity, links []link, changed map[string]struct{}, repository tree) []item {
	var results []item
	for _, candidate := range links {
		path, entity, ok := pathAndEntity(candidate)
		if !ok {
			continue
		}
		if _, isChanged := changed[path]; !isChanged {
			continue
		}
		results = append(results, item{
			provider: record.Provider.ID, entity: entities[entity], path: path, link: candidate,
			state:  verify(repository, path, candidate.relation.Blob),
			reason: fmt.Sprintf("changed path %s joined by %s relation %s", path, candidate.relation.Evidence, candidate.relation.Type),
		})
	}
	return results
}

func downstreamOf(record Record, entities map[string]Entity, links []link, results map[string]struct{}) []item {
	var downstream []item
	for _, candidate := range links {
		if candidate.from.isPath() || candidate.to.isPath() {
			continue
		}
		origin, next := candidate.from.entity, candidate.to.entity
		if _, isResult := results[origin]; !isResult {
			origin, next = next, origin
		}
		if _, isResult := results[origin]; !isResult {
			continue
		}
		if _, isResult := results[next]; isResult {
			continue
		}
		downstream = append(downstream, item{
			provider: record.Provider.ID, entity: entities[next], link: candidate, state: VerificationUnsupported,
			reason: fmt.Sprintf("one %s relation %s from result entity %s:%s", candidate.relation.Evidence, candidate.relation.Type, record.Provider.ID, origin),
		})
	}
	return downstream
}

func verificationOf(record Record, entities map[string]Entity, links []link, changed, listed map[string]struct{}, repository tree) []item {
	var verification []item
	for _, candidate := range links {
		if _, verifying := verificationTypes[candidate.relation.Type]; !verifying {
			continue
		}
		path, entity, ok := pathAndEntity(candidate)
		if !ok {
			continue
		}
		if _, isChanged := changed[path]; isChanged {
			continue
		}
		if _, isListed := listed[entity]; !isListed {
			continue
		}
		verification = append(verification, item{
			provider: record.Provider.ID, entity: entities[entity], path: path, link: candidate,
			state:  verify(repository, path, candidate.relation.Blob),
			reason: fmt.Sprintf("%s relation %s from path %s to listed entity %s:%s", candidate.relation.Evidence, candidate.relation.Type, path, record.Provider.ID, entity),
		})
	}
	return verification
}

func entitySet(items []item) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, entry := range items {
		set[entry.entity.ID] = struct{}{}
	}
	return set
}

func sortItems(items []item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].entity.ID != items[j].entity.ID {
			return items[i].entity.ID < items[j].entity.ID
		}
		return lessRelation(items[i].link.relation, items[j].link.relation)
	})
}

func lessRelation(a, b Relation) bool {
	if a.From != b.From {
		return a.From < b.From
	}
	if a.To != b.To {
		return a.To < b.To
	}
	return a.Type < b.Type
}

func relationMap(relation Relation) map[string]any {
	out := map[string]any{
		"from": relation.From, "to": relation.To, "type": relation.Type,
		"evidence": relation.Evidence, "rule": relation.Rule, "reference": relation.Reference,
	}
	if relation.Blob != "" {
		out["blob"] = relation.Blob
	}
	return out
}

func (entry item) toMap() map[string]any {
	out := map[string]any{
		"authority":    Authority,
		"provider":     entry.provider,
		"entity":       entry.provider + ":" + entry.entity.ID,
		"kind":         entry.entity.Kind,
		"summary":      entry.entity.Summary,
		"relation":     relationMap(entry.link.relation),
		"verification": entry.state,
		"reason":       entry.reason,
	}
	if entry.path != "" {
		out["path"] = entry.path
	}
	return out
}

func (entry unknown) toMap() map[string]any {
	return map[string]any{
		"authority": Authority,
		"provider":  entry.provider,
		"relation":  map[string]any{"from": entry.relation.From, "to": entry.relation.To, "type": entry.relation.Type, "evidence": entry.relation.Evidence},
		"state":     entry.state,
		"reason":    entry.reason,
	}
}
