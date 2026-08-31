package pipeline

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineBuildCmd returns the `pipeline build` command group, covering
// build runs (view/status/list/cancel/rebuild/last) and their runtime children
// stage runs (`build stage`) and job runs (`build job`).
func newPipelineBuildCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Manage pipeline build runs",
		Long: `View, list, cancel, rebuild and inspect gitee-go pipeline builds, plus the
stage and job runs they produce.

Builds, stages and jobs are all runtime entities of a pipeline run: they have
no create/config/delete commands and cannot exist outside a build run. The
stage and job subcommands operate on the runs of a single build (IDs come from
` + "`build view`" + ` / ` + "`build status`" + ` output).`,
	}
	cmd.AddCommand(newPipelineBuildViewCmd(f))
	cmd.AddCommand(newPipelineBuildStatusCmd(f))
	cmd.AddCommand(newPipelineBuildListCmd(f))
	cmd.AddCommand(newPipelineBuildCancelCmd(f))
	cmd.AddCommand(newPipelineBuildRebuildCmd(f))
	cmd.AddCommand(newPipelineBuildLastCmd(f))
	cmd.AddCommand(newPipelineStageCmd(f))
	cmd.AddCommand(newPipelineJobCmd(f))
	return cmd
}

func buildIDArg(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("build id is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid build id %q", args[0])
	}
	return id, nil
}

// resolveGoClient resolves owner/repo + go client with service-status check.
func resolveGoClient(f *cmdutil.Factory, cmd *cobra.Command) (*giteego.Client, string, string, error) {
	owner, repo, err := resolveOwnerRepo(f, cmd)
	if err != nil {
		return nil, "", "", err
	}
	if err := ensureServiceOpen(f, cmd, owner, repo); err != nil {
		return nil, "", "", err
	}
	client, err := repoPipelineClientForService(f, owner, repo, giteego.ServiceIPipe)
	if err != nil {
		return nil, "", "", err
	}
	return client, owner, repo, nil
}

func newPipelineBuildViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	var watch bool
	cmd := &cobra.Command{
		Use:   "view <build-id>",
		Short: "View a pipeline build",
		Example: `  gitee pipeline build view 123 -R owner/repo
  gitee pipeline build view 123 -R owner/repo --json
  gitee pipeline build view 123 -R owner/repo -w`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := buildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := resolveGoClient(f, cmd)
			if err != nil {
				return err
			}
			b, err := client.GetBuild(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get build: %w", err)
			}
			if b == nil {
				return fmt.Errorf("build %d not found", id)
			}
			if !watch || jsonFields != "" {
				return printBuild(f, jsonFields, b)
			}
			return watchBuild(f, client, id, b)
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Refresh build status every 2s until it reaches a terminal state")
	return cmd
}

// watchBuild re-fetches and re-renders a build every 2 seconds until it reaches
// a terminal status, overwriting the previous block in place.
func watchBuild(f *cmdutil.Factory, client *giteego.Client, id int64, initial *giteego.PipelineBuildVO) error {
	out := f.IOStreams.Out
	last := initial
	var lines int
	for {
		buf := &strings.Builder{}
		if err := renderBuild(buf, last); err != nil {
			return err
		}
		text := buf.String()
		if lines > 0 {
			clearLines(out, lines)
		}
		io.WriteString(out, text)
		lines = strings.Count(text, "\n")
		if isTerminalBuildStatus(last.Status) {
			return nil
		}
		time.Sleep(2 * time.Second)
		b, err := client.GetBuild(f.Context, id)
		if err != nil {
			return fmt.Errorf("failed to refresh build: %w", err)
		}
		if b == nil {
			return fmt.Errorf("build %d no longer exists", id)
		}
		last = b
	}
}

// clearLines moves the cursor up n lines and clears them (ANSI).
func clearLines(w io.Writer, n int) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(w, "\x1b[%dA\x1b[0J", n)
}

// isTerminalBuildStatus reports whether a build status is finished (no more
// runs). Mirrors the backend BuildStatus.isFinished(): SUCCESS/FAILED/CANCELLED/
// SKIPPED/TIMEOUT. Note the enum uses these full names (not SUCC/FAIL etc.).
func isTerminalBuildStatus(status string) bool {
	switch status {
	case "SUCCESS", "FAILED", "CANCELLED", "SKIPPED", "TIMEOUT":
		return true
	default:
		return false
	}
}

func printBuild(f *cmdutil.Factory, jsonFields string, b *giteego.PipelineBuildVO) error {
	if jsonFields != "" {
		fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
		if listFields {
			cmdutil.PrintJSONFieldList[giteego.PipelineBuildVO](f.IOStreams.Out)
			return nil
		}
		if full {
			return cmdutil.WriteJSON(f.IOStreams.Out, b)
		}
		result, err := cmdutil.SelectFields(b, fields)
		if err != nil {
			return err
		}
		return cmdutil.WriteJSON(f.IOStreams.Out, result)
	}
	return renderBuild(f.IOStreams.Out, b)
}

// renderBuild renders the default (non-JSON) build layout to w: header block,
// pipeline params, then stage/job tree with params and loggers.
func renderBuild(out io.Writer, b *giteego.PipelineBuildVO) error {
	// Header block.
	fmt.Fprintf(out, "Build #%d  [%s]\n", b.BuildNumber, b.Status)
	fmt.Fprintf(out, "ID:     %d\n", b.ID)
	fmt.Fprintf(out, "File:   %s\n", b.FileName)
	fmt.Fprintf(out, "Ref:    %s\n", b.Ref)
	fmt.Fprintf(out, "Start:  %s\n", timeStr(b.StartTime))
	fmt.Fprintf(out, "End:    %s\n", timeStr(b.EndTime))
	fmt.Fprintf(out, "%s: %s\n", i18n.T("pipeline.duration"), durationStr(b.StartTime, b.EndTime))
	if b.Trigger != nil {
		triggerType := ""
		if b.Trigger.Data != nil {
			triggerType = b.Trigger.Data.Type
		}
		if triggerType != "" || b.Trigger.Username != "" {
			who := b.Trigger.Username
			if who == "" {
				who = b.Trigger.UserID
			}
			fmt.Fprintf(out, "%s:  %s  by %s\n", i18n.T("pipeline.triggered_by"), triggerType, who)
		}
	}
	if b.Message != "" {
		fmt.Fprintf(out, "Msg:    %s\n", oneLine(b.Message))
	}

	if b.Param != nil && (len(b.Param.InParams) > 0 || len(b.Param.OutParams) > 0) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, i18n.T("pipeline.params_pipeline"))
		fmt.Fprintln(out, "--------")
		if len(b.Param.InParams) > 0 {
			printParamSection(out, i18n.T("pipeline.param_in"), b.Param.InParams)
		}
		if len(b.Param.OutParams) > 0 {
			printParamSection(out, i18n.T("pipeline.param_out"), b.Param.OutParams)
		}
	}

	for si, s := range b.Stages {
		fmt.Fprintln(out)
		bar := "└─"
		if si < len(b.Stages)-1 {
			bar = "├─"
		}
		fmt.Fprintf(out, "%s %s  [%s]  %s\n", bar, s.Name, s.Status, durationStr(s.StartTime, s.EndTime))

		if s.Param != nil && (len(s.Param.InParams) > 0 || len(s.Param.OutParams) > 0) {
			fmt.Fprintln(out)
			printParamSection(out, i18n.T("pipeline.param_stage_in"), s.Param.InParams)
			if len(s.Param.OutParams) > 0 {
				printParamSection(out, i18n.T("pipeline.param_stage_out"), s.Param.OutParams)
			}
		}

		for _, row := range s.Jobs {
			for _, j := range row {
				jobLine := fmt.Sprintf("    %s  [%s]  %s", j.Name, j.Status, durationStr(j.StartTime, j.EndTime))
				if j.Type != "" {
					jobLine += fmt.Sprintf("  (%s)", j.Type)
				}
				fmt.Fprintln(out, jobLine)
				if j.Message != "" {
					fmt.Fprintf(out, "      msg: %s\n", oneLine(j.Message))
				}
				if j.Data != nil && len(j.Data.Parameters) > 0 {
					fmt.Fprintln(out)
					printParamSection(out, i18n.T("pipeline.param_job"), j.Data.Parameters)
				}
				if j.Record != nil && len(j.Record.Loggers) > 0 {
					fmt.Fprintln(out)
					printLoggerSection(out, j.Record.Loggers)
				}
			}
		}
	}
	return nil
}

// timeStr formats a FlexTime or returns "-".
func timeStr(ft *giteego.FlexTime) string {
	if ft == nil || ft.IsZero() {
		return "-"
	}
	return ft.Format("2006-01-02 15:04:05")
}

// durationStr returns the human duration between two times, or "-".
func durationStr(start, end *giteego.FlexTime) string {
	if start == nil || end == nil || start.IsZero() || end.IsZero() {
		return "-"
	}
	d := end.Sub(start.Time)
	if d < 0 {
		return "-"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// printParamSection renders a titled parameter list as key = value rows.
func printParamSection(w io.Writer, title string, params []giteego.ParamValueVO) {
	fmt.Fprintf(w, "    %s:\n", title)
	for _, p := range params {
		fmt.Fprintf(w, "      %s = %s\n", p.Key, oneLine(string(p.Value)))
	}
}

// printConfigParams renders config-level parameters ([]ParameterStruct) as
// key = default rows, used by the program pipeline view command.
func printConfigParams(out io.Writer, title string, params []giteego.ParameterStruct) {
	if len(params) == 0 {
		return
	}
	fmt.Fprintf(out, "  %s:\n", title)
	for _, p := range params {
		key := p.Key
		if key == "" {
			key = p.Type
		}
		fmt.Fprintf(out, "    %s = %s\n", key, oneLine(formatParamValue(p.DefaultValue)))
	}
}

// formatParamValue renders a parameter default/value for display.
func formatParamValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []interface{}:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			parts = append(parts, formatParamValue(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// printLoggerSection renders a job's record loggers with their URLs.
func printLoggerSection(w io.Writer, loggers []giteego.JobLoggerVO) {
	fmt.Fprintf(w, "    %s:\n", i18n.T("pipeline.logs"))
	for _, lg := range loggers {
		fmt.Fprintf(w, "      %s  [%s]\n", lg.Name, lg.Status)
		if lg.Logger != "" {
			fmt.Fprintf(w, "        %s\n", lg.Logger)
		}
	}
}

// oneLine collapses newlines in a value for table-friendly display.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return s
}

// isPRSource reports whether a source's event indicates a pull-request trigger
// (PR opened or PR comment), matching the frontend CiTypes.PR_HOOK/PR_COMMENT_HOOK.
func isPRSource(s *giteego.BuildSourceDetail) bool {
	if s == nil {
		return false
	}
	return s.Event == "merge_request_hooks" || s.Event == "note_hooks"
}

// historySource renders the "source" column of a build history row: "PR #N" for
// pull-request triggers, else the branch name.
func historySource(sources []giteego.BuildSourceVO) string {
	for _, src := range sources {
		if src.Source == nil {
			continue
		}
		if isPRSource(src.Source) && src.Source.PrIID > 0 {
			return fmt.Sprintf("PR #%d", src.Source.PrIID)
		}
		if src.Source.Branch != "" {
			return src.Source.Branch
		}
	}
	return "-"
}

// historyCommit renders the commit message of a build history row from the
// source; on PR builds the message is the PR title (per the gitee-go backend).
func historyCommit(sources []giteego.BuildSourceVO) string {
	for _, src := range sources {
		if src.Source != nil && src.Source.Message != "" {
			return src.Source.Message
		}
	}
	return "-"
}

func newPipelineBuildStatusCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "status <build-id>",
		Short: "Show a pipeline build status tree",
		Example: `  gitee pipeline build status 123 -R owner/repo
  gitee pipeline build status 123 -R owner/repo --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := buildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := resolveGoClient(f, cmd)
			if err != nil {
				return err
			}
			st, err := client.GetBuildStatus(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get build status: %w", err)
			}
			if st == nil {
				return fmt.Errorf("build %d status not available", id)
			}
			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineBuildStatusVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, st)
				}
				result, err := cmdutil.SelectFields(st, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}
			out := f.IOStreams.Out
			fmt.Fprintf(out, "Build #%d  [%s]\n", st.ID, st.Status)
			if st.StartTime != nil {
				fmt.Fprintf(out, "Start:    %s\n", st.StartTime.Format("2006-01-02 15:04:05"))
			}
			if st.EndTime != nil {
				fmt.Fprintf(out, "End:      %s\n", st.EndTime.Format("2006-01-02 15:04:05"))
			}
			if len(st.Stages) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("pipeline.status_tree"))
				fmt.Fprintln(out, "------")
				for si, s := range st.Stages {
					bar := "├─"
					if si == len(st.Stages)-1 {
						bar = "└─"
					}
					fmt.Fprintf(out, "%s %s  [%s]\n", bar, s.Name, s.Status)
					for _, row := range s.Jobs {
						for _, j := range row {
							fmt.Fprintf(out, "    ├─%s  [%s]\n", j.Name, j.Status)
						}
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildStatusVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineBuildCancelCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cancel <build-id>",
		Short:   "Cancel a pipeline build",
		Example: `  gitee pipeline build cancel 123 -R owner/repo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := buildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := resolveGoClient(f, cmd)
			if err != nil {
				return err
			}
			if err := client.CancelBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to cancel build: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Build %d cancelled\n", id)
			return nil
		},
	}
	return cmd
}

func newPipelineBuildRebuildCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "rebuild <build-id>",
		Short:   "Re-run a pipeline build",
		Example: `  gitee pipeline build rebuild 123 -R owner/repo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := buildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := resolveGoClient(f, cmd)
			if err != nil {
				return err
			}
			b, err := client.RebuildBuild(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to rebuild: %w", err)
			}
			return printBuild(f, jsonFields, b)
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineBuildLastCmd(f *cmdutil.Factory) *cobra.Command {
	var ref, fileName string
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "last",
		Short:   "Show the most recent pipeline build",
		Example: `  gitee pipeline build last -R owner/repo --file ci.yml --ref master`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := resolveGoClient(f, cmd)
			if err != nil {
				return err
			}
			b, err := client.GetLastBuild(f.Context, fileName, ref)
			if err != nil {
				return fmt.Errorf("failed to get last build: %w", err)
			}
			if b == nil {
				return fmt.Errorf("no build found for %s @ %s", fileName, ref)
			}
			return printBuild(f, jsonFields, b)
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "", "Git branch or tag (required)")
	_ = cmd.MarkFlagRequired("ref")
	cmd.Flags().StringVar(&fileName, "file", "", "Pipeline YAML file name (required)")
	_ = cmd.MarkFlagRequired("file")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// newPipelineBuildListCmd returns the `build list` (history) command.
func newPipelineBuildListCmd(f *cmdutil.Factory) *cobra.Command {
	var ref, fileName, order, sort string
	var current, pageSize int
	var statuses string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline build history",
		Long: `List gitee-go pipeline build history with paging, optionally filtered by
file/ref/status. --order is the sort field (create_time/id/build_number, default
build_number) and --sort is the direction (asc/desc).`,
		Example: `  gitee pipeline build list -R owner/repo
  gitee pipeline build list -R owner/repo --file ci.yml --ref master
  gitee pipeline build list -R owner/repo --status FAILED --order create_time --sort desc --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if current < 1 {
				return fmt.Errorf("--page must be at least 1")
			}
			if pageSize < 1 {
				return fmt.Errorf("--page-size must be at least 1")
			}
			client, _, _, err := resolveGoClient(f, cmd)
			if err != nil {
				return err
			}
			sorted := []string{}
			if strings.TrimSpace(statuses) != "" {
				for _, s := range strings.Split(statuses, ",") {
					if t := strings.TrimSpace(s); t != "" {
						sorted = append(sorted, t)
					}
				}
			}
			page, err := client.ListBuildHistory(f.Context, fileName, ref, order, sort, current, pageSize, sorted)
			if err != nil {
				return fmt.Errorf("failed to list build history: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineBuildSimpleVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, page.Data)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, page.Data, fields)
			}

			if len(page.Data) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pipeline.no_results"))
				return nil
			}
			rows := [][]string{{"ID", "#", "YML", "Workflow", "STATUS", "START", "Code", "COMMIT"}}
			for _, b := range page.Data {
				rows = append(rows, []string{
					strconv.FormatInt(b.ID, 10),
					strconv.FormatInt(b.BuildNumber, 10),
					b.FileName,
					b.Ref,
					b.Status,
					timeStr(b.StartTime),
					historySource(b.Sources),
					oneLine(historyCommit(b.Sources)),
				})
			}
			if err := cmdutil.WriteTableBordered(f.IOStreams.Out, rows); err != nil {
				return err
			}
			if page.Total > 0 && page.PageSize > 0 {
				totalPages := (page.Total + page.PageSize - 1) / page.PageSize
				fmt.Fprintf(f.IOStreams.Out, "\npage %d/%d (pageSize %d, total %d)\n", page.Current, totalPages, page.PageSize, page.Total)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Filter by git ref (branch/tag)")
	cmd.Flags().StringVar(&fileName, "file", "", "Filter by pipeline YAML file name")
	cmd.Flags().StringVar(&order, "order", "", "Sort field: create_time|id|build_number (default build_number)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort direction: asc|desc")
	cmd.Flags().IntVarP(&current, "page", "p", 1, "Page number")
	cmd.Flags().IntVarP(&pageSize, "page-size", "L", 10, "Page size")
	cmd.Flags().StringVar(&statuses, "status", "", "Comma-separated build statuses to filter (e.g. FAILED,SUCCESS)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildSimpleVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
