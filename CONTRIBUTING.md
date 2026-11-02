# Contributing

## Test-driven workflow

1. Select one observable behavior of the pinned Python reference.
2. Capture its inputs and outputs, including boundary and error cases. Record
   the source commit, file and fixture-generation method.
3. Add a Go test and run it before implementing the behavior. Confirm that the
   failure demonstrates the missing behavior rather than a broken environment.
4. Implement the smallest faithful Go change, then run the focused tests.
5. Refactor with the tests passing. Run `go test ./...`, `go test -race ./...`
   and `go vet ./...` before completing the change.

Fixtures must be usable without Python, network access or a sibling checkout.
Python may be used to obtain development reference outputs, but it must not
be part of the delivered build, generation or execution chain.

## Compatibility expectations

Compare exact outputs where the upstream behavior is deterministic, including
ordering, Unicode positions, retained whitespace and empty trailing segments.
Preserve upstream quirks until an intentional deviation has a documented reason
and a dedicated test. Map upstream exceptions to explicit Go errors where the
public Go API returns an error.

Use English for prose, comments, identifiers and diagnostics. Keep linguistic
strings and upstream rule identifiers intact. Prefer a concrete counterexample
over broad claims of compatibility.

## Dependencies and command line

Prefer the standard library. The CLI uses Cobra; Viper is reserved for a future
configuration requirement. Any additional dependency needs a concrete use case.
The linguistic packages must remain usable without importing CLI packages.

Use `gofmt` and keep `go.mod` and `go.sum` in the repository. Tests should exercise
behavior rather than duplicate the implementation.
