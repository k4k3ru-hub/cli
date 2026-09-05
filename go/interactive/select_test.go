package interactive

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

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
