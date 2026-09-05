package interactive

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const (
	ansiCyan  = "\x1b[36m"
	ansiGreen = "\x1b[32m"
	ansiRed   = "\x1b[31m"
	ansiDim   = "\x1b[2m"
)

// MessageStyle identifies the presentation of an interactive message.
type MessageStyle uint8

const (
	// MessageStyleDefault uses the terminal's default presentation.
	MessageStyleDefault MessageStyle = iota
	// MessageStyleMuted uses a subdued presentation for progress messages.
	MessageStyleMuted
	// MessageStyleSuccess uses a green presentation for successful operations.
	MessageStyleSuccess
	// MessageStyleError uses a red presentation for errors.
	MessageStyleError
)

// WriteLine writes a styled interactive message followed by a line ending.
//
// Parameters:
//   - output: destination output stream
//   - message: user-facing message
//   - style: requested message presentation
//
// Returns:
//   - An output validation or write error.
//
// Version:
//   - 2026-08-28: Added.
func WriteLine(output io.Writer, message string, style MessageStyle) error {
	if output == nil || isNilPromptStream(output) {
		return fmt.Errorf("failed to write interactive message: output=null")
	}
	if message == "" {
		return fmt.Errorf("failed to write interactive message: message=empty")
	}
	if style > MessageStyleError {
		return fmt.Errorf("failed to write interactive message: style=invalid")
	}
	content := styledText(output, message, messageStyleANSI(style)) + "\r\n"
	if _, err := io.WriteString(output, content); err != nil {
		return fmt.Errorf("failed to write interactive message: %w", err)
	}
	return nil
}

func styledText(output io.Writer, value string, ansi string) string {
	if ansi == "" || !colorEnabled(output) {
		return value
	}
	return ansi + value + ansiReset
}

func messageStyleANSI(style MessageStyle) string {
	switch style {
	case MessageStyleMuted:
		return ansiDim
	case MessageStyleSuccess:
		return ansiGreen
	case MessageStyleError:
		return ansiRed
	default:
		return ""
	}
}

func colorEnabled(output io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	outputFile, ok := output.(*os.File)
	return ok && term.IsTerminal(int(outputFile.Fd()))
}
