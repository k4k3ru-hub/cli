package interactive_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/cli/go/interactive"
)

func TestPromptAsk(t *testing.T) {
	var output bytes.Buffer
	prompt, err := interactive.NewPrompt(strings.NewReader("user@example.com\n"), &output)
	if err != nil {
		t.Fatalf("NewPrompt() returned an unexpected error: %v", err)
	}

	value, err := prompt.Ask(context.Background(), "Email address")
	if err != nil {
		t.Fatalf("Ask() returned an unexpected error: %v", err)
	}
	if value != "user@example.com" {
		t.Fatalf("Ask() = %q, want %q", value, "user@example.com")
	}
	if output.String() != "Email address: " {
		t.Fatalf("output = %q, want %q", output.String(), "Email address: ")
	}
}

func TestPromptAskOTP(t *testing.T) {
	prompt, err := interactive.NewPrompt(strings.NewReader("123456\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewPrompt() returned an unexpected error: %v", err)
	}

	value, err := prompt.Ask(context.Background(), "OTP code")
	if err != nil {
		t.Fatalf("Ask() returned an unexpected error: %v", err)
	}
	if value != "123456" {
		t.Fatalf("Ask() = %q, want %q", value, "123456")
	}
}

func TestPromptAskRepeatedly(t *testing.T) {
	var output bytes.Buffer
	prompt, err := interactive.NewPrompt(strings.NewReader("user@example.com\n052784\n"), &output)
	if err != nil {
		t.Fatalf("NewPrompt() returned an unexpected error: %v", err)
	}
	email, err := prompt.Ask(context.Background(), "Email address")
	if err != nil {
		t.Fatalf("Ask() returned an unexpected email error: %v", err)
	}
	code, err := prompt.Ask(context.Background(), "OTP code")
	if err != nil {
		t.Fatalf("Ask() returned an unexpected OTP error: %v", err)
	}
	if email != "user@example.com" || code != "052784" {
		t.Fatalf("Ask() values = %q, %q, want email and OTP", email, code)
	}
	if output.String() != "Email address: OTP code: " {
		t.Fatalf("output = %q, want repeated prompt labels", output.String())
	}
}

func TestPromptAskCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	prompt, err := interactive.NewPrompt(strings.NewReader(""), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewPrompt() returned an unexpected error: %v", err)
	}

	_, err = prompt.Ask(ctx, "Email address")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Ask() error = %v, want context.Canceled", err)
	}
}

func TestNewPromptValidation(t *testing.T) {
	if _, err := interactive.NewPrompt(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("NewPrompt() error = nil for nil input")
	}
	if _, err := interactive.NewPrompt(strings.NewReader(""), nil); err == nil {
		t.Fatal("NewPrompt() error = nil for nil output")
	}
}
