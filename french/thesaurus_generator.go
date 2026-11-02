package french

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GenerateThesaurusData rebuilds the merged thesaurus from embedded source files.
func GenerateThesaurusData() ([]byte, error) {
	base, e := sourceReader("thes_fr.dat")
	if e != nil {
		return nil, e
	}
	defer base.Close()
	sets, e := sourceReader("synsets_fr.dat")
	if e != nil {
		return nil, e
	}
	defer sets.Close()
	return CompileThesaurus(base, sets)
}

// CompileThesaurus reads a MyThes dictionary and additional synonym sets.
// Duplicate removal, parenthetical labels and source ordering follow the reference.
func CompileThesaurus(base, synsets io.Reader) ([]byte, error) {
	result := map[string][]SynonymSet{}
	scanner := bufio.NewScanner(base)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	if !scanner.Scan() {
		return nil, fmt.Errorf("missing thesaurus encoding header")
	}
	if strings.TrimSpace(scanner.Text()) != "UTF-8" {
		return nil, fmt.Errorf("unsupported thesaurus encoding")
	}
	entry := ""
	remaining := 0
	line := 1
	for scanner.Scan() {
		line++
		s := strings.TrimSpace(scanner.Text())
		fields := strings.Split(s, "|")
		count := 0
		if len(fields) == 2 && fields[0] != "" {
			count, _ = strconv.Atoi(fields[1])
		}
		if count > 0 {
			if remaining != 0 {
				return nil, fmt.Errorf("line %d: incomplete synonym classes for %s", line, entry)
			}
			entry, remaining = fields[0], count
			result[entry] = []SynonymSet{}
		} else {
			if remaining <= 0 {
				return nil, fmt.Errorf("line %d: unexpected synonym class", line)
			}
			result[entry] = append(result[entry], SynonymSet{fields[0], deduplicateWords(fields[1:])})
			remaining--
		}
	}
	if e := scanner.Err(); e != nil {
		return nil, e
	}
	if remaining != 0 {
		return nil, fmt.Errorf("incomplete final synonym classes")
	}
	scanner = bufio.NewScanner(synsets)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		fields := strings.Split(strings.TrimSpace(scanner.Text()), "|")
		words := deduplicateWords(fields[1:])
		set := SynonymSet{fields[0], words}
		for _, word := range words {
			if strings.HasSuffix(word, "*") {
				continue
			}
			if start := strings.IndexByte(word, '('); start >= 0 {
				if end := strings.LastIndexByte(word, ')'); end >= start {
					word = strings.TrimSpace(word[:start] + word[end+1:])
				}
			}
			result[word] = append(result[word], set)
		}
	}
	if e := scanner.Err(); e != nil {
		return nil, e
	}
	return json.Marshal(result)
}
func deduplicateWords(words []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, word := range words {
		if !seen[word] {
			seen[word] = true
			out = append(out, strings.TrimSpace(word))
		}
	}
	return out
}
