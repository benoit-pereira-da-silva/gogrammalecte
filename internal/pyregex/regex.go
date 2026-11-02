// Package pyregex implements the regular-expression semantics needed by the
// pinned Grammalecte rules. Go's regexp/syntax parses ordinary expressions;
// explicit assertions and backreferences preserve Python's matching behavior.
// Unsupported constructs fail compilation rather than weakening a rule.
package pyregex

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"unicode"
)

// ErrMatchLimit protects callers from unbounded backtracking on hostile input.
var ErrMatchLimit = errors.New("regular expression matching work limit exceeded")

const maxSteps = 10000000

type special struct {
	kind         string
	tree         *syntax.Regexp
	group, width int
	fold, multi  bool
}

// Regexp is immutable and safe for concurrent matching.
type Regexp struct {
	tree   *syntax.Regexp
	extras map[string]*special
	groups int
	names  map[string]int
}

// Match uses Unicode code-point offsets, with [-1,-1] for an unmatched group.
type Match struct {
	Spans [][2]int
	input []rune
	names map[string]int
}

// GroupIndex resolves a named capture without exposing the internal name map.
func (m Match) GroupIndex(name string) (int, bool) { index, ok := m.names[name]; return index, ok }

func (m Match) Group(n int) string {
	if n < 0 || n >= len(m.Spans) || m.Spans[n][0] < 0 {
		return ""
	}
	return string(m.input[m.Spans[n][0]:m.Spans[n][1]])
}
func (m Match) Start(n int) int {
	if n < 0 || n >= len(m.Spans) {
		return -1
	}
	return m.Spans[n][0]
}
func (m Match) End(n int) int {
	if n < 0 || n >= len(m.Spans) {
		return -1
	}
	return m.Spans[n][1]
}

type parser struct {
	extras         map[string]*special
	groups, serial int
	names          map[string]int
}

func Compile(pattern string) (*Regexp, error) {
	p := &parser{extras: map[string]*special{}, names: map[string]int{}}
	tree, err := p.parse(normalizeClasses(pattern), "")
	if err != nil {
		return nil, fmt.Errorf("compile %q: %w", pattern, err)
	}
	for _, x := range p.extras {
		if x.kind == "ref" && (x.group < 1 || x.group > p.groups) {
			return nil, fmt.Errorf("invalid group reference %d", x.group)
		}
	}
	return &Regexp{tree, p.extras, p.groups, p.names}, nil
}

// Python permits a literal closing bracket as a class's first member.
// Escape that member before passing the class to Go's syntax parser.
func normalizeClasses(pattern string) string {
	var out strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		out.WriteByte(c)
		if c == '\\' && i+1 < len(pattern) {
			i++
			out.WriteByte(pattern[i])
			continue
		}
		if c == '[' && !inClass {
			inClass = true
			if i+1 < len(pattern) && pattern[i+1] == '^' {
				i++
				out.WriteByte('^')
			}
			if i+1 < len(pattern) && pattern[i+1] == ']' {
				i++
				out.WriteString(`\]`)
			}
		} else if c == ']' && inClass {
			inClass = false
		}
	}
	return out.String()
}
func (p *parser) placeholder(x *special) string {
	name := fmt.Sprintf("assert%d", p.serial)
	p.serial++
	p.extras[name] = x
	return "(?P<" + name + ">)"
}
func groupEnd(s string, start int) (int, error) {
	depth := 1
	class := false
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '[':
			class = true
		case ']':
			class = false
		case '(':
			if !class {
				depth++
			}
		case ')':
			if !class {
				depth--
				if depth == 0 {
					return i, nil
				}
			}
		}
	}
	return 0, errors.New("unterminated group")
}
func (p *parser) parse(source, flags string) (*syntax.Regexp, error) {
	var b strings.Builder
	if flags != "" {
		b.WriteString("(?" + flags + ")")
	}
	inClass := false
	classStart := 0
	for i := 0; i < len(source); {
		c := source[i]
		if c == '\\' {
			if i+1 >= len(source) {
				return nil, errors.New("trailing escape")
			}
			e := source[i+1]
			i += 2
			switch e {
			case 'w':
				if inClass {
					b.WriteString(`\pL\pN_`)
				} else {
					b.WriteString(`[\pL\pN_]`)
				}
			case 'W':
				if inClass {
					return nil, errors.New("complement escape inside character class is not supported")
				}
				b.WriteString(`[^\pL\pN_]`)
			case 'd':
				b.WriteString(`\p{Nd}`)
			case 'D':
				b.WriteString(`\P{Nd}`)
			case 's':
				if inClass {
					b.WriteString(`\t\n\v\f\r\x{1c}-\x{20}\x{85}\p{Z}`)
				} else {
					b.WriteString(`[\t\n\v\f\r\x{1c}-\x{20}\x{85}\p{Z}]`)
				}
			case 'S':
				if inClass {
					return nil, errors.New("complement escape inside character class is not supported")
				}
				b.WriteString(`[^\t\n\v\f\r\x{1c}-\x{20}\x{85}\p{Z}]`)
			case 'b', 'B':
				if inClass {
					if e == 'b' {
						b.WriteString(`\x08`)
					} else {
						return nil, errors.New("invalid class escape")
					}
				} else {
					b.WriteString(p.placeholder(&special{kind: string(e)}))
				}
			case 'Z':
				b.WriteString(`\z`)
			case 'u', 'U':
				count := 4
				if e == 'U' {
					count = 8
				}
				if i+count > len(source) {
					return nil, errors.New("short Unicode escape")
				}
				n, err := strconv.ParseUint(source[i:i+count], 16, 32)
				if err != nil || n > unicode.MaxRune {
					return nil, errors.New("invalid Unicode escape")
				}
				b.WriteString(regexp.QuoteMeta(string(rune(n))))
				i += count
			default:
				if e >= '1' && e <= '9' && !inClass {
					start := i - 1
					for i < len(source) && source[i] >= '0' && source[i] <= '9' {
						i++
					}
					n, _ := strconv.Atoi(source[start:i])
					b.WriteString(p.placeholder(&special{kind: "ref", group: n, fold: strings.Contains(flags, "i")}))
				} else {
					b.WriteByte('\\')
					b.WriteByte(e)
				}
			}
			continue
		}
		if c == '[' {
			if !inClass {
				classStart = b.Len()
			}
			inClass = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == ']' {
			if inClass && strings.Contains(flags, "i") {
				// Add Python's extra Turkish-I case-fold equivalents to the
				// positive members before syntax.Parse expands a negated class.
				// Adding them after complementation would incorrectly match
				// excluded letters in patterns such as (?i)[^i].
				class := b.String()[classStart:]
				if strings.HasPrefix(class, "[^") {
					class = "[" + class[2:]
				}
				node, err := syntax.Parse("(?i)"+class+"]", syntax.Perl)
				if err != nil {
					return nil, err
				}
				if classContainsTurkishI(node) {
					// A final unescaped hyphen is a literal. Keep it literal
					// when adding class members after it.
					built := b.String()
					if strings.HasSuffix(built, "-") {
						backslashes := 0
						for j := len(built) - 2; j >= classStart && built[j] == '\\'; j-- {
							backslashes++
						}
						if backslashes%2 == 0 {
							b.Reset()
							b.WriteString(built[:len(built)-1])
							b.WriteString(`\-`)
						}
					}
					b.WriteString("iIİı")
				}
			}
			inClass = false
			b.WriteByte(c)
			i++
			continue
		}
		if !inClass && c == '$' {
			b.WriteString(p.placeholder(&special{kind: "end", multi: strings.Contains(flags, "m")}))
			i++
			continue
		}
		if !inClass && c == '(' {
			// Assertions are parsed recursively, sharing the capture-number namespace.
			kind, prefix := "", 0
			for _, a := range []struct{ prefix, kind string }{{"(?<=", "behind"}, {"(?<!", "notbehind"}, {"(?=", "ahead"}, {"(?!", "notahead"}} {
				if strings.HasPrefix(source[i:], a.prefix) {
					kind = a.kind
					prefix = len(a.prefix)
					break
				}
			}
			if kind != "" {
				end, err := groupEnd(source, i+prefix)
				if err != nil {
					return nil, err
				}
				tree, err := p.parse(source[i+prefix:end], flags)
				if err != nil {
					return nil, err
				}
				x := &special{kind: kind, tree: tree}
				if strings.HasSuffix(kind, "behind") {
					w, ok := p.width(tree)
					if !ok {
						return nil, errors.New("lookbehind requires a fixed width")
					}
					x.width = w
				}
				b.WriteString(p.placeholder(x))
				i = end + 1
				continue
			}
			if strings.HasPrefix(source[i:], "(?P=") {
				return nil, errors.New("named backreferences are not supported")
			}
			if strings.HasPrefix(source[i:], "(?P<") {
				end := strings.IndexByte(source[i+4:], '>')
				if end < 0 {
					return nil, errors.New("invalid named group")
				}
				name := source[i+4 : i+4+end]
				p.groups++
				if _, ok := p.names[name]; ok {
					return nil, errors.New("duplicate named group")
				}
				p.names[name] = p.groups
				b.WriteString(fmt.Sprintf("(?P<capture%d>", p.groups))
				i += end + 5
				continue
			}
			if i+1 < len(source) && source[i+1] == '?' {
				j := i + 2
				for j < len(source) && strings.ContainsRune("ims", rune(source[j])) {
					j++
				}
				if j > i+2 && j < len(source) && source[j] == ')' {
					for _, f := range source[i+2 : j] {
						if !strings.ContainsRune(flags, f) {
							flags += string(f)
						}
					}
					b.WriteString(source[i : j+1])
					i = j + 1
					continue
				}
				if strings.HasPrefix(source[i:], "(?:") {
					b.WriteString("(?:")
					i += 3
					continue
				}
				return nil, errors.New("unsupported group construct")
			}
			p.groups++
			b.WriteString(fmt.Sprintf("(?P<capture%d>", p.groups))
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	return syntax.Parse(b.String(), syntax.Perl)
}

func classContainsTurkishI(node *syntax.Regexp) bool {
	for _, candidate := range "iIİı" {
		switch node.Op {
		case syntax.OpLiteral:
			if len(node.Rune) == 1 && equalFold(node.Rune[0], candidate) {
				return true
			}
		case syntax.OpCharClass:
			for i := 0; i < len(node.Rune); i += 2 {
				if candidate >= node.Rune[i] && candidate <= node.Rune[i+1] {
					return true
				}
			}
		case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
			return true
		}
	}
	return false
}
func (p *parser) width(n *syntax.Regexp) (int, bool) {
	switch n.Op {
	case syntax.OpLiteral:
		return len(n.Rune), true
	case syntax.OpCharClass, syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return 1, true
	case syntax.OpEmptyMatch, syntax.OpBeginText, syntax.OpEndText, syntax.OpBeginLine, syntax.OpEndLine:
		return 0, true
	case syntax.OpCapture:
		if x := p.extras[n.Name]; x != nil {
			if x.kind == "ref" {
				return 0, false
			}
			return 0, true
		}
		return p.width(n.Sub[0])
	case syntax.OpConcat:
		sum := 0
		for _, s := range n.Sub {
			w, ok := p.width(s)
			if !ok {
				return 0, false
			}
			sum += w
		}
		return sum, true
	case syntax.OpAlternate:
		w, ok := p.width(n.Sub[0])
		if !ok {
			return 0, false
		}
		for _, s := range n.Sub[1:] {
			v, ok := p.width(s)
			if !ok || v != w {
				return 0, false
			}
		}
		return w, true
	case syntax.OpRepeat:
		if n.Min != n.Max {
			return 0, false
		}
		w, ok := p.width(n.Sub[0])
		return w * n.Min, ok
	}
	return 0, false
}

type state struct {
	pos   int
	spans [][2]int
}
type machine struct {
	re    *Regexp
	input []rune
	ctx   context.Context
	steps int
	err   error
}

func clone(s state) state { spans := append([][2]int(nil), s.spans...); return state{s.pos, spans} }
func isWord(r rune) bool  { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }
func equalFold(a, b rune) bool {
	if a == b {
		return true
	}
	if strings.ContainsRune("iIİı", a) && strings.ContainsRune("iIİı", b) {
		return true
	}
	for r := unicode.SimpleFold(a); r != a; r = unicode.SimpleFold(r) {
		if r == b {
			return true
		}
	}
	return false
}
func (m *machine) tick() bool {
	m.steps++
	if m.steps > maxSteps {
		m.err = ErrMatchLimit
		return false
	}
	if m.steps%256 == 1 {
		m.err = m.ctx.Err()
	}
	return m.err == nil
}
func (m *machine) run(n *syntax.Regexp, s state, yield func(state) bool) bool {
	if !m.tick() {
		return false
	}
	switch n.Op {
	case syntax.OpNoMatch:
		return false
	case syntax.OpEmptyMatch:
		return yield(s)
	case syntax.OpLiteral:
		if s.pos+len(n.Rune) > len(m.input) {
			return false
		}
		for i, r := range n.Rune {
			c := m.input[s.pos+i]
			if c != r && (n.Flags&syntax.FoldCase == 0 || !equalFold(c, r)) {
				return false
			}
		}
		s.pos += len(n.Rune)
		return yield(s)
	case syntax.OpCharClass:
		if s.pos == len(m.input) {
			return false
		}
		c := m.input[s.pos]
		found := false
		for i := 0; i < len(n.Rune); i += 2 {
			if c >= n.Rune[i] && c <= n.Rune[i+1] {
				found = true
				break
			}
		}
		if !found && n.Flags&syntax.FoldCase != 0 && (c == 'İ' || c == 'ı') {
			for i := 0; i < len(n.Rune); i += 2 {
				if 'i' >= n.Rune[i] && 'i' <= n.Rune[i+1] || 'I' >= n.Rune[i] && 'I' <= n.Rune[i+1] {
					found = true
					break
				}
			}
		}
		if found {
			s.pos++
			return yield(s)
		}
		return false
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		if s.pos < len(m.input) && (n.Op == syntax.OpAnyChar || m.input[s.pos] != '\n') {
			s.pos++
			return yield(s)
		}
		return false
	case syntax.OpBeginText:
		if s.pos == 0 {
			return yield(s)
		}
		return false
	case syntax.OpEndText:
		if s.pos == len(m.input) {
			return yield(s)
		}
		return false
	case syntax.OpBeginLine:
		if s.pos == 0 || m.input[s.pos-1] == '\n' {
			return yield(s)
		}
		return false
	case syntax.OpEndLine:
		if s.pos == len(m.input) || m.input[s.pos] == '\n' {
			return yield(s)
		}
		return false
	case syntax.OpCapture:
		if x := m.re.extras[n.Name]; x != nil {
			return m.assert(x, s, yield)
		}
		group, _ := strconv.Atoi(strings.TrimPrefix(n.Name, "capture"))
		next := clone(s)
		next.spans[group] = [2]int{s.pos, -1}
		return m.run(n.Sub[0], next, func(result state) bool {
			result = clone(result)
			result.spans[group][1] = result.pos
			return yield(result)
		})
	case syntax.OpConcat:
		var next func(int, state) bool
		next = func(i int, s state) bool {
			if i == len(n.Sub) {
				return yield(s)
			}
			return m.run(n.Sub[i], s, func(out state) bool { return next(i+1, out) })
		}
		return next(0, s)
	case syntax.OpAlternate:
		for _, sub := range n.Sub {
			if m.run(sub, s, yield) {
				return true
			}
			if m.err != nil {
				return false
			}
		}
		return false
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		lo, hi := n.Min, n.Max
		switch n.Op {
		case syntax.OpStar:
			lo, hi = 0, -1
		case syntax.OpPlus:
			lo, hi = 1, -1
		case syntax.OpQuest:
			lo, hi = 0, 1
		}
		greedy := n.Flags&syntax.NonGreedy == 0
		var repeat func(int, state) bool
		repeat = func(count int, current state) bool {
			if !m.tick() {
				return false
			}
			if !greedy && count >= lo && yield(current) {
				return true
			}
			if hi < 0 || count < hi {
				if m.run(n.Sub[0], current, func(out state) bool {
					if out.pos == current.pos {
						if count+1 >= lo {
							return yield(out)
						}
						return repeat(count+1, out)
					}
					return repeat(count+1, out)
				}) {
					return true
				}
			}
			return greedy && count >= lo && yield(current)
		}
		return repeat(0, s)
	}
	m.err = fmt.Errorf("unhandled regular expression operation %v", n.Op)
	return false
}
func (m *machine) assert(x *special, s state, yield func(state) bool) bool {
	switch x.kind {
	case "b", "B":
		before := s.pos > 0 && isWord(m.input[s.pos-1])
		after := s.pos < len(m.input) && isWord(m.input[s.pos])
		ok := before != after
		if x.kind == "B" {
			ok = !ok
		}
		if ok {
			return yield(s)
		}
		return false
	case "end":
		if s.pos == len(m.input) || s.pos < len(m.input) && m.input[s.pos] == '\n' && (x.multi || s.pos == len(m.input)-1) {
			return yield(s)
		}
		return false
	case "ref":
		span := s.spans[x.group]
		if span[0] < 0 || span[1] < 0 {
			return false
		}
		size := span[1] - span[0]
		if s.pos+size > len(m.input) {
			return false
		}
		for i := 0; i < size; i++ {
			a, b := m.input[span[0]+i], m.input[s.pos+i]
			if a != b && (!x.fold || !equalFold(a, b)) {
				return false
			}
		}
		s.pos += size
		return yield(s)
	default:
		start := s
		behind := strings.HasSuffix(x.kind, "behind")
		if behind {
			start.pos -= x.width
		}
		found := false
		var result state
		if start.pos >= 0 {
			m.run(x.tree, start, func(out state) bool {
				if behind && out.pos != s.pos {
					return false
				}
				found = true
				result = out
				return true
			})
		}
		if strings.HasPrefix(x.kind, "not") {
			if !found && m.err == nil {
				return yield(s)
			}
			return false
		}
		if found {
			result.pos = s.pos
			return yield(result)
		}
		return false
	}
}

// FindAll returns non-overlapping matches, including Python's empty-match
// behavior. A negative limit means all matches. Resource exhaustion is an error.
func (r *Regexp) FindAll(ctx context.Context, input string, limit int) ([]Match, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := []Match{}
	m := &machine{re: r, input: []rune(input), ctx: ctx}
	pos, lastEmpty := 0, -1
	for pos <= len(m.input) && (limit < 0 || len(out) < limit) {
		spans := make([][2]int, r.groups+1)
		for i := range spans {
			spans[i] = [2]int{-1, -1}
		}
		var result state
		found := m.run(r.tree, state{pos, spans}, func(s state) bool {
			if s.pos == pos && lastEmpty == pos {
				return false
			}
			result = s
			return true
		})
		if m.err != nil {
			return nil, m.err
		}
		if found {
			result.spans = append([][2]int(nil), result.spans...)
			result.spans[0] = [2]int{pos, result.pos}
			out = append(out, Match{result.spans, m.input, r.names})
			if result.pos == pos {
				lastEmpty = pos
			} else {
				lastEmpty = -1
			}
			pos = result.pos
		} else {
			pos++
			lastEmpty = -1
		}
	}
	return out, nil
}

// Search finds the first match, or nil if there is none.
func (r *Regexp) Search(ctx context.Context, input string) (*Match, error) {
	m, err := r.FindAll(ctx, input, 1)
	if err != nil || len(m) == 0 {
		return nil, err
	}
	return &m[0], nil
}
