// cli.go
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
	action   CommandFunc
	commands []*Command
	name     string
	usage    string
	options  map[string]*Option
}
type CommandFunc func(*Command) error
type Option struct {
	Alias       string
	Value       string
	Description string
	IsFlagSet   bool
}

// New CLI
func NewCLI(defaultFunc CommandFunc) *CLI {
	// Set reserved options.
	options := make(map[string]*Option)
	options[OptHelpName] = &Option{
		Alias:       OptHelpAlias,
		Description: OptHelpDesc,
		IsFlagSet:   false,
	}
	options[OptVersionName] = &Option{
		Alias:       OptVersionAlias,
		Value:       OptVersionValue,
		Description: OptVersionDesc,
		IsFlagSet:   false,
	}

	// Create a root command.
	rootCommand := &Command{
		action:  defaultFunc,
		name:    filepath.Base(os.Args[0]),
		options: options,
	}

	return &CLI{
		Command: rootCommand,
	}
}

// New Command
func NewCommand(name string) *Command {
	return &Command{
		name:    name,
		options: make(map[string]*Option),
	}
}

// Set action.
func (cmd *Command) SetAction(action CommandFunc) {
	if cmd == nil {
		return
	}
	cmd.action = action
}

// Set usage.
func (cmd *Command) SetUsage(usage string) {
	if cmd == nil {
		return
	}
	cmd.usage = usage
}

// Add a subcommand.
func (cmd *Command) AddCommand(command *Command) error {
	if cmd == nil {
		return fmt.Errorf("failed to add command: command receiver=null")
	}
	if command == nil {
		return fmt.Errorf("failed to add command: command=null")
	}
	if strings.TrimSpace(command.name) == "" {
		return fmt.Errorf("failed to add command: command name=empty")
	}

	for _, existing := range cmd.commands {
		if existing != nil && existing.name == command.name {
			return fmt.Errorf("failed to add command: duplicate command name=%q", command.name)
		}
	}

	cmd.commands = append(cmd.commands, command)
	return nil
}

// Add an option.
func (cmd *Command) AddOption(name string, option *Option) error {
	if cmd == nil {
		return fmt.Errorf("failed to add option: command=null")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("failed to add option: option name=empty")
	}
	if option == nil {
		return fmt.Errorf("failed to add option: option=null name=%q", name)
	}
	if _, exists := cmd.options[name]; exists {
		return fmt.Errorf("failed to add option: duplicate option name=%q", name)
	}
	if option.Alias != "" {
		for existingName, existing := range cmd.options {
			if existing != nil && existing.Alias == option.Alias {
				return fmt.Errorf("failed to add option: duplicate option alias=%q names=%q,%q", option.Alias, existingName, name)
			}
		}
	}

	cmd.options[name] = option
	return nil
}

// Run CLI
func (cli *CLI) Run() error {
	args := os.Args[1:]

	// If there is no arguments provided, run the root command.
	if len(args) == 0 {
		if cli.Command.action != nil {
			return cli.Command.action(cli.Command)
		} else {
			cli.Command.ShowUsage()
		}
		return nil
	}

	// Check the help flag.
	if isHelpFlagSet(args) {
		cli.Command.ShowUsage()
		return nil
	}

	// Check the version flag.
	if isVersionFlagSet(args) {
		opt := cli.GetOption(OptVersionName)
		if opt != nil {
			fmt.Printf("Version: %s\n", opt.Value)
		}
		return nil
	}

	// Run command.
	return cli.Command.run(mergeOptions(nil, cli.Command.options), args)
}

// Set default config option.
func (cmd *Command) SetDefaultConfigOption() {
	cmd.options[OptConfigName] = &Option{
		Alias:       OptConfigAlias,
		Description: OptConfigDesc,
		IsFlagSet:   false,
	}
}

func (cmd *Command) GetOption(optionName string) *Option {
	if cmd == nil || optionName == "" {
		return nil
	}
	opt, ok := cmd.options[optionName]
	if opt == nil || !ok {
		return nil
	}
	return opt
}

// Get options without exposing the command's internal option map.
func (cmd *Command) Options() map[string]*Option {
	if cmd == nil {
		return nil
	}
	return mergeOptions(nil, cmd.options)
}

// Set version option
func (cli *CLI) SetVersion(version string) {
	opt := cli.GetOption(OptVersionName)
	if opt != nil {
		opt.Value = version
	}
}

// Get option.
func (cli *CLI) GetOption(name string) *Option {
	cmd := cli.Command
	if cmd == nil {
		fmt.Printf("missing required parameter: cli_command=null\n")
		return nil
	}
	opt, ok := cmd.options[OptVersionName]
	if opt == nil || !ok {
		fmt.Printf("missing required parameter: option_name=%s\n", OptVersionName)
		return nil
	}
	return opt
}

// Show usage of the command.
func (cmd *Command) ShowUsage() {
	// Usage section.
	var usage strings.Builder
	usage.WriteString("Usage: " + cmd.name)
	for optionName, option := range cmd.options {
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
	if len(cmd.commands) > 0 {
		usage.WriteString(" [")
		var commandNames []string
		for _, command := range cmd.commands {
			if command.name == "" {
				continue
			}
			commandNames = append(commandNames, command.name)
		}
		usage.WriteString(strings.Join(commandNames, "|") + "]")
	}

	fmt.Println(usage.String())

	// Options section.
	var desc strings.Builder
	var totalKeyLength int
	optionKeys := make([]string, 0, len(cmd.options))
	for key := range cmd.options {
		optionKeys = append(optionKeys, key)
		if totalKeyLength < len(key)+1 {
			totalKeyLength = len(key) + 1
		}
	}
	keyFormat := fmt.Sprintf("%%-%ds", totalKeyLength)
	sort.Strings(optionKeys)
	desc.WriteString("\n\nOptions:\n")
	for _, key := range optionKeys {
		option := cmd.options[key]
		desc.WriteString("  " + fmt.Sprintf(keyFormat, key+":") + " " + option.Description + "\n")
	}
	fmt.Println(desc.String())
}

// Get option by the argument.
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

// Merge options without modifying the source maps or options.
func mergeOptions(parent map[string]*Option, child map[string]*Option) map[string]*Option {
	merged := make(map[string]*Option, len(parent)+len(child))

	for name, option := range parent {
		merged[name] = cloneOption(option)
	}
	for name, option := range child {
		merged[name] = cloneOption(option)
	}

	return merged
}

// Clone an option.
func cloneOption(option *Option) *Option {
	if option == nil {
		return nil
	}

	cloned := *option
	return &cloned
}

// Check if help flag (--help or -h) is set in os.Args.
func isHelpFlagSet(args []string) bool {
	for _, arg := range args {
		if arg == "--"+OptHelpName || arg == "-"+OptHelpAlias {
			return true
		}
	}
	return false
}

// Check if version flag (--version or -v) is set in os.Args.
func isVersionFlagSet(args []string) bool {
	for _, arg := range args {
		if arg == "--"+OptVersionName || arg == "-"+OptVersionAlias {
			return true
		}
	}
	return false
}

// Run command
func (cmd *Command) run(options map[string]*Option, args []string) error {
	// Check the arguments.
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			foundOption := getOptionByArgument(arg, options)
			if foundOption == nil {
				cmd.ShowUsage()
				return fmt.Errorf("unknown option: %s", arg)
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
			for _, subCommand := range cmd.commands {
				if subCommand.name == arg {
					// Merge inherited and subcommand options.
					migratedOptions := mergeOptions(options, subCommand.options)

					// Run recursively.
					return subCommand.run(migratedOptions, args[i+1:])
				}
			}

			// Unsupported sub command.
			cmd.ShowUsage()
			return fmt.Errorf("unknown sub command: %s", arg)
		}
	}

	// Run the command action.
	if cmd.action != nil {
		executionCommand := *cmd
		executionCommand.options = options
		return executionCommand.action(&executionCommand)
	} else {
		cmd.ShowUsage()
	}
	return nil
}
