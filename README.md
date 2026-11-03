# gogrammalecte

A standalone Go implementation of Grammalecte's French grammar, spelling,
typography and resource-generation tools. One executable embeds its grammar,
dictionaries, linguistic tables, original lexical sources and all four frequency
corpora. It needs no Python, installed resource directory or network at runtime.

## Build and test

Requires Go 1.26.1 or later.

```sh
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/gogrammalecte ./cmd/gogrammalecte
```

The module builds independently on macOS, Windows and Linux. The unstripped
executable is approximately 115 MiB: the original frequency corpora account for
about 91 MiB compressed. These inputs are included so lexical regeneration also
works with only the executable. Race tests are available through
`go test -race ./...`; complete corpus instrumentation takes several minutes.

## Command line

```sh
printf 'Les chat sont ici.' | bin/gogrammalecte check
printf 'Il mues.' | bin/gogrammalecte check --full --include-context
printf 'Des chevaus.' | bin/gogrammalecte spell
bin/gogrammalecte suggest bonjouur
bin/gogrammalecte lookup bonjour
bin/gogrammalecte conjugate être :Ip :1s
printf 'Bonjour  monde.' | bin/gogrammalecte format
bin/gogrammalecte options
bin/gogrammalecte corpora
```

`check` emits grammar diagnostics with the original Python JSON field names.
`--full` adds sentence analysis and labeled tokens. Offsets count Unicode code
points in the original input, with an exclusive end. LF, CRLF and CR delimit
paragraphs; full-analysis token offsets remain relative to their sentence.
Options can be changed with `--enable`,
`--disable`, `--country` and `--context`; `--ignore` accepts rule identifiers.
`spell` emits unknown word tokens. `format` writes formatted text without adding
an extra newline. Input must be UTF-8. The CLI reads input into memory.

`paragraphs`, `sentences` and `rules` expose text segmentation and the embedded
rule inventory. Run any command with `--help` for its parameters.

## Text subtitles

`check`, `spell` and `format` accept an optional UTF-8 file path (or `-` for
standard input). Automatic detection uses its extension, then structural
signatures. Use `--format text` to disable detection, or select a format explicitly:

```sh
bin/gogrammalecte check film.srt
bin/gogrammalecte check --full --include-context film.vtt
bin/gogrammalecte spell lyrics.lrc
bin/gogrammalecte format film.ass > film-formatted.ass
bin/gogrammalecte check --format srt < film.srt
```

Supported formats are SRT, WebVTT (`vtt`, `webvtt`), ASS/SSA, SBV/SubViewer,
MicroDVD (textual `.sub`), LRC and TTML/DFXP. Only dialogue/lyrics are analyzed;
timing, cue identifiers, metadata, styles and drawing commands are excluded.
Physical lines within one cue are joined for grammar analysis; separate cues
remain independent. ASS/SSA requires Text to be the last declared event field.
Binary or bitmap subtitles and legacy non-UTF-8 encodings are not supported.
Incomplete or unrecognized blocks may be skipped; this is not a subtitle syntax,
timing or synchronization validator.

The public `subtitle` package exposes `Project`, `Check`, `Analyze`, `Spell` and
`Format`. The existing plain-paragraph APIs remain unchanged. Diagnostic offsets
refer to Unicode code points in the original file; full-analysis token offsets
remain relative to their original-source sentence. `Project` exposes byte spans
and `Cue.SourceRange` for adapters. Grammar suggestions crossing tags, entities,
line breaks or control escapes are withheld. Formatting changes only literal
text runs, preserving timing, markup, escapes and physical line breaks; rules
requiring context across these boundaries are conservatively skipped. Formatting
writes to standard output and does not modify the source file.

## Native generation

Every command below uses embedded source data and native Go generators:

```sh
bin/gogrammalecte generate rules > rules.json
bin/gogrammalecte build-dictionary > fr-allvars.json
bin/gogrammalecte generate lexicon > French.lex
bin/gogrammalecte generate lexical-table > French-full.txt
bin/gogrammalecte generate verb-source > dictConj.txt
bin/gogrammalecte generate declension-source > dictDecl.txt
bin/gogrammalecte generate inflections > inflections.json
bin/gogrammalecte generate conjugations > conjugations.json
bin/gogrammalecte generate phonetics > phonetics.json
bin/gogrammalecte generate thesaurus > thesaurus.json
bin/gogrammalecte generate hunspell --variant fr-classique > hunspell.zip
bin/gogrammalecte generate mythes > thesaurus.zip
bin/gogrammalecte generate statistics > frequency-report.txt
bin/gogrammalecte generate validation > lexical-warnings.json
bin/gogrammalecte generate ngrams --ngram-length 3 > trigrams.json
```

`build-dictionary --source path` accepts an external three-column lexicon;
`--source -` reads standard input. Hunspell generation supports all three
variants, morphology modes, aliases and simplified output. Lexical generation
recomputes frequencies from the four embedded original corpora.

## Compatibility evidence

The reference is Grammalecte revision
`08511c222029b3be20bf547563315d10fb48d44b`. Tests compare:

- All 14,160 grammar cases, including positions, identifiers, messages, colors,
  URLs and suggestions; 20 complete sentence/token-analysis snapshots.
- All 10 rule graphs and 4,396,046 expanded paths, plus 181 regex triggers.
- All 515,693 generated inflections and complete lexical outputs, including
  frequency allocation, phonetic codes, Hunspell and dictionary regeneration.
- Conjugation, inflection, morphology, phonetics, suggestions, typography,
  tokenization, dictionary queries and Unicode positions against Python fixtures.

Six grammar cases have suggestion order that changes between Python hash seeds.
Only those lists are compared as multisets; Go supplies deterministic ordering.
The standalone and platform checks are recorded in [VALIDATION.md](VALIDATION.md).
Other known boundaries are recorded in [PORTING.md](PORTING.md). Passing the
pinned corpus does not establish equivalence for every possible input.

The missing original frequency corpora were recovered from Fedora's archived
upstream source package. Their archive and individual hashes are verified;
[provenance](lexicon/data/provenance.md) records the original URLs and the limit
on verifying today's unversioned upstream copies.

## License

GNU GPL version 3; see [LICENSE](LICENSE). Source derivatives retain their
original notices, including MPL 2.0 for the applicable dictionary-generation
components. Grammalecte was created by Olivier R. See each resource's provenance
for its source and attribution. Code and development documentation are English;
French linguistic data and original diagnostic text are preserved.
