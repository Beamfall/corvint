# Build log

Append-only record of material design decisions, independent findings, failed evaluations, and
promotion evidence, newest entry first. Each entry carries a date heading and the requirement or
decision IDs it concerns, so `rg -n '^## ' docs/BUILD-LOG.md` is the index.

The public tree starts this log at the 0.4.0a4 alpha. Entries written before publication are internal
working records and are referenced from decisions and specifications as historical context only.

## 2026-09-18 ETS-V0 external test selection: synthetic conformance evaluation

Decision 0311. `TestSelectionEvaluation` over the 23 labelled cases in
`internal/extevidence/testdata/conformance-selection/cases.json`: precision 1.000 (16 of 16 selected
tests expected), unsafe-narrowing 0 of 20 cases that must not narrow, abstention accuracy 3 of 3,
latency p50 about 35 ms and max about 125 ms per case on one development host, and a largest
`test_selection` member of 4470 bytes. The corpus is synthetic and authored with the feature, so
these numbers show the fail-closed rules hold. They are not an adopter outcome, and promotion needs
a corpus drawn from a real change history.
