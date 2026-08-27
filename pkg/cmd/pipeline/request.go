package pipeline

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineRequestCmd builds the top-level `pipeline request` command. It has
// two modes:
//
//   - Pass an http(s) URL (e.g. a logger URL from `build view`): perform an
//     authenticated GET directly, no -R required.
//   - Pass a RemoteSelect component identifier + --type: resolve the plugin
//     scheme's RemoteSelect URL and call it (requires -R, like other pipeline
//     commands).
func newPipelineRequestCmd(f *cmdutil.Factory) *cobra.Command {
	var jobType string
	var params []string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "request <url|component> [--type <jobType>]",
		Short: "Perform an authenticated GET on a gitee-go URL or RemoteSelect component",
		Long: `Perform an authenticated GET against a gitee-go endpoint.

When the argument is an http(s) URL (e.g. a logger URL shown by 'pipeline build
view'), the request goes directly to that URL and no -R is needed.

Otherwise the argument is a RemoteSelect component identifier in a plugin scheme:
it resolves the scheme for --type, finds the component, and calls its
remote-select endpoint — this mode requires -R owner/repo.`,
		Example: `  gitee pipeline request "https://premium-k8s.gitee.cn/2/426/gitee-go/log-server/.../logs?recordUuid=..."
  gitee pipeline request PLUGIN_GCC_VERSION -R owner/repo --type gcc-build@v1.0.0 -p type=gcc`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := args[0]

			// URL mode: absolute http(s) URL, no -R required.
			if isAbsoluteURL(arg) {
				client, err := authOnlyClient(f)
				if err != nil {
					return err
				}
				body, err := client.FetchAbsoluteURL(f.Context, arg)
				if err != nil {
					return fmt.Errorf("failed to fetch %s: %w", arg, err)
				}
				if jsonFields != "" {
					return cmdutil.WriteJSON(f.IOStreams.Out, string(body))
				}
				fmt.Fprintf(f.IOStreams.Out, "%s\n", strings.TrimSpace(string(body)))
				return nil
			}

			// Component mode: RemoteSelect in a plugin scheme (requires -R).
			client, err := resolvePluginClient(f, cmd)
			if err != nil {
				return err
			}
			scheme, err := client.GetPluginScheme(f.Context, jobType)
			if err != nil {
				return fmt.Errorf("failed to get plugin scheme: %w", err)
			}
			if scheme == nil {
				return fmt.Errorf("no scheme for job type %q", jobType)
			}
			comp := findRemoteSelectComponent(scheme.Config, arg)
			if comp == nil {
				return fmt.Errorf("no RemoteSelect component %q in plugin %q", arg, jobType)
			}
			if comp.URL == nil || comp.URL.GitOps == "" {
				return fmt.Errorf("component %q has no gitOps remote-select URL", arg)
			}

			// Query params: scheme url.params (after ${ref} substitution), then
			// user -p values on top.
			query := map[string]string{}
			for k, v := range comp.Params {
				query[k] = expandRef(v, params)
			}
			for _, kv := range params {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("invalid param %q, expected key=value", kv)
				}
				query[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
			resp, err := client.RemoteSelectOptions(f.Context, comp.URL.GitOps, query)
			if err != nil {
				return fmt.Errorf("failed to fetch remote options: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.ComponentOptionVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, resp.List)
				}
				result, err := cmdutil.SelectFields(resp.List, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			if len(resp.List) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pipeline.no_options"))
				return nil
			}
			rows := [][]string{{"KEY", "LABEL", "VALUE"}}
			for _, opt := range resp.List {
				rows = append(rows, []string{opt.Key, opt.Label, componentValue(opt)})
			}
			return cmdutil.WriteTableBordered(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().StringVar(&jobType, "type", "", "Plugin JSON job type (required in component mode, from 'plugin list')")
	cmd.Flags().StringArrayVarP(&params, "param", "p", nil, "Endpoint param as key=value (repeated; overrides/supplies url.params)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.ComponentOptionVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// authOnlyClient builds a gitee-go client with auth (token) but no repo
// path — used to fetch absolute URLs that need a session but are not repo-scoped.
func authOnlyClient(f *cmdutil.Factory) (*giteego.Client, error) {
	c, err := f.GoAPIClient("", giteego.ServiceIPipe)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// isAbsoluteURL reports whether s is an http(s) absolute URL.
func isAbsoluteURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// findRemoteSelectComponent returns the first config (or nested children)
// component with the given identifier.
func findRemoteSelectComponent(config []giteego.PluginComponentScheme, identifier string) *giteego.PluginComponentScheme {
	for i := range config {
		if config[i].Identifier == identifier {
			return &config[i]
		}
		for j := range config[i].Children {
			if config[i].Children[j].Identifier == identifier {
				return &config[i].Children[j]
			}
		}
	}
	return nil
}

// expandRef substitutes ${identifier} references in a template using the given
// key=value pairs, e.g. "${certificate}" -> value of a "-p certificate=...".
func expandRef(tmpl string, kvPairs []string) string {
	for _, kv := range kvPairs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		ref := "${" + strings.TrimSpace(k) + "}"
		if strings.Contains(tmpl, ref) {
			tmpl = strings.ReplaceAll(tmpl, ref, strings.TrimSpace(v))
		}
	}
	return tmpl
}

func componentValue(opt giteego.ComponentOptionVO) string {
	if opt.Value == nil {
		return opt.Key
	}
	if s, ok := opt.Value.(string); ok {
		if s == "" {
			return opt.Key
		}
		return s
	}
	return fmt.Sprintf("%v", opt.Value)
}
