package grammar

import (
	"encoding/json"
	"testing"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
)

func TestTokenDescriptionSnapshot(t *testing.T) {
	checker, err := graphspell.Default()
	if err != nil {
		t.Fatal(err)
	}
	end := 2
	token := &Token{Index: 1, Type: "WORD", Value: "donne-le-moi", Start: 0, End: 12, Tags: map[string]bool{"verb": true}, MultiStartTo: &end}
	token.Multi = &MultiToken{Start: 1, End: 2, Tokens: []*Token{token}, Morphology: []string{":V"}}
	info := DescribeToken(token, checker)
	if _, err := json.Marshal(info); err != nil {
		t.Fatalf("cyclic token description: %v", err)
	}
	if len(info.SubTokens) != 3 || info.SubTokens[1].Value != "donne" || info.SubTokens[2].Value != "-le-moi" {
		t.Fatalf("incorrect decomposition: %#v", info.SubTokens)
	}
	if info.ValidToken == nil || !*info.ValidToken {
		t.Fatal("compound verb should be valid")
	}
	info.Tags[0] = "changed"
	info.Multi.Morphology[0] = "changed"
	*info.MultiStartTo = 8
	if !token.Tags["verb"] || token.Multi.Morphology[0] != ":V" || *token.MultiStartTo != 2 {
		t.Fatal("description shares mutable analysis state")
	}
	if token.Morphology != nil {
		t.Fatal("description changed token morphology cache")
	}
}
