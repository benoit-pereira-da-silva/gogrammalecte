package lexicon

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestEmbeddedCorpora(t *testing.T) {
	var records []struct {
		Name        string `json:"name"`
		Bytes       int64  `json:"bytes"`
		Lines       int64  `json:"lines"`
		Occurrences int64  `json:"occurrences"`
		SHA256      string `json:"sha256"`
	}
	data, err := os.ReadFile("testdata/corpora.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	names := CorpusNames()
	if len(names) != 4 {
		t.Fatalf("names = %v", names)
	}
	for _, record := range records {
		t.Run(record.Name, func(t *testing.T) {
			if !slices.Contains(names, record.Name) {
				t.Fatalf("missing %q", record.Name)
			}
			r, err := Corpus(record.Name)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			hash := sha256.New()
			counter := &byteCounter{}
			scanner := bufio.NewScanner(io.TeeReader(r, io.MultiWriter(hash, counter)))
			scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
			var lines, occurrences int64
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) != 2 {
					t.Fatalf("line %d: expected word and count", lines+1)
				}
				count, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					t.Fatal(err)
				}
				occurrences += count
				lines++
			}
			if err := scanner.Err(); err != nil {
				t.Fatal(err)
			}
			if lines != record.Lines || occurrences != record.Occurrences || counter.bytes != record.Bytes {
				t.Fatalf("lines/count/bytes = %d/%d/%d, want %d/%d/%d", lines, occurrences, counter.bytes, record.Lines, record.Occurrences, record.Bytes)
			}
			if got := hex.EncodeToString(hash.Sum(nil)); got != record.SHA256 {
				t.Fatalf("SHA-256 = %s, want %s", got, record.SHA256)
			}
		})
	}
}

type byteCounter struct{ bytes int64 }

func (c *byteCounter) Write(p []byte) (int, error) { c.bytes += int64(len(p)); return len(p), nil }

func TestCorpusRejectsUnknownNames(t *testing.T) {
	for _, name := range []string{"", "../data/stats_frwiki.txt", "stats_frwiki.txt.gz", "missing.txt"} {
		if r, err := Corpus(name); err == nil {
			r.Close()
			t.Errorf("Corpus(%q) unexpectedly succeeded", name)
		}
	}
	names := CorpusNames()
	names[0] = "changed"
	if CorpusNames()[0] == "changed" {
		t.Fatal("caller changed global corpus names")
	}
}

func TestEmbeddedSourceDigests(t *testing.T) {
	var fixture struct{ Files map[string]string }
	data, err := os.ReadFile("testdata/source.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"FRANCAIS.dic", "FRANCAIS_7.aff"} {
		file, err := dictionaryFiles.Open("data/" + name + ".gz")
		if err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			t.Fatal(err)
		}
		hash := sha256.New()
		_, err = io.Copy(hash, reader)
		reader.Close()
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := hex.EncodeToString(hash.Sum(nil)); got != fixture.Files["orthographe/"+name] {
			t.Fatalf("%s source SHA-256 = %s, want %s", name, got, fixture.Files["orthographe/"+name])
		}
	}
}
