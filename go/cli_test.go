//
// cli_test.go
//
package cli_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/cli-go"
)


//
// Test version option
//
func TestVersionOption(t *testing.T) {
	// Set test command arguments.
	os.Args = []string{"app", "--version"}

	// Capture the standard output of a command execution.
	output := captureStdout(func() {
		cli := cli.NewCLI(func(options map[string]*cli.Option) {})
		cli.SetVersion("1.0.0")
		cli.Run()
	})

	expected := "Version: 1.0.0\n"
    if output != expected {
        t.Errorf("Expected output '%s', got '%s'.", strings.ReplaceAll(expected, "\n", "\\n"), strings.ReplaceAll(output, "\n", "\\n"))
    }
}


//
// Capture the standard output of a command execution
//
func captureStdout(f func()) string {
	// Backup the standard output.
	oldStdout := os.Stdout

	// Change the standard output to pipeline.
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the specified function.
	f()

	// Revert the standard output.
	w.Close()
	os.Stdout = oldStdout

	// Get the captured content.
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
