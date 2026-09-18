# Contributing to Corvint

Corvint is an alpha. Contributions are welcome where they fit the product invariants in
[AGENTS.md](AGENTS.md) and the development contract in
[docs/SPEC-DRIVEN-DEVELOPMENT.md](docs/SPEC-DRIVEN-DEVELOPMENT.md).

## Before you start

- Read the [README](README.md) for what the product does and does not claim.
- Read [AGENTS.md](AGENTS.md); the eight product invariants are not negotiable in a change.
- Check [docs/specs/README.md](docs/specs/README.md) for the spec that governs the area you want
  to change. Substantive behaviour is spec-driven: a change that alters behaviour or a wire contract
  updates its spec in the same change.

## Building and testing

```sh
GOTOOLCHAIN=local go build ./...
GOTOOLCHAIN=local go test -count=1 -timeout 30m ./...
GOTOOLCHAIN=local go vet ./...
```

`make gate` runs the full release gate set. Individual gates are listed in the `Makefile`; the ones
most changes need are `make line-citations-check`, `make decision-numbers-check`, and
`make requirement-definitions-check`.

## Change hygiene

- One concern per change. Keep requirement IDs stable and trace implemented requirements to tests
  or measured evidence.
- Documentation line citations (`path:N-M@sha`) are checked; run `make line-citations-check` after
  editing any cited document.
- Material design decisions go in `docs/decisions/` using the next free number and the existing
  template; `make decision-numbers-check` enforces uniqueness.
- Do not add network dependencies, hosted services, embeddings, or a UI to the default local
  product (invariant 7).

## Licensing

Corvint is AGPL-3.0-or-later with an Apache-2.0 interoperability layer; the path boundary is
recorded in [LICENSING.md](LICENSING.md). By contributing you agree that your contribution is
licensed under the terms that apply to the path it lands in.
