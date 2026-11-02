// Package lexicon provides the French lexical generation resources and tools.
package lexicon

import (
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"slices"
)

//go:embed data/stats_*.txt.gz
var corpusFiles embed.FS

var corpusNames = []string{
	"stats_frwiki.txt",
	"stats_frwikisource.txt",
	"stats_google_ngram_1.txt",
	"stats_litterature.txt",
}

// CorpusNames returns the names of the four embedded original frequency corpora.
func CorpusNames() []string { return slices.Clone(corpusNames) }

// Corpus opens the unmodified, decompressed contents of a frequency corpus.
// Callers must close the returned reader. See data/provenance.md for its source.
func Corpus(name string) (io.ReadCloser, error) {
	if !slices.Contains(corpusNames, name) {
		return nil, fmt.Errorf("corpus %q: %w", name, fs.ErrNotExist)
	}
	file, err := corpusFiles.Open("data/" + name + ".gz")
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("corpus %q: %w", name, err)
	}
	return &corpusReader{Reader: reader, file: file}, nil
}

type corpusReader struct {
	*gzip.Reader
	file fs.File
}

func (r *corpusReader) Close() error {
	err := r.Reader.Close()
	if closeErr := r.file.Close(); err == nil {
		err = closeErr
	}
	return err
}
