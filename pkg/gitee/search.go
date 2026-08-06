package gitee

import (
	"context"
	"net/http"
	"strconv"
)

type SearchRepositoriesParams struct {
	Query    string
	Owner    string
	Fork     bool
	Language string
	Sort     string
	Order    string
	Page     int
	PerPage  int
}

func (p *SearchRepositoriesParams) toQuery() map[string]string {
	q := searchQuery(p.Query, p.Sort, p.Order, p.Page, p.PerPage)
	if p.Owner != "" {
		q["owner"] = p.Owner
	}
	if p.Fork {
		q["fork"] = strconv.FormatBool(p.Fork)
	}
	if p.Language != "" {
		q["language"] = p.Language
	}
	return q
}

func (c *Client) SearchRepositories(ctx context.Context, params *SearchRepositoriesParams) ([]Repository, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/search/repositories", params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var repos []Repository
	return repos, c.do(req, &repos)
}

type SearchIssuesParams struct {
	Query    string
	Repo     string
	Language string
	Label    string
	State    string
	Author   string
	Assignee string
	Sort     string
	Order    string
	Page     int
	PerPage  int
}

func (p *SearchIssuesParams) toQuery() map[string]string {
	q := searchQuery(p.Query, p.Sort, p.Order, p.Page, p.PerPage)
	optional := map[string]string{
		"repo": p.Repo, "language": p.Language, "label": p.Label,
		"state": p.State, "author": p.Author, "assignee": p.Assignee,
	}
	for key, value := range optional {
		if value != "" {
			q[key] = value
		}
	}
	return q
}

func (c *Client) SearchIssues(ctx context.Context, params *SearchIssuesParams) ([]Issue, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/search/issues", params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	return issues, c.do(req, &issues)
}

type SearchUsersParams struct {
	Query   string
	Sort    string
	Order   string
	Page    int
	PerPage int
}

func (p *SearchUsersParams) toQuery() map[string]string {
	return searchQuery(p.Query, p.Sort, p.Order, p.Page, p.PerPage)
}

func (c *Client) SearchUsersWithParams(ctx context.Context, params *SearchUsersParams) ([]User, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/search/users", params.toQuery(), nil)
	if err != nil {
		return nil, err
	}
	var users []User
	return users, c.do(req, &users)
}

func searchQuery(query, sort, order string, page, perPage int) map[string]string {
	q := map[string]string{"q": query}
	if sort != "" {
		q["sort"] = sort
	}
	if order != "" {
		q["order"] = order
	}
	if page > 0 {
		q["page"] = strconv.Itoa(page)
	}
	if perPage > 0 {
		q["per_page"] = strconv.Itoa(perPage)
	}
	return q
}
