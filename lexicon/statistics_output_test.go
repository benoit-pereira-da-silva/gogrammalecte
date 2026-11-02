package lexicon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestStatisticsDumpReference(t *testing.T) {
	var fixture struct {
		Words, Corpora []string
		Write          string
	}
	data, err := os.ReadFile("testdata/statistics.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	stats := NewStatistics(fixture.Words)
	for i, corpus := range fixture.Corpora {
		if err := stats.Add(strings.NewReader(corpus), fmt.Sprint(i), fmt.Sprint("corpus ", i)); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if err := stats.Write(&out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != fixture.Write {
		t.Fatalf("dump = %q, want %q", got, fixture.Write)
	}
}

func TestFrequencyChangeOutput(t *testing.T) {
	a := ParseEntry("mot/A.\tid:12\tfq:4")
	a.OldFrequencyIndex = "3"
	b := ParseEntry("stable\tid:15\tfq:5")
	b.OldFrequencyIndex = "5"
	d := &Dictionary{Entries: []*Entry{a, b}}
	var diff, notes bytes.Buffer
	if err := d.WriteFrequencyChanges(&diff, &notes); err != nil {
		t.Fatal(err)
	}
	if diff.String() != "12\t4\n" || notes.String() != "mot/A.\t3 > 4\n" {
		t.Fatalf("frequency changes = %q / %q", diff.String(), notes.String())
	}
}
