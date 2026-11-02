package grammar

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"
)

type fixtureLexicon map[string][]string

func (f fixtureLexicon) Morph(word string) []string  { return append([]string{}, f[word]...) }
func (f fixtureLexicon) IsValid(word string) bool    { _, ok := f[word]; return ok }
func (f fixtureLexicon) Lemmas(word string) []string { return nil }

func decodeFixture[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func decodeToken(t *testing.T, raw json.RawMessage) *Token {
	t.Helper()
	fields := decodeFixture[map[string]json.RawMessage](t, raw)
	var tags []string
	if v, ok := fields["aTags"]; ok {
		tags = decodeFixture[[]string](t, v)
		delete(fields, "aTags")
	}
	encoded, _ := json.Marshal(fields)
	token := decodeFixture[Token](t, encoded)
	if tags != nil {
		token.Tags = map[string]bool{}
		for _, tag := range tags {
			token.Tags[tag] = true
		}
	}
	return &token
}
func decodePositions(t *testing.T, raw json.RawMessage) map[int]*Token {
	out := map[int]*Token{}
	for key, value := range decodeFixture[map[string]json.RawMessage](t, raw) {
		n, err := strconv.Atoi(key)
		if err != nil {
			t.Fatal(err)
		}
		out[n] = decodeToken(t, value)
	}
	return out
}

func TestHelpersPythonParity(t *testing.T) {
	data, err := os.ReadFile("testdata/helpers.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Lexicon fixtureLexicon
		Cases   []struct {
			Function string
			Args     []json.RawMessage
			Expected json.RawMessage
			After    []json.RawMessage
		}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for i, c := range fixture.Cases {
		h := NewHelperContext(fixture.Lexicon)
		h.Options = map[string]bool{"active": true, "disabled": false}
		str := func(n int) string {
			if n >= len(c.Args) {
				return ""
			}
			return decodeFixture[string](t, c.Args[n])
		}
		num := func(n int) int { return decodeFixture[int](t, c.Args[n]) }
		boolean := func(n int) bool { return decodeFixture[bool](t, c.Args[n]) }
		bound := func(n int) *int {
			if n >= len(c.Args) {
				return nil
			}
			return decodeFixture[*int](t, c.Args[n])
		}
		tok := func(n int) *Token { return decodeToken(t, c.Args[n]) }
		var got bool
		var mutation *Token
		var err error
		switch c.Function {
		case "option":
			got = h.Option(str(0))
		case "look":
			got, err = h.Look(str(0), str(1), str(2))
		case "look_chk1":
			h.TokenPositions = decodePositions(t, c.Args[0])
			got, err = h.LookCheck1(str(1), num(2), str(3), str(4), str(5))
		case "analyse":
			got, err = h.Analyse(str(0), str(1), str(2))
		case "morph":
			h.TokenPositions = decodePositions(t, c.Args[0])
			var word *Word
			if string(c.Args[1]) != "null" {
				a := decodeFixture[[]json.RawMessage](t, c.Args[1])
				word = &Word{decodeFixture[int](t, a[0]), decodeFixture[string](t, a[1])}
			}
			got, err = h.Morph(word, str(2), str(3), boolean(4))
		case "g_value":
			got = h.GValue(tok(0), str(1), bound(2), bound(3))
		case "g_morph":
			got, err = h.GMorph(tok(0), str(1), str(2), bound(3), bound(4))
		case "g_morph0":
			got, err = h.GMorph0(tok(0), str(1), str(2), bound(3), bound(4))
		case "g_morphx":
			got, err = h.GMorphX(tok(0), str(1), str(2))
		case "g_morphVC":
			got, err = h.GMorphVC(tok(0), str(1), str(2))
		case "g_morph2":
			mutation = tok(0)
			got, err = h.GMorph2(mutation, tok(1), str(2), str(3), str(4), boolean(5))
		case "g_tag":
			got = h.GTag(tok(0), str(1))
		case "g_tagbefore":
			h.Tags = decodeFixture[map[string][2]int](t, c.Args[1])
			got = h.GTagBefore(tok(0), str(2))
		case "g_tagafter":
			h.Tags = decodeFixture[map[string][2]int](t, c.Args[1])
			got = h.GTagAfter(tok(0), str(2))
		case "g_meta":
			got = h.GMeta(tok(0), str(1))
		case "g_space":
			got = h.GSpace(tok(0), tok(1), num(2), bound(3))
		case "select":
			h.TokenPositions = decodePositions(t, c.Args[0])
			got, err = h.Select(num(1), str(2), str(3), str(4))
			if !reflect.DeepEqual(h.TokenPositions, decodePositions(t, c.After[0])) {
				t.Errorf("case %d selection mutation mismatch", i)
			}
		case "define":
			h.TokenPositions = decodePositions(t, c.Args[0])
			got = h.Define(num(1), str(2))
			if !reflect.DeepEqual(h.TokenPositions, decodePositions(t, c.After[0])) {
				t.Errorf("case %d definition mutation mismatch", i)
			}
		case "g_select":
			mutation = tok(0)
			got, err = h.GSelect(mutation, str(1), str(2))
		case "g_addmorph":
			mutation = tok(0)
			got = h.GAddMorph(mutation, str(1))
		case "g_rewrite":
			mutation = tok(0)
			got, err = h.GRewrite(mutation, str(1), str(2), boolean(3))
		case "g_define":
			mutation = tok(0)
			got = h.GDefine(mutation, str(1))
		case "g_definefrom":
			mutation = tok(0)
			got = h.GDefineFrom(mutation, bound(1), bound(2))
		case "g_setmeta":
			mutation = tok(0)
			got = h.GSetMeta(mutation, str(1))
		case "g_token":
			var tokens []*Token
			for _, raw := range decodeFixture[[]json.RawMessage](t, c.Args[0]) {
				tokens = append(tokens, decodeToken(t, raw))
			}
			if !reflect.DeepEqual(TokenAt(tokens, num(1)), decodeToken(t, c.Expected)) {
				t.Errorf("case %d token mismatch", i)
			}
			continue
		case "nextword", "prevword", "nextword1", "prevword1":
			var word *Word
			switch c.Function {
			case "nextword":
				word, err = h.NextWord(str(0), num(1), num(2))
			case "prevword":
				word, err = h.PreviousWord(str(0), num(1), num(2))
			case "nextword1":
				word, err = h.NextWord1(str(0), num(1))
			case "prevword1":
				word, err = h.PreviousWord1(str(0), num(1))
			}
			var expected *Word
			if string(c.Expected) != "null" {
				a := decodeFixture[[]json.RawMessage](t, c.Expected)
				expected = &Word{decodeFixture[int](t, a[0]), decodeFixture[string](t, a[1])}
			}
			if err != nil || !reflect.DeepEqual(word, expected) {
				t.Errorf("case %d %s %s got %v (%v), want %v", i, c.Function, c.Args, word, err, expected)
			}
			continue
		default:
			t.Fatalf("unknown fixture method %s", c.Function)
		}
		if err != nil || got != decodeFixture[bool](t, c.Expected) {
			t.Errorf("case %d %s %s got %v (%v), want %s", i, c.Function, c.Args, got, err, c.Expected)
		}
		if mutation != nil && !reflect.DeepEqual(mutation, decodeToken(t, c.After[0])) {
			t.Errorf("case %d %s mutation got %#v want %#v", i, c.Function, mutation, decodeToken(t, c.After[0]))
		}
	}
}

func TestHelperErrorsAndCancellation(t *testing.T) {
	h := NewHelperContext(fixtureLexicon{"word": {">word/:N:m:s"}})
	if _, err := h.Analyse("word", "(?R)", ""); err == nil {
		t.Fatal("expected unsupported pattern error")
	}
	if _, err := h.GRewrite(&Token{Value: "word"}, `(?P<tag>:N)`, `\g<tag>`, true); err == nil {
		t.Fatal("expected explicit unsupported named replacement error")
	}
	if _, err := h.NextWord("word", 0, 0); err == nil {
		t.Fatal("expected invalid word number error")
	}
	if _, err := Tokenize(string([]byte{0xff}), false); err == nil {
		t.Fatal("expected invalid UTF-8 error")
	}
	if _, err := SentenceBoundaries(string([]byte{0xff})); err == nil {
		t.Fatal("expected invalid UTF-8 error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.Context = ctx
	if _, err := h.Look("word", "word", ""); err == nil {
		t.Fatal("expected cancellation error")
	}
}
