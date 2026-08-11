package release

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newReleaseEditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		tagName    string
		name       string
		body       string
		prerelease bool
		jsonOut    bool
	)

	cmd := &cobra.Command{
		Use:   "edit <id-or-tag>",
		Short: "Edit a release",
		Long: `Edit a release's tag, name, description, or pre-release status.

At least one editing flag must be provided in non-interactive mode.
The existing release is loaded before updating because the Gitee v5 API
requires all editable release fields in every update request.`,
		Example: `  gitee release edit v1.2.0 --name "Version 1.2"
  gitee release edit 42 --tag v1.2.1 --body "Updated notes"
  gitee release edit 42 --prerelease
  gitee release edit 42 --prerelease=false
  gitee release edit v1.2.0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tagChanged := cmd.Flags().Changed("tag")
			nameChanged := cmd.Flags().Changed("name")
			bodyChanged := cmd.Flags().Changed("body")
			prereleaseChanged := cmd.Flags().Changed("prerelease")
			interactive := !tagChanged && !nameChanged && !bodyChanged && !prereleaseChanged
			if interactive && !f.IOStreams.IsStdinTerminal() {
				return cmdutil.FlagErrorf("at least one of --tag, --name, --body, --prerelease is required")
			}

			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			current, err := getReleaseForEdit(f, client, owner, repo, args[0])
			if err != nil {
				return err
			}
			if !tagChanged {
				tagName = current.TagName
			}
			if !nameChanged {
				name = current.Name
			}
			if !bodyChanged {
				body = current.Body
			}
			if !prereleaseChanged {
				prerelease = current.Prerelease
			}

			if interactive {
				if f.IsTUI() {
					if err := releaseEditForm(&tagName, &name, &body, &prerelease); err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
				} else {
					tagName, name, body, prerelease, err = promptReleaseEdit(f, tagName, name, body, prerelease)
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
				}
			}

			if strings.TrimSpace(tagName) == "" {
				return cmdutil.FlagErrorf("tag cannot be empty")
			}
			if strings.TrimSpace(name) == "" {
				return cmdutil.FlagErrorf("name cannot be empty")
			}

			updated, err := client.UpdateRelease(f.Context, owner, repo, current.ID, &gitee.UpdateReleaseParams{
				TagName:    tagName,
				Name:       name,
				Body:       body,
				Prerelease: prerelease,
			})
			if err != nil {
				return fmt.Errorf("failed to edit release %s: %w", args[0], err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, updated)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("release.updated", updated.TagName, updated.Name))
			return nil
		},
	}

	cmd.Flags().StringVar(&tagName, "tag", "", "New tag name")
	cmd.Flags().StringVarP(&name, "name", "n", "", "New release name")
	cmd.Flags().StringVarP(&body, "body", "b", "", "New release description")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Set or clear pre-release status")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func getReleaseForEdit(f *cmdutil.Factory, client *gitee.Client, owner, repo, idOrTag string) (*gitee.Release, error) {
	if id, err := strconv.Atoi(idOrTag); err == nil {
		r, err := client.GetRelease(f.Context, owner, repo, id)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch release %s: %w", idOrTag, err)
		}
		return r, nil
	}
	r, err := client.GetReleaseByTag(f.Context, owner, repo, idOrTag)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release %s: %w", idOrTag, err)
	}
	return r, nil
}

func releaseEditForm(tagName, name, body *string, prerelease *bool) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(i18n.T("form.release.tag")).
				Value(tagName).
				Validate(nonEmptyReleaseField("tag")),
			huh.NewInput().
				Title(i18n.T("form.release.name")).
				Value(name).
				Validate(nonEmptyReleaseField("name")),
			huh.NewText().
				Title(i18n.T("form.release.body")).
				Editor(strings.Fields(config.Editor())...).
				Value(body),
			huh.NewConfirm().
				Title(i18n.T("form.release.prerelease")).
				Value(prerelease),
		),
	).Run()
}

func nonEmptyReleaseField(name string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s cannot be empty", name)
		}
		return nil
	}
}

func promptReleaseEdit(f *cmdutil.Factory, tagName, name, body string, prerelease bool) (string, string, string, bool, error) {
	updatedTag, err := cmdutil.AskInput(i18n.T("form.release.tag"), tagName, false)
	if err != nil {
		return tagName, name, body, prerelease, err
	}
	updatedName, err := cmdutil.AskInput(i18n.T("form.release.name"), name, false)
	if err != nil {
		return tagName, name, body, prerelease, err
	}
	updatedBody, err := cmdutil.OpenEditor(f.IOStreams, "release-body-*.md", body)
	if err != nil {
		return tagName, name, body, prerelease, fmt.Errorf("could not open editor: %w", err)
	}

	preLabel := i18n.T("release.status_prerelease")
	stableLabel := i18n.T("release.status_stable")
	options := []string{stableLabel, preLabel}
	if prerelease {
		options = []string{preLabel, stableLabel}
	}
	status, err := cmdutil.AskSelect(i18n.T("release.status_prompt"), options, false)
	if err != nil {
		return tagName, name, body, prerelease, err
	}
	confirmed, err := cmdutil.AskConfirm(i18n.T("release.confirm_edit"), false)
	if err != nil {
		return tagName, name, body, prerelease, err
	}
	if !confirmed {
		return tagName, name, body, prerelease, huh.ErrUserAborted
	}
	return updatedTag, updatedName, updatedBody, status == preLabel, nil
}
