package interactive_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	cli "github.com/k4k3ru-hub/cli/go"
	"github.com/k4k3ru-hub/cli/go/interactive"
)

// TestSessionRunUntilEOF verifies that the session prompts for each input line.
func TestSessionRunUntilEOF(t *testing.T) {
	var output bytes.Buffer
	session, err := interactive.NewSession(
		strings.NewReader("first\nsecond\n"),
		&output,
		cli.NewCLIWithName("test", nil),
	)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	if err := session.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	if output.String() != "> > > " {
		t.Fatalf("output = %q, want %q", output.String(), "> > > ")
	}
}

// TestSessionRunCanceled verifies that cancellation stops a waiting session.
func TestSessionRunCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var output bytes.Buffer
	session, err := interactive.NewSession(strings.NewReader(""), &output, cli.NewCLIWithName("test", nil))
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	if err := session.Run(ctx); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// TestNewSessionValidation verifies required stream validation.
func TestNewSessionValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       *strings.Reader
		output      *bytes.Buffer
		application *cli.CLI
	}{
		{name: "nil input", output: &bytes.Buffer{}, application: cli.NewCLIWithName("test", nil)},
		{name: "nil output", input: strings.NewReader(""), application: cli.NewCLIWithName("test", nil)},
		{name: "nil cli", input: strings.NewReader(""), output: &bytes.Buffer{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := interactive.NewSession(test.input, test.output, test.application)
			if err == nil {
				t.Fatal("NewSession() error = nil, want an error")
			}
		})
	}
}

// TestSessionReadError verifies that input errors are propagated.
func TestSessionReadError(t *testing.T) {
	expected := errors.New("read failed")
	session, err := interactive.NewSession(
		errorReader{err: expected},
		&bytes.Buffer{},
		cli.NewCLIWithName("test", nil),
	)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	err = session.Run(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, expected)
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
