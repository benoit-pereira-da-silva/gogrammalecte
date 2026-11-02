// Package ruleexpr parses and evaluates Grammalecte's expression language.
// It provides no filesystem, process, network or arbitrary Python execution.
package ruleexpr

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

// Environment supplies only the names, token references and functions exposed
// by the grammar engine. Arguments retain reference metadata until consumed.
type Environment interface {
	Resolve(string) (any, error)
	Reference(string, int) (any, error)
	Call(string, []any) (any, error)
}

// TextValue allows token references to behave like strings without losing their
// identity when passed to morphological and disambiguation functions.
type TextValue interface{ Text() string }

// String returns a string or the textual value of a token reference.
func String(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case TextValue:
		return x.Text(), true
	}
	return "", false
}
func plain(v any) any {
	if s, ok := String(v); ok {
		return s
	}
	return v
}

// Truth follows the truth-value rules used by the source expression language.
func Truth(v any) bool {
	v = plain(v)
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case int:
		return x != 0
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case []string:
		return len(x) > 0
	}
	value := reflect.ValueOf(v)
	if value.IsValid() {
		switch value.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			return value.Len() != 0
		case reflect.Float32, reflect.Float64:
			return value.Float() != 0
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return value.Int() != 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return value.Uint() != 0
		}
	}
	return true
}

type node struct {
	kind, name string
	value      any
	children   []*node
}

// Expression is immutable and can be evaluated in independent environments.
type Expression struct{ root *node }

func Compile(source string) (*Expression, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := parser{tokens: tokens}
	root, err := p.expr()
	if err == nil && p.peek().kind != "eof" {
		err = fmt.Errorf("unexpected token %q", p.peek().text)
	}
	if err != nil {
		return nil, fmt.Errorf("expression %q: %w", source, err)
	}
	return &Expression{root}, nil
}
func (e *Expression) Eval(env Environment) (any, error) { return evaluate(e.root, env) }

type token struct {
	kind, text string
	value      any
}

func lex(s string) ([]token, error) {
	out := []token{}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		raw := false
		if (r == 'r' || r == 'R') && i+1 < len(s) && (s[i+1] == '\'' || s[i+1] == '"') {
			raw = true
			i++
			r = rune(s[i])
		}
		if r == '\'' || r == '"' {
			quote := byte(r)
			i++
			var b strings.Builder
			closed := false
			for i < len(s) {
				c := s[i]
				i++
				if c == quote {
					closed = true
					break
				}
				if c == '\\' && i < len(s) {
					e := s[i]
					i++
					if raw {
						b.WriteByte(c)
						b.WriteByte(e)
						continue
					}
					switch e {
					case '\\', '\'', '"':
						b.WriteByte(e)
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					case 'r':
						b.WriteByte('\r')
					case 'b':
						b.WriteByte('\b')
					case 'a':
						b.WriteByte('\a')
					case '\n':
						// A backslash followed by a newline continues the literal.
					case 'f':
						b.WriteByte('\f')
					case 'v':
						b.WriteByte('\v')
					case 'x', 'u', 'U':
						n := 2
						if e == 'u' {
							n = 4
						} else if e == 'U' {
							n = 8
						}
						if i+n > len(s) {
							return nil, fmt.Errorf("short escape")
						}
						value, err := strconv.ParseUint(s[i:i+n], 16, 32)
						if err != nil {
							return nil, err
						}
						if value > unicode.MaxRune || value >= 0xd800 && value <= 0xdfff {
							return nil, fmt.Errorf("Unicode escape cannot be represented as UTF-8")
						}
						b.WriteRune(rune(value))
						i += n
					default:
						if e >= '0' && e <= '7' {
							value := rune(e - '0')
							for count := 1; count < 3 && i < len(s) && s[i] >= '0' && s[i] <= '7'; count++ {
								value = value*8 + rune(s[i]-'0')
								i++
							}
							b.WriteRune(value)
						} else {
							b.WriteByte('\\')
							b.WriteByte(e)
						}
					}
				} else {
					b.WriteByte(c)
				}
			}
			if !closed {
				return nil, fmt.Errorf("unterminated string")
			}
			out = append(out, token{kind: "literal", value: b.String()})
			continue
		}
		if r == '\\' {
			i++
			start := i
			if i < len(s) && s[i] == '-' {
				i++
			}
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			n, err := strconv.Atoi(s[start:i])
			if err != nil {
				return nil, fmt.Errorf("invalid token reference")
			}
			out = append(out, token{kind: "ref", text: "\\", value: n})
			continue
		}
		if unicode.IsLetter(r) || r == '_' {
			start := i
			i += size
			for i < len(s) {
				r, n := utf8.DecodeRuneInString(s[i:])
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
					break
				}
				i += n
			}
			word := s[start:i]
			t := token{kind: "name", text: word}
			switch word {
			case "True":
				t = token{kind: "literal", value: true}
			case "False":
				t = token{kind: "literal", value: false}
			case "None":
				t = token{kind: "literal"}
			}
			out = append(out, t)
			continue
		}
		if r >= '0' && r <= '9' {
			start := i
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			n, err := strconv.Atoi(s[start:i])
			if err != nil {
				return nil, err
			}
			if s[start] == '0' && n != 0 {
				return nil, fmt.Errorf("leading zeros in decimal integer literal")
			}
			out = append(out, token{kind: "literal", value: n})
			continue
		}
		if i+1 < len(s) && strings.Contains(" == != <= >= // ** ", " "+s[i:i+2]+" ") {
			out = append(out, token{kind: s[i : i+2], text: s[i : i+2]})
			i += 2
			continue
		}
		if strings.ContainsRune("()[]{},.:+-*/%<>=@", r) {
			out = append(out, token{kind: string(r), text: string(r)})
			i += size
			continue
		}
		return nil, fmt.Errorf("unexpected character %q", r)
	}
	return append(out, token{kind: "eof"}), nil
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() token { return p.tokens[p.pos] }
func (p *parser) take() token { t := p.peek(); p.pos++; return t }
func (p *parser) accept(s string) bool {
	t := p.peek()
	if t.kind == s || t.kind == "name" && t.text == s {
		p.pos++
		return true
	}
	return false
}
func (p *parser) need(s string) error {
	if !p.accept(s) {
		return fmt.Errorf("expected %s, got %s", s, p.peek().text)
	}
	return nil
}
func (p *parser) expr() (*node, error) {
	a, err := p.binary(1)
	if err != nil {
		return nil, err
	}
	if p.accept("if") {
		condition, err := p.binary(1)
		if err != nil {
			return nil, err
		}
		if err := p.need("else"); err != nil {
			return nil, err
		}
		other, err := p.expr()
		if err != nil {
			return nil, err
		}
		return &node{kind: "if", children: []*node{condition, a, other}}, nil
	}
	return a, nil
}
func precedence(op string) int {
	switch op {
	case "or":
		return 1
	case "and":
		return 2
	case "==", "!=", "<", ">", "<=", ">=", "in", "not in", "is", "is not":
		return 4
	case "+", "-":
		return 5
	case "*", "//", "/", "%":
		return 6
	}
	return 0
}
func (p *parser) operator() (string, int) {
	t := p.peek()
	op := t.kind
	if t.kind == "name" {
		op = t.text
	}
	if op == "not" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].text == "in" {
		return "not in", 2
	}
	if op == "is" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].text == "not" {
		return "is not", 2
	}
	return op, 1
}
func (p *parser) binary(min int) (*node, error) {
	var left *node
	var err error
	if p.accept("not") {
		v, e := p.binary(3)
		left = &node{kind: "unary", name: "not", children: []*node{v}}
		err = e
	} else if p.accept("-") {
		v, e := p.binary(7)
		left = &node{kind: "unary", name: "-", children: []*node{v}}
		err = e
	} else if p.accept("+") {
		v, e := p.binary(7)
		left = &node{kind: "unary", name: "+", children: []*node{v}}
		err = e
	} else {
		left, err = p.atom()
	}
	if err != nil {
		return nil, err
	}
	for {
		op, count := p.operator()
		prec := precedence(op)
		if prec < min || prec == 0 {
			break
		}
		p.pos += count
		right, err := p.binary(prec + 1)
		if err != nil {
			return nil, err
		}
		if prec == 4 && left.kind == "compare" {
			left.children = append(left.children, right)
			left.name += "|" + op
		} else {
			kind := "binary"
			if prec == 4 {
				kind = "compare"
			}
			left = &node{kind: kind, name: op, children: []*node{left, right}}
		}
	}
	return left, nil
}
func (p *parser) atom() (*node, error) {
	t := p.take()
	var result *node
	switch t.kind {
	case "literal":
		result = &node{kind: "literal", value: t.value}
	case "name":
		result = &node{kind: "name", name: t.text}
	case "ref":
		result = &node{kind: "ref", name: t.text, value: t.value}
	case "@":
		result = &node{kind: "ref", name: "@", value: 0}
	case "/", ">", "<":
		n := p.take()
		value, ok := n.value.(int)
		if !ok {
			return nil, fmt.Errorf("expected reference index")
		}
		result = &node{kind: "ref", name: t.kind, value: value}
	case "(", "[", "{":
		end := map[string]string{"(": ")", "[": "]", "{": "}"}[t.kind]
		elements := []*node{}
		tuple := false
		if !p.accept(end) {
			for {
				value, err := p.expr()
				if err != nil {
					return nil, err
				}
				elements = append(elements, value)
				if p.accept(end) {
					break
				}
				if err := p.need(","); err != nil {
					return nil, err
				}
				tuple = true
				if p.accept(end) {
					break
				}
			}
		}
		if t.kind == "(" && len(elements) == 1 && !tuple {
			result = &node{kind: "group", children: elements}
		} else {
			result = &node{kind: "list", children: elements}
		}
	default:
		return nil, fmt.Errorf("expected value, got %q", t.text)
	}
	for {
		if p.accept(".") {
			name := p.take()
			if name.kind != "name" {
				return nil, fmt.Errorf("expected method name")
			}
			result = &node{kind: "attribute", name: name.text, children: []*node{result}}
			continue
		}
		if p.accept("(") {
			args := []*node{result}
			if !p.accept(")") {
				for {
					a, err := p.expr()
					if err != nil {
						return nil, err
					}
					args = append(args, a)
					if p.accept(")") {
						break
					}
					if err := p.need(","); err != nil {
						return nil, err
					}
					if p.accept(")") {
						break
					}
				}
			}
			result = &node{kind: "call", children: args}
			continue
		}
		if p.accept("[") {
			parts := []*node{result}
			slice := false
			var first *node
			if p.peek().kind != ":" {
				var err error
				first, err = p.expr()
				if err != nil {
					return nil, err
				}
			}
			parts = append(parts, first)
			if p.accept(":") {
				slice = true
				var end, step *node
				var err error
				if p.peek().kind != "]" && p.peek().kind != ":" {
					end, err = p.expr()
					if err != nil {
						return nil, err
					}
				}
				if p.accept(":") && p.peek().kind != "]" {
					step, err = p.expr()
					if err != nil {
						return nil, err
					}
				}
				parts = append(parts, end, step)
			}
			if err := p.need("]"); err != nil {
				return nil, err
			}
			kind := "index"
			if slice {
				kind = "slice"
			}
			result = &node{kind: kind, children: parts}
			continue
		}
		break
	}
	return result, nil
}
func evalChildren(nodes []*node, env Environment) ([]any, error) {
	out := make([]any, len(nodes))
	for i, n := range nodes {
		if n == nil {
			continue
		}
		v, err := evaluate(n, env)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}
func qualified(n *node) (string, bool) {
	if n.kind == "group" {
		return qualified(n.children[0])
	}
	if n.kind == "name" {
		return n.name, true
	}
	if n.kind == "attribute" {
		s, ok := qualified(n.children[0])
		return s + "." + n.name, ok
	}
	return "", false
}
func evaluate(n *node, env Environment) (any, error) {
	switch n.kind {
	case "literal":
		return n.value, nil
	case "ref":
		return env.Reference(n.name, n.value.(int))
	case "name":
		return env.Resolve(n.name)
	case "list":
		return evalChildren(n.children, env)
	case "group":
		return evaluate(n.children[0], env)
	case "if":
		v, err := evaluate(n.children[0], env)
		if err != nil {
			return nil, err
		}
		if Truth(v) {
			return evaluate(n.children[1], env)
		}
		return evaluate(n.children[2], env)
	case "unary":
		v, err := evaluate(n.children[0], env)
		if err != nil {
			return nil, err
		}
		if n.name == "not" {
			return !Truth(v), nil
		}
		if i, ok := integerValue(v); ok {
			if n.name == "+" {
				return i, nil
			}
			return -i, nil
		}
		return nil, fmt.Errorf("unary minus requires an integer")
	case "binary":
		a, err := evaluate(n.children[0], env)
		if err != nil {
			return nil, err
		}
		if n.name == "and" && !Truth(a) || n.name == "or" && Truth(a) {
			return a, nil
		}
		b, err := evaluate(n.children[1], env)
		if err != nil {
			return nil, err
		}
		if n.name == "and" || n.name == "or" {
			return b, nil
		}
		return arithmetic(n.name, plain(a), plain(b))
	case "compare":
		a, err := evaluate(n.children[0], env)
		if err != nil {
			return nil, err
		}
		for i, op := range strings.Split(n.name, "|") {
			b, err := evaluate(n.children[i+1], env)
			if err != nil {
				return nil, err
			}
			ok, err := compare(op, plain(a), plain(b))
			if err != nil || !ok {
				return ok, err
			}
			a = b
		}
		return true, nil
	case "call":
		args, err := evalChildren(n.children[1:], env)
		if err != nil {
			return nil, err
		}
		target := n.children[0]
		if name, ok := qualified(target); ok {
			if name == "len" && len(args) == 1 {
				if s, ok := String(args[0]); ok {
					return utf8.RuneCountInString(s), nil
				}
				value := reflect.ValueOf(args[0])
				if value.IsValid() && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array || value.Kind() == reflect.Map) {
					return value.Len(), nil
				}
				return nil, fmt.Errorf("len requires a string or collection")
			}
			if name == "str" && len(args) == 1 {
				return pythonString(plain(args[0]))
			}
			if name == "int" && len(args) == 1 {
				if s, ok := String(args[0]); ok {
					return pythonInteger(s)
				}
				if i, ok := integerValue(args[0]); ok {
					return i, nil
				}
				return nil, fmt.Errorf("int requires an integer, boolean or decimal string")
			}
			return env.Call(name, args)
		}
		if target.kind == "attribute" {
			receiver, err := evaluate(target.children[0], env)
			if err != nil {
				return nil, err
			}
			return method(receiver, target.name, args)
		}
		return nil, fmt.Errorf("unsupported callable")
	case "index", "slice":
		values, err := evalChildren(n.children, env)
		if err != nil {
			return nil, err
		}
		s, ok := String(values[0])
		if !ok {
			return nil, fmt.Errorf("indexing requires a string")
		}
		r := []rune(s)
		if n.kind == "index" {
			i, ok := values[1].(int)
			if !ok {
				return nil, fmt.Errorf("index must be integer")
			}
			if i < 0 {
				i += len(r)
			}
			if i < 0 || i >= len(r) {
				return nil, fmt.Errorf("string index out of range")
			}
			return string(r[i]), nil
		}
		return sliceRunes(r, values[1:])
	}
	return nil, fmt.Errorf("unsupported expression node %s", n.kind)
}
func arithmetic(op string, a, b any) (any, error) {
	x, xok := integerValue(a)
	y, yok := integerValue(b)
	if xok && yok {
		switch op {
		case "+":
			return x + y, nil
		case "-":
			return x - y, nil
		case "*":
			return x * y, nil
		case "//":
			if y != 0 {
				q := x / y
				if x%y != 0 && (x < 0) != (y < 0) {
					q--
				}
				return q, nil
			}
		case "%":
			if y != 0 {
				return ((x % y) + y) % y, nil
			}
		}
	}
	if x, ok := a.(string); ok {
		if op == "+" {
			if y, ok := b.(string); ok {
				return x + y, nil
			}
		}
		if op == "*" && yok {
			if y < 0 {
				y = 0
			}
			if len(x) != 0 && y > 10000000/len(x) {
				return nil, fmt.Errorf("string repetition exceeds limit")
			}
			return strings.Repeat(x, y), nil
		}
	}
	return nil, fmt.Errorf("unsupported operands for %s", op)
}
func compare(op string, a, b any) (bool, error) {
	switch op {
	case "==", "is":
		if op == "==" {
			return pythonEqual(a, b), nil
		}
		return reflect.DeepEqual(a, b), nil
	case "!=", "is not":
		if op == "!=" {
			return !pythonEqual(a, b), nil
		}
		return !reflect.DeepEqual(a, b), nil
	case "in", "not in":
		found := false
		switch value := b.(type) {
		case string:
			s, ok := String(a)
			if !ok {
				return false, fmt.Errorf("string membership requires string")
			}
			found = strings.Contains(value, s)
		case []any:
			for _, v := range value {
				if pythonEqual(a, plain(v)) {
					found = true
					break
				}
			}
		case []string:
			s, ok := String(a)
			if ok {
				for _, v := range value {
					if s == v {
						found = true
						break
					}
				}
			}
		default:
			return false, fmt.Errorf("unsupported membership value")
		}
		if op == "not in" {
			found = !found
		}
		return found, nil
	}
	order := 0
	if x, ok := integerValue(a); ok {
		y, ok := integerValue(b)
		if !ok {
			return false, fmt.Errorf("incomparable values")
		}
		if x < y {
			order = -1
		} else if x > y {
			order = 1
		}
	} else if x, ok := String(a); ok {
		y, ok := String(b)
		if !ok {
			return false, fmt.Errorf("incomparable values")
		}
		order = strings.Compare(x, y)
	} else {
		return false, fmt.Errorf("incomparable values")
	}
	switch op {
	case "<":
		return order < 0, nil
	case "<=":
		return order <= 0, nil
	case ">":
		return order > 0, nil
	case ">=":
		return order >= 0, nil
	}
	return false, fmt.Errorf("unknown comparison")
}
func sliceRunes(r []rune, args []any) (string, error) {
	step := 1
	if args[2] != nil {
		v, ok := args[2].(int)
		if !ok || v == 0 {
			return "", fmt.Errorf("invalid slice step")
		}
		step = v
	}
	bounds := []int{0, len(r)}
	if step < 0 {
		bounds = []int{len(r) - 1, -1}
	}
	for i := 0; i < 2; i++ {
		if args[i] != nil {
			v, ok := args[i].(int)
			if !ok {
				return "", fmt.Errorf("invalid slice bound")
			}
			if v < 0 {
				v += len(r)
			}
			lo, hi := 0, len(r)
			if step < 0 {
				lo, hi = -1, len(r)-1
			}
			bounds[i] = max(lo, min(v, hi))
		}
	}
	out := []rune{}
	for i := bounds[0]; step > 0 && i < bounds[1] || step < 0 && i > bounds[1]; {
		out = append(out, r[i])
		// Compare the remaining distance before adding a potentially huge
		// step, so valid Python slice bounds cannot wrap a machine integer.
		if step > 0 && step >= bounds[1]-i || step < 0 && step <= bounds[1]-i {
			break
		}
		i += step
	}
	return string(out), nil
}
func method(receiver any, name string, args []any) (any, error) {
	s, ok := String(receiver)
	if !ok {
		return nil, fmt.Errorf("method %s requires a string", name)
	}
	stringArg := func(i int) (string, error) {
		if i >= len(args) {
			return "", fmt.Errorf("missing argument for %s", name)
		}
		v, ok := String(args[i])
		if !ok {
			return "", fmt.Errorf("string argument required for %s", name)
		}
		return v, nil
	}
	if len(args) == 0 {
		switch name {
		case "lower":
			return strtransform.Lower(s), nil
		case "upper":
			return strtransform.Upper(s), nil
		case "capitalize":
			return strtransform.Capitalize(s), nil
		case "isupper":
			return strtransform.IsUpper(s), nil
		case "istitle":
			return strtransform.IsTitle(s), nil
		case "isdigit":
			return strtransform.IsDigit(s), nil
		case "isalpha":
			return strtransform.IsAlpha(s), nil
		case "islower":
			return strtransform.Lower(s) == s && strtransform.Upper(s) != s, nil
		case "__len__":
			return utf8.RuneCountInString(s), nil
		case "strip":
			return strings.TrimFunc(s, pythonSpace), nil
		case "rstrip":
			return strings.TrimRightFunc(s, pythonSpace), nil
		case "lstrip":
			return strings.TrimLeftFunc(s, pythonSpace), nil
		}
	}
	switch name {
	case "startswith", "endswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("%s requires one argument", name)
		}
		choices, ok := args[0].([]any)
		if !ok {
			choices = []any{args[0]}
		}
		for _, v := range choices {
			part, ok := String(v)
			if !ok {
				return nil, fmt.Errorf("prefix or suffix must be a string")
			}
			if name == "startswith" && strings.HasPrefix(s, part) || name == "endswith" && strings.HasSuffix(s, part) {
				return true, nil
			}
		}
		return false, nil
	case "replace":
		a, err := stringArg(0)
		if err != nil {
			return nil, err
		}
		b, err := stringArg(1)
		if err != nil {
			return nil, err
		}
		limit := -1
		if len(args) == 3 {
			n, ok := args[2].(int)
			if !ok {
				return nil, fmt.Errorf("replacement count must be integer")
			}
			limit = n
		}
		return strings.Replace(s, a, b, limit), nil
	case "strip", "rstrip", "lstrip":
		chars, err := stringArg(0)
		if err != nil {
			return nil, err
		}
		if name == "strip" {
			return strings.Trim(s, chars), nil
		}
		if name == "rstrip" {
			return strings.TrimRight(s, chars), nil
		}
		return strings.TrimLeft(s, chars), nil
	}
	return nil, fmt.Errorf("unsupported string method %s", name)
}
