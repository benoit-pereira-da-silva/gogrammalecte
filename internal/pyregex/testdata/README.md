# Python regular-expression reference

`patterns.json.gz` contains 3,003 matching cases captured with Python `re`.
They exercise all 181 expanded French regex triggers from Grammalecte revision
`08511c222029b3be20bf547563315d10fb48d44b`, positive and negative source test
sentences, Unicode, captures, lookarounds, backreferences and empty matches.
The regex compiler uses Go's `regexp/syntax`; missing Python assertions and
backreferences are evaluated explicitly with cancellation and a work limit.

This is the subset needed by the pinned rules, not a claim to implement every
Python regex extension. Unsupported constructs fail compilation. Unicode
categories follow Go's Unicode tables; equality with every Unicode version of
Python is not claimed. The French corpus and targeted Unicode cases are tested.
