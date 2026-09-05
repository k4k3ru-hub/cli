package interactive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"
)

const defaultSelectVisibleLimit = 8

// SelectOption describes one interactive selection candidate.
type SelectOption struct {
	Value       string
	Label       string
	Description string
}

// SelectConfig controls interactive selection rendering.
type SelectConfig struct {
	VisibleLimit int
}

// Select displays filterable options and returns the selected value.
//
// Parameters:
//   - ctx: selection lifetime context
//   - label: label displayed above the candidates
//   - options: selectable values and presentation text
//   - config: optional rendering configuration
//
// Returns:
//   - Selected option value.
//   - An error when selection is canceled, invalid, or input/output fails.
//
// Version:
//   - 2026-08-29: Added.
func (p *Prompt) Select(ctx context.Context, label string, options []SelectOption, config SelectConfig) (string, error) {
	if p == nil {
		return "", fmt.Errorf("failed to select interactive option: prompt=null")
	}
	if ctx == nil {
		return "", fmt.Errorf("failed to select interactive option: context=null")
	}
	if p.input == nil || isNilPromptStream(p.input) {
		return "", fmt.Errorf("failed to select interactive option: input=null")
	}
	if p.output == nil || isNilPromptStream(p.output) {
		return "", fmt.Errorf("failed to select interactive option: output=null")
	}
	if label == "" {
		return "", fmt.Errorf("failed to select interactive option: label=empty")
	}
	if len(options) == 0 {
		return "", fmt.Errorf("failed to select interactive option: options=empty")
	}
	for _, option := range options {
		if option.Value == "" {
			return "", fmt.Errorf("failed to select interactive option: option_value=empty")
		}
		if option.Label == "" {
			return "", fmt.Errorf("failed to select interactive option: option_label=empty")
		}
	}
	if config.VisibleLimit < 0 {
		return "", fmt.Errorf("failed to select interactive option: visible_limit=out_of_range min_value=0")
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("failed to select interactive option: %w", err)
	}
	visibleLimit := config.VisibleLimit
	if visibleLimit == 0 {
		visibleLimit = defaultSelectVisibleLimit
	}
	if inputFile, ok := p.input.(*os.File); ok && term.IsTerminal(int(inputFile.Fd())) {
		return p.selectTerminal(ctx, label, options, visibleLimit)
	}
	return p.selectLine(ctx, label, options)
}

func (p *Prompt) selectLine(ctx context.Context, label string, options []SelectOption) (string, error) {
	if _, err := io.WriteString(p.output, label+":\n"); err != nil {
		return "", fmt.Errorf("failed to select interactive option: failed to write label: %w", err)
	}
	for _, option := range options {
		line := "- " + option.Label
		if option.Description != "" {
			line += "  " + option.Description
		}
		if _, err := io.WriteString(p.output, line+"\n"); err != nil {
			return "", fmt.Errorf("failed to select interactive option: failed to write option: %w", err)
		}
	}
	if _, err := io.WriteString(p.output, "Selection: "); err != nil {
		return "", fmt.Errorf("failed to select interactive option: failed to write input label: %w", err)
	}
	value, err := p.askLine(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to select interactive option: %w", err)
	}
	if value == "" {
		return options[0].Value, nil
	}
	for _, option := range options {
		if value == option.Value || value == option.Label {
			return option.Value, nil
		}
	}
	return "", fmt.Errorf("failed to select interactive option: selection=invalid")
}

func (p *Prompt) selectTerminal(ctx context.Context, label string, options []SelectOption, visibleLimit int) (string, error) {
	filter := make([]rune, 0, 64)
	selected := 0
	decoder := &keyDecoder{}
	readBuffer := make([]byte, 1)
	if err := renderSelection(p.output, label, filter, options, selected, visibleLimit); err != nil {
		return "", err
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("failed to select interactive option: %w", err)
		}
		readCount, err := p.input.Read(readBuffer)
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("failed to select interactive option: %w", io.EOF)
		}
		if err != nil {
			return "", fmt.Errorf("failed to select interactive option: failed to read terminal input: %w", err)
		}
		if readCount == 0 {
			continue
		}
		event, complete := decoder.decode(readBuffer[0])
		if !complete {
			continue
		}
		filtered := filterSelectOptions(options, string(filter))
		switch event.kind {
		case keyInterrupt:
			if _, err := io.WriteString(p.output, "\r\x1b[J\r\n"); err != nil {
				return "", fmt.Errorf("failed to select interactive option: failed to write cancellation: %w", err)
			}
			return "", fmt.Errorf("failed to select interactive option: %w", context.Canceled)
		case keyEnter:
			if len(filtered) == 0 {
				continue
			}
			if selected >= len(filtered) {
				selected = 0
			}
			option := filtered[selected]
			if _, err := io.WriteString(p.output, "\r\x1b[J"+styledText(p.output, label+":", ansiCyan)+" "+option.Label+"\r\n"); err != nil {
				return "", fmt.Errorf("failed to select interactive option: failed to write selection: %w", err)
			}
			return option.Value, nil
		case keyArrowUp:
			selected = moveSelection(selected, len(filtered), -1)
		case keyArrowDown:
			selected = moveSelection(selected, len(filtered), 1)
		case keyBackspace:
			if len(filter) > 0 {
				filter = filter[:len(filter)-1]
				selected = 0
			}
		case keyCharacter:
			if unicode.IsPrint(event.character) {
				filter = append(filter, event.character)
				selected = 0
			}
		}
		if err := renderSelection(p.output, label, filter, options, selected, visibleLimit); err != nil {
			return "", err
		}
	}
}

func filterSelectOptions(options []SelectOption, filter string) []SelectOption {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return append([]SelectOption(nil), options...)
	}
	filtered := make([]SelectOption, 0, len(options))
	for _, option := range options {
		value := strings.ToLower(option.Value + " " + option.Label + " " + option.Description)
		if strings.Contains(value, filter) {
			filtered = append(filtered, option)
		}
	}
	return filtered
}

func moveSelection(selected int, count int, delta int) int {
	if count == 0 {
		return 0
	}
	selected = (selected + delta) % count
	if selected < 0 {
		selected += count
	}
	return selected
}

func selectionWindow(selected int, count int, visibleLimit int) (int, int) {
	if count <= visibleLimit {
		return 0, count
	}
	start := selected - visibleLimit/2
	if start < 0 {
		start = 0
	}
	if start+visibleLimit > count {
		start = count - visibleLimit
	}
	return start, start + visibleLimit
}

func renderSelection(output io.Writer, label string, filter []rune, options []SelectOption, selected int, visibleLimit int) error {
	filtered := filterSelectOptions(options, string(filter))
	if selected >= len(filtered) {
		selected = 0
	}
	var content strings.Builder
	content.WriteString("\r\x1b[J")
	content.WriteString(styledText(output, label+":", ansiCyan) + " " + string(filter))
	content.WriteString("\r\n\r\n")
	start, end := selectionWindow(selected, len(filtered), visibleLimit)
	if len(filtered) == 0 {
		content.WriteString(styledText(output, "  No matching options.", ansiRed))
	} else {
		for index := start; index < end; index++ {
			option := filtered[index]
			marker := "  "
			if index == selected {
				marker = "❯ "
			}
			line := marker + option.Label
			if option.Description != "" {
				line += "  " + option.Description
			}
			if index == selected {
				line = ansiBoldCyan + line + ansiReset
			}
			content.WriteString(line)
			content.WriteString("\r\n")
		}
		if len(filtered) > visibleLimit {
			content.WriteString(fmt.Sprintf("  %d-%d of %d\r\n", start+1, end, len(filtered)))
		}
	}
	content.WriteString("\r\n")
	content.WriteString(styledText(output, "Type to filter · ↑↓ Select · Enter Confirm", ansiDim))
	linesBelow := 3
	if len(filtered) == 0 {
		linesBelow++
	} else {
		linesBelow += end - start
		if len(filtered) > visibleLimit {
			linesBelow++
		}
	}
	content.WriteString(fmt.Sprintf("\x1b[%dA\r", linesBelow))
	cursorColumn := len([]rune(label)) + 2 + len(filter)
	if cursorColumn > 0 {
		content.WriteString(fmt.Sprintf("\x1b[%dC", cursorColumn))
	}
	if _, err := io.WriteString(output, content.String()); err != nil {
		return fmt.Errorf("failed to select interactive option: failed to render options: %w", err)
	}
	return nil
}
