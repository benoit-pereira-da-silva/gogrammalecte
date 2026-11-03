package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benoit-pereira-da-silva/gogrammalecte/grammar"
)

func TestSubtitleFileAndStdinCommands(t *testing.T) {
	const source = "1\n00:00:01,000 --> 00:00:03,000\nLes <i>chat</i> sont ici.\n\n2\n00:00:04,000 --> 00:00:06,000\nC'est bonjouur...\n"
	filename := filepath.Join(t.TempDir(), "dialogue.srt")
	if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"check", filename}, {"check", "--format", "srt"}, {"check"}} {
		var out bytes.Buffer
		if err := execute(args, strings.NewReader(source), &out); err != nil {
			t.Fatal(err)
		}
		var errors []grammar.Error
		if err := json.Unmarshal(out.Bytes(), &errors); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range errors {
			if e.Start == strings.Index(source, "chat") {
				found = true
			}
			if e.Start < strings.Index(source, "Les") {
				t.Fatalf("structure diagnosed: %+v", e)
			}
		}
		if !found {
			t.Fatalf("cue grammar absent: %s", out.String())
		}
	}
	var out bytes.Buffer
	if err := execute([]string{"check", "--full", filename}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var analysis grammar.Analysis
	if err := json.Unmarshal(out.Bytes(), &analysis); err != nil || len(analysis.Sentences) != 2 {
		t.Fatalf("full: %s %v", out.String(), err)
	}
	out.Reset()
	if err := execute([]string{"spell", filename}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "bonjouur") || strings.Contains(out.String(), `"sValue":"i"`) {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := execute([]string{"format", filename}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "C’est bonjouur…") || !strings.Contains(out.String(), "00:00:01,000 --> 00:00:03,000\nLes <i>chat</i>") {
		t.Fatal(out.String())
	}
	for _, command := range []string{"check", "spell", "format"} {
		out.Reset()
		if err := execute([]string{command, "--format", "unsupported"}, strings.NewReader(source), &out); err == nil {
			t.Fatal("accepted unknown format")
		}
	}
}
