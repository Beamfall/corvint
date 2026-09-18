---
name: bugs
description: Reproducible defects awaiting a fix
updated: 2026-09-18
---

# Bugs

Reproducible defects: something is broken, with a reproduction. Remove the entry once fixed. Entries are dated, newest first, and kept to one short paragraph. The public tree starts this backlog empty.

<!--
### YYYY-MM-DD <area>: <one-line title>
One paragraph: what, where (file:line), why it matters, and what done looks like.
-->

### 2026-09-18 extevidence: items are ordered by entity id before provider id
`EEP-V0-011` orders `results`, `downstream`, and `verification` by provider id, then entity id. `sortItems` (`internal/extevidence/compose.go`) compares the bare `entity.ID` and then the relation, and `assemble` calls it on the merged items of every provider. Two providers that declare the same entity id therefore interleave by relation instead of grouping by provider. Reproduce with two records that both declare `cap-x` linked to one changed path. The fix changes V0 section bytes for multi-provider runs, so it needs its own change and a note in the V0 spec's traceability.

