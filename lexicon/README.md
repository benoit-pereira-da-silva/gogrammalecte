# French lexical generator

This package ports the lexical core of Grammalecte's `genfrdic.py` to native Go.
The pinned dictionary, affixes and original frequency corpora are embedded. A
standalone executable can regenerate lexical resources without installed data
files, network access or Python.

```go
dictionary, err := lexicon.EmbeddedDictionary()
if err != nil { return err }
dictionary.SortNatural()
if err := dictionary.Generate(); err != nil { return err }
dictionary.CalculatePhonetics()
statistics, err := dictionary.EmbeddedStatistics()
if err != nil { return err }
dictionary.CalculateFrequencies(statistics)
dictionary.SortByFrequency()
return dictionary.WriteGrammarLexicon(output, "test")
```

`ReadDictionary` accepts custom dictionary and affix readers. `CorpusNames` and
`Corpus` provide the original uncompressed frequency streams. `NewStatistics`
and `Statistics.Add` permit caller-supplied corpora. Close every corpus reader.

The native generator covers:

- Dictionary and affix parsing, morphology, natural sorting, continuation flags,
  prefix/suffix expansion and duplicate/ambiguity counts.
- Original corpus accumulation, ordered ambiguity allocation, frequencies,
  frequency indices, full statistics reports and frequency-change exports. Entry order affects allocation; use `SortNatural` to
  reproduce the upstream build.
- Metagraphe and Double Metaphone, including original algorithm edge cases.
- Complete grammar-checker lexical, conjugation, declension and full lexical
  table output through `WriteGrammarLexicon`, `WriteConjugations`,
  `WriteDeclensions` and `WriteLexicon`.
- Classic, 1990 reform and all-variants Hunspell `.dic`/`.aff` output through
  `WriteHunspell`, with modes 0–2, AM/AF aliases and simplified output.
- Historical lexical ngram extraction and an ordered ngram index.
- Source entry validation warnings and alias-compression reports.
- MyThes DAT/IDX and source-ordered JSON exports for the complete thesaurus,
  reusing the French package’s original embedded sources.

Call `CalculatePhonetics` and `CalculateFrequencies` before writing the full
lexical table. Writing Hunspell files uses the current entry frequency fields;
frequency calculation updates those fields just as in the reference.

## Compatibility evidence

The native tests verify all 87,171 pinned entries, 6,003 affix rules, 177 flags,
and 515,693 generated inflections. Complete-output digests cover inflections,
phonetic codes, grammar-checker rows, frequency allocation, frequency ordering,
conjugation and declension data, and the complete 55,312,242-byte lexical table.
All 21 supported Hunspell variant/mode pairs
match Python output byte for byte. Small fixtures also exercise malformed
inputs and formatting edge cases. Tests use checked-in oracle outputs and
embedded data; they do not execute Python.

The original implementation's duplicate-corpus-row append behavior and lexical
ngram off-by-two range are preserved. Its dormant mixed prefix/suffix path can
produce an invalid two-field tuple; the Go port reports that case as an error.
No pinned source entry triggers it. Unsupported affix regex constructs are
returned as errors. Ngram word references have deterministic insertion order,
without depending on Python set iteration. Invalid IDs and malformed corpus
rows return Go errors instead of printing diagnostics.

Unicode classification follows the Go standard library; full case mapping and
normalization use `golang.org/x/text`. The pinned French resources and recorded
Unicode cases match. Equality across every Unicode code point is not claimed
when runtime Unicode table versions differ.

Application-specific LibreOffice/Mozilla packages and their templates are
outside this standalone library’s core lexical outputs. Validation uses a stable
`lx`, `se`, `et` field order instead of Python’s variable set iteration. Original
French diagnostics and report labels are retained as output-format data.

`CalculateFrequenciesReport(statistics, writer)` calculates frequencies and emits
the full statistical report in one pass; use it instead of a separate
`CalculateFrequencies` call when a report is needed. `WriteAliasReport` appends
compression rankings. `Statistics.Write` emits the original count dump, and
`WriteFrequencyChanges` emits frequency index changes and accompanying notes.
`EmbeddedThesaurus` returns ordered data supporting `WriteMyThes` and `WriteJSON`.

See [data provenance](data/provenance.md) for original archive hashes, per-corpus
hashes and the qualification concerning current unversioned upstream data.
The lexical-generator derivatives preserve their Mozilla Public License 2.0
attribution. The Double Metaphone file retains its algorithm authorship notices.
