package french

import (
	"context"
	"testing"
)

func TestTextFormatterPythonParity(t *testing.T) {
	var fixture struct {
		Rules []struct {
			Option        string
			Index         int
			Input, Output string
			Count         int
		}
		Full []struct {
			Input, Output string
			Options       map[string]bool
		}
	}
	readFixture(t, "textformatter", &fixture)
	f, err := NewTextFormatter()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Rules {
		got, count, err := f.applyRule(context.Background(), c.Input, c.Option, c.Index)
		if err != nil || got != c.Output || count != c.Count {
			t.Errorf("%s[%d] %q: got %q (%d,%v), want %q (%d)", c.Option, c.Index, c.Input, got, count, err, c.Output, c.Count)
		}
	}
	for _, c := range fixture.Full {
		got, err := f.FormatWithOptions(context.Background(), c.Input, c.Options)
		if err != nil || got != c.Output {
			t.Errorf("full %q options=%v: got %q (%v), want %q", c.Input, c.Options, got, err, c.Output)
		}
	}
}
func TestTextFormatterOptionsAndErrors(t *testing.T) {
	f, err := NewTextFormatter()
	if err != nil {
		t.Fatal(err)
	}
	options := f.DefaultOptions()
	options["start_of_paragraph"] = false
	if !f.DefaultOptions()["start_of_paragraph"] {
		t.Fatal("default options escaped by reference")
	}
	if _, err := f.FormatWithOptions(context.Background(), "", map[string]bool{"unknown": true}); err == nil {
		t.Fatal("expected unknown option error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.FormatText(ctx, "une phrase"); err == nil {
		t.Fatal("expected cancellation error")
	}
}
