# French lexical labels

`lexicographer.json` records Python 3.14.3 observations of
`graphspell/lexgraph_fr.py` and the labeling and analysis methods from
`graphspell/spellchecker.py`, pinned to Grammalecte commit
`08511c222029b3be20bf547563315d10fb48d44b` (Olivier R., GPLv3).

Tests cover 10,085 readable morphology strings, including every distinct tag
combination in the bundled original lexicon; 257 word decompositions and
analyses; and 4,641 labeled-token cases with absent, empty and explicit cached
morphologies. French labels are preserved as source linguistic data. Ordinary
tests require neither Python nor source checkouts.

`graphspell/data/lexicographer.json` embeds the source tag descriptions, value
descriptions and decomposition patterns. The native implementation intentionally
preserves historical behavior such as the past-participle display's initial
four-character removal, and the absence of a specialized label for HASHTAG
tokens (the source recognizes TAG instead).

Detailed grammar token snapshots sort tags and omit recursive multi-token
members while retaining their index bounds and morphology, allowing safe JSON
serialization without sharing mutable parser state.
