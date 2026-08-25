package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// NewPipelineCmd returns the gitee-go (repository pipeline / GitOps) command.
func NewPipelineCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Manage gitee-go repository pipelines",
		Long: `Manage gitee-go repository pipelines (GitOps), including pipeline
configuration, builds, stages and jobs.

Gitee-go is the CI/CD pipeline service for Gitee; commands resolve the go-api
gateway from the active host automatically (gitee.com uses go-api.gitee.com,
*.runjs.cn hosts use local-pipe-api.runjs.cn, other deployments keep the
Gitee host with a /go-api path segment).`,
	}

	cmd.PersistentFlags().StringP("repo", "R", "", "owner/repo (required)")

	cmd.AddCommand(newPipelineListCmd(f))
	cmd.AddCommand(newPipelineViewCmd(f))
	cmd.AddCommand(newPipelineExampleCmd(f))
	cmd.AddCommand(newPipelineRunCmd(f))
	cmd.AddCommand(newPipelineCommitCmd(f))
	cmd.AddCommand(newPipelineBuildCmd(f))
	cmd.AddCommand(newPipelinePluginCmd(f))
	cmd.AddCommand(newPipelineRequestCmd(f))
	cmd.AddCommand(newPipelineProgramCmd(f))
	return cmd
}

// resolveOwnerRepo requires an explicit -R owner/repo for repo pipelines;
// git-remote/default_repo inference is not used (matches the gitee CLI).
func resolveOwnerRepo(_ *cmdutil.Factory, cmd *cobra.Command) (string, string, error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	if repoFlag == "" {
		return "", "", fmt.Errorf("repo is required for pipeline commands: use -R owner/repo")
	}
	return cmdutil.ParseOwnerRepo(repoFlag)
}

// repoPipelineClientForService builds a gitee-go client for a service, scoped
// to the resolved owner/repo (the repo path sits between the go-api host and
// /gitee-go in the gateway URL).
func repoPipelineClientForService(f *cmdutil.Factory, owner, repo, service string) (*giteego.Client, error) {
	c, err := f.GoAPIClient(owner+"/"+repo, service)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ensureServiceOpen verifies gitee-go is enabled for the repo before a pipeline
// query. When it is not enabled, it errors out and points the user at the
// gitee-go enable page — there is no auto-open logic: the open URL cannot be
// driven from the CLI, so the user must open the page and enable gitee-go
// themselves, then re-run the command.
func ensureServiceOpen(f *cmdutil.Factory, _ *cobra.Command, owner, repo string) error {
	goClient, err := repoPipelineClientForService(f, owner, repo, giteego.ServiceIPipe)
	if err != nil {
		return err
	}
	svc, err := goClient.CheckServiceStatus(f.Context)
	if err != nil {
		return fmt.Errorf("check gitee-go service status: %w", err)
	}
	if svc != nil && svc.Enabled {
		return nil
	}

	openURL, _ := giteego.OpenURL(config.APIPrefixForHost(f.Hostname), owner, repo)
	if openURL == "" {
		return fmt.Errorf(i18n.T("pipeline.open_not_enabled"), owner, repo)
	}
	// Tf already sinks the args; wrap with a constant format for the vet
	// printf check.
	return fmt.Errorf("%s", i18n.Tf("pipeline.open_not_enabled_url", owner, repo, openURL))
}
