package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func execute(args []string, input io.Reader, output io.Writer) error {
	command := NewCommand("test-version")
	command.SetArgs(args)
	command.SetIn(input)
	command.SetOut(output)
	command.SetErr(io.Discard)
	return command.Execute()
}

func TestTextCommands(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        []string
	}{
		{"paragraphs", "Bonjour.\r\n\r\nÀ bientôt.\r", []string{"Bonjour.", "", "À bientôt.", ""}},
		{"paragraphs", "", []string{""}},
		{"sentences", "😀 Bonjour. À bientôt !", []string{"😀 Bonjour. ", "À bientôt !"}},
		{"sentences", "Un essai. puis la suite", []string{"Un essai. puis la suite"}},
	} {
		t.Run(tc.name+tc.input, func(t *testing.T) {
			var output bytes.Buffer
			if err := execute([]string{tc.name}, strings.NewReader(tc.input), &output); err != nil {
				t.Fatal(err)
			}
			var got []string
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	var output bytes.Buffer
	if err := execute([]string{"--version"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "test-version") {
		t.Fatalf("missing version: %q", output.String())
	}
}

func TestRejectsInvalidInput(t *testing.T) {
	for _, args := range [][]string{{"paragraphs"}, {"sentences"}} {
		var output bytes.Buffer
		err := execute(args, bytes.NewReader([]byte{0xff}), &output)
		if err == nil || !strings.Contains(err.Error(), "UTF-8") || output.Len() != 0 {
			t.Fatalf("got error %v and output %q", err, output.String())
		}
	}
}

func TestRejectsUnknownCommandsAndArguments(t *testing.T) {
	for _, args := range [][]string{{"check"}, {"paragraphs", "unexpected"}, {"sentences", "--unknown"}} {
		var output bytes.Buffer
		if err := execute(args, strings.NewReader(""), &output); err == nil {
			t.Fatalf("accepted unsupported arguments %q", args)
		}
		if output.Len() != 0 {
			t.Fatalf("unexpected output %q", output.String())
		}
	}
}

var errIO = errors.New("test I/O failure")

type brokenIO struct{}

func (brokenIO) Read([]byte) (int, error)  { return 0, errIO }
func (brokenIO) Write([]byte) (int, error) { return 0, errIO }

func TestPropagatesIOErrors(t *testing.T) {
	if err := execute([]string{"paragraphs"}, brokenIO{}, io.Discard); !errors.Is(err, errIO) {
		t.Fatalf("read error lost: %v", err)
	}
	if err := execute([]string{"sentences"}, strings.NewReader("Bonjour."), brokenIO{}); !errors.Is(err, errIO) {
		t.Fatalf("write error lost: %v", err)
	}
}
