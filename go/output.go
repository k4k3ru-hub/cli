package cli

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"unicode/utf8"
)

const tablePadding = 2

// OutputTable writes rows in table format to standard output.
//
// Parameters:
//   - headers: table column headers
//   - rows: table rows
//
// Returns:
//   - An error when table data is invalid or output fails.
//
// Version:
//   - 2026-08-18: Delegated output to OutputTableTo.
func OutputTable(headers []string, rows [][]any) error {
	return OutputTableTo(os.Stdout, headers, rows)
}

// OutputTableTo writes rows in table format to the supplied writer.
//
// Parameters:
//   - writer: table output destination
//   - headers: non-empty table column headers
//   - rows: rows whose column counts match the headers
//
// Returns:
//   - An error when input is invalid or output fails.
//
// Version:
//   - 2026-08-18: Added.
func OutputTableTo(writer io.Writer, headers []string, rows [][]any) error {
	if writer == nil || isNilWriter(writer) {
		return invalidParameterError("failed to output table", "writer", "null")
	}
	if len(headers) == 0 {
		return invalidParameterError("failed to output table", "headers", "empty")
	}

	formattedRows := make([][]string, len(rows))
	columnWidths := make([]int, len(headers))
	for columnIndex, header := range headers {
		if header == "" {
			return invalidParameterError(
				"failed to output table",
				fmt.Sprintf("headers[%d]", columnIndex),
				"empty",
			)
		}
		if !utf8.ValidString(header) {
			return invalidTableCellError(-1, columnIndex, "table cell contains invalid utf-8")
		}
		if containsTableControlCharacter(header) {
			return invalidTableCellError(-1, columnIndex, "table cell contains a control character")
		}
		columnWidths[columnIndex] = utf8.RuneCountInString(header)
	}

	for rowIndex, row := range rows {
		if len(row) != len(headers) {
			return columnCountError(rowIndex, len(row), len(headers))
		}
		formattedRows[rowIndex] = make([]string, len(row))
		for columnIndex, cell := range row {
			value := fmt.Sprint(cell)
			if !utf8.ValidString(value) {
				return invalidTableCellError(rowIndex, columnIndex, "table cell contains invalid utf-8")
			}
			if containsTableControlCharacter(value) {
				return invalidTableCellError(rowIndex, columnIndex, "table cell contains a control character")
			}
			formattedRows[rowIndex][columnIndex] = value
			if width := utf8.RuneCountInString(value); width > columnWidths[columnIndex] {
				columnWidths[columnIndex] = width
			}
		}
	}

	dataWriter := tabwriter.NewWriter(writer, 0, 0, tablePadding, ' ', 0)
	if err := writeTableRow(dataWriter, headers); err != nil {
		return err
	}
	separators := make([]string, len(columnWidths))
	for columnIndex, width := range columnWidths {
		separators[columnIndex] = strings.Repeat("-", width)
	}
	if err := writeTableRow(dataWriter, separators); err != nil {
		return err
	}
	for _, row := range formattedRows {
		if err := writeTableRow(dataWriter, row); err != nil {
			return err
		}
	}
	if err := dataWriter.Flush(); err != nil {
		return &CLIError{Kind: ErrOutput, Operation: "failed to output table", Cause: err}
	}
	return nil
}

func isNilWriter(writer io.Writer) bool {
	value := reflect.ValueOf(writer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func writeTableRow(writer io.Writer, cells []string) error {
	for _, cell := range cells {
		if _, err := fmt.Fprintf(writer, "%s\t", cell); err != nil {
			return &CLIError{Kind: ErrOutput, Operation: "failed to output table", Cause: err}
		}
	}
	if _, err := fmt.Fprintln(writer); err != nil {
		return &CLIError{Kind: ErrOutput, Operation: "failed to output table", Cause: err}
	}
	return nil
}

func containsTableControlCharacter(value string) bool {
	return strings.ContainsAny(value, "\t\r\n")
}

func columnCountError(rowIndex int, actual int, expected int) error {
	return &CLIError{
		Kind:                   ErrInvalidColumnCount,
		Operation:              "failed to output table",
		Reason:                 "row column count does not match headers",
		RowIndex:               rowIndex,
		ColumnCount:            actual,
		ExpectedColumnCount:    expected,
		hasRowIndex:            true,
		hasColumnCount:         true,
		hasExpectedColumnCount: true,
	}
}

func invalidTableCellError(rowIndex int, columnIndex int, reason string) error {
	err := &CLIError{
		Kind:           ErrInvalidTableCell,
		Operation:      "failed to output table",
		Reason:         reason,
		ColumnIndex:    columnIndex,
		hasColumnIndex: true,
	}
	if rowIndex >= 0 {
		err.RowIndex = rowIndex
		err.hasRowIndex = true
	}
	return err
}
