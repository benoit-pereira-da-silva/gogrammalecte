# Embedded spelling character tables

`characters.json` preserves the character substitution costs, ordered spelling
replacements, vowel/consonant/double-letter sets and keyboard proximity tables
from Grammalecte's `char_player.py` and `keyboard_chars_proximity.py`. The source
revision and SHA-256 hashes are included in the JSON. These tables were created
by Olivier R. and are distributed under GNU GPL version 3.

The Go package embeds this file into the binary. Public collection accessors
return copies to prevent callers from mutating the shared data. Character costs
are asymmetric where the reference defines asymmetric values; no symmetry is
added. Linguistic strings and keyboard neighbour sequences retain their original
ordering and spelling.
