# Text segmentation reference data

`text.json` contains 100 cases captured directly from Grammalecte's Python
`gc_core/py/text.py` at commit
`08511c222029b3be20bf547563315d10fb48d44b`. The fixture records the source file's
SHA-256 and the Python version used to capture it.

For every input, the expected values are the complete results of
`getParagraph`, `getSentenceBoundaries`, and `getSentence`. They were recorded
before implementing the corresponding Go functions. The Go tests read the
committed JSON and need neither Python nor an upstream checkout.

The cases cover empty and trailing paragraphs, CR/LF normalization, punctuation
runs, quotes, continuation capitals, ordinary and nonbreaking spaces, combining
characters, and characters outside the Basic Multilingual Plane. French text
is linguistic test data. Sentence positions count Unicode code points and use
an exclusive end position.

To refresh this reference, execute those three functions from the pinned
Python source on each stored input, then update the provenance with the source
revision, file checksum, and interpreter version. Review intentional upstream
behavior changes before replacing the expected values. Do not regenerate the
expected values from the Go implementation under test.

The original helpers were written by Olivier R. and are distributed by
Grammalecte under GPLv3; the port uses the repository's accompanying license.
