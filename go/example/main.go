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
	myCLI.Command.SetDefaultConfigOption()

	// Add `list` command.
	listCommand := cli.NewCommand("list")
	listCommand.SetUsage("List the configuration.")
	listCommand.SetAction(listFunc)
	if err := listCommand.AddOption("local", &cli.Option{
		Alias: "l",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := myCLI.Command.AddCommand(listCommand); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Add `push` command.
	pushCommand := cli.NewCommand("push")
	pushCommand.SetUsage("Push the source code.")
	if err := myCLI.Command.AddCommand(pushCommand); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Add `push > origin` command.
	pushOriginCommand := cli.NewCommand("origin")
	pushOriginCommand.SetUsage("Push the source code to the origin.")
	pushOriginCommand.SetAction(pushOringFunc)
	if err := pushOriginCommand.AddOption("url", &cli.Option{
		Alias: "u",
		Value: "https://exmaple.com",
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

func mainFunc(cmd *cli.Command) error {
	for _, o := range cmd.Options() {
		fmt.Printf("%v\n", o)
	}
	return nil
}

func listFunc(cmd *cli.Command) error {
	fmt.Printf("Started list func.\n")
	for _, o := range cmd.Options() {
		fmt.Printf("%v\n", o)
	}
	return nil
}

func pushOringFunc(cmd *cli.Command) error {
	fmt.Printf("Started push origin func.\n")
	for _, o := range cmd.Options() {
		fmt.Printf("%v\n", o)
	}
	return nil
}
