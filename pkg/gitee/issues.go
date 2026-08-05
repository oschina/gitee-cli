package gitee

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

type ListIssuesParams struct {
	State     string
	Labels    string
	Q         string
	Assignee  string
	Sort      string
	Direction string
	Since     string
	Page      int
	PerPage   int
}

func (p *ListIssuesParams) toQuery() map[string]string {
	q := map[string]string{}
	if p == nil {
		return q
	}
	if p.State != "" {
		q["state"] = p.State
	}
	if p.Labels != "" {
		q["labels"] = p.Labels
	}
	if p.Q != "" {
		q["q"] = p.Q
	}
	if p.Assignee != "" {
		q["assignee"] = p.Assignee
	}
	if p.Sort != "" {
		q["sort"] = p.Sort
	}
	if p.Direction != "" {
		q["direction"] = p.Direction
	}
	if p.Since != "" {
		q["since"] = p.Since
	}
	if p.Page > 0 {
		q["page"] = strconv.Itoa(p.Page)
	}
	if p.PerPage > 0 {
		q["per_page"] = strconv.Itoa(p.PerPage)
	}
	return q
}

func (c *Client) ListRepoIssues(ctx context.Context, owner, repo string, params *ListIssuesParams) ([]Issue, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/issues", owner, repo), params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	return issues, c.do(req, &issues)
}

func (c *Client) ListAllIssues(ctx context.Context, owner, repo string, params *ListIssuesParams) ([]Issue, error) {
	if params == nil {
		params = &ListIssuesParams{}
	}
	base := *params
	return paginateAll(ctx, func(ctx context.Context, page, perPage int) ([]Issue, error) {
		p := base
		p.Page = page
		p.PerPage = perPage
		return c.ListRepoIssues(ctx, owner, repo, &p)
	})
}

func (c *Client) GetIssue(ctx context.Context, owner, repo, number string) (*Issue, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number), nil, nil)
	if err != nil {
		return nil, err
	}
	var issue Issue
	return &issue, c.do(req, &issue)
}

type CreateIssueParams struct {
	Repo     string `json:"repo"`
	Title    string `json:"title"`
	Body     string `json:"body,omitempty"`
	Assignee string `json:"assignee,omitempty"`
	Labels   string `json:"labels,omitempty"`
}

func (c *Client) CreateIssue(ctx context.Context, owner, repo string, params *CreateIssueParams) (*Issue, error) {
	params.Repo = repo
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/issues", owner), nil, params)
	if err != nil {
		return nil, err
	}
	var issue Issue
	return &issue, c.do(req, &issue)
}

type UpdateIssueParams struct {
	Repo     string `json:"repo"`
	Title    string `json:"title,omitempty"`
	Body     string `json:"body,omitempty"`
	State    string `json:"state,omitempty"`
	Assignee string `json:"assignee,omitempty"`
}

func (c *Client) UpdateIssue(ctx context.Context, owner, repo, number string, params *UpdateIssueParams) (*Issue, error) {
	params.Repo = repo
	req, err := c.newRequest(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/issues/%s", owner, number), nil, params)
	if err != nil {
		return nil, err
	}
	var issue Issue
	return &issue, c.do(req, &issue)
}

func (c *Client) CreateIssueComment(ctx context.Context, owner, repo, number string, body string) (*Comment, error) {
	payload := map[string]string{"body": body}
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, number), nil, payload)
	if err != nil {
		return nil, err
	}
	var comment Comment
	return &comment, c.do(req, &comment)
}

func (c *Client) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/labels", owner, repo), nil, nil)
	if err != nil {
		return nil, err
	}
	var labels []Label
	return labels, c.do(req, &labels)
}
