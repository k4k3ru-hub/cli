package interactive

import (
	"bytes"
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestSelectionRedrawPreservesPromptRow verifies filtering never moves into prior console output.
//
// Version:
//   - 2026-09-16: Added.
func TestSelectionRedrawPreservesPromptRow(t *testing.T) {
	options := []SelectOption{{Value: "usdc", Label: "USDC"}, {Value: "weth", Label: "WETH"}}
	moveUp := regexp.MustCompile("\x1b\\[([0-9]+)A")
	row := 10 // Previous setup output occupies rows above this prompt.
	for _, filter := range []string{"", "1", "11", "111", "11", "1", "", "usd", "missing", ""} {
		var output bytes.Buffer
		if err := renderSelection(&output, "Approval token", []rune(filter), options, 0, 1); err != nil {
			t.Fatal(err)
		}
		frame := output.String()
		row += strings.Count(frame, "\n")
		moves := moveUp.FindAllStringSubmatch(frame, -1)
		if len(moves) != 1 {
			t.Fatalf("expected one cursor return: %q", frame)
		}
		for _, match := range moves {
			count, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			row -= count
		}
		if row != 10 {
			t.Fatalf("filter=%q returned to row %d, want prompt row 10", filter, row)
		}
	}
}

func TestPromptSelectLine(t *testing.T) {
	var output bytes.Buffer
	prompt, err := NewPrompt(strings.NewReader("product-2\n"), &output)
	if err != nil {
		t.Fatalf("NewPrompt() returned an unexpected error: %v", err)
	}
	selected, err := prompt.Select(context.Background(), "Select a product", []SelectOption{
		{Value: "product-1", Label: "product-1", Description: "1 USDC"},
		{Value: "product-2", Label: "product-2", Description: "10 USDC"},
	}, SelectConfig{})
	if err != nil {
		t.Fatalf("Select() returned an unexpected error: %v", err)
	}
	if selected != "product-2" {
		t.Fatalf("selected = %q, want %q", selected, "product-2")
	}
}

func TestFilterSelectOptions(t *testing.T) {
	options := []SelectOption{{Value: "alpha", Label: "Alpha"}, {Value: "beta", Label: "Beta", Description: "10 USDC"}}
	filtered := filterSelectOptions(options, "10 usdc")
	if len(filtered) != 1 || filtered[0].Value != "beta" {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestSelectionWindow(t *testing.T) {
	start, end := selectionWindow(9, 20, 8)
	if start != 5 || end != 13 {
		t.Fatalf("window = %d:%d, want 5:13", start, end)
	}
	start, end = selectionWindow(19, 20, 8)
	if start != 12 || end != 20 {
		t.Fatalf("end window = %d:%d, want 12:20", start, end)
	}
}
