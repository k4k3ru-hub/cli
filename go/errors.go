package cli

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidParameter indicates invalid API input or configuration.
	ErrInvalidParameter = errors.New("invalid parameter")
	// ErrDuplicateCommand indicates a duplicate sibling command name.
	ErrDuplicateCommand = errors.New("duplicate command")
	// ErrDuplicateOption indicates a duplicate option name or alias.
	ErrDuplicateOption = errors.New("duplicate option")
	// ErrInvalidCommandHierarchy indicates a cycle or multiple parent assignment.
	ErrInvalidCommandHierarchy = errors.New("invalid command hierarchy")
	// ErrReservedOption indicates use of a built-in option name or alias.
	ErrReservedOption = errors.New("reserved option")
	// ErrUnknownCommand indicates an unregistered command.
	ErrUnknownCommand = errors.New("unknown command")
	// ErrUnknownOption indicates an unregistered option.
	ErrUnknownOption = errors.New("unknown option")
	// ErrMissingOptionValue indicates a value-taking option without a value.
	ErrMissingOptionValue = errors.New("missing option value")
	// ErrUnexpectedOptionValue indicates a value assigned to a value-less flag.
	ErrUnexpectedOptionValue = errors.New("unexpected option value")
	// ErrInvalidArgument indicates an unsupported or malformed positional argument.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrInvalidArgumentCount indicates a positional argument count mismatch.
	ErrInvalidArgumentCount = errors.New("invalid argument count")
	// ErrInvalidColumnCount indicates a table row with an unexpected column count.
	ErrInvalidColumnCount = errors.New("invalid column count")
	// ErrInvalidTableCell indicates a cell containing unsupported control characters.
	ErrInvalidTableCell = errors.New("invalid table cell")
	// ErrOutput indicates failure to write CLI output.
	ErrOutput = errors.New("output failure")
)

// CLIError describes a categorized CLI failure.
type CLIError struct {
	Kind                   error
	Operation              string
	Reason                 string
	Cause                  error
	Parameter              string
	State                  string
	CommandName            string
	OptionName             string
	ExistingOptionName     string
	OptionAlias            string
	Argument               string
	ArgumentCount          int
	MinArgumentCount       int
	MaxArgumentCount       int
	RowIndex               int
	ColumnIndex            int
	ColumnCount            int
	ExpectedColumnCount    int
	hasArgumentCount       bool
	hasMinimum             bool
	hasMaximum             bool
	hasRowIndex            bool
	hasColumnIndex         bool
	hasColumnCount         bool
	hasExpectedColumnCount bool
}

// Error formats the categorized error.
//
// Returns:
//   - Formatted error text.
//
// Version:
//   - 2026-08-18: Added table row and column details.
//   - 2026-08-17: Added.
func (err *CLIError) Error() string {
	if err == nil {
		return ""
	}

	var message strings.Builder
	message.WriteString(err.Operation)
	if err.Reason != "" {
		message.WriteString(": " + err.Reason)
	}
	if err.Cause != nil {
		message.WriteString(": " + err.Cause.Error())
	}

	details := make([]string, 0, 13)
	if err.Parameter != "" && err.State != "" {
		details = append(details, fmt.Sprintf("%s=%s", err.Parameter, err.State))
	}
	if err.CommandName != "" {
		details = append(details, fmt.Sprintf("command_name=%q", err.CommandName))
	}
	if err.OptionName != "" {
		details = append(details, fmt.Sprintf("option_name=%q", err.OptionName))
	}
	if err.ExistingOptionName != "" {
		details = append(details, fmt.Sprintf("existing_option_name=%q", err.ExistingOptionName))
	}
	if err.OptionAlias != "" {
		details = append(details, fmt.Sprintf("option_alias=%q", err.OptionAlias))
	}
	if err.Argument != "" {
		details = append(details, fmt.Sprintf("argument=%q", err.Argument))
	}
	if err.hasArgumentCount {
		details = append(details, fmt.Sprintf("argument_count=%d", err.ArgumentCount))
	}
	if err.hasMinimum {
		details = append(details, fmt.Sprintf("min_argument_count=%d", err.MinArgumentCount))
	}
	if err.hasMaximum {
		details = append(details, fmt.Sprintf("max_argument_count=%d", err.MaxArgumentCount))
	}
	if err.hasRowIndex {
		details = append(details, fmt.Sprintf("row_index=%d", err.RowIndex))
	}
	if err.hasColumnIndex {
		details = append(details, fmt.Sprintf("column_index=%d", err.ColumnIndex))
	}
	if err.hasColumnCount {
		details = append(details, fmt.Sprintf("column_count=%d", err.ColumnCount))
	}
	if err.hasExpectedColumnCount {
		details = append(details, fmt.Sprintf("expected_column_count=%d", err.ExpectedColumnCount))
	}
	if len(details) > 0 {
		message.WriteString(": " + strings.Join(details, " "))
	}
	return message.String()
}

// Unwrap returns the category and underlying cause.
//
// Returns:
//   - Inspectable category and cause errors.
//
// Version:
//   - 2026-08-17: Added.
func (err *CLIError) Unwrap() []error {
	if err == nil {
		return nil
	}
	unwrapped := make([]error, 0, 2)
	if err.Kind != nil {
		unwrapped = append(unwrapped, err.Kind)
	}
	if err.Cause != nil {
		unwrapped = append(unwrapped, err.Cause)
	}
	return unwrapped
}

func invalidParameterError(operation string, parameter string, state string) error {
	return &CLIError{
		Kind:      ErrInvalidParameter,
		Operation: operation,
		Parameter: parameter,
		State:     state,
	}
}

func argumentCountError(operation string, reason string, actual int, minimum *int, maximum *int, commandName string) error {
	err := &CLIError{
		Kind:             ErrInvalidArgumentCount,
		Operation:        operation,
		Reason:           reason,
		CommandName:      commandName,
		ArgumentCount:    actual,
		hasArgumentCount: true,
	}
	if minimum != nil {
		err.MinArgumentCount = *minimum
		err.hasMinimum = true
	}
	if maximum != nil {
		err.MaxArgumentCount = *maximum
		err.hasMaximum = true
	}
	return err
}
