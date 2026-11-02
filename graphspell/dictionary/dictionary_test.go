package dictionary

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

type oracleQuery struct {
	Word          string
	Lookup, Valid bool
	TokenValid    bool `json:"token_valid"`
	Morph, Stems  []string
}
type oracleDictionary struct {
	Mode    string
	Data    json.RawMessage
	Entries [][]string
	Queries []oracleQuery
}
type oracleFixture struct {
	Lexicon      [][]string
	Dictionaries []oracleDictionary
}

func fixture(t *testing.T) oracleFixture {
	t.Helper()
	b, e := os.ReadFile("testdata/dictionaries.json")
	if e != nil {
		t.Fatal(e)
	}
	var f oracleFixture
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	return f
}
func checkOracle(t *testing.T, d *Dictionary, want oracleDictionary) {
	t.Helper()
	for _, q := range want.Queries {
		t.Run(q.Word, func(t *testing.T) {
			if got := d.Lookup(q.Word); got != q.Lookup {
				t.Errorf("Lookup = %v, want %v", got, q.Lookup)
			}
			if got := d.IsValid(q.Word); got != q.Valid {
				t.Errorf("IsValid = %v, want %v", got, q.Valid)
			}
			if got := d.IsValidToken(q.Word); got != q.TokenValid {
				t.Errorf("IsValidToken = %v, want %v", got, q.TokenValid)
			}
			if got := d.Morph(q.Word); !reflect.DeepEqual(got, q.Morph) {
				t.Errorf("Morph = %#v, want %#v", got, q.Morph)
			}
			if got := d.Stem(q.Word); !reflect.DeepEqual(got, q.Stems) {
				t.Errorf("Stem = %#v, want %#v", got, q.Stems)
			}
		})
	}
	got := d.Entries()
	if len(got) != len(want.Entries) {
		t.Fatalf("Entries length %d, want %d", len(got), len(want.Entries))
	}
	for i, e := range got {
		if !reflect.DeepEqual([]string{e.Word, e.Stem, e.Tag}, want.Entries[i]) {
			t.Errorf("Entries[%d] = %#v, want %#v", i, e, want.Entries[i])
		}
	}
}
func TestReadPythonDictionaries(t *testing.T) {
	for _, o := range fixture(t).Dictionaries {
		t.Run(o.Mode, func(t *testing.T) {
			d, e := Read(bytes.NewReader(o.Data))
			if e != nil {
				t.Fatal(e)
			}
			checkOracle(t, d, o)
			var b bytes.Buffer
			if e = d.Write(&b); e != nil {
				t.Fatal(e)
			}
			again, e := Read(&b)
			if e != nil {
				t.Fatal(e)
			}
			checkOracle(t, again, o)
		})
	}
}
func TestNativeBuildPythonParity(t *testing.T) {
	f := fixture(t)
	var entries []Entry
	for _, e := range f.Lexicon {
		entries = append(entries, Entry{Word: e[0], Stem: e[1], Tag: e[2]})
	}
	for _, o := range f.Dictionaries {
		t.Run(o.Mode, func(t *testing.T) {
			opts := BuildOptions{Stemming: o.Mode, Metadata: Metadata{LanguageCode: "fr", LanguageName: "French", Name: "fixture", Description: "Golden test dictionary"}}
			d, e := Build(entries, opts)
			if e != nil {
				t.Fatal(e)
			}
			checkOracle(t, d, o)
			d2, e := Build(entries, opts)
			if e != nil {
				t.Fatal(e)
			}
			var a, b bytes.Buffer
			d.Write(&a)
			d2.Write(&b)
			if !bytes.Equal(a.Bytes(), b.Bytes()) {
				t.Fatal("generation is not deterministic")
			}
			var py, goData map[string]any
			json.Unmarshal(o.Data, &py)
			json.Unmarshal(a.Bytes(), &goData)
			for _, k := range []string{"nChar", "nAff", "nTag", "nEntry", "nNode", "nArc", "lArcVal", "lByDic"} {
				if !reflect.DeepEqual(py[k], goData[k]) {
					t.Errorf("serialized %s differs from Python", k)
				}
			}
		})
	}
}
func TestLegacyBinary(t *testing.T) {
	o := fixture(t).Dictionaries[0]
	var data map[string]any
	json.Unmarshal(o.Data, &data)
	arcSize := int(data["nBytesArc"].(float64))
	addressSize := int(data["nBytesNodeAddress"].(float64))
	values := data["lByDic"].([]any)
	var binary []byte
	for i, v := range values {
		n := uint64(v.(float64))
		size := arcSize
		if i%2 == 1 {
			n = n * uint64(arcSize+addressSize) / 2
			size = addressSize
		}
		for shift := size - 1; shift >= 0; shift-- {
			binary = append(binary, byte(n>>uint(shift*8)))
		}
	}
	delete(data, "lByDic")
	data["sByDic"] = hex.EncodeToString(binary)
	raw, _ := json.Marshal(data)
	d, e := Read(bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	checkOracle(t, d, o)
}
func TestRejectCorruptData(t *testing.T) {
	o := fixture(t).Dictionaries[0]
	for _, mutate := range []func(map[string]any){
		func(d map[string]any) { d["sHeader"] = "bad" }, func(d map[string]any) { d["nBytesArc"] = 0 }, func(d map[string]any) { d["lByDic"] = []any{} }, func(d map[string]any) { d["lByDic"].([]any)[1] = 999999 }, func(d map[string]any) { d["lByDic"].([]any)[1] = 1 }, func(d map[string]any) { d["lByDic"].([]any)[1] = 0 }, func(d map[string]any) { d["cStemming"] = "X" }, func(d map[string]any) { d["nChar"] = 9999 }, func(d map[string]any) { d["dChar"].(map[string]any)["a"] = 0 },
	} {
		var data map[string]any
		json.Unmarshal(o.Data, &data)
		mutate(data)
		b, _ := json.Marshal(data)
		if _, e := Read(bytes.NewReader(b)); e == nil {
			t.Errorf("accepted corrupt data: %.120s", b)
		}
	}
	for _, raw := range []string{"null", "{}", "{", string(o.Data) + " {}"} {
		if _, e := Read(strings.NewReader(raw)); e == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}
func TestLexiconParsing(t *testing.T) {
	src := "## LangCode: fr\n## LangName: French\n## DicName: example\n# comment\nchat\tchat\t:N:m:s\nchats\tchat\t:N:m:p\n"
	entries, meta, e := ReadLexicon(strings.NewReader(src))
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) != 2 || meta.LanguageCode != "fr" || meta.Name != "example" {
		t.Fatalf("got %#v %#v", entries, meta)
	}
	for _, s := range []string{"no tabs", "word\tstem\t:tag\textra", "\xff\tword\t:tag"} {
		if _, _, e := ReadLexicon(strings.NewReader(s)); e == nil {
			t.Fatalf("accepted invalid lexicon %q", s)
		}
	}
}
func TestBuildValidationAndFilter(t *testing.T) {
	entries := []Entry{{"chat", "chat", ":N:m:s"}, {"chats", "chat", ":N:m:p"}}
	d, e := Build(entries, BuildOptions{TagPattern: ":p$"})
	if e != nil {
		t.Fatal(e)
	}
	if d.Lookup("chat") || !d.Lookup("chats") {
		t.Fatal("tag filter not applied")
	}
	for _, opts := range []BuildOptions{{Stemming: "X"}, {TagPattern: "["}, {TagPattern: "no-match"}} {
		if _, e := Build(entries, opts); e == nil {
			t.Fatal("accepted invalid build")
		}
	}
	if _, e := Build(nil, BuildOptions{}); e == nil {
		t.Fatal("accepted empty lexicon")
	}
	if _, e := Build([]Entry{{"\xff", "x", ":X"}}, BuildOptions{}); e == nil {
		t.Fatal("accepted invalid UTF-8")
	}
}
func TestSelections(t *testing.T) {
	d, e := Read(bytes.NewReader(fixture(t).Dictionaries[0].Data))
	if e != nil {
		t.Fatal(e)
	}
	entries, e := d.Select("^chat", ":p$")
	if e != nil {
		t.Fatal(e)
	}
	want := []Entry{{"chats", "chat", ":N:m:p"}}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("got %#v", entries)
	}
	if _, e := d.Select("(", ""); e == nil {
		t.Fatal("invalid pattern not reported")
	}
}

var errIO = errors.New("I/O failure")

type brokenIO struct{}

func (brokenIO) Read([]byte) (int, error)  { return 0, errIO }
func (brokenIO) Write([]byte) (int, error) { return 0, errIO }
func TestIOErrors(t *testing.T) {
	if _, e := Read(brokenIO{}); !errors.Is(e, errIO) {
		t.Fatal(e)
	}
	d, e := Read(bytes.NewReader(fixture(t).Dictionaries[0].Data))
	if e != nil {
		t.Fatal(e)
	}
	if e = d.Write(brokenIO{}); !errors.Is(e, errIO) {
		t.Fatal(e)
	}
}
func ExampleBuild() {
	d, err := Build([]Entry{{Word: "chats", Stem: "chat", Tag: ":N:m:p"}}, BuildOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println(d.Lookup("chats"))
	fmt.Println(d.Morph("chats"))
	// Output:
	// true
	// [>chat/:N:m:p]
}
