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

### 2026-09-18 liveverify/session: `TestRunningFailedPassed` misses its event wait under full-suite load
In a full `go test ./...` run, `session_test.go:132` timed out waiting for the next event after `running` (20.49s) and the package took 125s. It passed 2/2 alone (7.7s). This failure was also seen in the #8 run. Done means the wait is keyed on the run's own completion rather than elapsed time, or the budget is sized as a hang detector (decision 0082).
