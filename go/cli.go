// cli.go
package cli

import (
	"fmt"
	"io"
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

// CLI stores a command tree and its execution streams.
type CLI struct {
	root   *Command
	input  io.Reader
	output io.Writer
	errOut io.Writer
}

// Command defines a command independently from any individual execution.
type Command struct {
	action           CommandFunc
	commands         []*Command
	name             string
	usage            string
	options          map[string]Option
	parent           *Command
	minArguments     int
	maxArguments     int
	argumentRangeSet bool
}

// CommandFunc handles one command execution.
type CommandFunc func(*Context) error

// Option defines a command-line option.
type Option struct {
	Alias        string
	DefaultValue string
	Description  string
	IsFlag       bool
}

// OptionValue contains the value parsed for one execution.
type OptionValue struct {
	Value string
	IsSet bool
}

// Context contains immutable-by-copy state for one command execution.
type Context struct {
	commandPath string
	options     map[string]OptionValue
	arguments   []string
	input       io.Reader
	output      io.Writer
	errOut      io.Writer
}
type optionState struct {
	definition Option
	value      OptionValue
}

// NewCLI creates a CLI with a root command.
//
// Parameters:
//   - defaultFunc: action invoked when no subcommand is selected
//
// Returns:
//   - Configured CLI.
//
// Version:
//   - 2026-08-17: Added validation-safe defaults.
func NewCLI(defaultFunc CommandFunc) *CLI {
	return NewCLIWithName(filepath.Base(os.Args[0]), defaultFunc)
}

// NewCLIWithName creates a CLI with an explicit executable name.
//
// Parameters:
//   - name: executable name displayed in usage output
//   - defaultFunc: action invoked when no subcommand is selected
//
// Returns:
//   - Configured CLI.
//
// Version:
//   - 2026-08-17: Added.
func NewCLIWithName(name string, defaultFunc CommandFunc) *CLI {
	// Set reserved options.
	options := make(map[string]Option)
	options[OptHelpName] = Option{
		Alias:       OptHelpAlias,
		Description: OptHelpDesc,
		IsFlag:      true,
	}
	options[OptVersionName] = Option{
		Alias:        OptVersionAlias,
		DefaultValue: OptVersionValue,
		Description:  OptVersionDesc,
		IsFlag:       true,
	}

	// Create a root command.
	rootCommand := &Command{
		action:  defaultFunc,
		name:    name,
		options: options,
	}

	return &CLI{
		root:   rootCommand,
		input:  os.Stdin,
		output: os.Stdout,
		errOut: os.Stderr,
	}
}

// Root returns the root command definition.
//
// Returns:
//   - Root command.
//
// Version:
//   - 2026-08-17: Added.
func (cli *CLI) Root() *Command {
	if cli == nil {
		return nil
	}
	return cli.root
}

// NewCommand creates a command with the specified name.
//
// Parameters:
//   - name: command name
//
// Returns:
//   - Configured command.
//
// Version:
//   - 2026-08-17: Added.
func NewCommand(name string) *Command {
	return &Command{
		name:    name,
		options: make(map[string]Option),
	}
}

// SetAction sets the command action.
//
// Parameters:
//   - action: function invoked when the command runs
//
// Version:
//   - 2026-08-17: Added nil receiver handling.
func (cmd *Command) SetAction(action CommandFunc) {
	if cmd == nil {
		return
	}
	cmd.action = action
}

// SetUsage sets the command description displayed in help output.
//
// Parameters:
//   - usage: command description
//
// Version:
//   - 2026-08-17: Displayed the description in help output.
func (cmd *Command) SetUsage(usage string) {
	if cmd == nil {
		return
	}
	cmd.usage = usage
}

// Name returns the command name.
//
// Returns:
//   - Command name.
//
// Version:
//   - 2026-08-27: Added.
func (cmd *Command) Name() string {
	if cmd == nil {
		return ""
	}
	return cmd.name
}

// Usage returns the command description.
//
// Returns:
//   - Command description.
//
// Version:
//   - 2026-08-27: Added.
func (cmd *Command) Usage() string {
	if cmd == nil {
		return ""
	}
	return cmd.usage
}

// Commands returns a copy of the direct subcommand list.
//
// Returns:
//   - Direct subcommands in registration order.
//
// Version:
//   - 2026-08-27: Added.
func (cmd *Command) Commands() []*Command {
	if cmd == nil {
		return nil
	}
	return append([]*Command(nil), cmd.commands...)
}

// HasAction reports whether the command has an executable action.
//
// Returns:
//   - True when an action is registered.
//
// Version:
//   - 2026-08-28: Added.
func (cmd *Command) HasAction() bool {
	return cmd != nil && cmd.action != nil
}

// AddCommand adds a subcommand.
//
// Parameters:
//   - command: subcommand to add
//
// Returns:
//   - An error when the receiver or command is invalid or duplicated.
//
// Version:
//   - 2026-08-17: Added validation.
func (cmd *Command) AddCommand(command *Command) error {
	if cmd == nil {
		return invalidParameterError("failed to add command", "command", "null")
	}
	if command == nil {
		return invalidParameterError("failed to add command", "command", "null")
	}
	if strings.TrimSpace(command.name) == "" {
		return invalidParameterError("failed to add command", "command_name", "empty")
	}
	if command == cmd || command.containsCommand(cmd) {
		return &CLIError{
			Kind:        ErrInvalidCommandHierarchy,
			Operation:   "failed to add command",
			Reason:      "command hierarchy contains a cycle",
			CommandName: command.name,
		}
	}
	if command.parent != nil {
		return &CLIError{
			Kind:        ErrInvalidCommandHierarchy,
			Operation:   "failed to add command",
			Reason:      "command already has a parent",
			CommandName: command.name,
		}
	}

	for _, existing := range cmd.commands {
		if existing != nil && existing.name == command.name {
			return &CLIError{
				Kind:        ErrDuplicateCommand,
				Operation:   "failed to add command",
				Reason:      "duplicate command",
				CommandName: command.name,
			}
		}
	}
	if err := validateOptionTree(cmd.effectiveOptions(), command); err != nil {
		return fmt.Errorf("failed to add command: %w: command_name=%q", err, command.name)
	}

	command.parent = cmd
	cmd.commands = append(cmd.commands, command)
	return nil
}

// AddOption adds an option to the command.
//
// Parameters:
//   - name: option name
//   - option: option definition
//
// Returns:
//   - An error when the receiver, option, name, or alias is invalid or duplicated.
//
// Version:
//   - 2026-08-17: Added validation.
func (cmd *Command) AddOption(name string, option Option) error {
	if cmd == nil {
		return invalidParameterError("failed to add option", "command", "null")
	}
	if strings.TrimSpace(name) == "" {
		return invalidParameterError("failed to add option", "option_name", "empty")
	}
	if isReservedOption(name, option.Alias) {
		return &CLIError{
			Kind:        ErrReservedOption,
			Operation:   "failed to add option",
			Reason:      "reserved option",
			OptionName:  name,
			OptionAlias: option.Alias,
		}
	}
	if cmd.options == nil {
		cmd.options = make(map[string]Option)
	}
	if _, exists := cmd.options[name]; exists {
		return &CLIError{
			Kind:       ErrDuplicateOption,
			Operation:  "failed to add option",
			Reason:     "duplicate option",
			OptionName: name,
		}
	}
	if option.Alias != "" {
		for existingName, existing := range cmd.options {
			if existing.Alias == option.Alias {
				return &CLIError{
					Kind:               ErrDuplicateOption,
					Operation:          "failed to add option",
					Reason:             "duplicate option alias",
					OptionName:         name,
					ExistingOptionName: existingName,
					OptionAlias:        option.Alias,
				}
			}
		}
	}
	if err := validateOptionAgainstAncestors(cmd, name, option); err != nil {
		return fmt.Errorf("failed to add option: %w", err)
	}
	if err := validateOptionAgainstDescendants(cmd, name, option); err != nil {
		return fmt.Errorf("failed to add option: %w", err)
	}

	cmd.options[name] = option
	return nil
}

// Run parses process arguments and executes the selected command.
//
// Returns:
//   - An error when the CLI is invalid, parsing fails, or the action fails.
//
// Version:
//   - 2026-08-17: Added receiver validation and strict option parsing.
func (cli *CLI) Run() error {
	return cli.RunArgs(os.Args[1:])
}

// RunArgs parses the supplied arguments and executes the selected command.
//
// Parameters:
//   - args: arguments excluding the executable name
//
// Returns:
//   - An error when the CLI is invalid, parsing fails, output fails, or the action fails.
//
// Version:
//   - 2026-08-17: Added.
func (cli *CLI) RunArgs(args []string) error {
	if cli == nil {
		return invalidParameterError("failed to run cli", "cli", "null")
	}
	if cli.root == nil {
		return invalidParameterError("failed to run cli", "command", "null")
	}
	if cli.input == nil {
		return invalidParameterError("failed to run cli", "input", "null")
	}
	if cli.output == nil {
		return invalidParameterError("failed to run cli", "output", "null")
	}
	if cli.errOut == nil {
		return invalidParameterError("failed to run cli", "error_output", "null")
	}

	// Run command.
	options := mergeOptions(nil, cli.root.options)
	return cli.root.run(
		options,
		append([]string(nil), args...),
		cli.input,
		cli.output,
		cli.errOut,
	)
}

// SetIO sets the input, output, and error-output streams used during execution.
//
// Parameters:
//   - input: command input
//   - output: normal command output
//   - errorOutput: diagnostic command output
//
// Returns:
//   - An error when a stream is nil.
//
// Version:
//   - 2026-08-17: Added.
func (cli *CLI) SetIO(input io.Reader, output io.Writer, errorOutput io.Writer) error {
	if cli == nil {
		return invalidParameterError("failed to set cli io", "cli", "null")
	}
	if input == nil {
		return invalidParameterError("failed to set cli io", "input", "null")
	}
	if output == nil {
		return invalidParameterError("failed to set cli io", "output", "null")
	}
	if errorOutput == nil {
		return invalidParameterError("failed to set cli io", "error_output", "null")
	}

	cli.input = input
	cli.output = output
	cli.errOut = errorOutput
	return nil
}

// AddDefaultConfigOption adds the default configuration-file option.
//
// Returns:
//   - An error when the command is invalid or the option conflicts.
//
// Version:
//   - 2026-08-17: Added.
func (cmd *Command) AddDefaultConfigOption() error {
	return cmd.AddOption(OptConfigName, Option{
		Alias:       OptConfigAlias,
		Description: OptConfigDesc,
	})
}

// Option returns an option definition registered on the command.
//
// Parameters:
//   - optionName: option name
//
// Returns:
//   - Copied option definition and whether it exists.
//
// Version:
//   - 2026-08-17: Separated definitions from execution values.
func (cmd *Command) Option(optionName string) (Option, bool) {
	if cmd == nil || optionName == "" {
		return Option{}, false
	}
	opt, ok := cmd.options[optionName]
	return opt, ok
}

// Options returns copies of the command options.
//
// Returns:
//   - Copied options keyed by name.
//
// Version:
//   - 2026-08-17: Added nil receiver handling.
func (cmd *Command) Options() map[string]Option {
	if cmd == nil {
		return nil
	}
	return cloneOptions(cmd.options)
}

// Arguments returns copies of the positional arguments for the execution.
//
// Returns:
//   - Positional arguments in input order.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) Arguments() []string {
	if ctx == nil {
		return nil
	}
	return append([]string(nil), ctx.arguments...)
}

// Input returns the input stream for the current execution.
//
// Returns:
//   - Configured input stream, or nil outside an execution.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) Input() io.Reader {
	if ctx == nil {
		return nil
	}
	return ctx.input
}

// Output returns the normal output stream for the current execution.
//
// Returns:
//   - Configured output stream, or nil outside an execution.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) Output() io.Writer {
	if ctx == nil {
		return nil
	}
	return ctx.output
}

// ErrorOutput returns the diagnostic output stream for the current execution.
//
// Returns:
//   - Configured error-output stream, or nil outside an execution.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) ErrorOutput() io.Writer {
	if ctx == nil {
		return nil
	}
	return ctx.errOut
}

// Option returns a parsed option value for the execution.
//
// Parameters:
//   - name: option name
//
// Returns:
//   - Copied option value and whether it exists.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) Option(name string) (OptionValue, bool) {
	if ctx == nil || name == "" {
		return OptionValue{}, false
	}
	value, ok := ctx.options[name]
	return value, ok
}

// Options returns copies of all parsed option values for the execution.
//
// Returns:
//   - Option values keyed by name.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) Options() map[string]OptionValue {
	if ctx == nil {
		return nil
	}
	options := make(map[string]OptionValue, len(ctx.options))
	for name, value := range ctx.options {
		options[name] = value
	}
	return options
}

// CommandPath returns the selected command path.
//
// Returns:
//   - Space-separated command path.
//
// Version:
//   - 2026-08-17: Added.
func (ctx *Context) CommandPath() string {
	if ctx == nil {
		return ""
	}
	return ctx.commandPath
}

// SetArgumentCount sets the permitted positional argument count.
//
// Parameters:
//   - minimum: minimum number of arguments
//   - maximum: maximum number of arguments, or -1 for no limit
//
// Returns:
//   - An error when the range is invalid.
//
// Version:
//   - 2026-08-17: Added.
func (cmd *Command) SetArgumentCount(minimum int, maximum int) error {
	if cmd == nil {
		return invalidParameterError("failed to set argument count", "command", "null")
	}
	if minimum < 0 {
		return &CLIError{
			Kind:             ErrInvalidParameter,
			Operation:        "failed to set argument count",
			Reason:           "minimum is out of range",
			ArgumentCount:    minimum,
			MinArgumentCount: 0,
			hasArgumentCount: true,
			hasMinimum:       true,
		}
	}
	if maximum < -1 {
		return &CLIError{
			Kind:             ErrInvalidParameter,
			Operation:        "failed to set argument count",
			Reason:           "maximum is out of range",
			ArgumentCount:    maximum,
			MinArgumentCount: -1,
			hasArgumentCount: true,
			hasMinimum:       true,
		}
	}
	if maximum >= 0 && maximum < minimum {
		return &CLIError{
			Kind:             ErrInvalidParameter,
			Operation:        "failed to set argument count",
			Reason:           "maximum is less than minimum",
			MinArgumentCount: minimum,
			MaxArgumentCount: maximum,
			hasMinimum:       true,
			hasMaximum:       true,
		}
	}

	cmd.minArguments = minimum
	cmd.maxArguments = maximum
	cmd.argumentRangeSet = true
	return nil
}

// SetVersion sets the version displayed by the version flag.
//
// Parameters:
//   - version: version string
//
// Version:
//   - 2026-08-17: Stored the version in the option definition.
func (cli *CLI) SetVersion(version string) {
	if cli == nil || cli.root == nil {
		return
	}
	option, ok := cli.root.options[OptVersionName]
	if ok {
		option.DefaultValue = version
		cli.root.options[OptVersionName] = option
	}
}

// Option returns an option definition registered on the root command.
//
// Parameters:
//   - name: option name
//
// Returns:
//   - Copied option definition and whether it exists.
//
// Version:
//   - 2026-08-17: Separated definitions from execution values.
func (cli *CLI) Option(name string) (Option, bool) {
	if cli == nil || cli.root == nil || name == "" {
		return Option{}, false
	}
	return cli.root.Option(name)
}

// ShowUsage writes command usage to standard output.
//
// Returns:
//   - An error when the command is nil or output fails.
//
// Version:
//   - 2026-08-17: Returned output errors.
func (cmd *Command) ShowUsage() error {
	return cmd.writeUsage(os.Stdout)
}

func (cmd *Command) writeUsage(writer io.Writer) error {
	if cmd == nil {
		return invalidParameterError("failed to show command usage", "command", "null")
	}
	if writer == nil {
		return invalidParameterError("failed to show command usage", "output", "null")
	}

	// Usage section.
	var usage strings.Builder
	usage.WriteString("Usage: " + cmd.commandPath())
	optionNames := make([]string, 0, len(cmd.options))
	for optionName := range cmd.options {
		optionNames = append(optionNames, optionName)
	}
	sort.Strings(optionNames)
	for _, optionName := range optionNames {
		option := cmd.options[optionName]
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
			if command == nil || command.name == "" {
				continue
			}
			commandNames = append(commandNames, command.name)
		}
		sort.Strings(commandNames)
		usage.WriteString(strings.Join(commandNames, "|") + "]")
	}
	if cmd.argumentRangeSet || cmd.action != nil {
		usage.WriteString(" [arguments...]")
	}

	var output strings.Builder
	output.WriteString(usage.String() + "\n")
	if cmd.usage != "" {
		output.WriteString("\n" + cmd.usage + "\n")
	}

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
	output.WriteString(desc.String() + "\n")
	if _, err := io.WriteString(writer, output.String()); err != nil {
		return &CLIError{Kind: ErrOutput, Operation: "failed to show command usage", Cause: err}
	}
	return nil
}

// Get option by the argument.
func getOptionByArgument(arg string, options map[string]*optionState) (string, *optionState) {
	if strings.HasPrefix(arg, "--") {
		optionName := strings.SplitN(arg[2:], "=", 2)[0]
		if optionName == "" {
			return "", nil
		}
		for name, option := range options {
			if name == optionName {
				return name, option
			}
		}
	} else if strings.HasPrefix(arg, "-") {
		optionName := strings.SplitN(arg[1:], "=", 2)[0]
		if optionName == "" {
			return "", nil
		}
		for name, option := range options {
			if option != nil && option.definition.Alias == optionName {
				return name, option
			}
		}
	}
	return "", nil
}

// Merge options without modifying the source maps or options.
func mergeOptions(parent map[string]*optionState, child map[string]Option) map[string]*optionState {
	merged := make(map[string]*optionState, len(parent)+len(child))

	for name, option := range parent {
		if option == nil {
			continue
		}
		cloned := *option
		merged[name] = &cloned
	}
	for name, definition := range child {
		merged[name] = &optionState{
			definition: definition,
			value: OptionValue{
				Value: definition.DefaultValue,
			},
		}
	}

	return merged
}

func cloneOptions(options map[string]Option) map[string]Option {
	cloned := make(map[string]Option, len(options))
	for name, option := range options {
		cloned[name] = option
	}
	return cloned
}

func mergeOptionDefinitions(parent map[string]Option, child map[string]Option) map[string]Option {
	merged := cloneOptions(parent)
	for name, option := range child {
		merged[name] = option
	}
	return merged
}

func (cmd *Command) commandPath() string {
	if cmd == nil {
		return ""
	}

	names := []string{cmd.name}
	for parent := cmd.parent; parent != nil; parent = parent.parent {
		names = append(names, parent.name)
	}
	for left, right := 0, len(names)-1; left < right; left, right = left+1, right-1 {
		names[left], names[right] = names[right], names[left]
	}
	return strings.Join(names, " ")
}

func (cmd *Command) containsCommand(target *Command) bool {
	if cmd == nil || target == nil {
		return false
	}
	if cmd == target {
		return true
	}
	for _, child := range cmd.commands {
		if child != nil && child.containsCommand(target) {
			return true
		}
	}
	return false
}

func (cmd *Command) effectiveOptions() map[string]Option {
	if cmd == nil {
		return nil
	}

	var hierarchy []*Command
	for current := cmd; current != nil; current = current.parent {
		hierarchy = append(hierarchy, current)
	}
	options := make(map[string]Option)
	for index := len(hierarchy) - 1; index >= 0; index-- {
		options = mergeOptionDefinitions(options, hierarchy[index].options)
	}
	return options
}

func isReservedOption(name string, alias string) bool {
	return name == OptHelpName || name == OptVersionName || alias == OptHelpAlias || alias == OptVersionAlias
}

func validateOptionConflict(existingName string, existing Option, name string, option Option) error {
	if existingName == name {
		return nil
	}
	if option.Alias != "" && existing.Alias == option.Alias {
		return &CLIError{
			Kind:               ErrDuplicateOption,
			Operation:          "failed to validate option hierarchy",
			Reason:             "duplicate option alias",
			OptionName:         name,
			ExistingOptionName: existingName,
			OptionAlias:        option.Alias,
		}
	}
	return nil
}

func validateOptionAgainstAncestors(cmd *Command, name string, option Option) error {
	for parent := cmd.parent; parent != nil; parent = parent.parent {
		for existingName, existing := range parent.options {
			if err := validateOptionConflict(existingName, existing, name, option); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateOptionAgainstDescendants(cmd *Command, name string, option Option) error {
	for _, child := range cmd.commands {
		if child == nil {
			continue
		}
		for existingName, existing := range child.options {
			if err := validateOptionConflict(existingName, existing, name, option); err != nil {
				return err
			}
		}
		if err := validateOptionAgainstDescendants(child, name, option); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionTree(inherited map[string]Option, command *Command) error {
	options := cloneOptions(inherited)
	for name, option := range command.options {
		for existingName, existing := range options {
			if err := validateOptionConflict(existingName, existing, name, option); err != nil {
				return err
			}
		}
		options[name] = option
	}
	for _, child := range command.commands {
		if child == nil {
			continue
		}
		if err := validateOptionTree(options, child); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Command) acceptsArguments() bool {
	return cmd != nil && (cmd.action != nil || cmd.argumentRangeSet)
}

func (cmd *Command) validateArgumentCount(arguments []string) error {
	if cmd == nil || !cmd.argumentRangeSet {
		return nil
	}
	if len(arguments) < cmd.minArguments {
		return argumentCountError(
			"failed to validate command arguments",
			"too few positional arguments",
			len(arguments),
			&cmd.minArguments,
			nil,
			cmd.commandPath(),
		)
	}
	if cmd.maxArguments >= 0 && len(arguments) > cmd.maxArguments {
		return argumentCountError(
			"failed to validate command arguments",
			"too many positional arguments",
			len(arguments),
			nil,
			&cmd.maxArguments,
			cmd.commandPath(),
		)
	}
	return nil
}

// Run command
func (cmd *Command) run(
	options map[string]*optionState,
	args []string,
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
) error {
	if cmd == nil {
		return invalidParameterError("failed to run command", "command", "null")
	}

	var positionalArguments []string
	optionsEnded := false

	// Check the arguments.
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !optionsEnded && arg == "--" {
			optionsEnded = true
			continue
		}

		if !optionsEnded && (strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-")) {
			optionName, foundOption := getOptionByArgument(arg, options)
			if foundOption == nil {
				return &CLIError{
					Kind:      ErrUnknownOption,
					Operation: "failed to parse command arguments",
					Reason:    "unknown option",
					Argument:  arg,
				}
			}

			if optionName == OptHelpName {
				return showCommandUsage(cmd, options, output)
			}
			if optionName == OptVersionName {
				if _, err := fmt.Fprintf(output, "Version: %s\n", foundOption.value.Value); err != nil {
					return &CLIError{Kind: ErrOutput, Operation: "failed to show cli version", Cause: err}
				}
				return nil
			}

			// Check if the option has a value or not.
			parts := strings.SplitN(arg, "=", 2)
			if foundOption.definition.IsFlag {
				if len(parts) == 2 {
					return &CLIError{
						Kind:       ErrUnexpectedOptionValue,
						Operation:  "failed to parse command arguments",
						Reason:     "flag does not accept a value",
						OptionName: optionName,
					}
				}
				foundOption.value.IsSet = true
				continue
			}

			if len(parts) == 2 {
				if parts[1] == "" {
					return &CLIError{
						Kind:       ErrMissingOptionValue,
						Operation:  "failed to parse command arguments",
						Reason:     "missing option value",
						OptionName: optionName,
						Parameter:  "value",
						State:      "empty",
					}
				}
				foundOption.value.Value = parts[1]
				foundOption.value.IsSet = true
				continue
			}
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return &CLIError{
					Kind:       ErrMissingOptionValue,
					Operation:  "failed to parse command arguments",
					Reason:     "missing option value",
					OptionName: optionName,
				}
			}
			i++
			foundOption.value.Value = args[i]
			foundOption.value.IsSet = true
		} else {
			if !optionsEnded && len(positionalArguments) == 0 {
				for _, subCommand := range cmd.commands {
					if subCommand == nil || subCommand.name != arg {
						continue
					}
					// Merge inherited and subcommand options.
					migratedOptions := mergeOptions(options, subCommand.options)

					// Run recursively.
					return subCommand.run(migratedOptions, args[i+1:], input, output, errorOutput)
				}
			}
			if !cmd.acceptsArguments() {
				if optionsEnded {
					return &CLIError{
						Kind:        ErrInvalidArgument,
						Operation:   "failed to parse command arguments",
						Reason:      "positional arguments are not supported",
						Argument:    arg,
						CommandName: cmd.commandPath(),
					}
				}
				return &CLIError{
					Kind:        ErrUnknownCommand,
					Operation:   "failed to parse command arguments",
					Reason:      "unknown subcommand",
					CommandName: arg,
				}
			}
			positionalArguments = append(positionalArguments, arg)
		}
	}
	if err := cmd.validateArgumentCount(positionalArguments); err != nil {
		return err
	}

	// Run the command action.
	if cmd.action != nil {
		return cmd.action(&Context{
			commandPath: cmd.commandPath(),
			options:     optionValues(options),
			arguments:   append([]string(nil), positionalArguments...),
			input:       input,
			output:      output,
			errOut:      errorOutput,
		})
	} else {
		return showCommandUsage(cmd, options, output)
	}
}

func showCommandUsage(cmd *Command, options map[string]*optionState, output io.Writer) error {
	if cmd == nil {
		return invalidParameterError("failed to show command usage", "command", "null")
	}
	executionCommand := *cmd
	executionCommand.options = optionDefinitions(options)
	return executionCommand.writeUsage(output)
}

func optionValues(options map[string]*optionState) map[string]OptionValue {
	values := make(map[string]OptionValue, len(options))
	for name, option := range options {
		if option != nil {
			values[name] = option.value
		}
	}
	return values
}

func optionDefinitions(options map[string]*optionState) map[string]Option {
	definitions := make(map[string]Option, len(options))
	for name, option := range options {
		if option != nil {
			definitions[name] = option.definition
		}
	}
	return definitions
}
