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
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error { return nil })
	application.SetVersion("1.0.0")
	var output bytes.Buffer
	if err := application.SetIO(strings.NewReader(""), &output, io.Discard); err != nil {
		t.Fatalf("SetIO() returned an unexpected error: %v", err)
	}
	if err := application.RunArgs([]string{"--version"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}

	expected := "Version: 1.0.0\n"
	if output.String() != expected {
		t.Errorf("Expected output '%s', got '%s'.", strings.ReplaceAll(expected, "\n", "\\n"), strings.ReplaceAll(output.String(), "\n", "\\n"))
	}
}

// Test direct version field configuration.
func TestVersionField(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"app", "--version"}

	output := captureStdout(t, func() {
		application := cli.NewCLI(nil)
		application.SetVersion("2.0.0")
		if err := application.Run(); err != nil {
			t.Fatalf("Run() returned an unexpected error: %v", err)
		}
	})

	if output != "Version: 2.0.0\n" {
		t.Fatalf("version output = %q, want %q", output, "Version: 2.0.0\n")
	}
}

// Test CLI option lookup.
func TestCLIOption(t *testing.T) {
	application := cli.NewCLI(nil)
	if err := application.Root().AddOption("verbose", cli.Option{IsFlag: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}

	if _, ok := application.Option("verbose"); !ok {
		t.Fatal("Option(verbose) was not found")
	}
	if _, ok := application.Option("missing"); ok {
		t.Fatal("Option(missing) was unexpectedly found")
	}
}

// Test configured usage text is displayed.
func TestCommandUsageText(t *testing.T) {
	command := cli.NewCommand("deploy")
	command.SetUsage("Deploy the application.")

	output := captureStdout(t, func() {
		command.ShowUsage()
	})
	if !strings.Contains(output, "Deploy the application.") {
		t.Fatalf("ShowUsage() output = %q, want configured usage text", output)
	}
}

// TestCommandMetadata verifies command metadata accessors return safe values.
func TestCommandMetadata(t *testing.T) {
	root := cli.NewCommand("root")
	root.SetUsage("Root command.")
	child := cli.NewCommand("child")
	if err := root.AddCommand(child); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if root.Name() != "root" {
		t.Fatalf("Name() = %q, want %q", root.Name(), "root")
	}
	if root.Usage() != "Root command." {
		t.Fatalf("Usage() = %q, want %q", root.Usage(), "Root command.")
	}
	commands := root.Commands()
	if len(commands) != 1 || commands[0] != child {
		t.Fatalf("Commands() = %#v, want the registered child", commands)
	}
	commands[0] = nil
	if root.Commands()[0] != child {
		t.Fatal("Commands() exposed the command slice")
	}
	if root.HasAction() {
		t.Fatal("HasAction() = true before an action was registered")
	}
	root.SetAction(func(*cli.Context) error { return nil })
	if !root.HasAction() {
		t.Fatal("HasAction() = false after an action was registered")
	}

	var nilCommand *cli.Command
	if nilCommand.Name() != "" || nilCommand.Usage() != "" || nilCommand.Commands() != nil || nilCommand.HasAction() {
		t.Fatal("nil command metadata accessors returned non-zero values")
	}
}

// Test action error.
func TestActionError(t *testing.T) {
	expected := errors.New("action failed")

	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		return expected
	})

	if err := application.RunArgs(nil); !errors.Is(err, expected) {
		t.Fatalf("RunArgs() error = %v, want %v", err, expected)
	}
}

// Test subcommand action error.
func TestSubcommandActionError(t *testing.T) {
	expected := errors.New("subcommand action failed")
	application := cli.NewCLIWithName("app", nil)
	migrateCommand := cli.NewCommand("migrate")
	migrateCommand.SetAction(func(ctx *cli.Context) error {
		return expected
	})
	if err := application.Root().AddCommand(migrateCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"migrate"}); !errors.Is(err, expected) {
		t.Fatalf("RunArgs() error = %v, want %v", err, expected)
	}
}

// Test unknown command error.
func TestUnknownCommandError(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)

	if err := application.RunArgs([]string{"unknown"}); err == nil {
		t.Fatal("RunArgs() error = nil, want an unknown command error")
	}
}

// Test unknown option error.
func TestUnknownOptionError(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)

	if err := application.RunArgs([]string{"--unknown"}); err == nil {
		t.Fatal("RunArgs() error = nil, want an unknown option error")
	}
}

// Test an option requiring a value rejects a missing value.
func TestMissingOptionValue(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "end of arguments", args: []string{"app", "--config"}},
		{name: "followed by option", args: []string{"app", "--config", "--help"}},
		{name: "empty equals value", args: []string{"app", "--config="}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := cli.NewCLIWithName("app", nil)
			if err := application.Root().AddDefaultConfigOption(); err != nil {
				t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
			}

			err := application.RunArgs(test.args[1:])
			if err == nil || !strings.Contains(err.Error(), "missing option value") {
				t.Fatalf("Run() error = %v, want a missing option value error", err)
			}
		})
	}
}

// Test a flag rejects an assigned value.
func TestFlagRejectsValue(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.Root().AddOption("verbose", cli.Option{IsFlag: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}

	err := application.RunArgs([]string{"--verbose=true"})
	if err == nil || !strings.Contains(err.Error(), "flag does not accept a value") {
		t.Fatalf("Run() error = %v, want a flag value error", err)
	}
}

// Test an equals sign is preserved inside an option value.
func TestOptionValueContainingEquals(t *testing.T) {
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		option, ok := ctx.Option(cli.OptConfigName)
		if !ok || option.Value != "key=value" {
			return errors.New("option value containing equals was not parsed")
		}
		return nil
	})
	if err := application.Root().AddDefaultConfigOption(); err != nil {
		t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"--config=key=value"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
}

// Test option definitions and execution values remain separate.
func TestOptionDefinitionExecutionSeparation(t *testing.T) {
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		value, ok := ctx.Option("config")
		if !ok || value.Value != "config.yaml" || value.IsSet {
			return errors.New("default execution value was invalid")
		}
		values := ctx.Options()
		values["config"] = cli.OptionValue{Value: "modified", IsSet: true}
		value, _ = ctx.Option("config")
		if value.Value != "config.yaml" || value.IsSet {
			return errors.New("context option values exposed mutable state")
		}
		if ctx.CommandPath() != "app" {
			return errors.New("context command path was invalid")
		}
		return nil
	})
	if err := application.Root().AddOption("config", cli.Option{DefaultValue: "config.yaml"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}

	definitions := application.Root().Options()
	definitions["config"] = cli.Option{DefaultValue: "modified"}
	if err := application.RunArgs(nil); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
	definition, ok := application.Root().Option("config")
	if !ok || definition.DefaultValue != "config.yaml" {
		t.Fatalf("option definition = %+v, found = %t, want unchanged default", definition, ok)
	}
}

// Test a nil CLI receiver returns an error.
func TestNilCLIRun(t *testing.T) {
	var application *cli.CLI
	if err := application.Run(); err == nil {
		t.Fatal("Run() error = nil, want an error")
	}
}

// Test injected streams are available to command actions.
func TestInjectedCommandIO(t *testing.T) {
	input := strings.NewReader("request")
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		data, err := io.ReadAll(ctx.Input())
		if err != nil {
			return err
		}
		if _, err := ctx.Output().Write(data); err != nil {
			return err
		}
		if _, err := io.WriteString(ctx.ErrorOutput(), "diagnostic"); err != nil {
			return err
		}
		return nil
	})
	if err := application.SetIO(input, &output, &errorOutput); err != nil {
		t.Fatalf("SetIO() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs(nil); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
	if output.String() != "request" || errorOutput.String() != "diagnostic" {
		t.Fatalf("injected output = %q, error output = %q", output.String(), errorOutput.String())
	}
}

// Test RunArgs does not read process arguments.
func TestRunArgsIgnoresProcessArguments(t *testing.T) {
	originalArgs := os.Args
	os.Args = []string{"process", "--unknown"}
	t.Cleanup(func() { os.Args = originalArgs })

	called := false
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		called = true
		return nil
	})
	if err := application.RunArgs(nil); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
	if !called {
		t.Fatal("RunArgs() did not execute the root action")
	}
}

// Test RunArgs keeps concurrent execution state isolated.
func TestRunArgsConcurrent(t *testing.T) {
	results := make(chan string, 2)
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		option, ok := ctx.Option(cli.OptConfigName)
		if !ok {
			return errors.New("config option was not found")
		}
		results <- option.Value
		return nil
	})
	if err := application.Root().AddDefaultConfigOption(); err != nil {
		t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
	}
	errorsChannel := make(chan error, 2)

	go func() { errorsChannel <- application.RunArgs([]string{"--config", "first.yaml"}) }()
	go func() { errorsChannel <- application.RunArgs([]string{"--config", "second.yaml"}) }()
	for index := 0; index < 2; index++ {
		if err := <-errorsChannel; err != nil {
			t.Fatalf("RunArgs() returned an unexpected error: %v", err)
		}
	}

	values := map[string]bool{<-results: true, <-results: true}
	if !values["first.yaml"] || !values["second.yaml"] {
		t.Fatalf("concurrent option values = %v, want both execution values", values)
	}
}

// Test invalid injected streams are rejected.
func TestSetIOValidation(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	tests := []struct {
		name        string
		input       io.Reader
		output      io.Writer
		errorOutput io.Writer
	}{
		{name: "nil input", output: io.Discard, errorOutput: io.Discard},
		{name: "nil output", input: strings.NewReader(""), errorOutput: io.Discard},
		{name: "nil error output", input: strings.NewReader(""), output: io.Discard},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := application.SetIO(test.input, test.output, test.errorOutput); err == nil {
				t.Fatal("SetIO() error = nil, want an error")
			}
		})
	}
}

// Test help output failures are returned.
func TestHelpOutputError(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.SetIO(strings.NewReader(""), failingWriter{}, io.Discard); err != nil {
		t.Fatalf("SetIO() returned an unexpected error: %v", err)
	}

	err := application.RunArgs([]string{"--help"})
	if err == nil || !strings.Contains(err.Error(), "failed to show command usage") {
		t.Fatalf("RunArgs() error = %v, want an output error", err)
	}
}

// Test the option terminator is recognized.
func TestOptionTerminator(t *testing.T) {
	called := false
	application := cli.NewCLIWithName("app", func(ctx *cli.Context) error {
		called = true
		return nil
	})

	if err := application.RunArgs([]string{"--"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
	if !called {
		t.Fatal("root action was not called after the option terminator")
	}
}

// Test positional input after the option terminator is rejected explicitly.
func TestOptionTerminatorWithPositionalInput(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)

	err := application.RunArgs([]string{"--", "--literal"})
	if err == nil || !strings.Contains(err.Error(), "positional arguments are not supported") {
		t.Fatalf("Run() error = %v, want an unsupported positional argument error", err)
	}
}

// Test subcommand help displays its full command path and inherited options.
func TestSubcommandHelp(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.Root().AddDefaultConfigOption(); err != nil {
		t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
	}
	deployCommand := cli.NewCommand("deploy")
	deployCommand.SetUsage("Deploy the application.")
	if err := deployCommand.AddOption("force", cli.Option{Alias: "f", IsFlag: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	if err := application.Root().AddCommand(deployCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	var output bytes.Buffer
	if err := application.SetIO(strings.NewReader(""), &output, io.Discard); err != nil {
		t.Fatalf("SetIO() returned an unexpected error: %v", err)
	}
	if err := application.RunArgs([]string{"deploy", "--help"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
	for _, expected := range []string{"Usage: app deploy", "Deploy the application.", "config:", "force:"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("help output = %q, want it to contain %q", output.String(), expected)
		}
	}
}

// Test the global version flag is available after a subcommand.
func TestSubcommandVersion(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	application.SetVersion("2.1.0")
	if err := application.Root().AddCommand(cli.NewCommand("deploy")); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	var output bytes.Buffer
	if err := application.SetIO(strings.NewReader(""), &output, io.Discard); err != nil {
		t.Fatalf("SetIO() returned an unexpected error: %v", err)
	}
	if err := application.RunArgs([]string{"deploy", "--version"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
	if output.String() != "Version: 2.1.0\n" {
		t.Fatalf("version output = %q, want %q", output.String(), "Version: 2.1.0\n")
	}
}

// Test conflicting inherited option aliases are rejected.
func TestInheritedOptionAliasConflict(t *testing.T) {
	root := cli.NewCommand("root")
	if err := root.AddOption("output", cli.Option{Alias: "o"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	child := cli.NewCommand("child")
	if err := child.AddOption("overwrite", cli.Option{Alias: "o", IsFlag: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}

	if err := root.AddCommand(child); err == nil {
		t.Fatal("AddCommand() error = nil, want an inherited alias conflict error")
	}
}

// Test adding a parent option validates existing descendants.
func TestLateParentOptionAliasConflict(t *testing.T) {
	root := cli.NewCommand("root")
	child := cli.NewCommand("child")
	if err := child.AddOption("overwrite", cli.Option{Alias: "o", IsFlag: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	if err := root.AddCommand(child); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := root.AddOption("output", cli.Option{Alias: "o"}); err == nil {
		t.Fatal("AddOption() error = nil, want a descendant alias conflict error")
	}
}

// Test a child command can override an inherited option by name.
func TestChildOptionOverride(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.Root().AddDefaultConfigOption(); err != nil {
		t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
	}
	child := cli.NewCommand("child")
	if err := child.AddOption(cli.OptConfigName, cli.Option{Alias: "C", DefaultValue: "default.yaml"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	child.SetAction(func(ctx *cli.Context) error {
		option, ok := ctx.Option(cli.OptConfigName)
		if !ok || option.Value != "child.yaml" || !option.IsSet {
			return errors.New("overridden child option was not parsed")
		}
		return nil
	})
	if err := application.Root().AddCommand(child); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"child", "--config", "child.yaml"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
}

// Test options are inherited through multiple command levels.
func TestNestedOptionInheritance(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.Root().AddDefaultConfigOption(); err != nil {
		t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
	}
	serviceCommand := cli.NewCommand("service")
	if err := serviceCommand.AddOption("region", cli.Option{Alias: "r"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	deployCommand := cli.NewCommand("deploy")
	deployCommand.SetAction(func(ctx *cli.Context) error {
		config, configOK := ctx.Option(cli.OptConfigName)
		region, regionOK := ctx.Option("region")
		if !configOK || config.Value != "deploy.yaml" || !regionOK || region.Value != "ap-northeast-1" {
			return errors.New("nested options were not inherited")
		}
		return nil
	})
	if err := serviceCommand.AddCommand(deployCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}
	if err := application.Root().AddCommand(serviceCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"service", "deploy", "--config", "deploy.yaml", "--region", "ap-northeast-1"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
}

// Test reserved options cannot be redefined.
func TestReservedOptionValidation(t *testing.T) {
	command := cli.NewCommand("command")
	if err := command.AddOption(cli.OptHelpName, cli.Option{}); err == nil {
		t.Fatal("AddOption(help) error = nil, want a reserved option error")
	}
	if err := command.AddOption("custom", cli.Option{Alias: cli.OptVersionAlias}); err == nil {
		t.Fatal("AddOption(alias v) error = nil, want a reserved alias error")
	}
}

// Test positional arguments are delivered in input order.
func TestPositionalArguments(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	copyCommand := cli.NewCommand("copy")
	if err := copyCommand.SetArgumentCount(2, 2); err != nil {
		t.Fatalf("SetArgumentCount() returned an unexpected error: %v", err)
	}
	copyCommand.SetAction(func(ctx *cli.Context) error {
		arguments := ctx.Arguments()
		if len(arguments) != 2 || arguments[0] != "source.txt" || arguments[1] != "--literal" {
			return errors.New("positional arguments were not parsed")
		}
		arguments[0] = "modified"
		if ctx.Arguments()[0] != "source.txt" {
			return errors.New("Arguments returned mutable internal state")
		}
		return nil
	})
	if err := application.Root().AddCommand(copyCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"copy", "source.txt", "--", "--literal"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
}

// Test positional argument count validation.
func TestPositionalArgumentCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "too few", args: []string{"app", "copy"}, want: "too few positional arguments"},
		{name: "too many", args: []string{"app", "copy", "a", "b", "c"}, want: "too many positional arguments"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := cli.NewCLIWithName("app", nil)
			copyCommand := cli.NewCommand("copy")
			if err := copyCommand.SetArgumentCount(1, 2); err != nil {
				t.Fatalf("SetArgumentCount() returned an unexpected error: %v", err)
			}
			if err := application.Root().AddCommand(copyCommand); err != nil {
				t.Fatalf("AddCommand() returned an unexpected error: %v", err)
			}

			err := application.RunArgs(test.args[1:])
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Run() error = %v, want %q", err, test.want)
			}
		})
	}
}

// Test invalid positional argument ranges are rejected.
func TestArgumentCountValidation(t *testing.T) {
	command := cli.NewCommand("command")
	for _, counts := range [][2]int{{-1, 1}, {0, -2}, {2, 1}} {
		if err := command.SetArgumentCount(counts[0], counts[1]); err == nil {
			t.Fatalf("SetArgumentCount(%d, %d) error = nil, want an error", counts[0], counts[1])
		}
	}
}

// Test inherited option is available to a subcommand action.
func TestInheritedOption(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.Root().AddDefaultConfigOption(); err != nil {
		t.Fatalf("AddDefaultConfigOption() returned an unexpected error: %v", err)
	}

	migrateCommand := cli.NewCommand("migrate")
	migrateCommand.SetAction(func(ctx *cli.Context) error {
		option, ok := ctx.Option(cli.OptConfigName)
		if !ok {
			return errors.New("inherited config option was not found")
		}
		if option.Value != "migration.yaml" {
			return errors.New("inherited config option has an unexpected value")
		}
		return nil
	})
	if err := application.Root().AddCommand(migrateCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"migrate", "--config", "migration.yaml"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}
}

// Test option inheritance does not modify command definitions.
func TestOptionInheritanceDoesNotModifyDefinitions(t *testing.T) {
	application := cli.NewCLIWithName("app", nil)
	if err := application.Root().AddOption(cli.OptConfigName, cli.Option{DefaultValue: "root.yaml"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}

	childCommand := cli.NewCommand("child")
	if err := childCommand.AddOption(cli.OptConfigName, cli.Option{DefaultValue: "child.yaml"}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	if err := childCommand.AddOption("child-only", cli.Option{IsFlag: true}); err != nil {
		t.Fatalf("AddOption() returned an unexpected error: %v", err)
	}
	childCommand.SetAction(func(ctx *cli.Context) error {
		option, ok := ctx.Option(cli.OptConfigName)
		if !ok || option.Value != "runtime.yaml" {
			return errors.New("child config option was not parsed")
		}
		if _, ok := ctx.Option("child-only"); !ok {
			return errors.New("child option was not available")
		}
		return nil
	})
	if err := application.Root().AddCommand(childCommand); err != nil {
		t.Fatalf("AddCommand() returned an unexpected error: %v", err)
	}

	if err := application.RunArgs([]string{"child", "--config", "runtime.yaml"}); err != nil {
		t.Fatalf("RunArgs() returned an unexpected error: %v", err)
	}

	rootConfig, ok := application.Root().Option(cli.OptConfigName)
	if !ok {
		t.Fatal("root config option was not found")
	}
	if got := rootConfig.DefaultValue; got != "root.yaml" {
		t.Fatalf("root config option value = %q, want %q", got, "root.yaml")
	}
	if _, ok := application.Root().Option("child-only"); ok {
		t.Fatal("child option leaked into the root command")
	}
	childConfig, ok := childCommand.Option(cli.OptConfigName)
	if !ok {
		t.Fatal("child config option was not found")
	}
	if got := childConfig.DefaultValue; got != "child.yaml" {
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

	if err := command.AddOption("", cli.Option{}); err == nil {
		t.Fatal("AddOption(empty name) error = nil, want an error")
	}
	if err := command.AddOption("first", cli.Option{Alias: "f"}); err != nil {
		t.Fatalf("AddOption(first) returned an unexpected error: %v", err)
	}
	if err := command.AddOption("first", cli.Option{}); err == nil {
		t.Fatal("AddOption(duplicate name) error = nil, want an error")
	}
	if err := command.AddOption("second", cli.Option{Alias: "f"}); err == nil {
		t.Fatal("AddOption(duplicate alias) error = nil, want an error")
	}
}

// Capture the standard output of a command execution
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	// Backup the standard output.
	oldStdout := os.Stdout

	// Change the standard output to pipeline.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned an unexpected error: %v", err)
	}
	os.Stdout = w

	// Run the specified function.
	f()

	// Revert the standard output.
	if err := w.Close(); err != nil {
		t.Fatalf("Close() returned an unexpected error: %v", err)
	}
	os.Stdout = oldStdout

	// Get the captured content.
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy() returned an unexpected error: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close() returned an unexpected error: %v", err)
	}
	return buf.String()
}

type failingWriter struct{}

func (failingWriter) Write(data []byte) (int, error) {
	return 0, errors.New("write failed")
}
