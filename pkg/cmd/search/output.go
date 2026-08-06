package search

import (
	"time"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func writeJSON[T any](f *cmdutil.Factory, value string, items []T) error {
	fields, full, listFields := cmdutil.ParseJSONFlag(value)
	if listFields {
		cmdutil.PrintJSONFieldList[T](f.IOStreams.Out)
		return nil
	}
	if full {
		return cmdutil.WriteJSON(f.IOStreams.Out, items)
	}
	return cmdutil.WriteJSONFields(f.IOStreams.Out, items, fields)
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02")
}
