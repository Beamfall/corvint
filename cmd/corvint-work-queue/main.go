package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/Beamfall/corvint/internal/workqueue"
	"github.com/Beamfall/corvint/internal/worksource"
)

const (
	repositoryAuthority = "repo:corvint"
	queueAuthority      = "queue:corvint:worklist"
	capacityClass       = "capacity:corvint:worklist:agent"
)

type worklist struct {
	Profile string     `json:"profile"`
	Tickets []workItem `json:"tickets"`
}

type workItem struct {
	Body       string   `json:"body"`
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	TouchPaths []string `json:"touchPaths"`
}

func main() {
	if err := runProducer(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func runProducer(arguments []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(arguments) != 1 {
		return errors.New("usage: corvint-work-queue snapshot|details|verify")
	}
	switch arguments[0] {
	case "snapshot", "details", "verify":
	default:
		return errors.New("unsupported operation")
	}
	storage, err := producerOwnedScratch()
	if err != nil {
		return err
	}
	root, err := repositoryRoot(ctx, storage.parent)
	if err != nil {
		return err
	}
	if storage.parent != "" && storage.root != root {
		return errors.New("source scratch root mismatch")
	}
	qualified, err := worksource.AcquireWithScratch(ctx, root, storage.parent)
	if err != nil {
		return err
	}
	defer qualified.Close()
	if storage.parent != "" && storage.commit != qualified.Identity.Commit {
		return errors.New("source scratch commit mismatch")
	}
	policyRaw, err := sourceFile(qualified, ".corvint/work-queue-policy.json")
	if err != nil {
		return err
	}
	policy, err := workqueue.ParsePolicy(policyRaw)
	if err != nil {
		return err
	}
	snapshot, details, checkpoint, err := documentsFromSource(qualified, policy)
	if err != nil {
		return err
	}
	var output []byte
	switch arguments[0] {
	case "snapshot":
		output = snapshot.Canonical()
	case "details":
		output = details.Canonical()
	case "verify":
		output = checkpoint.Canonical()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = os.Stdout.Write(output)
	return err
}

func repositoryRoot(ctx context.Context, scratch string) (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return worksource.ResolveRootWithScratch(ctx, directory, scratch)
}

func documents(root string, policy *workqueue.Policy) (*workqueue.Snapshot, *workqueue.DetailsDocument, *workqueue.CheckpointDocument, error) {
	qualified, err := worksource.Acquire(context.Background(), root)
	if err != nil {
		return nil, nil, nil, err
	}
	defer qualified.Close()
	return documentsFromSource(qualified, policy)
}

func documentsFromSource(qualified *worksource.Source, policy *workqueue.Policy) (*workqueue.Snapshot, *workqueue.DetailsDocument, *workqueue.CheckpointDocument, error) {
	source := qualified.Identity
	raw, err := sourceFile(qualified, "docs/worklist.json")
	if err != nil {
		return nil, nil, nil, err
	}
	var list worklist
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&list); err != nil || list.Profile != "corvint-worklist/0" {
		return nil, nil, nil, errors.New("invalid worklist")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, nil, errors.New("invalid worklist")
	}
	tickets := make([]workqueue.TicketSummary, 0, len(list.Tickets))
	details := make([]workqueue.Detail, 0, len(list.Tickets))
	limit, err := workqueue.ParseCount(policy.DetailLimit)
	if err != nil {
		return nil, nil, nil, err
	}
	for index, item := range list.Tickets {
		itemRaw, _ := json.Marshal(item)
		payload := workqueue.DetailPayload{AcceptanceCriteria: []string{}, Body: &item.Body, EvidenceHandles: []string{}, Title: &item.Title}
		detailDigest := workqueue.DetailPayloadDigest(payload)
		ticket := workqueue.TicketSummary{
			AtomicRepositoryAuthorityIDs: []string{repositoryAuthority}, Authority: "COMPLETE",
			CapacityUses: []workqueue.CapacityUse{{ClassID: capacityClass, Units: 1}}, CollisionGroupIDs: []string{},
			DeclaredVersion: workqueue.SHA256Hex(itemRaw)[:16], DependencyTicketIDs: []string{}, DetailPayloadSHA256: &detailDigest,
			Lifecycle: "READY", QueueAuthorityID: queueAuthority, Rank: workqueue.Rank(index), RepositoryAuthorityID: repositoryAuthority,
			RouteAlternatives:   []workqueue.RouteAlternative{{ID: "route:corvint:worklist:codex", Requires: []string{"capability:corvint:worklist:codex"}}},
			SelectionFacts:      workqueue.SelectionFacts{Approvals: "CLEAR", Dependencies: "SATISFIED", Holds: "CLEAR", Lease: "ABSENT"},
			TicketContentSHA256: workqueue.SHA256Hex(itemRaw), TicketID: "ticket:corvint:worklist:" + item.ID,
			TouchPaths: append([]string(nil), item.TouchPaths...),
		}
		sort.Strings(ticket.TouchPaths)
		workqueue.RefreshTicket(&ticket)
		tickets = append(tickets, ticket)
		if workqueue.Count(index) >= limit {
			continue
		}
		detail := workqueue.Detail{Payload: payload, RepositoryAuthorityID: repositoryAuthority, TicketID: ticket.TicketID, TicketVersionID: ticket.TicketVersionID}
		workqueue.RefreshDetail(&detail)
		details = append(details, detail)
	}
	snapshot := &workqueue.Snapshot{
		AccessContextID: policy.AccessContextID, CapacityClasses: []workqueue.CapacityClass{{AvailableUnits: 4, ID: capacityClass}},
		Checkpoint: workqueue.Checkpoint{ID: "checkpoint:corvint:worklist", Version: source.Tree}, Leases: []workqueue.LeaseSummary{},
		PolicyID: policy.ID, Profile: workqueue.SnapshotProfile, QueueAuthorityID: queueAuthority, RepositoryAuthorityID: repositoryAuthority,
		RepositorySource: source, Scope: workqueue.Scope{Complete: true, ID: "scope:corvint:worklist", TicketCount: workqueue.Count(len(tickets))}, Tickets: tickets,
	}
	for index := range tickets {
		if workqueue.Count(index) >= limit {
			break
		}
		snapshot.DetailRequestTicketVersionIDs = append(snapshot.DetailRequestTicketVersionIDs, tickets[index].TicketVersionID)
	}
	sort.Strings(snapshot.DetailRequestTicketVersionIDs)
	workqueue.RefreshSnapshot(snapshot)
	detailDocument := &workqueue.DetailsDocument{Details: details, SnapshotID: snapshot.ID}
	workqueue.RefreshDetails(detailDocument)
	checkpoint := &workqueue.CheckpointDocument{Checkpoint: snapshot.Checkpoint, PolicyID: policy.ID, RepositorySource: source, SnapshotID: snapshot.ID}
	workqueue.RefreshCheckpoint(checkpoint)
	return snapshot, detailDocument, checkpoint, nil
}

func sourceFile(source *worksource.Source, path string) ([]byte, error) {
	for _, entry := range source.Entries {
		if entry.Path == path && entry.Mode == "100644" {
			return entry.Raw, nil
		}
	}
	return nil, fmt.Errorf("qualified source file unavailable: %s", path)
}
