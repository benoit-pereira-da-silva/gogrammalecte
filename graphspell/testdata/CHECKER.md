# Spelling checker reference

`spellchecker.json` was captured from `graphspell/spellchecker.py` at revision
`08511c222029b3be20bf547563315d10fb48d44b`, using the complete dictionary and the
French lexicographer on both the dictionary and checker. It covers paragraph
errors, Unicode offsets, word/lemma/unknown counts, abbreviation substitutions,
casing and filtered suggestion groups. The native dictionary has separate
full-entry and suggestion reference tests.

`graphspell/data/default_suggestions.json` contains the original `dSugg` table
from `graphspell/lexgraph_fr.py`, by Olivier R., GPLv3 or later. It is embedded
in the executable and never read from an external runtime file.
