//
// main.go
//
package main

import (
	"fmt"

	"github.com/k4k3ru-hub/cli-go"
)


func main() {
    // Initialize CLI.
    myCLI := cli.NewCLI(mainFunc)
    myCLI.SetVersion("1.0.0")
    myCLI.Command.SetDefaultConfigOption()

	// Add `list` command.
	listCommand := cli.NewCommand("list")
	listCommand.Usage = "List the configuration."
	listCommand.Action = listFunc
	listCommand.Options["local"] = &cli.Option{
        Alias: "l",
    }
	myCLI.Command.Commands = append(myCLI.Command.Commands, listCommand)

	// Add `push` command.
	pushCommand := cli.NewCommand("push")
	pushCommand.Usage = "Push the source code."
	myCLI.Command.Commands = append(myCLI.Command.Commands, pushCommand)

	// Add `push > origin` command.
	pushOriginCommand := cli.NewCommand("origin")
	pushOriginCommand.Usage = "Push the source code to the origin."
	pushOriginCommand.Action = pushOringFunc
	pushOriginCommand.Options["url"] = &cli.Option{
		Alias: "u",
		Value: "https://exmaple.com",
	}
	pushCommand.Commands = append(pushCommand.Commands, pushOriginCommand)

    // Run the CLI.
    myCLI.Run()
}


func mainFunc(cmd *cli.Command) {
	for _, o := range cmd.Options {
		fmt.Printf("%v\n", o)
	}
}


func listFunc(cmd *cli.Command) {
	fmt.Printf("Started list func.\n")
    for _, o := range cmd.Options {
        fmt.Printf("%v\n", o)
    }
}


func pushOringFunc(cmd *cli.Command) {
    fmt.Printf("Started push origin func.\n")
    for _, o := range cmd.Options {
        fmt.Printf("%v\n", o)
    }
}
