---
name: tests
description: Test coverage gaps and flaky tests to stabilise
updated: 2026-09-18
---

# Tests

Missing, weak, or flaky tests, named by file and behaviour. Remove the entry when the test exists and passes. Entries are dated, newest first, and kept to one short paragraph. The public tree starts this backlog empty.

<!--
### YYYY-MM-DD <area>: <one-line title>
One paragraph: what, where (file:line), why it matters, and what done looks like.
-->

### 2026-09-18 docmaintain: the SDD-V0-008 watch subtest is load-sensitive through its one-second wall clock
`TestWatchUnrelatedAndDirtyChangesDoNotWrite/SDD-V0-008_cited_source_eligibility` in `internal/docmaintain/watch_test.go` sets `policy.MaxWallClock = time.Second` and expects the watch to finish two cycles inside it. In a full `go test ./...` run on 2026-09-18 it failed with `StoppedReason:source-incomplete` after two cycles and one write, while the same test passed in isolation on the same tree and at the base commit. The wall clock is a hang detector standing in as a budget (compare decision 0082). Done means the subtest drives cycle completion by an explicit event rather than elapsed time, or the budget is sized so a loaded host cannot trip it.
