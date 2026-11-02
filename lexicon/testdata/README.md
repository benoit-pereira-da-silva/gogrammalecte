# Lexical generator oracle fixtures

These fixtures were captured with CPython 3.14.3 before implementing the
corresponding Go behavior. The source revision is
`08511c222029b3be20bf547563315d10fb48d44b` of the Grammalecte reference.
`source.json` records the lexical generator, dictionary and affix SHA-256 hashes.

- `corpora.json`: original byte counts, SHA-256 hashes, row counts, occurrence
  totals and compressed hashes for all four recovered frequency corpora.
- `generator.json`: 671 entry cases, all 6,003 rule representations, all 177 flags,
  source settings, and full ordered inflection/grammar-row digests.
- `statistics.json`: mixed recognized/unknown words, missing words and duplicate
  corpus-row behavior, including the original report formatting.
- `frequencies.json`: complete native-build checks against the original recovered
  corpora, including ordered ambiguity allocation, per-entry counts, sorted
  lexical output, conjugation and declension files.
- `hunspell.json`: full output digests for three dictionary variants, modes 0–2,
  with and without alias compression, and simplified mode 0.
- `phonetics.json`: 1,800 synthetic and real-word cases, source hashes, and a
  combined digest over phonetic codes for all 515,693 inflections.
- `thesaurus.json`: exact complete DAT, IDX and source-ordered JSON digests, plus
  label, whitespace, duplicate and entry-order edge cases.
- `validation.json`: all warning-bearing source entry cases and invalid synthetic
  inputs. The complete loaded dictionary also verifies without false warnings.
- `aliases.json`: compression frequency report formatting and ranking.
- `statistics-report.txt.gz`: original full report, retained for diagnosis in
  addition to its checked SHA-256.
- `outputs.json`: full lexical-table rows, headers, ngrams, flags, morphology
  combinations and field formatting.

The frequency oracle uses the archived original corpus files documented in
`../data/provenance.md`. Identical source README content corroborates their
provenance but does not establish equality with current unversioned upstream
storage. Every test runs from this Go repository alone.
