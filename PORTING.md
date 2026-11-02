# Compatibility and validation

## Reference

The behavioral reference is the Python implementation at revision
`08511c222029b3be20bf547563315d10fb48d44b`. Frozen fixtures are development
artifacts: normal builds, generation and tests do not invoke Python or require
a sibling checkout. Source and resource provenance live beside each component.

## Implemented native components

- Paragraph/sentence segmentation, Unicode normalization, string distances,
  affix encoding, character transformations and tokenization.
- DAWG reader/builder, dictionary variants, lexical queries, spelling suggestions,
  supplementary dictionaries, counts, lexical labels and compound analysis.
- French conjugation and verb tables, morphology, gender/plural, phonetics,
  thesaurus, date verification, agreement, grammatical suggestions and typography.
- Rule source/options/labels/colors parser, token expansion, minimal graph
  generation, source-expression parsing and evaluation, paragraph/sentence passes,
  graph matching, disambiguation, rewrites, priorities, immunity and detailed tokens.
- Lexical inflection, full frequency computation from original corpora, Metagraphe,
  Double Metaphone, Graphspell/Hunspell outputs and linguistic resource generators.
- Cobra commands for checking, spelling, formatting, inspection and generation.

## Deliberate boundaries

The Go CLI is organized as subcommands; it does not reproduce the original
interactive Python console's command syntax, Python HTTP server or application
integration packaging. LibreOffice, Mozilla and system UI bindings are separate
consumers of this standalone engine.

Python sets do not define a stable suggestion order. Go keeps first-seen or
explicitly sorted deterministic order where the reference uses sets. Four
Python hash-seed runs demonstrated six order-variable cases in the full grammar
corpus. Only those cases relax suggestion order in the differential test.
Diagnostic ordering and every other tested suggestion list remain strict.
Generator phonetic-group identifiers and derived-noun sets may also be ordered
deterministically; their contents and behavior are verified separately.

The regex implementation uses Go's `regexp/syntax` plus explicit Python
assertions, backreferences and Unicode boundaries. It implements the constructs
needed by the pinned grammar and formatter. It is not a general Python regex
replacement. Unsupported syntax, canceled work and excessive backtracking return
errors instead of silently dropping rules. Unicode categories follow Go's tables;
full case mapping and NFC use `golang.org/x/text`. Different Python Unicode
versions can disagree on newly introduced characters.

The Go ignore-rule option applies to both regex rules and graph action IDs.
This fixes an omission in the original Python graph dispatcher without changing
default analysis. The CLI processes paragraphs separately and returns offsets
in the original document; detailed token offsets stay sentence-relative.

Rune offsets follow the original text. Normalized token values can differ in
length from their original spans. Callers needing UTF-8 byte or UTF-16 offsets
must convert at their integration boundary.

The original lexical generator has a dormant malformed mixed prefix/suffix path;
Go reports it as an error. No pinned source entry triggers it. Original duplicate
corpus-row and ngram-range behavior is retained where the full source depends on
it. See [lexicon compatibility](lexicon/README.md) for export-specific details.

Recovered corpora are original archived upstream files, with verified hashes.
Their descriptive README is identical to the pinned checkout. The inaccessible
current Fossil service prevented confirming whether its unversioned payloads
have since changed. Native generation is reproducible against the recovered,
pinned inputs, not an unidentified latest corpus.

## Verification

`go test ./...` includes the complete grammar corpus and full-resource digest
checks. `go test -race ./...` checks concurrent use and the resource pipelines.
The normal executable and core generators are also exercised after copying only
the binary into an empty directory with no usable PATH. Cross-compilation checks
macOS, Windows and Linux; compilation alone does not establish runtime validation
on every operating system.

The development workflow is test-driven: add a frozen reference or explicit
behavior test, observe failure, implement, then run the relevant checks. See
[CONTRIBUTING.md](CONTRIBUTING.md).
