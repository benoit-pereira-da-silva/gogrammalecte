# Expression reference

`expressions.json` records Python values for operators, boolean short-circuiting,
conditional expressions, membership, Unicode indexing/slicing and string methods.
Additional tests compile every condition and computed action/message from the
pinned French regex and token rules. The full grammar differential suite verifies
the resulting expression behavior in its real token and morphology context.
The parser and evaluator are native Go and expose only named grammar helpers.
