package gitee

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type SearchRepositoriesParams struct {
	Query    string
	Language string
	Page     int
	PerPage  int
}

// Indexea /search/repos response shape; mapped back into Repository below.
type searchRepoResult struct {
	TotalCount int              `json:"total_count"`
	Data       []searchRepoItem `json:"data"`
}

type searchRepoItem struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Owner       string   `json:"owner"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Stars       int      `json:"stars"`
	Forks       int      `json:"forks"`
	License     string   `json:"license"`
	Languages   []string `json:"languages"`
	LastPushAt  string   `json:"last_push_at"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func (c *Client) SearchRepositories(ctx context.Context, params *SearchRepositoriesParams) ([]Repository, error) {
	query := map[string]string{"q": params.Query}
	if params.Language != "" {
		query["language"] = params.Language
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	query["size"] = strconv.Itoa(perPage)
	if params.Page > 1 {
		query["from"] = strconv.Itoa((params.Page - 1) * perPage)
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/search/repos", query, nil)
	if err != nil {
		return nil, err
	}
	var result searchRepoResult
	if err := c.do(req, &result); err != nil {
		return nil, err
	}
	return result.toRepositories(), nil
}

func (r searchRepoResult) toRepositories() []Repository {
	repos := make([]Repository, 0, len(r.Data))
	for _, item := range r.Data {
		repos = append(repos, item.toRepository())
	}
	return repos
}

func (i searchRepoItem) toRepository() Repository {
	repo := Repository{
		ID:              i.ID,
		Name:            i.Name,
		Path:            i.Path,
		HumanName:       i.Title,
		Description:     i.Description,
		HTMLURL:         i.URL,
		StargazersCount: i.Stars,
		ForksCount:      i.Forks,
		License:         i.License,
		Namespace:       Namespace{Path: i.Owner},
		Owner:           User{Login: i.Owner},
		UpdatedAt:       parseIndexeaTime(i.UpdatedAt),
		CreatedAt:       parseIndexeaTime(i.CreatedAt),
		PushedAt:        parseIndexeaTime(i.LastPushAt),
	}
	repo.FullName = repoFullName(i.Owner, i.Path)
	if len(i.Languages) > 0 {
		repo.Language = i.Languages[0]
	}
	return repo
}

func repoFullName(owner, path string) string {
	switch {
	case owner != "" && path != "":
		return owner + "/" + path
	case owner != "":
		return owner
	case path != "":
		return path
	default:
		return ""
	}
}

func parseIndexeaTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t
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
