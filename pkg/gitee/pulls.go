package gitee

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type ListPullsParams struct {
	State           string
	Head            string
	Base            string
	Sort            string
	Since           string
	Direction       string
	MilestoneNumber int
	Labels          string
	Page            int
	PerPage         int
	Author          string
	Assignee        string
	Tester          string
}

func (p *ListPullsParams) toQuery() map[string]string {
	q := map[string]string{}
	if p == nil {
		return q
	}
	if p.State != "" {
		q["state"] = p.State
	}
	if p.Head != "" {
		q["head"] = p.Head
	}
	if p.Base != "" {
		q["base"] = p.Base
	}
	if p.Sort != "" {
		q["sort"] = p.Sort
	}
	if p.Since != "" {
		q["since"] = p.Since
	}
	if p.Direction != "" {
		q["direction"] = p.Direction
	}
	if p.MilestoneNumber > 0 {
		q["milestone_number"] = strconv.Itoa(p.MilestoneNumber)
	}
	if p.Labels != "" {
		q["labels"] = p.Labels
	}
	if p.Page > 0 {
		q["page"] = strconv.Itoa(p.Page)
	}
	if p.PerPage > 0 {
		q["per_page"] = strconv.Itoa(p.PerPage)
	}
	if p.Author != "" {
		q["author"] = p.Author
	}
	if p.Assignee != "" {
		q["assignee"] = p.Assignee
	}
	if p.Tester != "" {
		q["tester"] = p.Tester
	}
	return q
}

func (c *Client) ListPulls(ctx context.Context, owner, repo string, params *ListPullsParams) ([]PullRequest, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var prs []PullRequest
	return prs, c.do(req, &prs)
}

func (c *Client) ListAllPulls(ctx context.Context, owner, repo string, params *ListPullsParams) ([]PullRequest, error) {
	if params == nil {
		params = &ListPullsParams{}
	}
	base := *params
	return paginateAll(ctx, func(ctx context.Context, page, perPage int) ([]PullRequest, error) {
		p := base
		p.Page = page
		p.PerPage = perPage
		return c.ListPulls(ctx, owner, repo, &p)
	})
}

func (c *Client) GetPull(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number), nil, nil)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	return &pr, c.do(req, &pr)
}

type CreatePullParams struct {
	Title             string `json:"title"`
	Head              string `json:"head"`
	Base              string `json:"base"`
	Body              string `json:"body,omitempty"`
	Draft             bool   `json:"draft"`
	PruneSourceBranch bool   `json:"prune_source_branch"`
	Assignees         string `json:"assignees,omitempty"`
	Testers           string `json:"testers,omitempty"`
	AssigneesNumber   int    `json:"assignees_number,omitempty"`
	TestersNumber     int    `json:"testers_number,omitempty"`
}

func (c *Client) CreatePull(ctx context.Context, owner, repo string, params *CreatePullParams) (*PullRequest, error) {
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), nil, params)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	return &pr, c.do(req, &pr)
}

type UpdatePullParams struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	State string `json:"state,omitempty"`
	Base  string `json:"base,omitempty"`
}

func (c *Client) UpdatePull(ctx context.Context, owner, repo string, number int, params *UpdatePullParams) (*PullRequest, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number), nil, params)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	return &pr, c.do(req, &pr)
}

type MergePullParams struct {
	MergeMethod       string `json:"merge_method"`
	PruneSourceBranch bool   `json:"prune_source_branch"`
}

func (c *Client) MergePull(ctx context.Context, owner, repo string, number int, mergeMethod string, pruneSourceBranch bool) error {
	if mergeMethod == "" {
		mergeMethod = "merge"
	}
	params := MergePullParams{MergeMethod: mergeMethod, PruneSourceBranch: pruneSourceBranch}
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, number), nil, params)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

type ReviewPullParams struct {
	Force bool `json:"force,omitempty"`
}

func (c *Client) ReviewPull(ctx context.Context, owner, repo string, number int, params *ReviewPullParams) error {
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls/%d/review", owner, repo, number), nil, params)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) CreatePullComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error) {
	payload := map[string]string{"body": body}
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number), nil, payload)
	if err != nil {
		return nil, err
	}
	var comment Comment
	return &comment, c.do(req, &comment)
}

func (c *Client) GetPullDiff(ctx context.Context, owner, repo string, number int) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls/%d.diff", owner, repo, number), nil, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gitee: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", parseErrorResponse(resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gitee: read diff: %w", err)
	}
	return string(data), nil
}

func (c *Client) GetPullDiffFiles(ctx context.Context, owner, repo string, number int) ([]DiffFile, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls/%d/files", owner, repo, number), nil, nil)
	if err != nil {
		return nil, err
	}
	var files []DiffFile
	return files, c.do(req, &files)
}
