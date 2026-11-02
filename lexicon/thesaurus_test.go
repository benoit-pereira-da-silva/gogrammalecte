package lexicon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMyThesReference(t *testing.T) {
	var fixture struct {
		Full    map[string]string
		Entries int
		Small   struct{ Base, Synsets, Dat, Idx, JSON string }
	}
	data, err := os.ReadFile("testdata/thesaurus.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	small, err := ReadThesaurus(strings.NewReader(fixture.Small.Base), strings.NewReader(fixture.Small.Synsets))
	if err != nil {
		t.Fatal(err)
	}
	var dat, idx, js bytes.Buffer
	if err := small.WriteMyThes(&dat, &idx); err != nil {
		t.Fatal(err)
	}
	if err := small.WriteJSON(&js); err != nil {
		t.Fatal(err)
	}
	if dat.String() != fixture.Small.Dat || idx.String() != fixture.Small.Idx || js.String() != fixture.Small.JSON {
		t.Fatalf("small thesaurus output differs:\nDAT %s\nIDX %s\nJSON %s", dat.String(), idx.String(), js.String())
	}
	full, err := EmbeddedThesaurus()
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Entries) != fixture.Entries {
		t.Fatalf("entries = %d, want %d", len(full.Entries), fixture.Entries)
	}
	datHash, idxHash, jsonHash := sha256.New(), sha256.New(), sha256.New()
	if err := full.WriteMyThes(datHash, idxHash); err != nil {
		t.Fatal(err)
	}
	if err := full.WriteJSON(jsonHash); err != nil {
		t.Fatal(err)
	}
	for name, hash := range map[string][]byte{"dat": datHash.Sum(nil), "idx": idxHash.Sum(nil), "json": jsonHash.Sum(nil)} {
		if got := hex.EncodeToString(hash); got != fixture.Full[name] {
			t.Errorf("thesaurus %s hash = %s, want %s", name, got, fixture.Full[name])
		}
	}
}
