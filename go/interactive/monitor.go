package interactive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Monitor displays asynchronously updated lines until Escape is pressed.
type Monitor struct {
	input  io.Reader
	output io.Writer
}

// MonitorLine contains one styled line in a monitor frame.
type MonitorLine struct {
	Text  string
	Style MessageStyle
}

// NewMonitor creates an interactive terminal monitor.
//
// Parameters:
//   - input: control-key input.
//   - output: monitor output.
//
// Returns:
//   - Configured monitor.
//   - Stream validation error.
//
// Version:
//   - 2026-08-29: Added.
func NewMonitor(input io.Reader, output io.Writer) (*Monitor, error) {
	if input == nil || isNilPromptStream(input) {
		return nil, fmt.Errorf("failed to create interactive monitor: input=null")
	}
	if output == nil || isNilPromptStream(output) {
		return nil, fmt.Errorf("failed to create interactive monitor: output=null")
	}
	return &Monitor{input: input, output: output}, nil
}

// Run displays monitor frames until Escape, Ctrl+C, cancellation, or input closure.
//
// Parameters:
//   - ctx: monitor lifetime context.
//   - frames: complete styled display frames; the latest frame replaces the previous terminal frame.
//
// Returns:
//   - Context, input, or output error. Escape and input closure return nil.
//
// Version:
//   - 2026-08-29: Added per-line message styles.
//   - 2026-08-29: Added.
func (m *Monitor) Run(ctx context.Context, frames <-chan []MonitorLine) error {
	if m == nil {
		return fmt.Errorf("failed to run interactive monitor: monitor=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to run interactive monitor: context=null")
	}
	if frames == nil {
		return fmt.Errorf("failed to run interactive monitor: frames=null")
	}
	keys := make(chan monitorKeyResult, 1)
	go readMonitorKey(m.input, keys)
	terminalOutput := false
	if outputFile, ok := m.output.(*os.File); ok {
		terminalOutput = term.IsTerminal(int(outputFile.Fd()))
	}
	previousLineCount := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to run interactive monitor: %w", ctx.Err())
		case result := <-keys:
			if result.err != nil && !errors.Is(result.err, io.EOF) {
				return fmt.Errorf("failed to run interactive monitor: failed to read input: %w", result.err)
			}
			switch result.value {
			case 3, 4:
				return fmt.Errorf("failed to run interactive monitor: %w", context.Canceled)
			default:
				return nil
			}
		case frame, ok := <-frames:
			if !ok {
				return nil
			}
			if err := renderMonitorFrame(m.output, frame, previousLineCount, terminalOutput); err != nil {
				return fmt.Errorf("failed to run interactive monitor: %w", err)
			}
			previousLineCount = len(frame)
		}
	}
}

type monitorKeyResult struct {
	value byte
	err   error
}

func readMonitorKey(input io.Reader, results chan<- monitorKeyResult) {
	buffer := make([]byte, 1)
	for {
		count, err := input.Read(buffer)
		if err != nil || count == 0 {
			results <- monitorKeyResult{err: err}
			return
		}
		if buffer[0] == 27 || buffer[0] == 3 || buffer[0] == 4 {
			results <- monitorKeyResult{value: buffer[0]}
			return
		}
	}
}

func renderMonitorFrame(output io.Writer, lines []MonitorLine, previousLineCount int, terminal bool) error {
	if terminal && previousLineCount > 0 {
		if _, err := fmt.Fprintf(output, "\r\x1b[%dA", previousLineCount); err != nil {
			return fmt.Errorf("failed to position monitor frame: %w", err)
		}
	}
	for _, line := range lines {
		if line.Style > MessageStyleError {
			return fmt.Errorf("failed to render monitor frame: style=invalid")
		}
		if terminal {
			if _, err := io.WriteString(output, "\x1b[2K\r"); err != nil {
				return fmt.Errorf("failed to clear monitor line: %w", err)
			}
		}
		if _, err := io.WriteString(output, styledText(output, line.Text, messageStyleANSI(line.Style))+"\r\n"); err != nil {
			return fmt.Errorf("failed to write monitor line: %w", err)
		}
	}
	staleLineCount := previousLineCount - len(lines)
	for index := 0; terminal && index < staleLineCount; index++ {
		if _, err := io.WriteString(output, "\x1b[2K\r\r\n"); err != nil {
			return fmt.Errorf("failed to clear stale monitor line: %w", err)
		}
	}
	if terminal && staleLineCount > 0 {
		if _, err := fmt.Fprintf(output, "\x1b[%dA", staleLineCount); err != nil {
			return fmt.Errorf("failed to restore monitor frame position: %w", err)
		}
	}
	return nil
}
