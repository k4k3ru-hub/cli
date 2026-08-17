package cli_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/cli/go"
)

// Test parse errors support category and detail inspection.
func TestParseErrorInspection(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*testing.T, *cli.CLI)
		args       []string
		kind       error
		optionName string
		argument   string
	}{
		{
			name:     "unknown command",
			args:     []string{"unknown"},
			kind:     cli.ErrUnknownCommand,
			argument: "unknown",
		},
		{
			name:     "unknown option",
			args:     []string{"--unknown"},
			kind:     cli.ErrUnknownOption,
			argument: "--unknown",
		},
		{
			name: "missing option value",
			configure: func(t *testing.T, application *cli.CLI) {
				if err := application.Root().AddDefaultConfigOption(); err != nil {
					t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
				}
			},
			args:       []string{"--config"},
			kind:       cli.ErrMissingOptionValue,
			optionName: cli.OptConfigName,
		},
		{
			name: "unexpected flag value",
			configure: func(t *testing.T, application *cli.CLI) {
				if err := application.Root().AddOption("verbose", cli.Option{IsFlag: true}); err != nil {
					t.Fatalf("AddOption() returned an unexpected error: %v", err)
				}
			},
			args:       []string{"--verbose=true"},
			kind:       cli.ErrUnexpectedOptionValue,
			optionName: "verbose",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := cli.NewCLIWithName("app", nil)
			var output bytes.Buffer
			if err := application.SetIO(strings.NewReader(""), &output, io.Discard); err != nil {
				t.Fatalf("SetIO() returned an unexpected error: %v", err)
			}
			if test.configure != nil {
				test.configure(t, application)
			}

			err := application.RunArgs(test.args)
			if !errors.Is(err, test.kind) {
				t.Fatalf("RunArgs() error = %v, want category %v", err, test.kind)
			}
			var cliError *cli.CLIError
			if !errors.As(err, &cliError) {
				t.Fatalf("RunArgs() error = %T, want *cli.CLIError", err)
			}
			if cliError.OptionName != test.optionName {
				t.Fatalf("CLIError.OptionName = %q, want %q", cliError.OptionName, test.optionName)
			}
			if test.kind == cli.ErrUnknownCommand {
				if cliError.CommandName != test.argument {
					t.Fatalf("CLIError.CommandName = %q, want %q", cliError.CommandName, test.argument)
				}
			} else if cliError.Argument != test.argument {
				t.Fatalf("CLIError.Argument = %q, want %q", cliError.Argument, test.argument)
			}
			if output.Len() != 0 {
				t.Fatalf("parse error wrote unexpected output %q", output.String())
			}
		})
	}
}

// Test registration errors expose stable categories.
func TestRegistrationErrorCategories(t *testing.T) {
	root := cli.NewCommand("root")
	child := cli.NewCommand("child")
	if err := root.AddCommand(child); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := root.AddCommand(cli.NewCommand("child")); !errors.Is(err, cli.ErrDuplicateCommand) {
		t.Fatalf("AddCommand() error = %v, want ErrDuplicateCommand", err)
	}
	if err := child.AddCommand(root); !errors.Is(err, cli.ErrInvalidCommandHierarchy) {
		t.Fatalf("AddCommand() error = %v, want ErrInvalidCommandHierarchy", err)
	}
	if err := root.AddOption("help", cli.Option{}); !errors.Is(err, cli.ErrReservedOption) {
		t.Fatalf("AddOption() error = %v, want ErrReservedOption", err)
	}
	if err := root.AddOption("first", cli.Option{Alias: "f"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	if err := root.AddOption("second", cli.Option{Alias: "f"}); !errors.Is(err, cli.ErrDuplicateOption) {
		t.Fatalf("AddOption() error = %v, want ErrDuplicateOption", err)
	}
}

// Test output errors preserve their category and underlying cause.
func TestOutputErrorInspection(t *testing.T) {
	underlying := errors.New("write failed")
	application := cli.NewCLIWithName("app", nil)
	if err := application.SetIO(strings.NewReader(""), errorWriter{err: underlying}, io.Discard); err != nil {
		t.Fatalf("SetIO() returned an unexpected error: %v", err)
	}

	err := application.RunArgs([]string{"--help"})
	if !errors.Is(err, cli.ErrOutput) {
		t.Fatalf("RunArgs() error = %v, want ErrOutput", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("RunArgs() error = %v, want underlying error", err)
	}
}

// Test argument count errors expose their limits.
func TestArgumentCountErrorInspection(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	copyCommand := cli.NewCommand("copy")
	copyCommand.SetAction(func(ctx *cli.Context) error { return nil })
	if err := copyCommand.SetArgumentCount(2, 3); err != nil {
		t.Fatalf("SetArgumentCount() returned an unexpected error: %v", err)
	}
	if err := application.Root().AddCommand(copyCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	err := application.RunArgs([]string{"copy", "source.txt"})
	if !errors.Is(err, cli.ErrInvalidArgumentCount) {
		t.Fatalf("RunArgs() error = %v, want ErrInvalidArgumentCount", err)
	}
	var cliError *cli.CLIError
	if !errors.As(err, &cliError) {
		t.Fatalf("RunArgs() error = %T, want *cli.CLIError", err)
	}
	if cliError.ArgumentCount != 1 || cliError.MinArgumentCount != 2 {
		t.Fatalf("argument count = %d, minimum = %d, want 1 and 2", cliError.ArgumentCount, cliError.MinArgumentCount)
	}
}

// Test invalid parameters expose a stable category and validation state.
func TestInvalidParameterInspection(t *testing.T) {
	var application *cli.CLI
	err := application.RunArgs(nil)
	if !errors.Is(err, cli.ErrInvalidParameter) {
		t.Fatalf("RunArgs() error = %v, want ErrInvalidParameter", err)
	}
	var cliError *cli.CLIError
	if !errors.As(err, &cliError) {
		t.Fatalf("RunArgs() error = %T, want *cli.CLIError", err)
	}
	if cliError.Parameter != "cli" || cliError.State != "null" {
		t.Fatalf("parameter = %q, state = %q, want cli=null", cliError.Parameter, cliError.State)
	}
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write(data []byte) (int, error) {
	return 0, writer.err
}
