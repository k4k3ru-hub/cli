# CLI (Command Line Interface) Utility for Go

This project provides a customizable CLI (Command Line Interface) tool written in Go. It supports multiple commands, options, and subcommands with versioning and configuration capabilities.


## Features

- Custom command execution
- Subcommand support
- Global option inheritance
- Positional arguments and argument-count validation
- Command-specific help output
- Version flag (--version | -v)
- Help flag (--help | -h)


## Installation

Importing this module.
```console
import "github.com/k4k3ru-hub/cli/go"
```


## Usage

1. Run as default

There are reserved flags:
- --version | -v: Show the version of the CLI tool.
- --help | -h: Display a list of available commands and options.

When you run like:
```console
go run main.go --version
```

It would be output:
```output
Version: 1.0.0
```

2. Initialize CLI

```go
myCLI := cli.NewCLI(mainFunc)

func mainFunc(ctx *cli.Context) error {
	// Here is default function.
	return nil
}
```

3. Set options

```go
// Version.
myCLI.SetVersion("1.0.0")

// Default config option.
if err := myCLI.Root().AddDefaultConfigOption(); err != nil {
	// Handle the registration error.
}

// Customized option.
if err := myCLI.Root().AddOption("local", cli.Option{
	Alias:  "l",
	IsFlag: true,
}); err != nil {
	// Handle the registration error.
}
```

4. Run the CLI

```go
if err := myCLI.Run(); err != nil {
	// Handle the execution error.
}
```

5. Positional arguments

Commands with actions accept positional arguments. Use `SetArgumentCount` when
the command requires a specific range. Set the maximum to `-1` for no limit.

```go
copyCommand := cli.NewCommand("copy")
if err := copyCommand.SetArgumentCount(2, 2); err != nil {
	// Handle the validation error.
}
copyCommand.SetAction(func(ctx *cli.Context) error {
	arguments := ctx.Arguments()
	fmt.Printf("copy %s to %s\n", arguments[0], arguments[1])
	return nil
})
```

Use `--` to stop option parsing. All following values are treated as positional
arguments, including values beginning with `-`.

```console
app copy source.txt -- --destination.txt
```

Options registered on a parent command are inherited by its subcommands. A
subcommand may override an inherited option with the same name. Reusing an
inherited alias for a different option is rejected.

6. Explicit arguments and I/O

Use `RunArgs` when arguments come from somewhere other than `os.Args`, such as
tests or an embedding application. The executable name is not included.

```go
myCLI := cli.NewCLIWithName("app", mainFunc)
if err := myCLI.RunArgs([]string{"list", "--local"}); err != nil {
	// Handle the execution error.
}
```

By default, the CLI uses standard input, standard output, and standard error.
They can be replaced without modifying process-global state.

```go
var output bytes.Buffer
if err := myCLI.SetIO(strings.NewReader(""), &output, io.Discard); err != nil {
	// Handle the configuration error.
}
```

Injected streams are available inside an action through `Input`, `Output`, and
`ErrorOutput`. Output errors should be returned to the caller.

```go
command.SetAction(func(ctx *cli.Context) error {
	if _, err := fmt.Fprintln(ctx.Output(), "completed"); err != nil {
		return fmt.Errorf("failed to output command result: %w", err)
	}
	return nil
})
```

7. Table output

Use `OutputTableTo` to write a table to an explicit destination. `OutputTable`
is a convenience wrapper that writes to standard output.

```go
var output bytes.Buffer
if err := cli.OutputTableTo(
	&output,
	[]string{"name", "status"},
	[][]any{{"alpha", "ready"}, {"beta", nil}},
); err != nil {
	// Handle validation or output errors.
}
```

Headers must be non-empty, and every row must contain exactly the same number
of columns as the headers. Nil cells are rendered as `<nil>`. Invalid UTF-8,
tabs, and line breaks in headers or cells are rejected because the table format
is single-line per row.

Unicode text is preserved and alignment is calculated by Unicode code-point
count. The standard-library formatter treats every code point as the same
width, so visual alignment of East Asian wide characters or combining
characters can vary by terminal and font.

8. Error inspection

CLI errors can be classified with `errors.Is`. Action errors are returned
unchanged, and output errors preserve their underlying cause.

```go
err := myCLI.RunArgs(args)
switch {
case errors.Is(err, cli.ErrUnknownCommand):
	// Handle an unknown command.
case errors.Is(err, cli.ErrUnknownOption):
	// Handle an unknown option.
case errors.Is(err, cli.ErrMissingOptionValue):
	// Request the missing value.
case errors.Is(err, cli.ErrInvalidArgumentCount):
	// Report the expected argument count.
case errors.Is(err, cli.ErrOutput):
	// Handle an output failure.
case err != nil:
	// Handle an action or another CLI error.
}
```

Structured details are available through `CLIError`.

```go
var cliError *cli.CLIError
if errors.As(err, &cliError) {
	fmt.Printf(
		"operation=%s command=%s option=%s argument=%s\n",
		cliError.Operation,
		cliError.CommandName,
		cliError.OptionName,
		cliError.Argument,
	)
}
```

Option definitions and execution values are separate. `Option` configures a
command without storing runtime state. Parsed values are read from `Context`.

```go
if err := command.AddOption("config", cli.Option{
	Alias:        "c",
	DefaultValue: "config.yaml",
}); err != nil {
	// Handle the registration error.
}

command.SetAction(func(ctx *cli.Context) error {
	config, ok := ctx.Option("config")
	if !ok {
		return errors.New("config option was not found")
	}
	fmt.Printf("value=%s explicitly_set=%t\n", config.Value, config.IsSet)
	return nil
})
```


## Support me
I am a Japanese developer, and your support is a great encouragement for my work!
In addition to support, feel free to reach out with comments, feature requests, or development inquiries!

Thank you for your support😊

[![Support on Ko-fi](https://img.shields.io/badge/Ko--fi-Support%20Me-blue?style=flat-square&logo=ko-fi)](https://ko-fi.com/k4k3ru)
[![Support on Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-Support%20Me-yellow?style=flat-square&logo=buy-me-a-coffee)](https://buymeacoffee.com/k4k3ru)


## License
This repository is open-source and distributed under the MIT License.
