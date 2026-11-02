# Tokenizer reference cases

`tokens.json` contains 1,048 token streams captured from the pinned Python source.
The `source` object records the revision, checksums and Python version. Inputs
cover both French and default rules, with and without boundary sentinels,
Unicode combining characters, non-BMP symbols, paths, links, ordinal numbers,
clocks, elisions, punctuation and seeded random combinations.

The Go implementation uses the standard regular-expression library. Python's
Unicode word/digit classes are expanded explicitly. The word-pattern negative
lookahead excludes underscore through a character class. Hour/ordinal end
boundaries are validated explicitly, including the shorter-hour alternative
when optional seconds cannot satisfy the original boundary. Matching order,
normalization and Unicode code-point offsets are compared exactly.

Unknown languages intentionally select default rules. The Python constructor
sets a fallback language but accidentally indexes the original unknown key and
raises `KeyError`. This constructor inconsistency is not reproduced.

The source module was written by Olivier R. and is distributed under GNU GPL
version 3. Tests read the recorded data directly and require no interpreter or
upstream checkout. Inputs to the Go API must be valid UTF-8.
