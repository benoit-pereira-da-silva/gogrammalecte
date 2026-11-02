# Validation record

Reference: Grammalecte `08511c222029b3be20bf547563315d10fb48d44b`.
Toolchain: Go 1.26.1. Development host: macOS arm64.

## Behavioral coverage

The full normal suite passes with `go test ./...`. It includes every one of the
14,160 source grammar cases, exact diagnostic fields and ordering, full token
analysis snapshots, complete dictionary and lexical outputs, all rule graph
transitions and native resource regeneration. Only the six suggestion lists
proven unstable across four Python hash seeds use multiset comparison.
`go vet ./...` and whitespace checks also pass.

## Standalone executable

The executable was built with:

```sh
CGO_ENABLED=0 go build -trimpath -o bin/gogrammalecte ./cmd/gogrammalecte
```

Only that file was copied to a new empty directory. Each invocation ran from
that directory with an environment containing only `PATH=/nonexistent`.
The following operations succeeded without external resources or executables:

- Grammar checking, multiline Unicode offsets and detailed token analysis.
- Spelling suggestions, conjugation and typography formatting.
- Access to all four original frequency corpora.
- Complete dictionary regeneration: 515,637 distinct entries.
- Complete lexical-table regeneration: 55,312,242 bytes for resource version
  `test`, SHA-256 `b6b10890ae5e5b8a72cd34ed0b518c820c199f91beeaa546313318f5033df5e5`.
- Frequency-aware Hunspell generation with mode 2 and aliases, matching the
  Python dictionary and affix digests.
- MyThes dictionary/index generation as a ZIP archive.

The verified macOS arm64 executable contains 120,322,450 bytes, SHA-256
`43f7c5386630322478a280f4a8165a40137ac42230effc3f999fa0818db8b350`.
Size and digest depend on the build flags, toolchain, platform and source tree.

## Platforms

The final source builds with `CGO_ENABLED=0` for macOS arm64 and amd64,
Linux amd64 and arm64, and Windows amd64. Only the macOS arm64 executable was
executed during this validation; cross-compilation is not runtime testing.

## Concurrency

The full race-instrumented suite passes with `go test -race ./...`.
Its corpus and graph-generation tests take substantially longer than the normal
suite. Targeted concurrency tests also exercise independent grammar analyses,
shared dictionaries, detailed token snapshots and resource helpers.
