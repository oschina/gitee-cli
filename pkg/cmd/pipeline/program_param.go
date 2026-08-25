package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineProgramParamCmd returns the `pipeline program param` command group
// for project (enterprise) parameter templates. These are reusable parameter
// sets scoped to the -E/-P project, listed/shared across program pipelines.
func newPipelineProgramParamCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "param",
		Short: "Manage program parameter templates",
		Long:  `List, create, view, update, clone and delete gitee-go program parameter templates.`,
	}
	cmd.AddCommand(newPipelineProgramParamListCmd(f))
	cmd.AddCommand(newPipelineProgramParamViewCmd(f))
	cmd.AddCommand(newPipelineProgramParamCreateCmd(f))
	cmd.AddCommand(newPipelineProgramParamEditCmd(f))
	cmd.AddCommand(newPipelineProgramParamCloneCmd(f))
	cmd.AddCommand(newPipelineProgramParamDeleteCmd(f))
	return cmd
}

func programParamIDArg(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("param id is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid param id %q", args[0])
	}
	return id, nil
}

func newPipelineProgramParamListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List program parameter templates",
		Example: `  gitee pipeline program param list -E 2 -P 423
  gitee pipeline program param list -E 2 -P 423 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			params, err := client.ListProgramParams(f.Context)
			if err != nil {
				return fmt.Errorf("failed to list program params: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.ParamVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, params)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, params, fields)
			}

			if len(params) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pipeline.program.no_params"))
				return nil
			}
			rows := [][]string{{"ID", "NAME", "DESCRIPTION", "PARAMS", "UPDATED"}}
			for _, p := range params {
				rows = append(rows, []string{
					strconv.FormatInt(p.ID, 10),
					p.Name,
					oneLine(p.Description),
					strconv.Itoa(len(p.Parameters)),
					timeStr(p.UpdateTime),
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ParamVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramParamViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "view <param-id>",
		Short: "View a program parameter template",
		Example: `  gitee pipeline program param view 12 -E 2 -P 423
  gitee pipeline program param view 12 -E 2 -P 423 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programParamIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			p, err := client.GetProgramParam(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get program param: %w", err)
			}
			if p == nil {
				return fmt.Errorf("program param %d not found", id)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.ParamVO](f.IOStreams.Out)
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
			fmt.Fprintf(out, "ID:          %d\n", p.ID)
			fmt.Fprintf(out, "Name:        %s\n", p.Name)
			if p.Description != "" {
				fmt.Fprintf(out, "Description: %s\n", p.Description)
			}
			printConfigParams(out, i18n.T("pipeline.params_pipeline"), p.Parameters)
			if len(p.Labels) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Labels")
				for _, l := range p.Labels {
					fmt.Fprintln(out, "  "+l.Name)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ParamVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// paramFlagsFromCMD builds the ParamRequest body from shared flags.
func paramRequestFromFlags(name, description, body string) (giteego.ParamRequest, error) {
	req := giteego.ParamRequest{Name: name, Description: description}
	if body != "" {
		data, err := os.ReadFile(body)
		if err != nil {
			return req, fmt.Errorf("failed to read body file: %w", err)
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return req, fmt.Errorf("failed to parse body JSON: %w", err)
		}
		if name != "" {
			req.Name = name
		}
		if description != "" {
			req.Description = description
		}
	}
	return req, nil
}

func attachParamFlags(cmd *cobra.Command, name, description, body *string) {
	cmd.Flags().StringVar(name, "name", "", "Parameter template name")
	cmd.Flags().StringVar(description, "description", "", "Parameter template description")
	cmd.Flags().StringVar(body, "body", "", "JSON body file to submit (ParamRequest)")
}

func newPipelineProgramParamCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var name, description, body string
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a program parameter template",
		Example: `  gitee pipeline program param create -E 2 -P 423 --name "build-params"
  gitee pipeline program param create -E 2 -P 423 --body ./params.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := paramRequestFromFlags(name, description, body)
			if err != nil {
				return err
			}
			if req.Name == "" {
				return fmt.Errorf("--name is required")
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			p, err := client.CreateProgramParam(f.Context, req)
			if err != nil {
				return fmt.Errorf("failed to create program param: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, p)
			}
			fmt.Fprintf(f.IOStreams.Out, "Created program param %d (%s)\n", p.ID, p.Name)
			return nil
		},
	}
	attachParamFlags(cmd, &name, &description, &body)
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ParamVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramParamEditCmd(f *cmdutil.Factory) *cobra.Command {
	var name, description, body string
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "edit <param-id>",
		Aliases: []string{"update"},
		Short:   "Edit a program parameter template",
		Example: `  gitee pipeline program param edit 12 -E 2 -P 423 --name "new-name"
  gitee pipeline program param edit 12 -E 2 -P 423 --body ./params.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programParamIDArg(args)
			if err != nil {
				return err
			}
			req, err := paramRequestFromFlags(name, description, body)
			if err != nil {
				return err
			}
			if req.Name == "" && req.Description == "" && len(req.Parameters) == 0 {
				return fmt.Errorf("nothing to edit: pass --name, --description or --body")
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			p, err := client.UpdateProgramParam(f.Context, id, req)
			if err != nil {
				return fmt.Errorf("failed to update program param: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, p)
			}
			fmt.Fprintf(f.IOStreams.Out, "Edited program param %d (%s)\n", p.ID, p.Name)
			return nil
		},
	}
	attachParamFlags(cmd, &name, &description, &body)
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ParamVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramParamCloneCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "clone <param-id>",
		Short:   "Clone a program parameter template",
		Example: `  gitee pipeline program param clone 12 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programParamIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			p, err := client.CloneProgramParam(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to clone program param: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, p)
			}
			fmt.Fprintf(f.IOStreams.Out, "Cloned program param %d from %d\n", p.ID, id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ParamVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramParamDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <param-id>",
		Short:   "Delete a program parameter template",
		Example: `  gitee pipeline program param delete 12 -E 2 -P 423 --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programParamIDArg(args)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Delete program param %d?", id))
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
			if err := client.DeleteProgramParam(f.Context, id); err != nil {
				return fmt.Errorf("failed to delete program param: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Deleted program param %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}
