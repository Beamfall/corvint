---
name: ideas
description: Broader future work, features, and unscheduled directions
updated: 2026-09-18
---

# Ideas

Future features and larger directions. Promote to `docs/specs/` when an idea matures; remove the entry once promoted or shipped. Entries are dated, newest first, and kept to one short paragraph. The public tree starts this backlog empty.

<!--
### YYYY-MM-DD <area>: <one-line title>
One paragraph: what, where (file:line), why it matters, and what done looks like.
-->

### 2026-09-18 extevidence: test selection beyond one hop and into checkout worktrees (ETS-V0 follow-up)
`docs/specs/external-test-selection-v0.md` stops obligations one relation hop downstream of a changed entity, and never reads a bound checkout's worktree (rows say `checkout-worktree-not-inspected`). Both keep the selection fail-closed only as far as the record is complete. Done means a bounded transitive obligation walk and a per-checkout dirty read, each with conformance cases proving they only ever widen.

### 2026-09-18 extevidence: command, MCP, and remote provider transports (EEP slice 2)
Decision 0309 ships only the file transport for `docs/specs/external-evidence-provider-v0.md`. A provider that is a local command, an MCP tool, or a remote service reaches outside the local boundary (AGENTS.md invariant 7), so each transport needs its own accepted profile under the analyzer capability contract before `--provider` accepts anything but a file path. Done means an accepted profile plus the same strict record decode, freshness, and separation tests running over the new transport.

### 2026-09-18 extevidence: external obligations as a Change Frontier sidecar (EEP slice 4)
Provider-reported capabilities and journeys downstream of a change are obligations a reviewer should see, but CEM 0.2 rejects unknown fields, so they cannot enter the CEM. Decision 0309 points at a `docs/specs/change-frontier-v0.md` sidecar keyed by the impact receipt digest. Done means the frontier report can cite the sidecar without the sidecar carrying authority or altering frontier state.
