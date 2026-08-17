// main.go
package main

import (
	"fmt"
	"os"

	"github.com/k4k3ru-hub/cli/go"
)

func main() {
	// Initialize CLI.
	myCLI := cli.NewCLI(mainFunc)
	myCLI.SetVersion("1.0.0")
	if err := myCLI.Root().AddDefaultConfigOption(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Add `list` command.
	listCommand := cli.NewCommand("list")
	listCommand.SetUsage("List the configuration.")
	listCommand.SetAction(listFunc)
	if err := listCommand.AddOption("local", cli.Option{
		Alias:  "l",
		IsFlag: true,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := myCLI.Root().AddCommand(listCommand); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Add `push` command.
	pushCommand := cli.NewCommand("push")
	pushCommand.SetUsage("Push the source code.")
	if err := myCLI.Root().AddCommand(pushCommand); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Add `push > origin` command.
	pushOriginCommand := cli.NewCommand("origin")
	pushOriginCommand.SetUsage("Push the source code to the origin.")
	pushOriginCommand.SetAction(pushOringFunc)
	if err := pushOriginCommand.AddOption("url", cli.Option{
		Alias:        "u",
		DefaultValue: "https://example.com",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := pushCommand.AddCommand(pushOriginCommand); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Run the CLI.
	if err := myCLI.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func mainFunc(ctx *cli.Context) error {
	for _, o := range ctx.Options() {
		if _, err := fmt.Fprintf(ctx.Output(), "%v\n", o); err != nil {
			return fmt.Errorf("failed to output root command option: %w", err)
		}
	}
	return nil
}

func listFunc(ctx *cli.Context) error {
	if _, err := fmt.Fprintln(ctx.Output(), "Started list func."); err != nil {
		return fmt.Errorf("failed to output list command status: %w", err)
	}
	for _, o := range ctx.Options() {
		if _, err := fmt.Fprintf(ctx.Output(), "%v\n", o); err != nil {
			return fmt.Errorf("failed to output list command option: %w", err)
		}
	}
	return nil
}

func pushOringFunc(ctx *cli.Context) error {
	if _, err := fmt.Fprintln(ctx.Output(), "Started push origin func."); err != nil {
		return fmt.Errorf("failed to output push origin command status: %w", err)
	}
	for _, o := range ctx.Options() {
		if _, err := fmt.Fprintf(ctx.Output(), "%v\n", o); err != nil {
			return fmt.Errorf("failed to output push origin command option: %w", err)
		}
	}
	return nil
}
