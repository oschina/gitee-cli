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

// newPipelineProgramTemplateCmd returns the `pipeline program template` command
// group for project (enterprise) pipeline templates. Templates bundle a full
// pipeline configuration (Config *PipelineVO) plus a name/description.
func newPipelineProgramTemplateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage program pipeline templates",
		Long:  `List, create, view, update, delete, enable, disable and inspect gitee-go program pipeline templates.`,
	}
	cmd.AddCommand(newPipelineProgramTemplateListCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateViewCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateCreateCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateEditCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateDeleteCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateEnableCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateDisableCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateCategoriesCmd(f))
	return cmd
}

func programTemplateIDArg(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("template id is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid template id %q", args[0])
	}
	return id, nil
}

func newPipelineProgramTemplateListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List program pipeline templates",
		Example: `  gitee pipeline program template list -E 2 -P 423
  gitee pipeline program template list -E 2 -P 423 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			templates, err := client.ListProgramTemplates(f.Context)
			if err != nil {
				return fmt.Errorf("failed to list program templates: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.ProgramPipelineTemplateVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, templates)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, templates, fields)
			}

			if len(templates) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pipeline.program.no_templates"))
				return nil
			}
			rows := [][]string{{"ID", "NAME", "CATEGORY", "STATUS", "UPDATED"}}
			for _, t := range templates {
				category := "-"
				if t.Category != nil {
					category = t.Category.Name
				}
				status := "enabled"
				if t.Disabled {
					status = "disabled"
				}
				rows = append(rows, []string{
					strconv.FormatInt(t.ID, 10),
					t.Name,
					category,
					status,
					timeStr(t.UpdateTime),
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ProgramPipelineTemplateVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramTemplateViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "view <template-id>",
		Short: "View a program pipeline template",
		Example: `  gitee pipeline program template view 12 -E 2 -P 423
  gitee pipeline program template view 12 -E 2 -P 423 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programTemplateIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			t, err := client.GetProgramTemplate(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get program template: %w", err)
			}
			if t == nil {
				return fmt.Errorf("program template %d not found", id)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.ProgramPipelineTemplateVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, t)
				}
				result, err := cmdutil.SelectFields(t, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "ID:          %d\n", t.ID)
			fmt.Fprintf(out, "Name:        %s\n", t.Name)
			if t.Description != "" {
				fmt.Fprintf(out, "Description: %s\n", t.Description)
			}
			if t.Category != nil {
				fmt.Fprintf(out, "Category:    %s\n", t.Category.Name)
			}
			status := "enabled"
			if t.Disabled {
				status = "disabled"
			}
			fmt.Fprintf(out, "Status:      %s\n", status)
			if t.Config != nil {
				fmt.Fprintf(out, "Ref:         %s\n", t.Config.Ref)
				printConfigParams(out, i18n.T("pipeline.params_pipeline"), t.Config.Parameters)
				if len(t.Config.Stages) > 0 {
					fmt.Fprintln(out)
					fmt.Fprintln(out, "Stages")
					for _, s := range t.Config.Stages {
						fmt.Fprintln(out, "  "+s.Name)
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ProgramPipelineTemplateVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// templateRequestFromFlags builds the template body, letting the --config file
// supply the full pipeline configuration (name/description union when set).
func templateRequestFromFlags(name, description, config string) (giteego.ProgramPipelineTemplateRequest, error) {
	req := giteego.ProgramPipelineTemplateRequest{Name: name, Description: description}
	if config != "" {
		data, err := os.ReadFile(config)
		if err != nil {
			return req, fmt.Errorf("failed to read config file: %w", err)
		}
		var cfg giteego.PipelineVO
		if err := json.Unmarshal(data, &cfg); err != nil {
			return req, fmt.Errorf("failed to parse config JSON: %w", err)
		}
		req.Config = &cfg
	}
	return req, nil
}

func newPipelineProgramTemplateCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var name, description, config string
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a program pipeline template",
		Example: `  gitee pipeline program template create -E 2 -P 423 --name "deploy-template"
  gitee pipeline program template create -E 2 -P 423 --name deploy --config ./pipeline.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := templateRequestFromFlags(name, description, config)
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
			t, err := client.CreateProgramTemplate(f.Context, req)
			if err != nil {
				return fmt.Errorf("failed to create program template: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, t)
			}
			fmt.Fprintf(f.IOStreams.Out, "Created program template %d (%s)\n", t.ID, t.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Template name")
	cmd.Flags().StringVar(&description, "description", "", "Template description")
	cmd.Flags().StringVar(&config, "config", "", "JSON body file supplying the pipeline configuration (PipelineVO)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ProgramPipelineTemplateVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramTemplateEditCmd(f *cmdutil.Factory) *cobra.Command {
	var name, description, config string
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "edit <template-id>",
		Aliases: []string{"update"},
		Short:   "Edit a program pipeline template",
		Example: `  gitee pipeline program template edit 12 -E 2 -P 423 --name "new-name"
  gitee pipeline program template edit 12 -E 2 -P 423 --config ./pipeline.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programTemplateIDArg(args)
			if err != nil {
				return err
			}
			req, err := templateRequestFromFlags(name, description, config)
			if err != nil {
				return err
			}
			if req.Name == "" && req.Description == "" && req.Config == nil {
				return fmt.Errorf("nothing to edit: pass --name, --description or --config")
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			t, err := client.UpdateProgramTemplate(f.Context, id, req)
			if err != nil {
				return fmt.Errorf("failed to update program template: %w", err)
			}
			if jsonFields != "" {
				return cmdutil.WriteJSON(f.IOStreams.Out, t)
			}
			fmt.Fprintf(f.IOStreams.Out, "Edited program template %d (%s)\n", t.ID, t.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New template name")
	cmd.Flags().StringVar(&description, "description", "", "New template description")
	cmd.Flags().StringVar(&config, "config", "", "JSON body file supplying the pipeline configuration (PipelineVO)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ProgramPipelineTemplateVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineProgramTemplateDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <template-id>",
		Short:   "Delete a program pipeline template",
		Example: `  gitee pipeline program template delete 12 -E 2 -P 423 --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programTemplateIDArg(args)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Delete program template %d?", id))
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
			if err := client.DeleteProgramTemplate(f.Context, id); err != nil {
				return fmt.Errorf("failed to delete program template: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Deleted program template %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newPipelineProgramTemplateEnableCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <template-id>",
		Short:   "Enable a disabled program pipeline template",
		Example: `  gitee pipeline program template enable 12 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programTemplateIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if err := client.EnableProgramTemplate(f.Context, id); err != nil {
				return fmt.Errorf("failed to enable program template: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Enabled program template %d\n", id)
			return nil
		},
	}
	return cmd
}

func newPipelineProgramTemplateDisableCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "disable <template-id>",
		Short:   "Disable a program pipeline template",
		Example: `  gitee pipeline program template disable 12 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := programTemplateIDArg(args)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Disable program template %d?", id))
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
			if err := client.DisableProgramTemplate(f.Context, id); err != nil {
				return fmt.Errorf("failed to disable program template: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Disabled program template %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newPipelineProgramTemplateCategoriesCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "List program pipeline template categories",
		Example: `  gitee pipeline program template categories -E 2 -P 423
  gitee pipeline program template categories -E 2 -P 423 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			categories, err := client.ListProgramTemplateCategories(f.Context)
			if err != nil {
				return fmt.Errorf("failed to list template categories: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineTemplateCategoryVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, categories)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, categories, fields)
			}

			if len(categories) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pipeline.program.no_categories"))
				return nil
			}
			rows := [][]string{{"ID", "NAME", "ICON", "SORT", "UPDATED"}}
			for _, c := range categories {
				rows = append(rows, []string{
					strconv.FormatInt(c.ID, 10),
					c.Name,
					c.Icon,
					strconv.FormatInt(c.Sort, 10),
					timeStr(c.UpdateTime),
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineTemplateCategoryVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
