package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewAPICmd(f *cmdutil.Factory) *cobra.Command {
	var (
		method      string
		headers     []string
		fields      []string
		rawBody     string
		pretty      bool
		searchQuery string
		searchLimit int
		searchPage  int
	)

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated request to the Gitee API",
		Long: `Make an authenticated request to any Gitee V5 API endpoint.

Use --search to find API endpoints by keyword (fetches the OpenAPI spec):
  gitee api --search "list issues"
  gitee api --search "create pull request"
  gitee api --search "issues" --limit 50
  gitee api --search "issues" --page 2

API reference: https://gitee.com/api/v5/swagger

Common endpoints:
  /user                                    - authenticated user
  /users/{username}                        - user profile
  /repos/{owner}/{repo}                    - repository info
  /repos/{owner}/{repo}/issues             - issues
  /repos/{owner}/{repo}/pulls              - pull requests
  /repos/{owner}/{repo}/releases           - releases
  /repos/{owner}/{repo}/contents/{path}    - file content
  /search/issues?q={query}                 - search issues
  /search/users?q={query}                  - search users`,
		Example: `  gitee api --search "create issue"
  gitee api --search "list pull requests"
  gitee api --search "issues" --limit 50
  gitee api --search "issues" --page 2
  gitee api /user
  gitee api /repos/owner/repo
  gitee api /repos/owner/repo/issues?state=open&per_page=5
  gitee api /repos/owner/repo/issues --method POST -f title="Bug" -f body="desc"
  gitee api /repos/owner/repo/pulls -X PATCH --body '{"state":"closed"}'
  gitee api /search/issues?q=bug -p`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Search mode: no endpoint arg, just --search
			if searchQuery != "" {
				client := &http.Client{Timeout: 15 * time.Second}
				doc, err := fetchSwaggerDoc(client)
				if err != nil {
					return err
				}
				all := searchEndpoints(doc, searchQuery, 0)
				pageMatches, total := paginate(all, searchPage, searchLimit)
				printEndpoints(f.IOStreams.Out, pageMatches)
				printPagination(f.IOStreams.Out, total, searchPage, searchLimit)
				return nil
			}

			if len(args) == 0 {
				return cmdutil.FlagErrorf("endpoint is required (use --search to find endpoints)")
			}
			endpoint := args[0]

			hostname := f.Hostname
			baseURL := config.APIPrefixForHost(hostname)
			fullURL := buildURL(baseURL, endpoint)

			bodyData, err := buildBody(fields, rawBody)
			if err != nil {
				return err
			}

			req, err := http.NewRequest(strings.ToUpper(method), fullURL, bodyData)
			if err != nil {
				return fmt.Errorf("invalid request: %w", err)
			}

			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")

			for _, h := range headers {
				k, v, ok := strings.Cut(h, ":")
				if !ok {
					return fmt.Errorf("invalid header %q: expected Key: Value", h)
				}
				req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			}

			if pretty && json.Valid(body) {
				var buf bytes.Buffer
				if err := json.Indent(&buf, body, "", "  "); err == nil {
					body = buf.Bytes()
				}
			}

			fmt.Fprintf(f.IOStreams.Out, "%s\n", body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Add a request header (Key: Value)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Add a JSON body field (key=value)")
	cmd.Flags().StringVar(&rawBody, "body", "", "Raw JSON request body")
	cmd.Flags().BoolVarP(&pretty, "pretty", "p", false, "Pretty-print JSON response")
	cmd.Flags().StringVar(&searchQuery, "search", "", "Search the Gitee API for endpoints matching a keyword")
	cmd.Flags().IntVar(&searchLimit, "limit", 20, "Number of search results per page (default: 20)")
	cmd.Flags().IntVar(&searchPage, "page", 1, "Page number for search results")
	return cmd
}

func buildURL(base, endpoint string) string {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func buildBody(fields []string, rawBody string) (io.Reader, error) {
	if rawBody != "" && len(fields) > 0 {
		return nil, fmt.Errorf("--body and --field cannot be used together")
	}
	if rawBody != "" {
		return strings.NewReader(rawBody), nil
	}
	if len(fields) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			return nil, fmt.Errorf("invalid field %q: expected key=value", f)
		}
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}
