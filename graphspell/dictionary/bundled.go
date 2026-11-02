package dictionary

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"sync"
)

// BundledLexiconSHA256 identifies the uncompressed original French lexicon.
const BundledLexiconSHA256 = "5938d8180b64e4f5320c358dab64e2eea1298c6c5967b147d5506f0ac4562c4a"

//go:embed data/fr-allvars.json.gz
var bundledDictionary []byte

//go:embed data/French.lex.gz
var bundledLexicon []byte

// Variant selects the spelling conventions accepted by the French dictionary.
type Variant string

const (
	AllVariants Variant = "all"
	Classic     Variant = "classic"
	Reform1990  Variant = "reform"
)

var loadDefault = sync.OnceValues(func() (*Dictionary, error) {
	r, err := gzip.NewReader(bytes.NewReader(bundledDictionary))
	if err != nil {
		return nil, fmt.Errorf("open bundled dictionary: %w", err)
	}
	defer r.Close()
	return Read(r)
})
var loadClassic = sync.OnceValues(func() (*Dictionary, error) { return BuildBundled(Classic) })
var loadReform = sync.OnceValues(func() (*Dictionary, error) { return BuildBundled(Reform1990) })

// Default returns the shared immutable French dictionary embedded in the binary.
// Loading does not read files, start a process, or access the network.
func Default() (*Dictionary, error) { return loadDefault() }

// Bundled returns an embedded dictionary variant. Classic and reform variants
// are built once in memory from the bundled source lexicon, then shared.
func Bundled(variant Variant) (*Dictionary, error) {
	switch variant {
	case "", AllVariants:
		return Default()
	case Classic:
		return loadClassic()
	case Reform1990:
		return loadReform()
	default:
		return nil, fmt.Errorf("unknown dictionary variant %q", variant)
	}
}

// BundledLexicon opens an independent stream over the original embedded TSV.
// Callers must close the returned reader. No filesystem access is needed.
func BundledLexicon() (io.ReadCloser, error) { return gzip.NewReader(bytes.NewReader(bundledLexicon)) }

// BuildBundled regenerates a dictionary using the native Go builder and the
// embedded original lexicon. Unlike Bundled, it does not use the cached graph.
func BuildBundled(variant Variant) (*Dictionary, error) {
	opts := BuildOptions{Stemming: "S", Metadata: Metadata{LanguageCode: "fr", LanguageName: "French", SourceFile: "French.lex"}}
	switch variant {
	case "", AllVariants:
		opts.Metadata.Name = "fr-allvars"
		opts.Metadata.Description = "French (all spelling variants)"
	case Classic:
		opts.Metadata.Name = "fr-classic"
		opts.Metadata.Description = "French (classic spelling)"
		opts.TagPattern = "[*CMPX]$"
	case Reform1990:
		opts.Metadata.Name = "fr-reform"
		opts.Metadata.Description = "French (1990 spelling reform)"
		opts.TagPattern = "[*RPX]$"
	default:
		return nil, fmt.Errorf("unknown dictionary variant %q", variant)
	}
	r, err := BundledLexicon()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	entries, _, err := ReadLexicon(r)
	if err != nil {
		return nil, err
	}
	return Build(entries, opts)
}
