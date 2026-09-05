package interactive_test

import (
	"bytes"
	"testing"

	"github.com/k4k3ru-hub/cli/go/interactive"
)

func TestWriteLineWithoutTerminalColor(t *testing.T) {
	for _, style := range []interactive.MessageStyle{
		interactive.MessageStyleDefault,
		interactive.MessageStyleMuted,
		interactive.MessageStyleSuccess,
		interactive.MessageStyleError,
	} {
		var output bytes.Buffer
		if err := interactive.WriteLine(&output, "message", style); err != nil {
			t.Fatalf("WriteLine() returned an unexpected error: %v", err)
		}
		if output.String() != "message\r\n" {
			t.Fatalf("output = %q, want %q", output.String(), "message\r\n")
		}
	}
}

func TestWriteLineValidation(t *testing.T) {
	if err := interactive.WriteLine(nil, "message", interactive.MessageStyleDefault); err == nil {
		t.Fatal("WriteLine() error = nil for nil output")
	}
	if err := interactive.WriteLine(&bytes.Buffer{}, "", interactive.MessageStyleDefault); err == nil {
		t.Fatal("WriteLine() error = nil for empty message")
	}
	if err := interactive.WriteLine(&bytes.Buffer{}, "message", interactive.MessageStyle(255)); err == nil {
		t.Fatal("WriteLine() error = nil for invalid style")
	}
}
