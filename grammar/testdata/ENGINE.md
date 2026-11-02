# Grammar diagnostic reference

`engine.json.gz` contains every one of the 14,160 `TEST:` inputs in the pinned
French source, with complete Python diagnostic objects. Source revision:
`08511c222029b3be20bf547563315d10fb48d44b`.

The oracle uses the original Python regex compiler, engine, helper plugins,
linguistic modules and tokenizer. Token graphs come from the separately verified
full Python graph fixture. The dictionary is the full native-generated graph,
whose lexical contents and graph encoding have independent Python parity tests.
Modules are loaded under their installed `grammalecte` package names so the
French lexicographer is active. Default Python options, country FR and original
RGB diagnostic colors are used. A test's explicit option is enabled for that
case and restored afterward. Error markers and expected-suggestion annotations
are removed as in the original test runner.

Four independent oracle runs used `PYTHONHASHSEED=0,1,2,3`. Exactly six cases
changed suggestion order while retaining identical suggestions and all other
diagnostic fields. Those cases alone carry `unordered_suggestions` identifying
the affected rule. The test compares those lists as sorted multisets; all other
suggestions and all diagnostic ordering remain exact. Go retains deterministic
first-seen lemma order rather than reproducing Python's randomized set order.

The complete Go test fails on any unexplained difference. Python is not needed
to build or run it. Additional full-information fixtures exercise sentence
metadata, token mutation, labels and diagnostic context.
