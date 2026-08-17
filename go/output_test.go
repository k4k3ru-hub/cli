package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/cli/go"
)

// Test table output writes headers, separators, and rows.
func TestOutputTableTo(t *testing.T) {
	var output bytes.Buffer
	err := cli.OutputTableTo(
		&output,
		[]string{"name", "status"},
		[][]any{{"alpha", "ready"}, {"beta", nil}},
	)
	if err != nil {
		t.Fatalf("OutputTableTo() returned an unexpected error: %v", err)
	}

	expected := "name   status  \n-----  ------  \nalpha  ready   \nbeta   <nil>   \n"
	if output.String() != expected {
		t.Fatalf("OutputTableTo() output = %q, want %q", output.String(), expected)
	}
}

// Test table output accepts no data rows.
func TestOutputTableToWithoutRows(t *testing.T) {
	var output bytes.Buffer
	if err := cli.OutputTableTo(&output, []string{"name"}, nil); err != nil {
		t.Fatalf("OutputTableTo() returned an unexpected error: %v", err)
	}
	if output.String() != "name  \n----  \n" {
		t.Fatalf("OutputTableTo() output = %q, want a header and separator", output.String())
	}
}

// Test table output preserves Unicode text.
func TestOutputTableToUnicode(t *testing.T) {
	var output bytes.Buffer
	if err := cli.OutputTableTo(
		&output,
		[]string{"名前", "状態"},
		[][]any{{"猫", "有効"}, {"アルファ", "停止"}},
	); err != nil {
		t.Fatalf("OutputTableTo() returned an unexpected error: %v", err)
	}
	for _, value := range []string{"名前", "状態", "猫", "有効", "アルファ", "停止"} {
		if !strings.Contains(output.String(), value) {
			t.Fatalf("OutputTableTo() output = %q, want Unicode value %q", output.String(), value)
		}
	}
}

// Test table output validates its required inputs.
func TestOutputTableToValidation(t *testing.T) {
	tests := []struct {
		name    string
		writer  *bytes.Buffer
		headers []string
		rows    [][]any
		kind    error
	}{
		{name: "nil writer", headers: []string{"name"}, kind: cli.ErrInvalidParameter},
		{name: "empty headers", writer: &bytes.Buffer{}, kind: cli.ErrInvalidParameter},
		{name: "empty header", writer: &bytes.Buffer{}, headers: []string{""}, kind: cli.ErrInvalidParameter},
		{name: "short row", writer: &bytes.Buffer{}, headers: []string{"name", "status"}, rows: [][]any{{"alpha"}}, kind: cli.ErrInvalidColumnCount},
		{name: "wide row", writer: &bytes.Buffer{}, headers: []string{"name"}, rows: [][]any{{"alpha", "ready"}}, kind: cli.ErrInvalidColumnCount},
		{name: "header control", writer: &bytes.Buffer{}, headers: []string{"na\nme"}, kind: cli.ErrInvalidTableCell},
		{name: "cell control", writer: &bytes.Buffer{}, headers: []string{"name"}, rows: [][]any{{"alpha\tbeta"}}, kind: cli.ErrInvalidTableCell},
		{name: "invalid utf-8", writer: &bytes.Buffer{}, headers: []string{"name"}, rows: [][]any{{string([]byte{0xff})}}, kind: cli.ErrInvalidTableCell},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var writer = test.writer
			err := cli.OutputTableTo(writer, test.headers, test.rows)
			if !errors.Is(err, test.kind) {
				t.Fatalf("OutputTableTo() error = %v, want category %v", err, test.kind)
			}
			if test.writer != nil && test.writer.Len() != 0 {
				t.Fatalf("OutputTableTo() wrote partial output %q", test.writer.String())
			}
		})
	}
}

// Test column-count errors expose row and column details.
func TestOutputTableColumnCountError(t *testing.T) {
	err := cli.OutputTableTo(&bytes.Buffer{}, []string{"name"}, [][]any{{"value", "extra"}})
	if !errors.Is(err, cli.ErrInvalidColumnCount) {
		t.Fatalf("OutputTableTo() error = %v, want ErrInvalidColumnCount", err)
	}
	var cliError *cli.CLIError
	if !errors.As(err, &cliError) {
		t.Fatalf("OutputTableTo() error = %T, want *cli.CLIError", err)
	}
	if cliError.RowIndex != 0 || cliError.ColumnCount != 2 || cliError.ExpectedColumnCount != 1 {
		t.Fatalf(
			"column error details = row:%d actual:%d expected:%d, want 0, 2, 1",
			cliError.RowIndex,
			cliError.ColumnCount,
			cliError.ExpectedColumnCount,
		)
	}
}

// Test table output preserves writer failures.
func TestOutputTableWriterError(t *testing.T) {
	underlying := errors.New("write failed")
	err := cli.OutputTableTo(errorWriter{err: underlying}, []string{"name"}, nil)
	if !errors.Is(err, cli.ErrOutput) {
		t.Fatalf("OutputTableTo() error = %v, want ErrOutput", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("OutputTableTo() error = %v, want underlying error", err)
	}
}
