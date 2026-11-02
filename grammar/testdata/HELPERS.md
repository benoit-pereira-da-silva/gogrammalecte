# Grammar helper reference fixtures

`helpers.json` contains 5,135 observations captured from Python 3.14.3 before
the native helper implementation. The source is Grammalecte commit
`08511c222029b3be20bf547563315d10fb48d44b`, principally
`gc_core/py/lang_core/gc_functions.py`, with compound-verb slicing from
`gc_lang/fr/modules/gce_analyseur.py` (Olivier R., GPLv3).

The fixture includes a small, explicit morphology lexicon. Each case records
the input arguments, return value and arguments after execution so tests check
both predicates and mutations. Coverage includes cache precedence, empty
analyses, Unicode casing and slicing, positive and negative morphology filters,
word navigation, capture offsets, metadata, tags, spacing and disambiguation.
The Go tests consume JSON only and require no Python installation or checkout.

Pattern matching uses the native Python-regex compatibility package. Unsupported
expressions return errors. Morphology rewriting currently supports literal
replacement, numerical group references and common replacement escapes. Named
replacement references return an explicit tested error; the pinned grammar has
no named references in morphology rewrites. Diagnostic-only `info` and `g_info`
callbacks are left to the calling engine's logging interface.
