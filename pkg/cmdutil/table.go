package cmdutil

import (
	"bufio"
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
