# gogrammalecte

A standalone Go port of Grammalecte, targeting behavioral equivalence with the
Python implementation. The final engine and all rule and resource generators
will be written in Go.

**Status: initial parity-tested components. Full grammar checking, spelling
suggestions, dictionaries and rule generation are not implemented yet.**

## Build and test

Requires Go 1.26.1 or later.

```sh
go test ./...
go test -race ./...
go vet ./...
go build -o bin/gogrammalecte ./cmd/gogrammalecte
```

Building and running the tests require neither Python nor a Grammalecte checkout.
The same Go module builds on macOS, Windows and Linux.

## Current components

- `text`: paragraph splitting, sentence boundaries and sentence splitting.
  Boundaries use Unicode code-point offsets with an exclusive end.
- `graphspell/strtransform`: suffix and affix encoding and decoding.
- `cmd/gogrammalecte`: a Cobra CLI exposing the available text segmentation.

```sh
printf 'Bonjour. À bientôt !' | go run ./cmd/gogrammalecte sentences
# ["Bonjour. ","À bientôt !"]

printf 'First paragraph\r\nSecond paragraph' | go run ./cmd/gogrammalecte paragraphs
# ["First paragraph","Second paragraph"]
```

Both commands read valid UTF-8 from standard input and write a JSON array to
standard output. Invalid UTF-8, invalid arguments and I/O failures return errors.
Input is currently read into memory. These commands expose ported helpers;
the complete upstream CLI interface remains to be implemented.

## Compatibility and development

The Python source at commit
`08511c222029b3be20bf547563315d10fb48d44b` of
[Grammalecte](https://github.com/benoit-pereira-da-silva/grammalecte)
is the initial behavioral reference. Tests consume checked-in outputs captured
from that source. French test data is preserved verbatim.

Development follows a failing-test-first workflow. Standard-library `regexp`
is preferred; unsupported Python constructs require explicit matching logic
and parity tests. No alternative regex dependency is currently used.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow and
[PORTING.md](PORTING.md) for the remaining compatibility work.

## License

GNU GPL version 3; see [LICENSE](LICENSE). Ported components identify their
upstream source. Grammalecte was created by Olivier R.
