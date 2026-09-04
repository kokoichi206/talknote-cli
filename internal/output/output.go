// Package output は CLI の出力整形を担う。
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatText  Format = "text"
)

func Parse(value string) (Format, error) {
	switch Format(value) {
	case FormatJSON, FormatTable, FormatText:
		return Format(value), nil
	default:
		return "", fmt.Errorf("unknown output format %q (json|table|text)", value)
	}
}

func Default(isTTY bool) Format {
	if isTTY {
		return FormatTable
	}
	return FormatJSON
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func WriteTable(w io.Writer, header []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		for i, cell := range row {
			row[i] = sanitizeCell(cell)
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

func sanitizeCell(value string) string {
	value = StripControl(value)
	value = strings.ReplaceAll(value, "\t", " ")
	return strings.ReplaceAll(value, "\n", " ")
}

func StripControl(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

func Truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "…"
}
