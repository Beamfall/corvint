# Corvint licensing

Corvint uses a deliberately small split-license boundary: the product is copyleft, and the
vendor-neutral material anyone needs in order to interoperate with it is permissive.

## Corvint product layer — AGPL-3.0

Every repository path not listed in the next section is licensed under the **GNU Affero General
Public License, version 3.0 or later**. The full text is in `LICENSE`.

You may use, study, modify, and redistribute Corvint under those terms. AGPL-3.0 section 13
additionally requires that if you run a modified Corvint to provide a service over a network, you
offer that service's users the corresponding source of your modified version.

## Apache-2.0 interoperability layer

The following paths are licensed under the **Apache License, Version 2.0**, with no copyleft
condition. The full text is in `LICENSE-APACHE-2.0`.

- `protocol/**`
- `conformance/**`
- `interop/**`
- `examples/**`
- `docs/CHANGE-EVIDENCE-MAP.md`
- `docs/CEM-CI.md`
- `docs/cem-0.1.schema.json`
- `docs/cem-0.2.schema.json`
- `docs/lrf-0.schema.json`
- `docs/tcq-0.schema.json`

These are the vendor-neutral protocol descriptions, schemas, conformance material, example
integrations, and independent implementation probes. Anyone may use them to add Corvint-compatible
support to an agent, IDE, CI system, context engine, or commercial product under ordinary
Apache-2.0 terms, including in proprietary software, without the AGPL obligations above ever
attaching.

`protocol/**` is the canonical home for future vendor-neutral protocol, specification, and schema
material. A new Apache-2.0 file must be placed there; extending the boundary to any other path
requires an explicit amendment to `docs/decisions/0002-future-publication-transition.md`.

## Rules

- A file's licence is determined by its path, per the boundary above, unless the file carries a
  more specific notice.
- Contributions are licensed according to the destination path.
- Combining an AGPL-3.0 path with an Apache-2.0 path produces an AGPL-3.0 work; the reverse is not
  true, which is the entire point of the boundary.
- Third-party dependencies keep their own licences; this document governs only Corvint's own source.

This boundary is the one recorded in `docs/decisions/0002-future-publication-transition.md`.

## Go runtime notice

The native Corvint and companion executables include the Go runtime and standard library.
Those Go components retain their upstream BSD-3-Clause terms. This notice supplements the
Corvint source-license boundary above. It is reproduced unchanged from the
[Go 1.27.0 license](https://github.com/golang/go/blob/go1.27.0/LICENSE), also used by that
release's vendored `golang.org/x/crypto`, `x/net`, `x/sys` and `x/text` packages.

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
