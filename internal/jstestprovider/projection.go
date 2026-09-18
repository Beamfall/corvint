package jstestprovider

import (
	"strconv"

	"github.com/Beamfall/corvint/internal/tcq"
	"github.com/Beamfall/corvint/internal/testvalidity"
)

// ToTestProjection projects one TestOutcome through the shared
// testvalidity.Project (internal/testvalidity/projection.go). Ordinary
// per-test states (passed/failed/skipped/flaky) carry ClaimFacts, since they
// are TCQ-shaped per-test report rows with a known association anchor.
// Harness-level incomplete states (timedOut/interrupted/infrastructure) that
// mean the test itself never produced a real report instead carry
// ExecutionFacts with an INCOMPLETE outcome, matching the LPCV/GLTP cause
// vocabulary Project already understands. No branch invents a "valid"
// verdict; every branch states exactly the fact this provider observed.
func ToTestProjection(outcome TestOutcome) testvalidity.Projection {
	anchors := anchorStrings(outcome.Anchor)
	switch outcome.State {
	case StatePassed, StateFailed, StateSkipped, StateFlaky:
		claim := testvalidity.ClaimFacts{
			AssociationState: tcq.AssociationAssociated,
			HygieneState:     tcq.HygieneEligible,
			ReportState:      reportStateFor(outcome.State),
			Anchors:          anchors,
		}
		if outcome.State == StateFlaky {
			claim.Reasons = []string{"flaky-retry"}
		}
		return testvalidity.Project(testvalidity.Input{Claim: &claim})
	case StateTimedOut, StateInterrupted, StateInfrastructure:
		execution := testvalidity.ExecutionFacts{
			Outcome: "INCOMPLETE",
			Cause:   causeFor(outcome.State),
			Anchors: anchors,
		}
		return testvalidity.Project(testvalidity.Input{Execution: &execution})
	default:
		return testvalidity.Project(testvalidity.Input{})
	}
}

func reportStateFor(state ExecutionState) string {
	switch state {
	case StatePassed, StateFlaky:
		return tcq.ReportPassed
	case StateFailed:
		return tcq.ReportFailed
	case StateSkipped:
		return tcq.ReportSkipped
	default:
		return tcq.ReportError
	}
}

func causeFor(state ExecutionState) string {
	switch state {
	case StateTimedOut:
		return "TIMEOUT"
	case StateInterrupted:
		return "CANCELLATION"
	case StateInfrastructure:
		return "INFRASTRUCTURE"
	default:
		return ""
	}
}

func anchorStrings(a *Anchor) []string {
	if a == nil || a.File == "" {
		return nil
	}
	return []string{anchorString(*a)}
}

func anchorString(a Anchor) string {
	if a.Line == 0 {
		return a.File
	}
	return a.File + ":" + strconv.Itoa(a.Line)
}

// ReceiptRunProjection projects the receipt's own run-level facts: a
// run-level InfrastructureFailure (bad command, no tests found, missing
// browser at collection time) and stale-app-build freshness. It is separate
// from per-test projections because a run-level infrastructure failure may
// exist with zero test outcomes, and staleness is a fact about the whole
// run's served build, not any one test.
func ReceiptRunProjection(r Receipt) testvalidity.Projection {
	var execution *testvalidity.ExecutionFacts
	switch {
	case r.Infrastructure != nil:
		execution = &testvalidity.ExecutionFacts{Outcome: "INCOMPLETE", Cause: "INFRASTRUCTURE"}
	case r.Cancelled:
		execution = &testvalidity.ExecutionFacts{Outcome: "INCOMPLETE", Cause: "CANCELLATION"}
	}
	if execution == nil && !r.StaleAppBuild {
		return testvalidity.Project(testvalidity.Input{})
	}
	if execution == nil {
		execution = &testvalidity.ExecutionFacts{}
	}
	if r.StaleAppBuild {
		execution.Currency = "STALE"
	} else if execution.Outcome != "" {
		execution.Currency = "CURRENT"
	}
	return testvalidity.Project(testvalidity.Input{Execution: execution})
}
