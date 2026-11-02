# String primitive reference data

`primitives.json` records Python outputs from the source commit, file checksums
and interpreter version in its `source` field. It covers 248 input strings and
510 pairs. The strings include targeted Unicode/casing examples, seeded random
strings and a sample of the French lexical data. Distances are compared with a
small floating-point tolerance; string and predicate outputs are exact.

The tests were written and observed failing before the implementations. The
weighted edit-distance scratch matrix is now local to each call, preserving
results while permitting parallel callers. Jaro-Winkler retains the unusual
index ordering of the source implementation. Sift4 retains its original
candidate traversal order. Diagnostic matrix-printing functions are not public
library operations.

`digits.go` contains the non-decimal characters for which Python 3.14.3 reports
`isdigit()`. NFC and full case conversion use Unicode data from `x/text`.
Unicode versions can differ across toolchains; the captured fixture scope does
not establish equality over all future Unicode additions.

The reference functions are in `graphspell/str_transform.py` by Olivier R.,
distributed under GNU GPL version 3. Python is a development oracle only;
normal tests use the stored observations without invoking an interpreter.
