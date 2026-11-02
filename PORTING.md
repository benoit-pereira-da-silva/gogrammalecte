# Porting plan and compatibility boundaries

## Reference and objective

Target the Python behavior of Grammalecte commit
`08511c222029b3be20bf547563315d10fb48d44b`. The JavaScript implementation is
useful evidence, but it already differs from Python in Unicode handling and
some spelling transformations. It is not a second simultaneous oracle.

The complete target includes the grammatical engine, dictionaries, suggestions,
rule compiler, graph construction and linguistic resource generators. Current
tests cover only the components listed in the README.

## Standard-library regex strategy

Go's `regexp` cannot directly express every Python pattern. Keep regex matching
in the standard library where possible, and model missing assertions explicitly:

- Find candidate matches, then check preceding or following context without
  consuming that context.
- Preserve Python's greedy matching, capture spans and retry behavior when a
  context check rejects a candidate. Filtering one match per position can miss
  valid shorter alternatives and is not a general solution.
- Handle backreferences through explicit captured-text comparisons only when
  equivalence has been demonstrated for the affected rule.
- Match Python character classes, case behavior and word boundaries explicitly;
  Go's default `\w` and `\b` are not Python's Unicode definitions.
- Fail clearly for a construct that has not been ported. Never disable a rule
  or return an incomplete analysis as if it were complete.

Do not build a general replacement for Python's regex engine before the actual
rule inventory establishes what is needed. Each lowering must have differential
tests against the reference, including overlapping candidates and Unicode.

## Text, state and ordering

Preserve Unicode code-point positions internally and specify offset units at
each public boundary. Converting to UTF-8 byte or UTF-16 offsets belongs at an
explicit boundary. Normalization must retain the mapping to the original text.

Represent ordered arcs, rules and suggestions as sequences. Map iteration is
not a replacement for Python dictionary insertion order. Keep mutable parse
state and scratch buffers isolated; shared dictionaries should be immutable,
with a documented cache policy.

## Subsequent increments

1. Extend reference fixtures and port the remaining text and string utilities.
2. Implement DAWG decoding and verify lexical entries, lemmata and morphologies.
3. Port linguistic helpers and spelling suggestions, including their ordering.
4. Parse the rule language into a typed representation; generate Go functions
   and rule graphs with source locations and explicit unsupported-construct errors.
5. Implement grammatical passes, rewriting, disambiguation, options and priority.
6. Port dictionary and linguistic-data builders, then the full CLI behavior.
7. Run a blocking differential suite and measure resource usage on all targets.

The upstream grammar corpus declares 14,160 cases. Its original runner can
print discrepancies without failing the process; the Go parity runner must
fail on unexplained differences.

Four frequency corpora used by the upstream dictionary generator are absent
from the source checkout. Reproducing that entire generation chain requires
recovering and pinning those inputs. Existing lexical data can support early
reader tests, but does not prove complete generator reproducibility.
