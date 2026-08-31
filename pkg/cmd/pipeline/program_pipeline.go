package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// pluginEntry pairs a plugin category name with a plugin, used to build the
// job-type selector in the program pipeline wizard.
type pluginEntry struct {
	category string
	plugin   giteego.PluginVO
}

// label renders the selector option for a plugin entry.
func (pe pluginEntry) label() string {
	name := pe.plugin.Name
	if name == "" {
		name = pe.plugin.Type
	}
	if pe.plugin.Type == "" {
		return name
	}
	if pe.category != "" {
		return fmt.Sprintf("[%s] %s (%s)", pe.category, name, pe.plugin.Type)
	}
	return fmt.Sprintf("%s (%s)", name, pe.plugin.Type)
}

func newPipelineProgramListCmd(f *cmdutil.Factory) *cobra.Command {
	var current, pageSize int
	var order, sort, search string
	var jsonFields string
	var showUUID, showFull bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List program pipelines",
		Example: `  gitee pipeline program list -E 2 -P 423
  gitee pipeline program list -E 2 -P 423 --search build --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			page, err := client.ListProgramPipelines(f.Context, giteego.ProgramListQuery{
				Current:  current,
				PageSize: pageSize,
				Order:    order,
				Sort:     sort,
				Search:   search,
			})
			if err != nil {
				return fmt.Errorf("failed to list program pipelines: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineSummaryVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, page.Data)
				}
				result, err := cmdutil.SelectFields(page.Data, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			if len(page.Data) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pipeline.program.no_pipelines"))
				return nil
			}

			if err := renderProgramPipelineList(f, page.Data, page.Total, showUUID, showFull); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&current, "page", "p", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Number of pipelines per page")
	cmd.Flags().StringVar(&order, "order", "", "Sort order: asc or desc")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort field")
	cmd.Flags().StringVar(&search, "search", "", "Search pipeline by name")
	cmd.Flags().BoolVar(&showUUID, "uuid", false, "Show full pipeline uuid")
	cmd.Flags().BoolVar(&showFull, "full", false, "Show full details (uuid, group id, labels)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineSummaryVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "view <pipeline-id>",
		Short: "View a program pipeline configuration",
		Example: `  gitee pipeline program view 706 -E 2 -P 423
  gitee pipeline program view pipeline.ops.pipeline.706 -E 2 -P 423 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineIDArg(args[0])
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			p, err := client.GetProgramPipeline(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get program pipeline: %w", err)
			}
			if p == nil {
				return fmt.Errorf("program pipeline %d not found", id)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, p)
				}
				result, err := cmdutil.SelectFields(p, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "ID:         %d\n", pipelineIDOf(p))
			if p.UUID != "" {
				fmt.Fprintf(out, "UUID:       %s\n", p.UUID)
			}
			fmt.Fprintf(out, "Name:       %s\n", p.Name)
			if p.Ref != "" {
				fmt.Fprintf(out, "Ref:        %s\n", p.Ref)
			}
			printConfigParams(out, i18n.T("pipeline.params_pipeline"), p.Parameters)
			if len(p.Stages) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Stages")
				for _, s := range p.Stages {
					fmt.Fprintln(out, "  "+s.Name)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var name, body, configFile string
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a program pipeline",
		Long: `Create a program pipeline.

A bare --name is accepted in non-interactive mode, but the backend requires
at least one stage with a job (a bare {"name": "..."} request is rejected
with an HTTP 400). For human use pass a complete PipelineRequest JSON via
--config/--body, or omit all flags in a terminal to launch the interactive
wizard. --body - reads the JSON from stdin.`,
		Example: `  gitee pipeline program create -E 2 -P 423 --name "build-all"
  gitee pipeline program create -E 2 -P 423 --config ./pipeline.json
  gitee pipeline program create -E 2 -P 423 --body - < ./pipeline.json
  gitee pipeline program create -E 2 -P 423   # interactive wizard`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ent, prog, err := programClient(f, cmd)
			if err != nil {
				return err
			}

			nameChanged := cmd.Flags().Changed("name")
			bodyChanged := cmd.Flags().Changed("body")
			configChanged := cmd.Flags().Changed("config")

			// Interactive wizard: no flags given in a real terminal.
			if !nameChanged && !bodyChanged && !configChanged {
				if f.IsTUI() {
					req, err := programPipelineWizard(f, client, nil, i18n.T("pipeline.wizard.confirm"), i18n.T("pipeline.wizard.option_submit_create"))
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
							return nil
						}
						return err
					}
					if len(req.Stages) == 0 {
						return errors.New(i18n.T("pipeline.wizard.empty_config"))
					}
					return createProgramPipelineCmd(f, client, cmd, req, ent, prog)
				}
				if err := warnNoStagesHint(f); err != nil {
					return err
				}
			}

			req, err := pipelineRequestFromFlags(f, cmd, name, body, configFile)
			if err != nil {
				return err
			}
			if req.Name == "" {
				if configFile != "" || (body != "" && body != "-") {
					// Name may legitimately come from the file; don't require --name.
				} else {
					return cmdutil.FlagErrorf("--name is required")
				}
			}
			return createProgramPipelineCmd(f, client, cmd, req, ent, prog)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Pipeline name")
	cmd.Flags().StringVar(&body, "body", "", "PipelineRequest JSON body file to submit (use - for stdin)")
	cmd.Flags().StringVar(&configFile, "config", "", "PipelineRequest JSON config file to submit (same as --body)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// createProgramPipelineCmd performs the actual create request and renders the
// confirmation/output, shared by the flags path and the wizard.
func createProgramPipelineCmd(f *cmdutil.Factory, client *giteego.Client, cmd *cobra.Command, req giteego.PipelineRequest, _, _ string) error {
	jsonFields, _ := cmd.Flags().GetString("json")
	// Keep a bare --name request working (advisory warning) but warn that the
	// backend will reject it without stages.
	if len(req.Stages) == 0 {
		hint := pipelineRequestExample()
		fmt.Fprintf(f.IOStreams.ErrOut, "%s\n", i18n.Tf("pipeline.create.stages_hint", hint))
	}
	p, err := client.CreateProgramPipeline(f.Context, req)
	if err != nil {
		return friendlyAPIError(fmt.Errorf("failed to create program pipeline: %w", err))
	}
	id := pipelineIDOf(p)
	if jsonFields != "" {
		return cmdutil.WriteJSON(f.IOStreams.Out, p)
	}
	fmt.Fprintf(f.IOStreams.Out, "Created program pipeline %d (%s)\n", id, p.Name)
	return nil
}

// warnNoStagesHint emits the advisory stages hint to stderr for a request that
// has no stages, before a bare --name request is submitted.
func warnNoStagesHint(f *cmdutil.Factory) error {
	hint := pipelineRequestExample()
	fmt.Fprintf(f.IOStreams.ErrOut, "%s\n", i18n.Tf("pipeline.create.stages_hint", hint))
	return nil
}

func newPipelineProgramEditCmd(f *cmdutil.Factory) *cobra.Command {
	var name, body, configFile string
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "edit <pipeline-id>",
		Aliases: []string{"update"},
		Short:   "Edit a program pipeline",
		Long: `Edit a program pipeline.

The edit is a full replacement on the backend: every stage/job/source/
trigger is replaced by what you submit. In a terminal, omitting all flags
launches the interactive wizard pre-filled from the current configuration.`,
		Example: `  gitee pipeline program edit 706 -E 2 -P 423 --name "new-name"
  gitee pipeline program edit 706 -E 2 -P 423 --config ./pipeline.json
  gitee pipeline program edit 706 -E 2 -P 423   # interactive wizard`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineIDArg(args[0])
			if err != nil {
				return err
			}
			client, ent, prog, err := programClient(f, cmd)
			if err != nil {
				return err
			}

			nameChanged := cmd.Flags().Changed("name")
			bodyChanged := cmd.Flags().Changed("body")
			configChanged := cmd.Flags().Changed("config")

			// Interactive wizard: no flags given in a real terminal; pre-filled
			// from the current configuration.
			if !nameChanged && !bodyChanged && !configChanged && f.IsTUI() {
				current, err := client.GetProgramPipeline(f.Context, id)
				if err != nil {
					return fmt.Errorf("failed to fetch program pipeline %d: %w", id, err)
				}
				prefill := pipelineRequestFromVO(current)
				req, err := programPipelineUpdateWizard(f, client, &prefill)
				if err != nil {
					if cmdutil.IsUserCancelled(err) {
						fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
						return nil
					}
					return err
				}
				if len(req.Stages) == 0 {
					return errors.New(i18n.T("pipeline.wizard.empty_config"))
				}
				return updateProgramPipelineCmd(f, client, cmd, id, req, ent, prog)
			}

			if !nameChanged && !bodyChanged && !configChanged {
				return cmdutil.FlagErrorf("nothing to edit: pass --name or --body")
			}

			req, err := pipelineRequestFromFlags(f, cmd, name, body, configFile)
			if err != nil {
				return err
			}
			if req.Name == "" && !nameChanged {
				// The name may come from the file; only --name requires it.
			}
			if req.Name == "" && nameChanged {
				return cmdutil.FlagErrorf("--name cannot be empty")
			}
			return updateProgramPipelineCmd(f, client, cmd, id, req, ent, prog)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New pipeline name")
	cmd.Flags().StringVar(&body, "body", "", "PipelineRequest JSON body file to submit (use - for stdin)")
	cmd.Flags().StringVar(&configFile, "config", "", "PipelineRequest JSON config file to submit (same as --body)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// updateProgramPipelineCmd performs the actual update request and renders the
// confirmation/output, shared by the flags path and the wizard.
func updateProgramPipelineCmd(f *cmdutil.Factory, client *giteego.Client, cmd *cobra.Command, id int64, req giteego.PipelineRequest, _, _ string) error {
	jsonFields, _ := cmd.Flags().GetString("json")
	p, err := client.UpdateProgramPipeline(f.Context, id, req)
	if err != nil {
		return friendlyAPIError(fmt.Errorf("failed to update program pipeline: %w", err))
	}
	if jsonFields != "" {
		return cmdutil.WriteJSON(f.IOStreams.Out, p)
	}
	fmt.Fprintf(f.IOStreams.Out, "Edited program pipeline %d (%s)\n", pipelineIDOf(p), p.Name)
	return nil
}

// pipelineRequestFromFlags assembles a PipelineRequest from the --name / --body
// / --config flags. --body and --config both load a PipelineRequest JSON (from a
// file, or stdin when the value is "-"); --name overrides the resulting name.
func pipelineRequestFromFlags(f *cmdutil.Factory, cmd *cobra.Command, name, body, configFile string) (giteego.PipelineRequest, error) {
	req := giteego.PipelineRequest{Name: name}
	source := body
	if source == "" {
		source = configFile
	}
	if source != "" {
		var data []byte
		var err error
		if source == "-" {
			data, err = io.ReadAll(f.IOStreams.In)
		} else {
			data, err = os.ReadFile(source)
		}
		if err != nil {
			return req, fmt.Errorf("failed to read body: %w", err)
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return req, fmt.Errorf("failed to parse body JSON: %w", err)
		}
		if name != "" {
			req.Name = name
		}
	}
	_ = cmd
	return req, nil
}

// pipelineRequestExample is a minimal-but-valid PipelineRequest JSON example
// shown to users who submit a bare --name request. Job parameters follow the
// gitee-go data contract: each scheme field becomes a data.parameters
// {key, value} entry (key = scheme identifier), except convertor.parameters:false
// fields which sit at data.<identifier>.
func pipelineRequestExample() string {
	example := giteego.PipelineRequest{
		Name: "build-all",
		Stages: []giteego.StageRequest{
			{
				Name: "build",
				Jobs: [][]giteego.JobRequest{
					{{
						Name: "compile",
						Type: "JENKINS_JOB",
						Data: map[string]interface{}{
							"parameters": []map[string]interface{}{
								{"key": "certificate", "value": "{{certificate}}"},
								{"key": "jobName", "value": "my-job"},
								{"key": "params", "value": "{}"},
							},
						},
					}},
				},
			},
		},
	}
	b, _ := json.MarshalIndent(example, "", "  ")
	return string(b)
}

// programPipelineUpdateWizard is the interactive path for `pipeline update`. It
// asks how the user wants to update the existing pipeline — edit the full JSON
// configuration (pre-filled from the current one) or rebuild stages with the
// wizard — then dispatches to the matching flow.
func programPipelineUpdateWizard(f *cmdutil.Factory, client *giteego.Client, prefill *giteego.PipelineRequest) (giteego.PipelineRequest, error) {
	req := giteego.PipelineRequest{}
	if prefill != nil {
		req = *prefill
	}

	var mode string
	if err := huh.NewSelect[string]().
		Title(i18n.T("pipeline.wizard.update_mode")).
		Options(
			huh.NewOption(i18n.T("pipeline.wizard.update_mode_json"), "json"),
			huh.NewOption(i18n.T("pipeline.wizard.update_mode_rebuild"), "rebuild"),
		).
		Value(&mode).
		Run(); err != nil {
		return req, err
	}

	if mode == "json" {
		// Full JSON edit, pre-filled from the current configuration.
		return reviewPipelineRequest(f, req, i18n.T("pipeline.wizard.confirm_update"), i18n.T("pipeline.wizard.option_submit_update"))
	}
	// Rebuild: drop the current stages so the wizard starts from a clean slate
	// (other fields — name, sources, etc. — stay prefilled).
	req.Stages = nil
	return programPipelineWizard(f, client, &req, i18n.T("pipeline.wizard.confirm_update"), i18n.T("pipeline.wizard.option_submit_update"))
}

// programPipelineWizard interactively builds a PipelineRequest. With a non-nil
// prefill the wizard starts from the current configuration (used by update).
// confirmTitle is the final confirmation prompt and submitLabel the label of
// the "submit" choice (create vs. update wording).
func programPipelineWizard(f *cmdutil.Factory, client *giteego.Client, prefill *giteego.PipelineRequest, confirmTitle, submitLabel string) (giteego.PipelineRequest, error) {
	req := giteego.PipelineRequest{}
	if prefill != nil {
		req = *prefill
	}

	// Name.
	if err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T("pipeline.wizard.name")).
			Prompt("> ").
			Placeholder(i18n.T("pipeline.wizard.name_hint")).
			Value(&req.Name).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New(i18n.T("pipeline.wizard.name_hint"))
				}
				return nil
			}),
	)).Run(); err != nil {
		return req, err
	}

	// Plugins used by the job-type selector; fetched once.
	var plugins []pluginEntry
	if client != nil {
		cats, err := client.ListProgramPlugins(f.Context)
		if err != nil {
			return req, friendlyAPIError(fmt.Errorf("failed to list plugins: %w", err))
		}
		for _, cat := range cats {
			for _, p := range cat.Plugins {
				plugins = append(plugins, pluginEntry{category: cat.Name, plugin: p})
			}
		}
	}
	if len(plugins) == 0 {
		return req, errors.New(i18n.T("pipeline.wizard.no_plugins"))
	}

	// Stages & jobs.
	for {
		stage := giteego.StageRequest{}
		if err := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title(i18n.Tf("pipeline.wizard.stage_number", len(req.Stages)+1)).
				Prompt("> ").
				Placeholder(i18n.T("pipeline.wizard.stage_name_hint")).
				Value(&stage.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New(i18n.T("pipeline.wizard.stage_name_hint"))
					}
					return nil
				}),
		)).Run(); err != nil {
			return req, err
		}

		// Jobs in this stage (one group per job; parallel groups are not
		// supported by the CLI wizard).
		var jobs []giteego.JobRequest
		for {
			job, err := wizardJob(f, client, plugins)
			if err != nil {
				return req, err
			}
			jobs = append(jobs, job)

			// Explicit "keep adding" vs "done" choices. The finish option is
			// preselected so a plain Enter advances — users are never trapped
			// in the job loop — while arrow-up + Enter keeps adding.
			addMore := "no"
			if err := huh.NewSelect[string]().
				Title(i18n.Tf("pipeline.wizard.add_job", stage.Name)).
				Options(
					huh.NewOption(i18n.T("pipeline.wizard.add_job_more"), "yes"),
					huh.NewOption(i18n.T("pipeline.wizard.add_job_done"), "no"),
				).
				Value(&addMore).
				Run(); err != nil {
				return req, err
			}
			if addMore != "yes" {
				break
			}
		}
		if len(jobs) > 0 {
			stage.Jobs = [][]giteego.JobRequest{jobs}
			req.Stages = append(req.Stages, stage)
		}

		addStage := "no"
		if err := huh.NewSelect[string]().
			Title(i18n.T("pipeline.wizard.add_stage")).
			Options(
				huh.NewOption(i18n.T("pipeline.wizard.add_stage_more"), "yes"),
				huh.NewOption(i18n.T("pipeline.wizard.add_stage_done"), "no"),
			).
			Value(&addStage).
			Run(); err != nil {
			return req, err
		}
		if addStage != "yes" {
			break
		}
	}

	// Final review: full PipelineRequest JSON + submit/edit/cancel choices.
	return reviewPipelineRequest(f, req, confirmTitle, submitLabel)
}

// reviewPipelineRequest is the final step of the wizard. It prints the full
// PipelineRequest JSON, then asks how to finish: submit as-is, edit the full
// JSON in an external editor (with a save hint), or cancel. submitLabel names
// the submit choice (create vs. update wording).
func reviewPipelineRequest(f *cmdutil.Factory, req giteego.PipelineRequest, confirmTitle, submitLabel string) (giteego.PipelineRequest, error) {
	reqJSONBytes, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return req, err
	}
	reqJSON := string(reqJSONBytes)
	fmt.Fprintln(f.IOStreams.ErrOut, i18n.T("pipeline.wizard.final"))
	fmt.Fprintln(f.IOStreams.ErrOut, reqJSON)

	for {
		var choice string
		if err := huh.NewSelect[string]().
			Title(confirmTitle).
			Options(
				huh.NewOption(submitLabel, "submit"),
				huh.NewOption(i18n.T("pipeline.wizard.option_edit_json"), "edit"),
				huh.NewOption(i18n.T("pipeline.wizard.option_cancel"), "cancel"),
			).
			Value(&choice).
			Run(); err != nil {
			return req, err
		}

		switch choice {
		case "submit":
			return req, nil
		case "cancel":
			return req, huh.ErrUserAborted
		}

		// Edit the full JSON in the external editor. Print a save/quit hint
		// first — this is the step users most often get stuck on.
		fmt.Fprintln(f.IOStreams.ErrOut, i18n.T("pipeline.wizard.editor_hint"))
		for {
			edited, err := cmdutil.OpenEditor(f.IOStreams, "pipeline-request-*.json", reqJSON)
			if err != nil {
				return req, fmt.Errorf("could not open editor: %w", err)
			}
			var parsed giteego.PipelineRequest
			if err := json.Unmarshal([]byte(edited), &parsed); err != nil {
				fmt.Fprintf(f.IOStreams.ErrOut, "%s\n", i18n.Tf("pipeline.wizard.invalid_json", err))
				retry := false
				if err := huh.NewConfirm().
					Title(i18n.T("pipeline.wizard.edit_again")).
					Affirmative("yes").
					Negative("no").
					Value(&retry).
					Run(); err != nil {
					return req, err
				}
				if !retry {
					return req, cmdutil.FlagErrorf("%s", i18n.Tf("pipeline.wizard.invalid_json", err))
				}
				reqJSON = edited
				continue
			}
			if parsed.Name == "" {
				fmt.Fprintln(f.IOStreams.ErrOut, i18n.T("pipeline.wizard.name_hint"))
				reqJSON = edited
				continue
			}
			req = parsed
			if b, mErr := json.MarshalIndent(req, "", "  "); mErr == nil {
				reqJSON = string(b)
			}
			fmt.Fprintln(f.IOStreams.ErrOut, reqJSON)
			break
		}
		// Loop back to the submit/edit/cancel choice with the edited config.
	}
}

// wizardJob prompts for a single job: plugin type (from the program plugin
// list), a display name, and an interactive data form rendered from the plugin
// scheme (see wizardJobDataForm). The job identifier is intentionally NOT
// collected: on create the backend assigns one, and on update identifiers of
// kept jobs are preserved untouched (JSON edit mode); the CLI never invents or
// overwrites a job identifier.
func wizardJob(f *cmdutil.Factory, client *giteego.Client, plugins []pluginEntry) (giteego.JobRequest, error) {
	var opts []huh.Option[string]
	seen := map[string]bool{}
	for _, pe := range plugins {
		if pe.plugin.Type == "" {
			continue
		}
		if seen[pe.plugin.Type] {
			continue // same type may appear under multiple categories
		}
		seen[pe.plugin.Type] = true
		opts = append(opts, huh.NewOption(pe.label(), pe.plugin.Type))
	}
	sort.Slice(opts, func(i, j int) bool {
		return opts[i].Key < opts[j].Key
	})

	var jobType string
	if err := huh.NewSelect[string]().
		Title(i18n.T("pipeline.wizard.job_type")).
		Options(opts...).
		Value(&jobType).
		Run(); err != nil {
		return giteego.JobRequest{}, err
	}

	var name string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T("pipeline.wizard.job_name")).
			Prompt("> ").
			Placeholder(i18n.T("pipeline.wizard.job_name_hint")).
			Value(&name).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New(i18n.T("pipeline.wizard.job_name_hint"))
				}
				return nil
			}),
	)).Run(); err != nil {
		return giteego.JobRequest{}, err
	}

	// Job data: an interactive form rendered from the plugin scheme, mirroring
	// the gitee-go web editor's component mapping (PluginTypeRelation.ts):
	// Input/Textarea/Password/Select/Radio/Checkbox/Command/ObjectArray/...
	var data interface{} = map[string]interface{}{}
	if client != nil {
		schemes, err := client.GetProgramPluginSchemes(f.Context, jobType)
		if err != nil {
			if isGiteeGoQuota(err) {
				return giteego.JobRequest{}, friendlyAPIError(err)
			}
			fmt.Fprintf(f.IOStreams.ErrOut, "%s\n", i18n.Tf("pipeline.wizard.scheme_missing", err))
		} else {
			for _, sch := range schemes {
				if sch == nil {
					continue
				}
				formData, err := wizardJobDataForm(f, client, sch.Config)
				if err != nil {
					return giteego.JobRequest{}, err
				}
				data = formData
				break
			}
		}
	}

	job := giteego.JobRequest{
		Name: name,
		Type: jobType,
		Data: data,
	}
	if job.Data == nil {
		job.Data = map[string]interface{}{}
	}
	// Echo the assembled data so the user sees exactly what was saved.
	saved, _ := json.MarshalIndent(job.Data, "", "  ")
	fmt.Fprintln(f.IOStreams.ErrOut, i18n.T("pipeline.wizard.job_data"))
	fmt.Fprintf(f.IOStreams.ErrOut, "%s\n", string(saved))
	return job, nil
}

// wizardJobDataForm interactively collects a job's data fields by rendering
// the plugin scheme's component list as huh form screens, one component per
// step. The component→form mapping mirrors gitee-go's PluginTypeRelation.ts:
//
//	Input/Password/Textarea/Number/Certification/HostSelect/UserSelect → text input
//	Select/Radio                                        → single select
//	Checkbox (with options) / Select.multiple           → multi select
//	Checkbox (no options) / Switch                      → yes/no toggle
//	Command                                             → multi-line text
//	RemoteSelect                                        → options fetched over the API (fallback: text input)
//	Text (html:false)                                   → read-only note
//	Text (html:true)                                    → skipped (static HTML for the web editor)
//	Compose/Object                                      → nested group of children
//	Compose array/ObjectArray                           → repeatable nested group
//	RefValue                                            → auto-resolved reference (${sibling} substitution, no prompt)
//	RefExport                                           → skipped (upstream task exports unavailable in the CLI)
func wizardJobDataForm(f *cmdutil.Factory, client *giteego.Client, config []giteego.PluginComponentScheme) (map[string]interface{}, error) {
	fmt.Fprintln(f.IOStreams.ErrOut, i18n.T("pipeline.wizard.data_form_hint"))
	// Collect one value per top-level scheme component into an ordered list;
	// the final placement (data.parameters vs data.<identifier>) is decided by
	// assembleJobData from each component's convertor.
	var fields []collectedComponent
	for _, c := range config {
		if c.Identifier == "" {
			continue
		}
		bucket := map[string]interface{}{}
		if err := wizardDataComponent(f, client, bucket, c); err != nil {
			return nil, err
		}
		v, ok := bucket[c.Identifier]
		if !ok {
			continue // skipped component (RefValue, html Text, …)
		}
		fields = append(fields, collectedComponent{comp: c, val: v})
	}
	return assembleJobData(fields)
}

// collectedComponent is one top-level scheme component together with the value
// its wizard form collected.
type collectedComponent struct {
	comp giteego.PluginComponentScheme
	val  interface{}
}

// passwordParamType marks Password scheme components in the job's
// data.parameters entries so the gitee-go form editor renders them as
// secret/password fields.
const passwordParamType = "PASSWORD"

// jobParameterVO is one entry of the job's data.parameters array, the shape
// the Gitee Go web editor submits (Converter.toFormGeneric) and the backend
// stores: an ordered list of {key, value} pairs keyed by scheme identifier.
// Password components additionally carry type "PASSWORD" (see
// jobParameterEntry), so the editor knows to render the field as a password.
type jobParameterVO struct {
	Key   string      `json:"key"`
	Type  string      `json:"type,omitempty"`
	Value interface{} `json:"value"`
}

// jobParameterEntry builds one data.parameters entry for a scheme component.
// Password components carry type "PASSWORD"; all other components omit type.
func jobParameterEntry(c giteego.PluginComponentScheme, value interface{}) jobParameterVO {
	entry := jobParameterVO{Key: c.Identifier, Value: value}
	if c.Type == "Password" {
		entry.Type = passwordParamType
	}
	return entry
}

// assembleJobData builds the submitted job data from the values collected per
// top-level scheme component, mirroring gitee-go's Converter.toFormGeneric:
//
//   - convertor.parameters explicitly false → data.<identifier> (top level);
//   - otherwise (true or unset, which is the common case) → an entry of the
//     data.parameters array, in scheme order; children of Compose components
//     never become separate entries — their values ride inside the parent's
//     JSON-stringified value;
//   - Password components get "type":"PASSWORD" on their parameters entry;
//   - switch → "true"/"false" string; map/array values (Compose, multi-select,
//     array inputs) → JSON string; single Input → trimmed; other scalars as-is.
func assembleJobData(fields []collectedComponent) (map[string]interface{}, error) {
	params := []jobParameterVO{}
	top := map[string]interface{}{}
	// A duplicate identifier (e.g. a RefValue variant plus the real Input
	// variant of the same field) keeps the LAST occurrence — the frontend's
	// step map (getStepFromParameters) is last-wins as well.
	seen := map[string]bool{}
	for _, fld := range fields {
		value := jobParamValue(fld.comp, fld.val)
		if fld.comp.Convertor != nil && !fld.comp.Convertor.Parameters {
			// Explicit parameters:false → top-level data field.
			top[fld.comp.Identifier] = value
			seen[fld.comp.Identifier] = true
			continue
		}
		if seen[fld.comp.Identifier] {
			// Replace the previously appended parameter entry, keeping order.
			for i := len(params) - 1; i >= 0; i-- {
				if params[i].Key == fld.comp.Identifier {
					params[i] = jobParameterEntry(fld.comp, value)
					break
				}
			}
			continue
		}
		seen[fld.comp.Identifier] = true
		params = append(params, jobParameterEntry(fld.comp, value))
	}
	data := map[string]interface{}{}
	for k, v := range top {
		data[k] = v
	}
	data["parameters"] = params
	return data, nil
}

// jobParamValue converts one collected component value the way the gitee-go
// frontend's toString does when serializing parameters (see Converter.ts).
func jobParamValue(c giteego.PluginComponentScheme, v interface{}) interface{} {
	switch tv := v.(type) {
	case nil:
		return ""
	case map[string]interface{}, []interface{}:
		// Compose/objects and arrays are JSON-stringified in parameters
		// (toString → JSON.stringify), e.g. maven settings/artifacts payloads.
		if b, err := json.Marshal(tv); err == nil {
			return string(b)
		}
		return v
	}
	switch c.Type {
	case "Switch":
		// toString: JSON-stringified boolean.
		return strconv.FormatBool(isTruthy(v))
	case "Input":
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return v
}

// resolveRefValue resolves a RefValue component's defaultValue template the way
// gitee-go's RefValueItem does: every ${siblingIdentifier} placeholder is
// replaced with the value already collected for that sibling (empty siblings
// leave the placeholder unresolved). Unresolvable refs stay as-is.
func resolveRefValue(c giteego.PluginComponentScheme, data map[string]interface{}) string {
	s, ok := c.DefaultValue.(string)
	if !ok || s == "" {
		return ""
	}
	for k, v := range data {
		if k == c.Identifier || v == nil {
			continue
		}
		sv := schemeValueString(v)
		if sv == "" {
			continue
		}
		s = strings.ReplaceAll(s, "${"+k+"}", sv)
	}
	return s
}

// wizardDataComponent renders one scheme component and stores its value at
// data[identifier].
func wizardDataComponent(f *cmdutil.Factory, client *giteego.Client, data map[string]interface{}, c giteego.PluginComponentScheme) error {
	if c.Identifier == "" {
		return nil
	}
	// Disabled fields are fixed values on the web; keep their default without
	// prompting (e.g. the artifacts "type" in the maven plugin).
	if c.Disabled {
		if c.DefaultValue != nil {
			data[c.Identifier] = c.DefaultValue
		} else {
			data[c.Identifier] = ""
		}
		return nil
	}
	typ := c.Type
	// Compose maps to Object / ObjectArray exactly like gitee-go's
	// transformNewSchemaField.
	if typ == "Compose" {
		if c.Array {
			typ = "ObjectArray"
		} else {
			typ = "Object"
		}
	}
	switch typ {
	// Text (html:true) is static HTML for the web editor's display only → skip.
	// Text (html:false) is a read-only note.
	case "Text":
		if c.HTML {
			return nil
		}
		// Read-only note; the value is display-only but still participates in
		// data.parameters with its (resolved) default, like the web editor.
		note := huh.NewNote().
			Title(componentLabel(c)).
			Description(fmt.Sprintf("%v", c.DefaultValue))
		if err := huh.NewForm(huh.NewGroup(note)).Run(); err != nil {
			return err
		}
		if d := usableDefault(c.DefaultValue); d != nil {
			data[c.Identifier] = schemeValueString(d)
		}
		return nil
	case "RefValue":
		// Auto-resolved reference (RefValueItem): the defaultValue is a
		// template that gets ${siblingIdentifier} placeholders substituted with
		// the already-collected sibling values; it is never user input.
		if resolved := resolveRefValue(c, data); resolved != "" {
			data[c.Identifier] = resolved
		}
		return nil
	case "Object":
		sub := map[string]interface{}{}
		if m, ok := c.DefaultValue.(map[string]interface{}); ok {
			for k, v := range m {
				sub[k] = v
			}
		}
		for _, child := range c.Children {
			if err := wizardDataComponent(f, client, sub, child); err != nil {
				return err
			}
		}
		data[c.Identifier] = sub
		return nil
	case "ObjectArray", "ObjectList":
		entries := []interface{}{}
		for {
			entry := map[string]interface{}{}
			for _, child := range c.Children {
				if err := wizardDataComponent(f, client, entry, child); err != nil {
					return err
				}
			}
			entries = append(entries, entry)
			// Finish is preselected so a plain Enter stops adding entries.
			more := "no"
			if err := huh.NewSelect[string]().
				Title(i18n.Tf("pipeline.wizard.entries_more", componentLabel(c))).
				Options(
					huh.NewOption(i18n.T("pipeline.wizard.entries_add"), "yes"),
					huh.NewOption(i18n.T("pipeline.wizard.entries_done"), "no"),
				).
				Value(&more).
				Run(); err != nil {
				return err
			}
			if more != "yes" {
				break
			}
		}
		data[c.Identifier] = entries
		return nil
	case "Select":
		return wizardSelectField(f, data, c, c.Multiple)
	case "Radio":
		return wizardSelectField(f, data, c, false)
	case "Checkbox":
		if len(c.Options) > 0 {
			return wizardSelectField(f, data, c, true)
		}
		// Single boolean checkbox.
		return wizardBoolField(f, data, c)
	case "Switch":
		// gitee-go renders Switch as a boolean toggle.
		return wizardBoolField(f, data, c)
	case "Command":
		return wizardTextAreaField(f, data, c)
	case "RemoteSelect":
		// Fetch the options over the API (like the web's RemoteItem) and render
		// a real select; fall back to a manual input when the fetch fails.
		if client != nil && c.URL != nil && c.URL.PipelineOps != "" {
			resp, err := client.RemoteSelectOptions(f.Context, c.URL.PipelineOps, nil)
			if err == nil && len(resp.List) > 0 {
				synth := c
				synth.Options = resp.List
				return wizardSelectField(f, data, synth, c.Multiple)
			}
		}
		return wizardTextField(f, data, c)
	default:
		return wizardTextField(f, data, c)
	}
}

// wizardTextField renders single-line (or repeated) text-like components:
// Input/Password/Textarea/Number/Certification/HostSelect/RemoteSelect and
// unknown types. Array components collect a list of strings until the user
// enters an empty value or stops adding.
func wizardTextField(f *cmdutil.Factory, data map[string]interface{}, c giteego.PluginComponentScheme) error {
	label := componentLabel(c)
	title := label
	if note := kindNote(c.Type); note != "" {
		title = label + " " + note
	}

	// Repeated (array) text input.
	if c.Array {
		list := []interface{}{}
		if arr, ok := c.DefaultValue.([]interface{}); ok {
			for _, v := range arr {
				if usableDefault(v) != nil {
					list = append(list, v)
				}
			}
		}
		for {
			var s string
			input := huh.NewInput().
				Title(fmt.Sprintf("%s [%d]", title, len(list))).
				Prompt("> ").
				Value(&s)
			input.Placeholder(schemePlaceholder(c))
			if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
				return err
			}
			if strings.TrimSpace(s) == "" {
				break
			}
			list = append(list, typedValue(c.Type, s))
			more := "no"
			if err := huh.NewSelect[string]().
				Title(i18n.Tf("pipeline.wizard.entries_more", label)).
				Options(
					huh.NewOption(i18n.T("pipeline.wizard.entries_add"), "yes"),
					huh.NewOption(i18n.T("pipeline.wizard.entries_done"), "no"),
				).
				Value(&more).
				Run(); err != nil {
				return err
			}
			if more != "yes" {
				break
			}
		}
		if list == nil {
			list = []interface{}{}
		}
		data[c.Identifier] = list
		return nil
	}

	var s string
	if d := usableDefault(c.DefaultValue); d != nil {
		s = schemeValueString(d)
	}
	if c.Type == "Textarea" {
		return wizardTextAreaField(f, data, c)
	}
	input := huh.NewInput().
		Title(title).
		Prompt("> ").
		Value(&s)
	input.Placeholder(schemePlaceholder(c))
	if c.Type == "Password" {
		input.EchoMode(huh.EchoModePassword)
	}
	input.Validate(schemeValidate(c, i18n.T("pipeline.wizard.field_required")))
	if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
		return err
	}
	data[c.Identifier] = typedValue(c.Type, s)
	return nil
}

// wizardTextAreaField renders a multi-line component (Command/Textarea) as a
// huh text field.
func wizardTextAreaField(f *cmdutil.Factory, data map[string]interface{}, c giteego.PluginComponentScheme) error {
	var s string
	if d := usableDefault(c.DefaultValue); d != nil {
		// Command defaults may be given as a list of lines; join them.
		if arr, ok := d.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, v := range arr {
				parts = append(parts, schemeValueString(v))
			}
			s = strings.Join(parts, "\n")
		} else {
			s = schemeValueString(d)
		}
	}
	text := huh.NewText().
		Title(componentLabel(c)).
		CharLimit(-1).
		Value(&s)
	if c.Placeholder != "" && !isTemplateString(c.Placeholder) {
		text.Placeholder(c.Placeholder)
	}
	text.Validate(schemeValidate(c, i18n.T("pipeline.wizard.field_required")))
	if err := huh.NewForm(huh.NewGroup(text)).Run(); err != nil {
		return err
	}
	data[c.Identifier] = s
	return nil
}

// wizardBoolField renders a boolean component (single Checkbox / Switch) as a
// yes/no confirm.
func wizardBoolField(f *cmdutil.Factory, data map[string]interface{}, c giteego.PluginComponentScheme) error {
	var val bool
	if c.DefaultValue != nil {
		switch d := c.DefaultValue.(type) {
		case bool:
			val = d
		default:
			val = schemeValueString(d) == "true"
		}
	}
	if err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(componentLabel(c)).
			Affirmative("yes").
			Negative("no").
			Value(&val),
	)).Run(); err != nil {
		return err
	}
	data[c.Identifier] = val
	return nil
}

// wizardSelectField renders a single- or multi-select component from its
// options, preserving the raw option values on submit.
func wizardSelectField(f *cmdutil.Factory, data map[string]interface{}, c giteego.PluginComponentScheme, multi bool) error {
	opts, raw, defKey := schemeSelectOptions(c)
	if len(opts) == 0 {
		// No options to choose from (e.g. an empty Checkbox) — fall back to a
		// plain input so the field can still be filled.
		return wizardTextField(f, data, c)
	}

	var validate func(string) error
	rules := c.Rules
	validate = func(key string) error {
		for _, r := range rules {
			if r.Required && key == "" {
				if r.ErrorMsg != "" {
					return errors.New(r.ErrorMsg)
				}
				return errors.New(i18n.T("pipeline.wizard.name_hint"))
			}
		}
		return nil
	}

	if multi {
		picked := seedMultiSelection(c)
		ms := huh.NewMultiSelect[string]().
			Title(componentLabel(c)).
			Options(opts...).
			Value(&picked)
		if err := ms.Run(); err != nil {
			return err
		}
		vals := make([]interface{}, 0, len(picked))
		for _, k := range picked {
			vals = append(vals, raw[k])
		}
		if vals == nil {
			vals = []interface{}{}
		}
		data[c.Identifier] = vals
		return nil
	}

	key := defKey
	sel := huh.NewSelect[string]().
		Title(componentLabel(c)).
		Options(opts...).
		Value(&key).
		Validate(validate)
	if err := sel.Run(); err != nil {
		return err
	}
	if key == "" {
		data[c.Identifier] = nil
		return nil
	}
	data[c.Identifier] = raw[key]
	return nil
}

// schemeSelectOptions converts a component's options into huh options plus a
// key→raw map (preserving the original value type) and the default option key.
func schemeSelectOptions(c giteego.PluginComponentScheme) ([]huh.Option[string], map[string]interface{}, string) {
	var opts []huh.Option[string]
	raw := map[string]interface{}{}
	var defKey string
	for _, o := range c.Options {
		key := schemeValueString(o.Value)
		if key == "" {
			key = o.Key
		}
		label := o.Label
		if label == "" {
			label = o.Key
		}
		if label == "" {
			label = key
		}
		rawVal := o.Value
		if rawVal == nil {
			rawVal = o.Key
		}
		raw[key] = rawVal
		opts = append(opts, huh.NewOption(label, key))
		if defKey == "" && c.DefaultValue != nil && schemeValueString(c.DefaultValue) == key {
			defKey = key
		}
	}
	return opts, raw, defKey
}

// seedMultiSelection derives the initially checked keys of a multi-select from
// the component's default value: an array of values, an object whose (truthy)
// keys are checked (e.g. the maven settings Checkbox), or a single value.
func seedMultiSelection(c giteego.PluginComponentScheme) []string {
	var picked []string
	switch d := c.DefaultValue.(type) {
	case []interface{}:
		for _, v := range d {
			if k := schemeValueString(v); k != "" {
				picked = append(picked, k)
			}
		}
	case map[string]interface{}:
		for k, v := range d {
			if isTruthy(v) {
				picked = append(picked, k)
			}
		}
	default:
		if c.DefaultValue != nil {
			if k := schemeValueString(c.DefaultValue); k != "" {
				picked = append(picked, k)
			}
		}
	}
	return picked
}

// typedValue converts an entered string to the JSON-friendly type of the
// component: numbers become int64/float64, everything else stays a string.
func typedValue(typ, s string) interface{} {
	if typ == "Number" && s != "" {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if fl, err := strconv.ParseFloat(s, 64); err == nil {
			return fl
		}
	}
	return s
}

// schemeValidate builds a huh input validator from the component's rules
// (required + regex), mirroring gitee-go's antd rule conversion.
func schemeValidate(c giteego.PluginComponentScheme, requiredHint string) func(string) error {
	return func(s string) error {
		val := strings.TrimSpace(s)
		for _, r := range c.Rules {
			if r.Required && val == "" {
				if r.ErrorMsg != "" {
					return errors.New(r.ErrorMsg)
				}
				if requiredHint != "" {
					return errors.New(requiredHint)
				}
				return errors.New("required")
			}
			if r.Regex != "" && val != "" {
				re, err := regexp.Compile(r.Regex)
				if err == nil && !re.MatchString(val) {
					if r.ErrorMsg != "" {
						return errors.New(r.ErrorMsg)
					}
					return errors.New("invalid value")
				}
			}
		}
		return nil
	}
}

// kindNote returns the localized suffix explaining how a component type is
// entered in the terminal where the web editor offers a richer control.
func kindNote(typ string) string {
	key := ""
	switch typ {
	case "Password":
		key = "pipeline.wizard.field_password"
	case "Certification":
		key = "pipeline.wizard.field_certification"
	case "HostSelect":
		key = "pipeline.wizard.field_host_select"
	case "RemoteSelect":
		key = "pipeline.wizard.field_remote_select"
	case "Number":
		key = "pipeline.wizard.field_number"
	case "UserSelect":
		key = "pipeline.wizard.field_user_select"
	}
	if key == "" {
		return ""
	}
	note := i18n.T(key)
	if note == key {
		return ""
	}
	return note
}

// schemeValueString renders a scheme value (option or default) as the stable
// string key used by the TUI selector. JSON numbers decode as float64, so
// integer-valued floats render without a decimal point.
func schemeValueString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		if float64(t) == math.Trunc(float64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(float64(t), 'f', -1, 32)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// componentLabel returns the display label of a scheme component. The backend
// often sends i18n templates (e.g. "#{{build.maven.jdkVersion.displayName}}")
// that the CLI cannot resolve; in that case the scheme's yaml field name or
// the raw identifier is shown instead.
func componentLabel(c giteego.PluginComponentScheme) string {
	if c.Name != "" && !isTemplateString(c.Name) {
		return c.Name
	}
	if c.Convertor != nil && c.Convertor.YAMLField != "" && !isTemplateString(c.Convertor.YAMLField) {
		return c.Convertor.YAMLField
	}
	if c.Identifier != "" {
		return c.Identifier
	}
	if c.Type != "" {
		return c.Type
	}
	return "?"
}

// isTemplateString reports whether s is an unresolved scheme i18n template
// ("#{{key}}") that the CLI cannot translate.
func isTemplateString(s string) bool {
	return strings.HasPrefix(s, "#{{")
}

// usableDefault returns the component's default value unless it is an
// unresolvable i18n template; single-brace "{...}" refs are user-side
// references and are kept.
func usableDefault(v interface{}) interface{} {
	if s, ok := v.(string); ok && isTemplateString(s) {
		return nil
	}
	return v
}

// schemePlaceholder returns a usable placeholder for a component, stripping
// unresolved i18n templates.
func schemePlaceholder(c giteego.PluginComponentScheme) string {
	if c.Placeholder != "" && !isTemplateString(c.Placeholder) {
		return c.Placeholder
	}
	return i18n.T("pipeline.wizard.field_optional")
}

// isTruthy reports whether a JSON-ish value is truthy (used to seed checkbox
// defaults that are objects of booleans).
func isTruthy(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != "" && t != "false"
	case float64:
		return t != 0
	default:
		return true
	}
}

// isGiteeGoQuota reports whether err is a gitee-go quota (429
// insufficient_quota) response.
func isGiteeGoQuota(err error) bool {
	var apiErr *giteego.APIError
	return errors.As(err, &apiErr) && apiErr.IsQuota()
}

// friendlyAPIError augments gitee-go API errors with a translated, actionable
// hint so quota/permission failures explain themselves at the command level.
func friendlyAPIError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *giteego.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch {
	case apiErr.IsQuota():
		return fmt.Errorf("%w\n%s", err, i18n.T("pipeline.wizard.quota_hint"))
	case apiErr.IsPermission():
		return fmt.Errorf("%w\n%s", err, i18n.T("pipeline.wizard.permission_hint"))
	}
	return err
}

// pipelineRequestFromVO converts a PipelineVO into a PipelineRequest for the
// update wizard prefill. Struct fields are carried over as-is.
func pipelineRequestFromVO(vo *giteego.PipelineVO) giteego.PipelineRequest {
	req := giteego.PipelineRequest{}
	if vo == nil {
		return req
	}
	req.Name = vo.Name
	req.UUID = vo.UUID
	req.Group = vo.Group
	req.Strategy = vo.Strategy
	req.Notifications = vo.Notifications
	req.Parameters = vo.Parameters
	for _, l := range vo.Labels {
		req.Labels = append(req.Labels, l.ID)
	}
	for _, s := range vo.Sources {
		req.Sources = append(req.Sources, giteego.SourceRequest{
			Name:       s.Name,
			Type:       s.Type,
			Identifier: s.Identifier,
			Setting:    s.Setting,
		})
	}
	for _, t := range vo.Triggers {
		req.Triggers = append(req.Triggers, giteego.TriggerRequest{
			Name:  t.Name,
			Type:  t.Type,
			Auto:  t.Auto,
			Rules: t.Rules,
		})
	}
	for _, st := range vo.Stages {
		stage := giteego.StageRequest{
			Name:          st.Name,
			Strategy:      st.Strategy,
			Notifications: st.Notifications,
		}
		if len(st.Jobs) > 0 {
			for _, group := range st.Jobs {
				var jobs []giteego.JobRequest
				for _, j := range group {
					jobs = append(jobs, giteego.JobRequest{
						Name:          j.Name,
						Identifier:    j.Identifier,
						Type:          j.Type,
						Data:          j.Data,
						Sources:       j.Sources,
						Notifications: j.Notifications,
						Strategy:      j.Strategy,
					})
				}
				stage.Jobs = append(stage.Jobs, jobs)
			}
		}
		req.Stages = append(req.Stages, stage)
	}
	return req
}

func newPipelineProgramCloneCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "clone <pipeline-id>",
		Short:   "Clone a program pipeline",
		Example: `  gitee pipeline program clone 706 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineIDArg(args[0])
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			p, err := client.CloneProgramPipeline(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to clone program pipeline: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, p)
			}
			fmt.Fprintf(f.IOStreams.Out, "Cloned program pipeline %d from %d\n", pipelineIDOf(p), id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <pipeline-id>",
		Short:   "Delete a program pipeline",
		Example: `  gitee pipeline program delete 706 -E 2 -P 423 --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineIDArg(args[0])
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Delete program pipeline %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteProgramPipeline(f.Context, id); err != nil {
				return fmt.Errorf("failed to delete program pipeline: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Deleted program pipeline %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newPipelineProgramHistoryCmd(f *cmdutil.Factory) *cobra.Command {
	var current, pageSize int
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "history <pipeline-id>",
		Short:   "List configuration (version) history of a program pipeline",
		Example: `  gitee pipeline program history 706 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineIDArg(args[0])
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			page, err := client.ListProgramPipelineHistory(f.Context, id, current, pageSize)
			if err != nil {
				return fmt.Errorf("failed to list program pipeline history: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineHistoryVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, page.Data)
				}
				result, err := cmdutil.SelectFields(page.Data, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			out := f.IOStreams.Out
			if len(page.Data) == 0 {
				fmt.Fprintln(out, i18n.T("pipeline.no_history"))
				return nil
			}
			fmt.Fprintf(out, "Total: %d\n", page.Total)
			fmt.Fprintln(out, "VERSION  HITS  CREATOR      CREATED             ONLINE")
			for _, h := range page.Data {
				online := "no"
				if h.Online {
					online = "yes"
				}
				created := ""
				if h.CreateTime != nil {
					created = h.CreateTime.Format("2006-01-02 15:04:05")
				}
				fmt.Fprintf(out, "%d       %d    %-12s  %s  %s\n", h.Version, h.Hits, h.Creator, created, online)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&current, "page", "p", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Number of versions per page")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineHistoryVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// programHistoryScan* bound the ownership pre-check below.
const (
	programHistoryScanPages = 20
	programHistoryScanSize  = 100
)

// programPipelineOwnsHistory checks historyID against pipelineID's history
// listing before an apply. The apply endpoint is scoped by history id alone,
// so without this check a mismatched --history would silently reconfigure
// whichever pipeline really owns that record. Beyond the scan window the
// listing cannot rule out ownership and conclusive is false, leaving the
// decision to the caller.
func programPipelineOwnsHistory(ctx context.Context, client *giteego.Client, pipelineID, historyID int64) (owned, conclusive bool, err error) {
	for cur := 1; cur <= programHistoryScanPages; cur++ {
		page, err := client.ListProgramPipelineHistory(ctx, pipelineID, cur, programHistoryScanSize)
		if err != nil {
			return false, true, err
		}
		for i := range page.Data {
			if page.Data[i].ID == historyID {
				return true, true, nil
			}
		}
		if len(page.Data) == 0 || cur*programHistoryScanSize >= page.Total {
			return false, true, nil
		}
	}
	return false, false, nil
}

func newPipelineProgramHistoryApplyCmd(f *cmdutil.Factory) *cobra.Command {
	var historyID int64
	var yes bool
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "history-apply <pipeline-id>",
		Short:   "Apply a saved configuration version to a program pipeline",
		Example: `  gitee pipeline program history-apply 706 -E 2 -P 423 --history 3 --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pipelineIDArg(args[0])
			if err != nil {
				return err
			}
			if historyID <= 0 {
				return fmt.Errorf("--history <history-id> is required (see 'gitee pipeline program history')")
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Apply history %d to program pipeline %d?", historyID, id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			owned, conclusive, err := programPipelineOwnsHistory(f.Context, client, id, historyID)
			if err != nil {
				return fmt.Errorf("failed to verify history %d of program pipeline %d: %w", historyID, id, err)
			}
			if conclusive && !owned {
				return fmt.Errorf("history %d does not belong to program pipeline %d; refusing to apply (see 'gitee pipeline program history %d')", historyID, id, id)
			}
			if !conclusive {
				fmt.Fprintf(f.IOStreams.ErrOut, "warning: pipeline %d history exceeds %d versions; could not confirm history %d belongs to it\n", id, programHistoryScanPages*programHistoryScanSize, historyID)
			}
			p, err := client.ApplyProgramPipelineHistory(f.Context, historyID)
			if err != nil {
				return fmt.Errorf("failed to apply history version: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, p)
			}
			if p == nil {
				fmt.Fprintf(f.IOStreams.Out, "Applied history %d to program pipeline %d\n", historyID, id)
				return nil
			}
			fmt.Fprintf(f.IOStreams.Out, "Applied history %d to program pipeline %d (%s)\n", historyID, id, p.Name)
			return nil
		},
	}
	cmd.Flags().Int64Var(&historyID, "history", 0, "History record id to apply (required, see 'history')")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// programNoGroupName is the backend placeholder for pipelines that belong to no
// group; it is rendered as "-" in the list.
const programNoGroupName = "GITEE_PIPELINE_NO_GROUP_NAME"

// renderProgramPipelineList renders program pipeline summaries as an aligned
// table. Columns: ID (from the pipelineOps identifier), NAME, GROUP, STATUS,
// BUILD (#latest run), START, DURATION, TRIGGER, and UUID (truncated unless
// --uuid/--full is set).
func renderProgramPipelineList(f *cmdutil.Factory, data []giteego.PipelineSummaryVO, total int, showUUID, showFull bool) error {
	out := f.IOStreams.Out
	fmt.Fprintf(out, "Total: %d\n", total)
	rows := [][]string{{"ID", "PIPELINE", "GROUP", "STATUS", "BUILD", "START", "DURATION", "TRIGGER", "UUID"}}
	for _, p := range data {
		id, _ := pipelineIDFromIdentifier(p.Identifier)
		status, buildNum, start, dur, trigger := "-", "-", "-", "-", "-"
		if p.Build != nil {
			if p.Build.Status != "" {
				status = p.Build.Status
			}
			if p.Build.BuildNumber > 0 {
				buildNum = strconv.FormatInt(p.Build.BuildNumber, 10)
			}
			start = timeStr(p.Build.StartTime)
			dur = durationStr(p.Build.StartTime, p.Build.EndTime)
			if p.Build.Trigger != nil && p.Build.Trigger.Username != "" {
				trigger = p.Build.Trigger.Username
			}
		}
		grp := "-"
		if p.Group != nil && p.Group.Name != "" && p.Group.Name != programNoGroupName {
			grp = p.Group.Name
		}
		uuid := "-"
		if p.UUID != "" {
			uuid = p.UUID
			if !showUUID && !showFull && len(uuid) > 8 {
				uuid = uuid[:8] + "…"
			}
		}
		rows = append(rows, []string{
			strconv.FormatInt(id, 10),
			p.Name,
			grp,
			status,
			buildNum,
			start,
			dur,
			trigger,
			uuid,
		})
		if showFull && p.Group != nil && p.Group.ID > 0 {
			rows = append(rows, []string{
				"", fmt.Sprintf("gid:%d", p.Group.ID),
				groupLabels(p.Labels), "", "", "", "", "", "",
			})
		}
	}
	return cmdutil.WriteTableBordered(out, rows)
}

// groupLabels renders pipeline labels compactly (label1,label2).
func groupLabels(labels []giteego.LabelVO) string {
	if len(labels) == 0 {
		return ""
	}
	names := make([]string, 0, len(labels))
	for _, l := range labels {
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	return strings.Join(names, ",")
}
