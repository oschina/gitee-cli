package gitee

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ListReposParams struct {
	Visibility  string
	Affiliation string
	Type        string
	Sort        string
	Direction   string
	Q           string
	Page        int
	PerPage     int
}

func (p *ListReposParams) toQuery() map[string]string {
	q := map[string]string{}
	if p == nil {
		return q
	}
	if p.Visibility != "" {
		q["visibility"] = p.Visibility
	}
	if p.Affiliation != "" {
		q["affiliation"] = p.Affiliation
	}
	if p.Type != "" {
		q["type"] = p.Type
	}
	if p.Sort != "" {
		q["sort"] = p.Sort
	}
	if p.Direction != "" {
		q["direction"] = p.Direction
	}
	if p.Q != "" {
		q["q"] = p.Q
	}
	if p.Page > 0 {
		q["page"] = strconv.Itoa(p.Page)
	}
	if p.PerPage > 0 {
		q["per_page"] = strconv.Itoa(p.PerPage)
	}
	return q
}

func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*Repository, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s", owner, repo), nil, nil)
	if err != nil {
		return nil, err
	}
	var r Repository
	return &r, c.do(req, &r)
}

func (c *Client) ListUserRepos(ctx context.Context, params *ListReposParams) ([]Repository, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/user/repos", params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var repos []Repository
	return repos, c.do(req, &repos)
}

func (c *Client) ListAllUserRepos(ctx context.Context, params *ListReposParams) ([]Repository, error) {
	if params == nil {
		params = &ListReposParams{}
	}
	base := *params
	return paginateAll(ctx, func(ctx context.Context, page, perPage int) ([]Repository, error) {
		p := base
		p.Page = page
		p.PerPage = perPage
		return c.ListUserRepos(ctx, &p)
	})
}

func (c *Client) ListOwnerRepos(ctx context.Context, owner string, params *ListReposParams) ([]Repository, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/users/%s/repos", owner), params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var repos []Repository
	return repos, c.do(req, &repos)
}

type CreateRepoParams struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Private     bool   `json:"private"`
	HasIssues   bool   `json:"has_issues"`
	HasWiki     bool   `json:"has_wiki"`
	AutoInit    bool   `json:"auto_init"`
}

func (c *Client) CreateRepo(ctx context.Context, params *CreateRepoParams) (*Repository, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/user/repos", nil, params)
	if err != nil {
		return nil, err
	}
	var r Repository
	return &r, c.do(req, &r)
}

func (c *Client) DeleteRepo(ctx context.Context, owner, repo string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/repos/%s/%s", owner, repo), nil, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) ForkRepo(ctx context.Context, owner, repo, organization string) (*Repository, error) {
	body := map[string]string{}
	if organization != "" {
		body["organization"] = organization
	}
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/forks", owner, repo), nil, body)
	if err != nil {
		return nil, err
	}
	var r Repository
	return &r, c.do(req, &r)
}

// FileContent represents a file returned by the contents API.
type FileContent struct {
	Type    string `json:"type"`
	Content string `json:"content"` // base64-encoded
}

// GetFileContent fetches the content of a single file from a repository at the
// given ref (branch/tag/commit). Returns ("", nil) when the file does not exist.
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	query := map[string]string{}
	if ref != "" {
		query["ref"] = ref
	}
	req, err := c.newRequest(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path),
		query, nil)
	if err != nil {
		return "", err
	}

	// The API returns [] (empty array) when the path does not exist, so we
	// unmarshal into a FileContent and treat a non-"file" type as missing.
	var fc FileContent
	if err := c.do(req, &fc); err != nil {
		return "", err
	}
	if fc.Type != "file" || fc.Content == "" {
		return "", nil
	}

	decoded, err := base64DecodeContent(fc.Content)
	if err != nil {
		return "", fmt.Errorf("gitee: decode file content: %w", err)
	}
	return decoded, nil
}

func base64DecodeContent(s string) (string, error) {
	s = strings.ReplaceAll(s, "\n", "")
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
