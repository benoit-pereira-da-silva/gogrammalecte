// Ported from Grammalecte thes_build.py by Olivier R.
// Distributed with the French lexical generator under the Mozilla Public License 2.0.

package lexicon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
)

// Thesaurus retains source entry order alongside the merged synonym classes.
type Thesaurus struct {
	Entries []string
	Classes map[string][]french.SynonymSet
}

// EmbeddedThesaurus reuses the embedded French source files without duplicating data.
func EmbeddedThesaurus() (*Thesaurus, error) {
	base, err := french.Source("thes_fr.dat")
	if err != nil {
		return nil, err
	}
	defer base.Close()
	sets, err := french.Source("synsets_fr.dat")
	if err != nil {
		return nil, err
	}
	defer sets.Close()
	return ReadThesaurus(base, sets)
}

var thesaurusEntry = regexp.MustCompile(`^[^|]+\|[1-9][0-9]*$`)
var synonymLabel = regexp.MustCompile(`\(.*\)`)

// ReadThesaurus uses the French compiler for merging and preserves the source
// insertion order needed for byte-identical MyThes and Python JSON outputs.
func ReadThesaurus(base, synsets io.Reader) (*Thesaurus, error) {
	baseData, err := io.ReadAll(newUniversalReader(base))
	if err != nil {
		return nil, err
	}
	setData, err := io.ReadAll(newUniversalReader(synsets))
	if err != nil {
		return nil, err
	}
	data, err := french.CompileThesaurus(bytes.NewReader(baseData), bytes.NewReader(setData))
	if err != nil {
		return nil, err
	}
	thesaurus := &Thesaurus{Classes: map[string][]french.SynonymSet{}}
	if err := json.Unmarshal(data, &thesaurus.Classes); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	add := func(word string) {
		if !seen[word] {
			seen[word] = true
			thesaurus.Entries = append(thesaurus.Entries, word)
		}
	}
	scanner := newScanner(bytes.NewReader(baseData))
	scanner.Scan()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if thesaurusEntry.MatchString(line) {
			word, _, _ := strings.Cut(line, "|")
			add(word)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	scanner = newScanner(bytes.NewReader(setData))
	for scanner.Scan() {
		parts := strings.Split(strings.TrimSpace(scanner.Text()), "|")
		for _, word := range parts[1:] {
			word = strings.TrimSpace(word)
			if strings.HasSuffix(word, "*") {
				continue
			}
			if strings.Contains(word, "(") {
				word = strings.TrimSpace(synonymLabel.ReplaceAllString(word, ""))
			}
			add(word)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return thesaurus, nil
}

// WriteMyThes writes the original DAT format and a sorted byte-offset IDX file.
func (t *Thesaurus) WriteMyThes(data, index io.Writer) error {
	dat, idx := bufio.NewWriter(data), bufio.NewWriter(index)
	fmt.Fprint(dat, "UTF-8\n")
	offset := int64(6)
	offsets := make(map[string]int64, len(t.Entries))
	for _, word := range t.Entries {
		classes := t.Classes[word]
		offsets[word] = offset
		line := fmt.Sprintf("%s|%d\n", word, len(classes))
		io.WriteString(dat, line)
		offset += int64(len(line))
		for _, class := range classes {
			line := class.Label + "|" + strings.Join(class.Words, "|") + "\n"
			io.WriteString(dat, line)
			offset += int64(len(line))
		}
	}
	fmt.Fprintf(idx, "UTF-8\n%d\n", len(t.Entries))
	words := slices.Clone(t.Entries)
	slices.Sort(words)
	for _, word := range words {
		fmt.Fprintf(idx, "%s|%d\n", word, offsets[word])
	}
	if err := dat.Flush(); err != nil {
		return err
	}
	return idx.Flush()
}

// WriteJSON preserves the source's insertion order, Unicode and separator spacing.
func (t *Thesaurus) WriteJSON(writer io.Writer) error {
	w := bufio.NewWriter(writer)
	w.WriteByte('{')
	for i, word := range t.Entries {
		if i > 0 {
			io.WriteString(w, ", ")
		}
		writePythonJSONString(w, word)
		io.WriteString(w, ": [")
		for j, class := range t.Classes[word] {
			if j > 0 {
				io.WriteString(w, ", ")
			}
			w.WriteByte('[')
			writePythonJSONString(w, class.Label)
			io.WriteString(w, ", [")
			for k, synonym := range class.Words {
				if k > 0 {
					io.WriteString(w, ", ")
				}
				writePythonJSONString(w, synonym)
			}
			io.WriteString(w, "]]")
		}
		w.WriteByte(']')
	}
	w.WriteByte('}')
	return w.Flush()
}

func writePythonJSONString(w *bufio.Writer, value string) {
	w.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			w.WriteByte('\\')
			w.WriteRune(r)
		case '\b':
			io.WriteString(w, `\b`)
		case '\f':
			io.WriteString(w, `\f`)
		case '\n':
			io.WriteString(w, `\n`)
		case '\r':
			io.WriteString(w, `\r`)
		case '\t':
			io.WriteString(w, `\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(w, `\u%04x`, r)
			} else {
				w.WriteRune(r)
			}
		}
	}
	w.WriteByte('"')
}
