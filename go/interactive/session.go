// Package interactive provides reusable interactive terminal sessions.
package interactive

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode"

	cli "github.com/k4k3ru-hub/cli/go"
	"golang.org/x/term"
)

const defaultPrompt = "> "

const aiAgentUnderDevelopmentMessage = "The AI agent is under development. Type / to select a command."

const (
	ansiBoldCyan = "\x1b[1;36m"
	ansiReset    = "\x1b[0m"
)

// Session runs an interactive terminal input loop.
type Session struct {
	application *cli.CLI
	input       io.Reader
	output      io.Writer
	prompt      string
}

type commandCandidate struct {
	command *cli.Command
	path    string
}

// NewSession creates an interactive terminal session.
//
// Parameters:
//   - input: session input
//   - output: session output
//   - application: CLI containing the command tree and actions
//
// Returns:
//   - Configured session.
//   - An error when a required stream is nil.
//
// Version:
//   - 2026-08-28: Required the CLI used for completion and execution.
//   - 2026-08-27: Added.
func NewSession(input io.Reader, output io.Writer, application *cli.CLI) (*Session, error) {
	if input == nil || isNilStream(input) {
		return nil, fmt.Errorf("failed to create interactive session: input=null")
	}
	if output == nil || isNilStream(output) {
		return nil, fmt.Errorf("failed to create interactive session: output=null")
	}
	if application == nil {
		return nil, fmt.Errorf("failed to create interactive session: cli=null")
	}
	if application.Root() == nil {
		return nil, fmt.Errorf("failed to create interactive session: root_command=null")
	}
	if err := application.SetIO(input, output, output); err != nil {
		return nil, fmt.Errorf("failed to create interactive session: %w", err)
	}

	return &Session{
		application: application,
		input:       input,
		output:      output,
		prompt:      defaultPrompt,
	}, nil
}

// Run waits for interactive input until the context is canceled or input closes.
//
// Parameters:
//   - ctx: session lifetime context
//
// Returns:
//   - An error when input or output fails.
//
// Version:
//   - 2026-08-28: Kept command errors inside the interactive session.
//   - 2026-08-27: Added.
func (s *Session) Run(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("failed to run interactive session: session=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to run interactive session: context=null")
	}
	if s.input == nil || isNilStream(s.input) {
		return fmt.Errorf("failed to run interactive session: input=null")
	}
	if s.output == nil || isNilStream(s.output) {
		return fmt.Errorf("failed to run interactive session: output=null")
	}
	if s.application == nil {
		return fmt.Errorf("failed to run interactive session: cli=null")
	}
	if inputFile, ok := s.input.(*os.File); ok && term.IsTerminal(int(inputFile.Fd())) {
		return s.runTerminal(ctx, inputFile)
	}

	lines := make(chan error)
	go s.readLines(ctx, lines)

	for {
		if _, err := io.WriteString(s.output, styledText(s.output, s.prompt, ansiGreen)); err != nil {
			return fmt.Errorf("failed to run interactive session: failed to write prompt: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-lines:
			if !ok {
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to run interactive session: failed to read input: %w", err)
			}
		}
	}
}

func (s *Session) runTerminal(ctx context.Context, input *os.File) error {
	state, err := term.MakeRaw(int(input.Fd()))
	if err != nil {
		return fmt.Errorf("failed to run interactive session: failed to enable raw terminal mode: %w", err)
	}
	defer term.Restore(int(input.Fd()), state)

	inputBuffer := make([]rune, 0, 64)
	selectedCandidate := 0
	if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
		return err
	}
	decoder := &keyDecoder{}
	readBuffer := make([]byte, 1)
	for {
		if ctx.Err() != nil {
			return nil
		}

		readCount, err := input.Read(readBuffer)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to run interactive session: failed to read terminal input: %w", err)
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
			if _, err := io.WriteString(s.output, "\r\n"); err != nil {
				return fmt.Errorf("failed to run interactive session: failed to write terminal exit: %w", err)
			}
			return nil
		case keyEnter:
			input := string(inputBuffer)
			command := s.resolveCommand(input)
			if command != nil && command.HasAction() {
				if err := s.writeSubmittedCommand(inputBuffer); err != nil {
					return err
				}
				if err := s.executeCommand(input); err != nil {
					if ctx.Err() != nil && errors.Is(err, context.Canceled) {
						return nil
					}
					if err := s.writeCommandError(err); err != nil {
						return err
					}
				}
				if ctx.Err() != nil {
					return nil
				}
				inputBuffer = inputBuffer[:0]
				selectedCandidate = 0
				if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
					return err
				}
			} else if completedInput := s.completeInput(input, selectedCandidate); completedInput != input {
				inputBuffer = []rune(completedInput)
				selectedCandidate = 0
				if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
					return err
				}
			} else if command == nil && len(inputBuffer) > 1 && inputBuffer[0] == '/' && len(s.matchingCommands(input)) == 0 {
				if err := s.writeNoMatchingCommands(inputBuffer); err != nil {
					return err
				}
				inputBuffer = inputBuffer[:0]
				selectedCandidate = 0
				if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
					return err
				}
			}
			continue
		case keyTab:
			completedInput := s.completeInput(string(inputBuffer), selectedCandidate)
			if completedInput == string(inputBuffer) {
				continue
			}
			inputBuffer = []rune(completedInput)
			selectedCandidate = 0
			if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
				return err
			}
		case keyArrowUp, keyArrowDown:
			candidateCount := len(s.commandCandidates(string(inputBuffer)))
			if candidateCount == 0 {
				continue
			}
			selectedCandidate = moveCandidateSelection(selectedCandidate, candidateCount, event.kind)
			if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
				return err
			}
		case keyBackspace:
			if len(inputBuffer) == 0 {
				continue
			}
			inputBuffer = inputBuffer[:len(inputBuffer)-1]
			selectedCandidate = 0
			if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
				return err
			}
		case keyCharacter:
			r := event.character
			if !unicode.IsPrint(r) {
				continue
			}
			inputBuffer = append(inputBuffer, r)
			selectedCandidate = 0
			if err := s.renderTerminal(inputBuffer, selectedCandidate); err != nil {
				return err
			}
		}
	}
}

func (s *Session) executeCommand(input string) error {
	command := s.resolveCommand(input)
	if command == nil {
		return fmt.Errorf("failed to execute interactive command: command=invalid")
	}
	if !command.HasAction() {
		return fmt.Errorf("failed to execute interactive command: command_action=null")
	}
	if err := s.application.RunArgs(strings.Fields(strings.TrimPrefix(input, "/"))); err != nil {
		return err
	}
	return nil
}

func (s *Session) writeCommandError(commandErr error) error {
	if commandErr == nil {
		return fmt.Errorf("failed to run interactive session: command_error=null")
	}
	if _, err := io.WriteString(s.output, styledText(s.output, commandErr.Error(), ansiRed)+"\r\n"); err != nil {
		return fmt.Errorf("failed to run interactive session: failed to display command error: %w", err)
	}
	return nil
}

func (s *Session) writeSubmittedCommand(inputBuffer []rune) error {
	output := "\r\x1b[J" + styledText(s.output, s.prompt, ansiGreen) + string(inputBuffer) + "\r\n"
	if _, err := io.WriteString(s.output, output); err != nil {
		return fmt.Errorf("failed to run interactive session: failed to display submitted command: %w", err)
	}
	return nil
}

func moveCandidateSelection(current int, candidateCount int, direction keyKind) int {
	if candidateCount <= 0 {
		return 0
	}
	if direction == keyArrowUp {
		return (current - 1 + candidateCount) % candidateCount
	}
	if direction == keyArrowDown {
		return (current + 1) % candidateCount
	}
	return current
}

func (s *Session) writeNoMatchingCommands(inputBuffer []rune) error {
	output := "\r\x1b[J" + styledText(s.output, s.prompt, ansiGreen) + string(inputBuffer) + "\r\n" + styledText(s.output, "No matching commands.", ansiRed) + "\r\n\r\n"
	if _, err := io.WriteString(s.output, output); err != nil {
		return fmt.Errorf("failed to run interactive session: failed to display unmatched command: %w", err)
	}
	return nil
}

func (s *Session) renderTerminal(inputBuffer []rune, selectedCandidate int) error {
	input := string(inputBuffer)
	candidateContent := s.candidateContent(input, selectedCandidate, true)

	var output strings.Builder
	output.WriteString("\r\x1b[J")
	output.WriteString(styledText(s.output, s.prompt, ansiGreen) + input)

	linesBelowPrompt := 0
	if candidateContent != "" {
		output.WriteString("\r\n\r\n" + strings.ReplaceAll(candidateContent, "\n", "\r\n"))
		linesBelowPrompt = 2 + strings.Count(candidateContent, "\n")
	}
	if linesBelowPrompt > 0 {
		output.WriteString(fmt.Sprintf("\x1b[%dA\r", linesBelowPrompt))
		cursorColumn := len([]rune(s.prompt)) + len(inputBuffer)
		if cursorColumn > 0 {
			output.WriteString(fmt.Sprintf("\x1b[%dC", cursorColumn))
		}
	}

	if _, err := io.WriteString(s.output, output.String()); err != nil {
		return fmt.Errorf("failed to run interactive session: failed to render terminal: %w", err)
	}
	return nil
}

func (s *Session) candidateContent(input string, selectedCandidate int, color bool) string {
	if input == "" {
		return ""
	}
	if input[0] != '/' {
		return aiAgentUnderDevelopmentMessage
	}
	return formatCommandList(s.commandCandidates(input), selectedCandidate, color)
}

func (s *Session) isExactCommand(input string) bool {
	return s.resolveCommand(input) != nil
}

func (s *Session) resolveCommand(input string) *cli.Command {
	if len(input) <= 1 || input[0] != '/' {
		return nil
	}
	command := s.application.Root()
	for _, name := range strings.Fields(strings.TrimPrefix(input, "/")) {
		command = findSubcommand(command, name)
		if command == nil {
			return nil
		}
	}
	if command == s.application.Root() || strings.HasSuffix(input, " ") {
		return nil
	}
	return command
}

func (s *Session) completeInput(input string, selectedCandidate int) string {
	if input == "" || input[0] != '/' {
		return input
	}

	candidates := s.commandCandidates(input)
	if selectedCandidate < 0 || selectedCandidate >= len(candidates) {
		return input
	}
	completedInput := candidates[selectedCandidate].path
	if len(candidates[selectedCandidate].command.Commands()) > 0 {
		completedInput += " "
	}
	return completedInput
}

func (s *Session) commandCandidates(input string) []commandCandidate {
	if input == "" || input[0] != '/' {
		return nil
	}

	value := strings.TrimPrefix(input, "/")
	parts := strings.Fields(value)
	endsWithSpace := strings.HasSuffix(value, " ")
	parent := s.application.Root()
	parentPath := make([]string, 0, len(parts))

	completePartCount := len(parts) - 1
	prefix := ""
	if endsWithSpace {
		completePartCount = len(parts)
	} else if len(parts) > 0 {
		prefix = parts[len(parts)-1]
	}
	for index := 0; index < completePartCount; index++ {
		parent = findSubcommand(parent, parts[index])
		if parent == nil {
			return nil
		}
		parentPath = append(parentPath, parts[index])
	}

	if !endsWithSpace && prefix != "" {
		exact := findSubcommand(parent, prefix)
		if exact != nil && len(exact.Commands()) > 0 {
			parent = exact
			parentPath = append(parentPath, prefix)
			prefix = ""
		}
	}

	candidates := make([]commandCandidate, 0)
	for _, command := range parent.Commands() {
		if command == nil || !strings.HasPrefix(command.Name(), prefix) {
			continue
		}
		pathParts := append(append([]string(nil), parentPath...), command.Name())
		candidates = append(candidates, commandCandidate{
			command: command,
			path:    "/" + strings.Join(pathParts, " "),
		})
	}
	sort.Slice(candidates, func(i int, j int) bool {
		return candidates[i].path < candidates[j].path
	})
	return candidates
}

func (s *Session) matchingCommands(input string) []*cli.Command {
	candidates := s.commandCandidates(input)
	commands := make([]*cli.Command, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.command != nil {
			commands = append(commands, candidate.command)
		}
	}
	return commands
}

func findSubcommand(parent *cli.Command, name string) *cli.Command {
	if parent == nil || name == "" {
		return nil
	}
	for _, command := range parent.Commands() {
		if command != nil && command.Name() == name {
			return command
		}
	}
	return nil
}

func formatCommandList(candidates []commandCandidate, selectedCandidate int, color bool) string {
	nameWidth := 0
	for _, candidate := range candidates {
		if candidate.command == nil {
			continue
		}
		if len(candidate.path) > nameWidth {
			nameWidth = len(candidate.path)
		}
	}
	if nameWidth == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("Commands\n")
	for index, candidate := range candidates {
		if candidate.command == nil || candidate.command.Name() == "" {
			continue
		}
		marker := "  "
		if index == selectedCandidate {
			marker = "❯ "
		}
		if index == selectedCandidate && color {
			output.WriteString(ansiBoldCyan + marker + fmt.Sprintf("%-*s", nameWidth, candidate.path) + ansiReset)
		} else {
			output.WriteString(marker + fmt.Sprintf("%-*s", nameWidth, candidate.path))
		}
		if candidate.command.Usage() != "" {
			output.WriteString("  " + candidate.command.Usage())
		}
		output.WriteString("\n")
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func isNilStream(stream any) bool {
	value := reflect.ValueOf(stream)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (s *Session) readLines(ctx context.Context, lines chan<- error) {
	defer close(lines)

	reader := bufio.NewReader(s.input)
	for {
		_, err := reader.ReadString('\n')
		switch {
		case err == nil:
			select {
			case lines <- nil:
			case <-ctx.Done():
				return
			}
		case errors.Is(err, io.EOF):
			return
		default:
			select {
			case lines <- err:
			case <-ctx.Done():
			}
			return
		}
	}
}
