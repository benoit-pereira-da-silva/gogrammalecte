# Native rule compiler

`FrenchSource` reads the embedded original `rules.grx` at reference revision
`08511c222029b3be20bf547563315d10fb48d44b`. `Parse` preserves regex rules,
ordered actions, graph source groups, interleaved passes, options, labels and
colors. `CompileGraphs` expands and minimizes all graph rules entirely in Go.
`FrenchGraphs` loads the native-generated embedded result for faster startup.
Conditions and computed replacements are typed source expressions evaluated by
the native engine; no Python or JavaScript is generated or executed.

The source contains 181 regex triggers, 237 regex actions, 1,353 graph groups,
10 token graphs and 14,160 grammar tests. Graph expansion yields 4,396,046 paths.
Tests compare every minimized transition and action against the original Python
compiler and independently regenerate the embedded graphs. See
[data/graphs_provenance.json](data/graphs_provenance.json) for source hashes and
per-graph counts. The source rules and compiler semantics are by Olivier R.,
GPLv3 or later.

Unsupported statements and patterns produce located errors. Compiling custom
rules does not grant expressions access to processes, files or network calls.
