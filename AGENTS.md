# Development instructions

- This repository is a standalone Go library and command line program. Its
  build, tests and runtime must work from an independent checkout.
- Use the Go toolchain directly: `go build`, `go test`, `go vet` and `gofmt`.
- Write all code, identifiers, comments and documentation in English. Preserve
  French linguistic data, rule identifiers and test inputs where fidelity requires it.
- Follow test-driven development: write the behavioral test, observe a failure,
  implement the smallest faithful change, then refactor with tests passing.
- Use the pinned Python implementation as the behavioral reference during
  development. Check reference outputs into portable fixtures. Normal builds
  and tests must not require Python or a sibling repository.
- Prefer the standard library, including `regexp`. When a Python regex uses
  unsupported constructs, implement explicit, tested matching logic or a
  semantics-preserving lowering. Never silently discard a rule or weaken it.
- Use Cobra for the CLI. Introduce Viper when configuration requirements justify it.
- Preserve Unicode code-point positions, rule order, suggestions and error
  behavior. Record intentional API differences and incomplete coverage.
- Keep generated linguistic data traceable to its source revision. The final
  rule and resource generation toolchain must also be written in Go.
- Keep the module independent of host applications and their build frameworks.
  Do not add their SDKs, manifests, generated service contracts or dependencies.
- Preserve upstream attribution and the repository license when porting source.
