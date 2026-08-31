package pipeline

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineProgramRunCmd triggers a program (enterprise) pipeline build via
// --pipeline <pipeline-id>, scoped by --enterprise/-E and --program/-P.
func newPipelineProgramRunCmd(f *cmdutil.Factory) *cobra.Command {
	var params []string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a program pipeline build",
		Long:  `Trigger a gitee-go program pipeline build by its pipeline id (see 'gitee pipeline program list'). Scoped by --enterprise/-E and --program/-P.`,
		Example: `  gitee pipeline program run --pipeline 706 -E 2 -P 423
  gitee pipeline program run --pipeline 706 -E 2 -P 423 -p KEY=VALUE -p OTHER=x --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineFlagID(cmd)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}

			req := giteego.PipelineOpsBuildRequest{PipelineID: id}
			for _, kv := range params {
				kv = strings.TrimSpace(kv)
				if kv == "" {
					continue
				}
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("invalid param %q, expected KEY=VALUE", kv)
				}
				req.Params = append(req.Params, giteego.ParameterStruct{Key: strings.TrimSpace(k), DefaultValue: strings.TrimSpace(v)})
			}

			b, err := client.TriggerProgramBuild(f.Context, req)
			if err != nil {
				return fmt.Errorf("failed to trigger program build: %w", err)
			}
			if b == nil {
				return fmt.Errorf("trigger build returned no build")
			}

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

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Build #%d triggered\n", b.BuildNumber)
			fmt.Fprintf(out, "ID:     %d\n", b.ID)
			fmt.Fprintf(out, "Status: %s\n", b.Status)
			if len(req.Params) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("pipeline.param_in"))
				fmt.Fprintln(out, "----")
				for _, p := range req.Params {
					fmt.Fprintf(out, "  %s=%s\n", p.Key, formatParamValue(p.DefaultValue))
				}
			}
			return nil
		},
	}

	cmd.Flags().Int64("pipeline", 0, "Program pipeline id (required, see 'gitee pipeline program list')")
	_ = cmd.MarkFlagRequired("pipeline")
	cmd.Flags().StringArrayVarP(&params, "param", "p", nil, "Build param as KEY=VALUE (repeated)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// newPipelineProgramBuildCmd returns the `pipeline program build` command group,
// covering program build runs and their runtime children stage runs
// (`program build stage`) and job runs (`program build job`).
func newPipelineProgramBuildCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Manage program pipeline build runs",
		Long: `View, list, cancel, rebuild and inspect gitee-go program pipeline builds, plus
the stage and job runs they produce.

Builds, stages and jobs are all runtime entities of a pipeline run: they have
no create/config/delete commands and cannot exist outside a build run. The
stage and job subcommands operate on the runs of a single build (IDs come from
` + "`build view`" + ` / ` + "`build status`" + ` output).`,
	}
	cmd.AddCommand(newPipelineProgramBuildListCmd(f))
	cmd.AddCommand(newPipelineProgramBuildViewCmd(f))
	cmd.AddCommand(newPipelineProgramBuildStatusCmd(f))
	cmd.AddCommand(newPipelineProgramBuildCancelCmd(f))
	cmd.AddCommand(newPipelineProgramBuildRebuildCmd(f))
	cmd.AddCommand(newPipelineProgramBuildLastCmd(f))
	cmd.AddCommand(newPipelineProgramStageCmd(f))
	cmd.AddCommand(newPipelineProgramJobCmd(f))
	return cmd
}

func programBuildIDArg(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("build id is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid build id %q", args[0])
	}
	return id, nil
}

func newPipelineProgramBuildListCmd(f *cmdutil.Factory) *cobra.Command {
	var current, pageSize int
	var order, sort, statuses string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List program pipeline build history",
		Example: `  gitee pipeline program build list --pipeline 706 -E 2 -P 423
  gitee pipeline program build list --pipeline 706 -E 2 -P 423 --status FAILED --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if current < 1 {
				return fmt.Errorf("--page must be at least 1")
			}
			if pageSize < 1 {
				return fmt.Errorf("--page-size must be at least 1")
			}
			id, err := pipelineFlagID(cmd)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
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
			page, err := client.ListProgramBuildHistory(f.Context, id, giteego.ProgramListQuery{
				Current:  current,
				PageSize: pageSize,
				Order:    order,
				Sort:     sort,
				Statuses: sorted,
			})
			if err != nil {
				return fmt.Errorf("failed to list program build history: %w", err)
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
			rows := [][]string{{"ID", "#", "Workflow", "STATUS", "START", "Code", "COMMIT"}}
			for _, b := range page.Data {
				rows = append(rows, []string{
					strconv.FormatInt(b.ID, 10),
					strconv.FormatInt(b.BuildNumber, 10),
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

	cmd.Flags().Int64("pipeline", 0, "Program pipeline id (required)")
	_ = cmd.MarkFlagRequired("pipeline")
	cmd.Flags().IntVarP(&current, "page", "p", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Number of builds per page")
	cmd.Flags().StringVar(&order, "order", "", "Sort field: create_time|id|build_number")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort direction: asc|desc")
	cmd.Flags().StringVar(&statuses, "status", "", "Comma-separated build statuses to filter (e.g. FAILED,SUCCESS)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildSimpleVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramBuildViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	var watch bool
	cmd := &cobra.Command{
		Use:   "view <build-id>",
		Short: "View a program pipeline build",
		Example: `  gitee pipeline program build view 123 -E 2 -P 423
  gitee pipeline program build view 123 -E 2 -P 423 -w`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programBuildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			b, err := client.GetProgramBuild(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get build: %w", err)
			}
			if b == nil {
				return fmt.Errorf("build %d not found", id)
			}
			if !watch || jsonFields != "" {
				return printBuild(f, jsonFields, b)
			}
			return watchProgramBuild(f, client, id, b)
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Refresh build status every 2s until it reaches a terminal state")
	return cmd
}

func newPipelineProgramBuildStatusCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "status <build-id>",
		Short:   "Show a program pipeline build status tree",
		Example: `  gitee pipeline program build status 123 -E 2 -P 423 --json`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programBuildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			st, err := client.GetProgramBuildStatus(f.Context, id)
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

func newPipelineProgramBuildCancelCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "cancel <build-id>",
		Short:   "Cancel a program pipeline build",
		Example: `  gitee pipeline program build cancel 123 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programBuildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Cancel build %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.CancelProgramBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to cancel build: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Build %d cancelled\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineProgramBuildRebuildCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "rebuild <build-id>",
		Short:   "Re-run a program pipeline build",
		Example: `  gitee pipeline program build rebuild 123 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programBuildIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			b, err := client.RebuildProgramBuild(f.Context, id)
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

func newPipelineProgramBuildLastCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "last",
		Short:   "Show the most recent program pipeline build",
		Example: `  gitee pipeline program build last --pipeline 706 -E 2 -P 423`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineFlagID(cmd)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			b, err := client.GetProgramLastBuild(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get last build: %w", err)
			}
			if b == nil {
				return fmt.Errorf("no build found for pipeline %d", id)
			}
			return printBuild(f, jsonFields, b)
		},
	}
	cmd.Flags().Int64("pipeline", 0, "Program pipeline id (required)")
	_ = cmd.MarkFlagRequired("pipeline")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// watchProgramBuild is the program variant of watchBuild: re-fetches a program
// build every 2 seconds until it reaches a terminal status, overwriting in place.
func watchProgramBuild(f *cmdutil.Factory, client *giteego.Client, id int64, initial *giteego.PipelineBuildVO) error {
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
		if _, err := fmt.Fprint(out, text); err != nil {
			return err
		}
		lines = strings.Count(text, "\n")
		if isTerminalBuildStatus(last.Status) {
			return nil
		}
		time.Sleep(2 * time.Second)
		b, err := client.GetProgramBuild(f.Context, id)
		if err != nil {
			return fmt.Errorf("failed to refresh build: %w", err)
		}
		if b == nil {
			return fmt.Errorf("build %d no longer exists", id)
		}
		last = b
	}
}
