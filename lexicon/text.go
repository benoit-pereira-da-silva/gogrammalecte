package lexicon

import (
	"bufio"
	"io"
)

// universalReader preserves Python text-mode newline handling for source files:
// LF, CRLF and CR all become LF without changing the embedded source bytes.
type universalReader struct {
	reader *bufio.Reader
	skipLF bool
}

func newUniversalReader(r io.Reader) *universalReader {
	return &universalReader{reader: bufio.NewReader(r)}
}

func (r *universalReader) Read(buffer []byte) (int, error) {
	written := 0
	for written < len(buffer) {
		b, err := r.reader.ReadByte()
		if err != nil {
			return written, err
		}
		if r.skipLF {
			r.skipLF = false
			if b == '\n' {
				continue
			}
		}
		if b == '\r' {
			b = '\n'
			r.skipLF = true
		}
		buffer[written] = b
		written++
	}
	return written, nil
}
