# French language resources and compatibility

All upstream files are pinned to Grammalecte commit
`08511c222029b3be20bf547563315d10fb48d44b` in
`benoit-pereira-da-silva/grammalecte`.

## Runtime

The package ports `conj.py`, `mfsp.py`, `phonet.py`, `cregex.py` and
`thesaurus.py` from `gc_lang/fr/modules`. It also ports the date helper,
the independent analyser predicates, and all grammatical suggestion helpers from
`gce_date_verif.py`, `gce_analyseur.py` and `gce_suggestions.py`.
The experimental `conj_generator.py` is also ported, including its original
limitation to first- and second-group generation templates.
`textformatter.py` is ported with all 53 options and 186 ordered replacements.
These modules were written by Olivier R. and are distributed with Grammalecte
under GPLv3. Original French labels and strings remain linguistic data.

Tables are embedded with Go's `embed` package and decoded lazily. Public lookup
results are copied so callers cannot mutate the shared tables. No runtime
resource path, Python interpreter, external process or parent checkout is needed.

`data/conj.json`, `data/mfsp.json` and `data/phonet.json` were taken from the
pinned JavaScript resource files only after checking that each JSON value
exactly matched its Python resource counterpart. `data/thesaurus.json` is a JSON
serialization of the pinned Python `dThesaurus` table.
`data/conj_generator.json` serializes the original static `oConj` rule table;
its six regular-expression constraints all use supported standard-library syntax.
`data/textformatter.json` serializes the original options, defaults and ordered
replacement rules. Simple patterns use Go's standard-library regexp; Python
Unicode classes and lookaround use the tested native compatibility engine.

## Native generation

The package includes compressed original source inputs and native Go generators:

| Generator | Source | Verification |
|---|---|---|
| `GenerateInflectionData` | `data/dictDecl.txt` | Complete JSON value equality with the pinned tables. |
| `GenerateConjugationData` | `data/dictConj.txt` and a morphology lexicon | Complete pattern/type/tag/verb equality; derived-noun set equality. |
| `GeneratePhoneticData` | `data/phonet_simil.txt`, conjugations and a morphology lexicon | Every indexed phonetic group and morphology equals the pinned result. |
| `GenerateThesaurusData` | `dictionnaire/thesaurus/thes_fr.dat` and `synsets_fr.dat` | Complete merged JSON value equality. |

The generator behavior derives from `gc_lang/fr/build_data.py` (Olivier R.,
MPL 2.0) and `gc_lang/fr/dictionnaire/thes_build.py`. Source archives use gzip
with a fixed timestamp. The generator itself is Go and reads only embedded
inputs unless a caller supplies its own reader to a `Compile` function.

Two build-output ordering differences are intentional and explicit. The upstream
conjugation generator converts a Python set into derived-noun tuples, making
that order dependent on Python's hash seed; the Go generator sorts these names.
The phonetic generator traverses set members in sorted order, stabilizing indices
of additional merged groups. The resulting group memberships and morphologies
match the pinned tables. Runtime tables preserve the original pinned ordering.

These functions rebuild the language tables from the committed linguistic
inputs. They do not recreate the earlier Hunspell lexicon/frequency-corpus
production pipeline or the grammar-rule compiler.

## Parity tests

The tests were written before the runtime implementations and use Python 3.14.3
reference observations captured from the pinned source. Ordinary Go tests need
no Python and do not regenerate expected outputs from the implementation under test.

- All 8,403 verbs are checked across 143 tense/person combinations using a
  per-verb SHA-256 of NUL-separated verb, tense, person, result and presence flag.
  Raw verb classifications, derived names and selected infinitive-similar forms
  are also compared individually.
- Every registered masculine form and irregular plural is compared with the
  Python helper output.
- Every phonetic word is checked, including six morphology filters and selected
  capitalization/unknown-word cases.
- 384 full display conjugation tables cover twelve verbs and every combination
  of the five boolean formatting options.
- Morphology predicates, named patterns, date/calendar behavior, analyser
  predicates and typography transformations use targeted oracle fixtures.
- Thesaurus lookups use a deterministic sample across the complete word index;
  native regeneration compares the complete table.
- Generator integration uses the independently embedded dictionary package.
- 9,583 grammatical suggestion cases cover conjugation, gender, number,
  agreement, phonetic substitutions and compound verbs. The reference oracle
  uses first-seen lemma order to remove the nondeterminism of Python sets.
- 16,906 generated conjugation cases cover every known verb, both participle
  modes, uppercase and novel infinitives, with ordered form/tag hashes. Separate
  fixtures verify rule selection for every group-limit option.
- 1,413 individual formatter cases include a positive witness for every
  replacement rule, and 444 complete formatting cases exercise the defaults,
  option overrides and the original formatter test document's text.

## Public API boundaries

`GetConjugation` reports unknown verbs and tenses as errors. A known verb's
missing form is an empty string without error. `HasConjugation` returns false
for unknown verbs, matching the Python helper. Invalid formatted-conjugation
arguments return Go errors; display tables retain upstream labels and fallback
values.

Phonetic filtering accepts standard-library regexp syntax and returns a compile
error for unsupported expressions. It does not silently weaken Python regexes.
A general Python-regex compatibility layer, if needed by the grammar engine,
should supply separately tested matching rather than assuming this API accepts
all Python patterns.

The historical weekday helper intentionally follows Grammalecte's fixed shift
before the French calendar transition, including its empty weekday for
December 10–19, 1582. It is not presented as a general Julian-calendar library.

Grammar parsing and rule execution are separate components. Compound-verb morphology slicing
is integrated with the grammar token helpers.
