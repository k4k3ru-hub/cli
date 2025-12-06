//
// cli.go
//
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
)


const (
    OptConfigName  = "config"
    OptConfigAlias = "c"
    OptConfigDesc  = "Specify the configuration file to use. Supported formats: JSON, YAML, TOML."

    OptHelpName  = "help"
    OptHelpAlias = "h"
    OptHelpDesc  = "Display a list of available commands and global options."

    OptVersionName  = "version"
    OptVersionAlias = "v"
    OptVersionDesc  = "Show the version of the CLI tool."
    OptVersionValue = "1.0.0"
)


type CLI struct {
    Command  *Command
    Version  string
    execName string
}
type Command struct {
    Action   CommandFunc
    Commands []*Command
    Name     string
    Usage    string
    Options  map[string]*Option
}
type CommandFunc func(*Command)
type Option struct {
    Alias       string
    Value       string
    Description string
    IsFlagSet   bool
}


//
// New CLI
//
func NewCLI(defaultFunc func(*Command)) *CLI {
    // Set reserved options.
    options := make(map[string]*Option)
    options[OptHelpName] = &Option{
        Alias: OptHelpAlias,
        Description: OptHelpDesc,
        IsFlagSet: false,
    }
    options[OptVersionName] = &Option{
        Alias: OptVersionAlias,
        Value: OptVersionValue,
        Description: OptVersionDesc,
        IsFlagSet: false,
    }

    // Create a root command.
    rootCommand := &Command{
        Action: defaultFunc,
        Name: filepath.Base(os.Args[0]),
        Options: options,
    }

    return &CLI{
        Command: rootCommand,
    }
}


//
// New Command
//
func NewCommand(name string) *Command {
    return &Command{
        Name: name,
        Options: make(map[string]*Option),
    }
}


//
// Run CLI
//
func (cli *CLI) Run() {
    args := os.Args[1:]

    // If there is no arguments provided, run the root command.
    if len(args) == 0 {
        if cli.Command.Action != nil {
            cli.Command.Action(cli.Command)
        } else {
            cli.Command.ShowUsage()
        }
        return
    }

    // Check the help flag.
    if isHelpFlagSet(args) {
        cli.Command.ShowUsage()
        return
    }

    // Check the version flag.
    if isVersionFlagSet(args) {
        opt := cli.GetOption(OptVersionName)
        if opt != nil {
            fmt.Printf("Version: %s\n", opt.Value)
        }
        return
    }

    // Run command.
    cli.Command.run(cli.Command.Options, args)
}


//
// Set default config option.
//
func (cmd *Command) SetDefaultConfigOption() {
    cmd.Options[OptConfigName] = &Option{
        Alias: OptConfigAlias,
        Description: OptConfigDesc,
        IsFlagSet: false,
    }
}


//
// Set version option
//
func (cli *CLI) SetVersion(version string) {
    opt := cli.GetOption(OptVersionName)
    if opt != nil {
        opt.Value = version
    }
}


//
// Get option.
//
func (cli *CLI) GetOption(name string) *Option {
    cmd := cli.Command
    if cmd == nil {
        fmt.Printf("missing required parameter: cli_command=null\n")
        return nil
    }
    opt, ok := cmd.Options[OptVersionName]
    if opt == nil || !ok {
        fmt.Printf("missing required parameter: option_name=%s\n", OptVersionName)
        return nil
    }
    return opt
}


//
// Show usage of the command.
//
func (cmd *Command) ShowUsage() {
    // Usage section.
    var usage strings.Builder
    usage.WriteString("Usage: " + cmd.Name)
    for optionName, option := range cmd.Options {
        if optionName != "" && option.Alias != "" {
            usage.WriteString(" [--" + optionName + "|-" + option.Alias + "]")
        } else if optionName != "" {
            usage.WriteString(" [--" + optionName + "]")
        } else if option.Alias != "" {
            usage.WriteString(" [-" + option.Alias + "]")
        } else {
            continue
        }
    }
    if len(cmd.Commands) > 0 {
        usage.WriteString(" [")
        var commandNames []string
        for _, command := range cmd.Commands {
            if command.Name == "" {
                continue
            }
            commandNames = append(commandNames, command.Name)
        }
        usage.WriteString(strings.Join(commandNames, "|") + "]")
    }

    fmt.Println(usage.String())

    // Options section.
    var desc strings.Builder
    var totalKeyLength int
    optionKeys := make([]string, 0, len(cmd.Options))
    for key := range cmd.Options {
        optionKeys = append(optionKeys, key)
        if totalKeyLength < len(key)+1 {
            totalKeyLength = len(key)+1
        }
    }
    keyFormat := fmt.Sprintf("%%-%ds", totalKeyLength)
    sort.Strings(optionKeys)
    desc.WriteString("\n\nOptions:\n")
    for _, key := range optionKeys {
        option := cmd.Options[key]
        desc.WriteString("  " + fmt.Sprintf(keyFormat, key+":") + " " + option.Description + "\n")
    }
    fmt.Println(desc.String())
}


//
// Get option by the argument.
//
func getOptionByArgument(arg string, options map[string]*Option) *Option {
    if strings.HasPrefix(arg, "--") {
        optionName := strings.SplitN(arg[2:], "=", 2)[0]
        if optionName == "" {
            return nil
        }
        for name, option := range options {
            if name == optionName {
                return option
            }
        }
    } else if strings.HasPrefix(arg, "-") {
        optionName := strings.SplitN(arg[1:], "=", 2)[0]
        if optionName == "" {
            return nil
        }
        for _, option := range options {
            if option.Alias == optionName {
                return option
            }
        }
    }
    return nil
}


//
// Check if help flag (--help or -h) is set in os.Args.
//
func isHelpFlagSet(args []string) bool {
    for _, arg := range args {
        if arg == "--" + OptHelpName || arg == "-" + OptHelpAlias {
            return true
        }
    }
    return false
}


//
// Check if version flag (--version or -v) is set in os.Args.
//
func isVersionFlagSet(args []string) bool {
    for _, arg := range args {
        if arg == "--" + OptVersionName || arg == "-" + OptVersionAlias {
            return true
        }
    }
    return false
}


//
// Run command
//
func (cmd *Command) run(options map[string]*Option, args []string) {
    // Check the arguments.
    for i := 0; i < len(args); i++ {
        arg := args[i]

        if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
            foundOption := getOptionByArgument(arg, cmd.Options)
            if foundOption == nil {
                fmt.Printf("Unknown option: %s\n\n", arg)
                cmd.ShowUsage()
                return
            }

            // Check if the option has a value or not.
            if !foundOption.IsFlagSet {
                // Override to the option value if the arg has `=`.
                if strings.Count(arg, "=") == 1 {
                    parts := strings.SplitN(arg, "=", 2)
                    if len(parts) == 2 {
                        foundOption.Value = parts[1]
                    }
                } else {
                    if i+1 < len(args) {
                        if !strings.HasPrefix(args[i+1], "-") {
                            foundOption.Value = args[i+1]
                        }
                        i++
                    }
                }
            }
        } else {
            // Check if the sub command has been registered or not.
            for _, subCommand := range cmd.Commands {
                if subCommand.Name == arg {
                    // Migrate options.
                    migratedOptions := options
                    for subCommandOptionName, subCommandOption := range subCommand.Options {
                        // Append / Override an option.
                        migratedOptions[subCommandOptionName] = subCommandOption
                    }

                    // Run recursively.
                    subCommand.run(migratedOptions, args[i+1:])
                    return
                }
            }

            // Unsupported sub command.
            fmt.Printf("Unknown sub command: %s\n\n", arg)
            cmd.ShowUsage()
            return
        }
    }

    // Run the command action.
    if cmd.Action != nil {
        cmd.Action(cmd)
    } else {
        cmd.ShowUsage()
    }
}
