package interactive

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// Prompt collects reusable interactive input.
type Prompt struct {
	input      io.Reader
	output     io.Writer
	lineReader *bufio.Reader
}

// NewPrompt creates an interactive input prompt.
//
// Parameters:
//   - input: prompt input
//   - output: prompt output
//
// Returns:
//   - Configured prompt.
//   - An error when a required stream is nil.
//
// Version:
//   - 2026-08-28: Preserved buffered input across repeated prompts.
//   - 2026-08-28: Added.
func NewPrompt(input io.Reader, output io.Writer) (*Prompt, error) {
	if input == nil || isNilPromptStream(input) {
		return nil, fmt.Errorf("failed to create interactive prompt: input=null")
	}
	if output == nil || isNilPromptStream(output) {
		return nil, fmt.Errorf("failed to create interactive prompt: output=null")
	}
	return &Prompt{input: input, output: output}, nil
}

// Ask displays a label and returns one line of visible user input.
//
// Parameters:
//   - ctx: prompt lifetime context
//   - label: label displayed before the input
//
// Returns:
//   - Input without its line ending.
//   - An error when the prompt is canceled or input/output fails.
//
// Version:
//   - 2026-08-28: Styled prompt labels on color-capable terminals.
//   - 2026-08-28: Added.
func (p *Prompt) Ask(ctx context.Context, label string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("failed to ask interactive prompt: prompt=null")
	}
	if ctx == nil {
		return "", fmt.Errorf("failed to ask interactive prompt: context=null")
	}
	if p.input == nil || isNilPromptStream(p.input) {
		return "", fmt.Errorf("failed to ask interactive prompt: input=null")
	}
	if p.output == nil || isNilPromptStream(p.output) {
		return "", fmt.Errorf("failed to ask interactive prompt: output=null")
	}
	if label == "" {
		return "", fmt.Errorf("failed to ask interactive prompt: label=empty")
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("failed to ask interactive prompt: %w", err)
	}
	if _, err := io.WriteString(p.output, styledText(p.output, label+":", ansiCyan)+" "); err != nil {
		return "", fmt.Errorf("failed to ask interactive prompt: failed to write label: %w", err)
	}

	if inputFile, ok := p.input.(*os.File); ok && term.IsTerminal(int(inputFile.Fd())) {
		return p.askTerminal(ctx)
	}
	return p.askLine(ctx)
}

func (p *Prompt) askLine(ctx context.Context) (string, error) {
	if p.lineReader == nil {
		p.lineReader = bufio.NewReader(p.input)
	}
	line, err := p.lineReader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to ask interactive prompt: failed to read input: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("failed to ask interactive prompt: %w", err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func (p *Prompt) askTerminal(ctx context.Context) (string, error) {
	inputBuffer := make([]rune, 0, 64)
	decoder := &keyDecoder{}
	readBuffer := make([]byte, 1)
	for {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("failed to ask interactive prompt: %w", err)
		}
		readCount, err := p.input.Read(readBuffer)
		if errors.Is(err, io.EOF) {
			return string(inputBuffer), nil
		}
		if err != nil {
			return "", fmt.Errorf("failed to ask interactive prompt: failed to read terminal input: %w", err)
		}
		if readCount == 0 {
			continue
		}
		event, complete := decoder.decode(readBuffer[0])
		if !complete {
			continue
		}
		switch event.kind {
		case keyInterrupt:
			if _, err := io.WriteString(p.output, "\r\n"); err != nil {
				return "", fmt.Errorf("failed to ask interactive prompt: failed to write cancellation: %w", err)
			}
			return "", fmt.Errorf("failed to ask interactive prompt: %w", context.Canceled)
		case keyEnter:
			if _, err := io.WriteString(p.output, "\r\n"); err != nil {
				return "", fmt.Errorf("failed to ask interactive prompt: failed to write line ending: %w", err)
			}
			return string(inputBuffer), nil
		case keyBackspace:
			if len(inputBuffer) == 0 {
				continue
			}
			inputBuffer = inputBuffer[:len(inputBuffer)-1]
			if _, err := io.WriteString(p.output, "\b \b"); err != nil {
				return "", fmt.Errorf("failed to ask interactive prompt: failed to erase input: %w", err)
			}
		case keyCharacter:
			if !unicode.IsPrint(event.character) {
				continue
			}
			inputBuffer = append(inputBuffer, event.character)
			if _, err := io.WriteString(p.output, string(event.character)); err != nil {
				return "", fmt.Errorf("failed to ask interactive prompt: failed to write input: %w", err)
			}
		}
	}
}

func isNilPromptStream(stream any) bool {
	value := reflect.ValueOf(stream)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
