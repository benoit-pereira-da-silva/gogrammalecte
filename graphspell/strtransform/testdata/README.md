# Affix reference fixtures

`affix.json` records the results of the original Python implementation of
`defineSuffixCode`, `defineAffixCode`, `changeWordWithSuffixCode`, and
`changeWordWithAffixCode`. Its `source` object identifies the repository, commit,
source-file SHA-256, and Python version used to produce these observations.

The 285 word/stem pairs contain 29 targeted cases and 256 entries selected from
`lexicons/French.lex`. For the lexical sample, read nonempty, non-comment lines,
split each line on tabs, retain its first two fields, and select entry
`i * (entry_count - 1) // 255` for each integer `i` from 0 through 255.

The targeted cases cover accents, decomposed characters, non-BMP characters,
empty strings, longest-substring ties, removals longer than nine characters,
literal replacements, and slash-containing codes. Seventeen additional decoding
cases capture malformed codes and Python's negative/clamped slice behavior.
Reference exceptions are recorded by class name; the Go API reports them as
errors matching `ErrInvalidCode`.

These are frozen observations, not expectations computed by the Go code under
test. Normal tests need only the Go toolchain and this repository. Python and
the upstream checkout were used to capture the reference results; neither is
required to run the test suite. French words remain unchanged because they are
linguistic data.

The fixtures prove this selected transformation slice. They do not establish
parity of the complete dictionary or grammar engine.
