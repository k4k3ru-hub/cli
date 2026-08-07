// cli_test.go
package cli_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/cli/go"
)

// Test version option
func TestVersionOption(t *testing.T) {
	// Set test command arguments.
	os.Args = []string{"app", "--version"}

	// Capture the standard output of a command execution.
	output := captureStdout(func() {
		cli := cli.NewCLI(func(command *cli.Command) error { return nil })
		cli.SetVersion("1.0.0")
		if err := cli.Run(); err != nil {
			t.Fatalf("Run() returned an unexpected error: %v", err)
		}
	})

	expected := "Version: 1.0.0\n"
	if output != expected {
		t.Errorf("Expected output '%s', got '%s'.", strings.ReplaceAll(expected, "\n", "\\n"), strings.ReplaceAll(output, "\n", "\\n"))
	}
}

// Test action error.
func TestActionError(t *testing.T) {
	expected := errors.New("action failed")
	os.Args = []string{"app"}

	application := cli.NewCLI(func(command *cli.Command) error {
		return expected
	})

	if err := application.Run(); !errors.Is(err, expected) {
		t.Fatalf("Run() error = %v, want %v", err, expected)
	}
}

// Test subcommand action error.
func TestSubcommandActionError(t *testing.T) {
	expected := errors.New("subcommand action failed")
	os.Args = []string{"app", "migrate"}

	application := cli.NewCLI(nil)
	migrateCommand := cli.NewCommand("migrate")
	migrateCommand.SetAction(func(command *cli.Command) error {
		return expected
	})
	if err := application.Command.AddCommand(migrateCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.Run(); !errors.Is(err, expected) {
		t.Fatalf("Run() error = %v, want %v", err, expected)
	}
}

// Test unknown command error.
func TestUnknownCommandError(t *testing.T) {
	os.Args = []string{"app", "unknown"}
	application := cli.NewCLI(nil)

	if err := application.Run(); err == nil {
		t.Fatal("Run() error = nil, want an unknown command error")
	}
}

// Test unknown option error.
func TestUnknownOptionError(t *testing.T) {
	os.Args = []string{"app", "--unknown"}
	application := cli.NewCLI(nil)

	if err := application.Run(); err == nil {
		t.Fatal("Run() error = nil, want an unknown option error")
	}
}

// Test inherited option is available to a subcommand action.
func TestInheritedOption(t *testing.T) {
	os.Args = []string{"app", "migrate", "--config", "migration.yaml"}
	application := cli.NewCLI(nil)
	application.Command.SetDefaultConfigOption()

	migrateCommand := cli.NewCommand("migrate")
	migrateCommand.SetAction(func(command *cli.Command) error {
		option := command.GetOption(cli.OptConfigName)
		if option == nil {
			return errors.New("inherited config option was not found")
		}
		if option.Value != "migration.yaml" {
			return errors.New("inherited config option has an unexpected value")
		}
		return nil
	})
	if err := application.Command.AddCommand(migrateCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// Test option inheritance does not modify command definitions.
func TestOptionInheritanceDoesNotModifyDefinitions(t *testing.T) {
	os.Args = []string{"app", "child", "--config", "runtime.yaml"}
	application := cli.NewCLI(nil)
	if err := application.Command.AddOption(cli.OptConfigName, &cli.Option{Value: "root.yaml"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}

	childCommand := cli.NewCommand("child")
	if err := childCommand.AddOption(cli.OptConfigName, &cli.Option{Value: "child.yaml"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	if err := childCommand.AddOption("child-only", &cli.Option{IsFlagSet: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	childCommand.SetAction(func(command *cli.Command) error {
		option := command.GetOption(cli.OptConfigName)
		if option == nil || option.Value != "runtime.yaml" {
			return errors.New("child config option was not parsed")
		}
		if command.GetOption("child-only") == nil {
			return errors.New("child option was not available")
		}
		return nil
	})
	if err := application.Command.AddCommand(childCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	if got := application.Command.GetOption(cli.OptConfigName).Value; got != "root.yaml" {
		t.Fatalf("root config option value = %q, want %q", got, "root.yaml")
	}
	if application.Command.GetOption("child-only") != nil {
		t.Fatal("child option leaked into the root command")
	}
	if got := childCommand.GetOption(cli.OptConfigName).Value; got != "child.yaml" {
		t.Fatalf("child config option value = %q, want %q", got, "child.yaml")
	}
}

// Test command validation.
func TestAddCommandValidation(t *testing.T) {
	root := cli.NewCommand("root")

	if err := root.AddCommand(nil); err == nil {
		t.Fatal("AddCommand(nil) error = nil, want an error")
	}
	if err := root.AddCommand(cli.NewCommand("")); err == nil {
		t.Fatal("AddCommand(empty name) error = nil, want an error")
	}
	if err := root.AddCommand(cli.NewCommand("child")); err != nil {
		t.Fatalf("AddCommand(child) returned an unexpected error: %v", err)
	}
	if err := root.AddCommand(cli.NewCommand("child")); err == nil {
		t.Fatal("AddCommand(duplicate) error = nil, want an error")
	}
}

// Test option validation.
func TestAddOptionValidation(t *testing.T) {
	command := cli.NewCommand("command")

	if err := command.AddOption("option", nil); err == nil {
		t.Fatal("AddOption(nil) error = nil, want an error")
	}
	if err := command.AddOption("", &cli.Option{}); err == nil {
		t.Fatal("AddOption(empty name) error = nil, want an error")
	}
	if err := command.AddOption("first", &cli.Option{Alias: "f"}); err != nil {
		t.Fatalf("AddOption(first) returned an unexpected error: %v", err)
	}
	if err := command.AddOption("first", &cli.Option{}); err == nil {
		t.Fatal("AddOption(duplicate name) error = nil, want an error")
	}
	if err := command.AddOption("second", &cli.Option{Alias: "f"}); err == nil {
		t.Fatal("AddOption(duplicate alias) error = nil, want an error")
	}
}

// Capture the standard output of a command execution
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
