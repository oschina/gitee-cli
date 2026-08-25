package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

const tableColumnSpacing = 2

// WriteTable writes plain-text rows aligned by their terminal display width.
func WriteTable(w io.Writer, rows [][]string) error {
	widths := make([]int, 0)
	for _, row := range rows {
		if len(row) > len(widths) {
			widths = append(widths, make([]int, len(row)-len(widths))...)
		}
		for column, cell := range row {
			widths[column] = max(widths[column], runewidth.StringWidth(cell))
		}
	}

	bw := bufio.NewWriter(w)
	for _, row := range rows {
		for column, cell := range row {
			if _, err := io.WriteString(bw, cell); err != nil {
				return err
			}
			if column < len(row)-1 {
				padding := widths[column] - runewidth.StringWidth(cell) + tableColumnSpacing
				if _, err := io.WriteString(bw, strings.Repeat(" ", padding)); err != nil {
					return err
				}
			}
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// WriteTableBordered writes rows as a bordered pipe-separated table, with a
// separator header row under the column names. Example:
//
//	| ID | BUILD | FILE | ... |
//	|----|-------|------|-----|
//	| 8  | 6     | ...  | ... |
func WriteTableBordered(w io.Writer, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	widths := make([]int, 0)
	for _, row := range rows {
		if len(row) > len(widths) {
			widths = append(widths, make([]int, len(row)-len(widths))...)
		}
		for column, cell := range row {
			widths[column] = max(widths[column], runewidth.StringWidth(cell))
		}
	}

	bw := bufio.NewWriter(w)
	header := rows[0]
	data := rows[1:]

	// Render header row.
	writeBorderedRow(bw, header, widths, false)
	// Render separator row.
	writeBorderedSeparator(bw, widths)
	// Render data rows.
	for _, row := range data {
		writeBorderedRow(bw, row, widths, false)
	}
	return bw.Flush()
}

func writeBorderedRow(bw *bufio.Writer, row []string, widths []int, _ bool) {
	_, _ = io.WriteString(bw, "│")
	for column, cell := range row {
		_, _ = io.WriteString(bw, " ")
		_, _ = io.WriteString(bw, cell)
		padding := widths[column] - runewidth.StringWidth(cell)
		if padding > 0 {
			_, _ = io.WriteString(bw, strings.Repeat(" ", padding))
		}
		_, _ = io.WriteString(bw, " │")
	}
	_ = bw.WriteByte('\n')
}

func writeBorderedSeparator(bw *bufio.Writer, widths []int) {
	_, _ = io.WriteString(bw, "├")
	for i, w := range widths {
		_, _ = io.WriteString(bw, strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			_, _ = io.WriteString(bw, "┼")
		}
	}
	_, _ = io.WriteString(bw, "┤")
	_ = bw.WriteByte('\n')
}

// WriteTableBorderedCompact is like WriteTableBordered but omits the separator
// line between header and data rows for a lighter look.
func WriteTableBorderedCompact(w io.Writer, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	widths := make([]int, 0)
	for _, row := range rows {
		if len(row) > len(widths) {
			widths = append(widths, make([]int, len(row)-len(widths))...)
		}
		for column, cell := range row {
			widths[column] = max(widths[column], runewidth.StringWidth(cell))
		}
	}

	bw := bufio.NewWriter(w)
	for _, row := range rows {
		writeBorderedRow(bw, row, widths, false)
	}
	return bw.Flush()
}

// WriteTableBorderedAll writes a full bordered table with top, separator, and
// bottom lines. Example:
//
//	+-------+------+-----+
//	| ID    | FILE | REF |
//	+-------+------+-----+
//	| 8     | ...  | ... |
//	+-------+------+-----+
func WriteTableBorderedAll(w io.Writer, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	widths := make([]int, 0)
	for _, row := range rows {
		if len(row) > len(widths) {
			widths = append(widths, make([]int, len(row)-len(widths))...)
		}
		for column, cell := range row {
			widths[column] = max(widths[column], runewidth.StringWidth(cell))
		}
	}

	bw := bufio.NewWriter(w)

	// Top border.
	_, _ = fmt.Fprint(bw, borderLine(widths, "┌", "┬", "┐"))
	// Header row.
	writeBorderedRow(bw, rows[0], widths, false)
	// Separator.
	_, _ = fmt.Fprint(bw, borderLine(widths, "├", "┼", "┤"))
	// Data rows.
	for _, row := range rows[1:] {
		writeBorderedRow(bw, row, widths, false)
	}
	// Bottom border.
	_, _ = fmt.Fprint(bw, borderLine(widths, "└", "┴", "┘"))
	return bw.Flush()
}

func borderLine(widths []int, left, cross, right string) string {
	var b strings.Builder
	b.WriteString(left)
	for i, w := range widths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			b.WriteString(cross)
		}
	}
	b.WriteString(right)
	b.WriteByte('\n')
	return b.String()
}
