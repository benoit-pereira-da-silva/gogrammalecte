# Diagnostic rendering reference

`rendering.json` contains Python 3.14.3 observations captured before implementing
the corresponding Go functions. The source is `gc_core/py/text.py` at Grammalecte
commit `08511c222029b3be20bf547563315d10fb48d44b` (Olivier R., GPLv3).

The fixture covers 153 source-preserving paragraph wraps, 459 message wraps,
6 individual readable errors, 48 wrapped diagnostic groups, 120 annotated
paragraphs, 6 numbered-line merges and 30 coordinate conversions. Tests read
JSON only, with no interpreter or checkout dependency.

Message wrapping reproduces the Python standard library's default `textwrap`
semantics: tab expansion, ASCII whitespace replacement, Unicode whitespace
trimming, hyphen-aware word splitting, long-word breaking and hanging indents.
Its native implementation uses explicit scanning rather than a regex fallback.

`GenerateParagraph` preserves reference mutation of the supplied diagnostics by
assigning `iError`. `ConvertToXY` replaces absolute offsets in the supplied maps
with line and column coordinates. All offsets count Unicode code points.

Go APIs return explicit errors for nonpositive widths and malformed diagnostic
records. The former prevents the source wrapper's lack of forward progress;
the latter replaces Python's fallback string containing a dictionary repr.
