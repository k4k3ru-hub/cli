package interactive

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	cli "github.com/k4k3ru-hub/cli/go"
)

func TestCommandCandidates(t *testing.T) {
	application := cli.NewCLIWithName("test", nil)
	account := cli.NewCommand("account")
	for _, name := range []string{"signup", "api-key"} {
		if err := account.AddCommand(cli.NewCommand(name)); err != nil {
			t.Fatalf("AddCommand() returned an unexpected error: %v", err)
		}
	}
	if err := application.Root().AddCommand(account); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	for _, name := range []string{"execute", "exit"} {
		if err := application.Root().AddCommand(cli.NewCommand(name)); err != nil {
			t.Fatalf("AddCommand() returned an unexpected error: %v", err)
		}
	}
	session, err := NewSession(strings.NewReader(""), &bytes.Buffer{}, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	tests := []struct {
		input string
		want  []string
	}{
		{input: "/", want: []string{"/account", "/execute", "/exit"}},
		{input: "/a", want: []string{"/account"}},
		{input: "/account", want: []string{"/account api-key", "/account signup"}},
		{input: "/account ", want: []string{"/account api-key", "/account signup"}},
		{input: "/account s", want: []string{"/account signup"}},
		{input: "/account signup", want: []string{"/account signup"}},
		{input: "/e", want: []string{"/execute", "/exit"}},
		{input: "/account unknown"},
		{input: "/unknown"},
		{input: "help"},
	}
	for _, test := range tests {
		candidates := session.commandCandidates(test.input)
		if len(candidates) != len(test.want) {
			t.Fatalf("commandCandidates(%q) count = %d, want %d", test.input, len(candidates), len(test.want))
		}
		for index, candidate := range candidates {
			if candidate.path != test.want[index] {
				t.Fatalf("commandCandidates(%q)[%d] = %q, want %q", test.input, index, candidate.path, test.want[index])
			}
		}
	}
}

func TestFormatCommandList(t *testing.T) {
	help := cli.NewCommand("help")
	help.SetUsage("Show help.")

	got := formatCommandList([]commandCandidate{{command: help, path: "/help"}}, 0, false)
	want := "Commands\n❯ /help  Show help."
	if got != want {
		t.Fatalf("formatCommandList() = %q, want %q", got, want)
	}
}

func TestFormatCommandListSelectionAndColor(t *testing.T) {
	execute := cli.NewCommand("execute")
	exit := cli.NewCommand("exit")

	got := formatCommandList([]commandCandidate{
		{command: execute, path: "/execute"},
		{command: exit, path: "/exit"},
	}, 1, true)
	want := "Commands\n  /execute\n" + ansiBoldCyan + "❯ /exit   " + ansiReset
	if got != want {
		t.Fatalf("formatCommandList() = %q, want %q", got, want)
	}
}

func TestCandidateContent(t *testing.T) {
	application := cli.NewCLIWithName("test", nil)
	help := cli.NewCommand("help")
	help.SetUsage("Show help.")
	if err := application.Root().AddCommand(help); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	session, err := NewSession(strings.NewReader(""), &bytes.Buffer{}, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty"},
		{name: "slash", input: "/", want: "Commands\n❯ /help  Show help."},
		{name: "natural language", input: "h", want: aiAgentUnderDevelopmentMessage},
		{name: "matching command", input: "/h", want: "Commands\n❯ /help  Show help."},
		{name: "unmatched command", input: "/unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := session.candidateContent(test.input, 0, false); got != test.want {
				t.Fatalf("candidateContent(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestCompleteInput(t *testing.T) {
	application := cli.NewCLIWithName("test", nil)
	account := cli.NewCommand("account")
	for _, name := range []string{"signup", "api-key"} {
		if err := account.AddCommand(cli.NewCommand(name)); err != nil {
			t.Fatalf("AddCommand() returned an unexpected error: %v", err)
		}
	}
	if err := application.Root().AddCommand(account); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	for _, name := range []string{"exit", "execute"} {
		if err := application.Root().AddCommand(cli.NewCommand(name)); err != nil {
			t.Fatalf("AddCommand() returned an unexpected error: %v", err)
		}
	}
	session, err := NewSession(strings.NewReader(""), &bytes.Buffer{}, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "hello", want: "hello"},
		{input: "/", want: "/account "},
		{input: "/a", want: "/account "},
		{input: "/account", want: "/account api-key"},
		{input: "/account s", want: "/account signup"},
		{input: "/e", want: "/execute"},
		{input: "/unknown", want: "/unknown"},
	}
	for _, test := range tests {
		if got := session.completeInput(test.input, 0); got != test.want {
			t.Fatalf("completeInput(%q) = %q, want %q", test.input, got, test.want)
		}
	}
	if got := session.completeInput("/e", 1); got != "/exit" {
		t.Fatalf("completeInput(%q, 1) = %q, want %q", "/e", got, "/exit")
	}
}

func TestCompleteInputUsesSelectedNestedCommand(t *testing.T) {
	application := cli.NewCLIWithName("test", nil)
	account := cli.NewCommand("account")
	for _, name := range []string{"signup", "status"} {
		if err := account.AddCommand(cli.NewCommand(name)); err != nil {
			t.Fatalf("AddCommand() returned an unexpected error: %v", err)
		}
	}
	if err := application.Root().AddCommand(account); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	session, err := NewSession(strings.NewReader(""), &bytes.Buffer{}, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	if got := session.completeInput("/account s", 1); got != "/account status" {
		t.Fatalf("completeInput(%q, 1) = %q, want %q", "/account s", got, "/account status")
	}
}

func TestIsExactCommand(t *testing.T) {
	application := cli.NewCLIWithName("test", nil)
	account := cli.NewCommand("account")
	if err := account.AddCommand(cli.NewCommand("signup")); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	if err := application.Root().AddCommand(account); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	session, err := NewSession(strings.NewReader(""), &bytes.Buffer{}, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	tests := []struct {
		input string
		want  bool
	}{
		{input: "/account", want: true},
		{input: "/account signup", want: true},
		{input: "/account signup ", want: false},
		{input: "/account unknown", want: false},
		{input: "/unknown", want: false},
		{input: "account signup", want: false},
	}
	for _, test := range tests {
		if got := session.isExactCommand(test.input); got != test.want {
			t.Fatalf("isExactCommand(%q) = %t, want %t", test.input, got, test.want)
		}
	}
}

func TestExecuteSubcommand(t *testing.T) {
	executed := false
	application := cli.NewCLIWithName("test", nil)
	account := cli.NewCommand("account")
	signup := cli.NewCommand("signup")
	signup.SetAction(func(*cli.Context) error {
		executed = true
		return nil
	})
	if err := account.AddCommand(signup); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	if err := application.Root().AddCommand(account); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	session, err := NewSession(strings.NewReader(""), &bytes.Buffer{}, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	if err := session.executeCommand("/account signup"); err != nil {
		t.Fatalf("executeCommand() returned an unexpected error: %v", err)
	}
	if !executed {
		t.Fatal("executeCommand() did not execute the nested command action")
	}
}

func TestExecuteCommandReturnsActionErrorWithoutInteractiveWrapper(t *testing.T) {
	expected := errors.New("failed to run account signup: err_code=\"already_created\"")
	application := cli.NewCLIWithName("test", nil)
	account := cli.NewCommand("account")
	signup := cli.NewCommand("signup")
	signup.SetAction(func(*cli.Context) error { return expected })
	if err := account.AddCommand(signup); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	if err := application.Root().AddCommand(account); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	var output bytes.Buffer
	session, err := NewSession(strings.NewReader(""), &output, application)
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}
	commandErr := session.executeCommand("/account signup")
	if !errors.Is(commandErr, expected) {
		t.Fatalf("executeCommand() error = %v, want %v", commandErr, expected)
	}
	if commandErr.Error() != expected.Error() {
		t.Fatalf("executeCommand() error = %q, want %q", commandErr.Error(), expected.Error())
	}
	if err := session.writeCommandError(commandErr); err != nil {
		t.Fatalf("writeCommandError() returned an unexpected error: %v", err)
	}
	if output.String() != expected.Error()+"\r\n" {
		t.Fatalf("output = %q, want %q", output.String(), expected.Error()+"\r\n")
	}
}

func TestRenderTerminalWithoutCandidatesLeavesCursorAtInputEnd(t *testing.T) {
	var output bytes.Buffer
	session, err := NewSession(strings.NewReader(""), &output, cli.NewCLIWithName("test", nil))
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	if err := session.renderTerminal([]rune(""), 0); err != nil {
		t.Fatalf("renderTerminal() returned an unexpected error: %v", err)
	}
	want := "\r\x1b[J> "
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestWriteNoMatchingCommands(t *testing.T) {
	var output bytes.Buffer
	session, err := NewSession(strings.NewReader(""), &output, cli.NewCLIWithName("test", nil))
	if err != nil {
		t.Fatalf("NewSession() returned an unexpected error: %v", err)
	}

	if err := session.writeNoMatchingCommands([]rune("/asdf")); err != nil {
		t.Fatalf("writeNoMatchingCommands() returned an unexpected error: %v", err)
	}
	want := "\r\x1b[J> /asdf\r\nNo matching commands.\r\n\r\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestMoveCandidateSelection(t *testing.T) {
	tests := []struct {
		name      string
		current   int
		count     int
		direction keyKind
		want      int
	}{
		{name: "down", current: 0, count: 2, direction: keyArrowDown, want: 1},
		{name: "down wraps", current: 1, count: 2, direction: keyArrowDown, want: 0},
		{name: "up", current: 1, count: 2, direction: keyArrowUp, want: 0},
		{name: "up wraps", current: 0, count: 2, direction: keyArrowUp, want: 1},
		{name: "empty", current: 0, count: 0, direction: keyArrowDown, want: 0},
	}
	for _, test := range tests {
		if got := moveCandidateSelection(test.current, test.count, test.direction); got != test.want {
			t.Fatalf("moveCandidateSelection() = %d, want %d", got, test.want)
		}
	}
}
