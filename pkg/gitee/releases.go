package gitee

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type Release struct {
	ID              int       `json:"id"`
	TagName         string    `json:"tag_name"`
	Name            string    `json:"name"`
	Body            string    `json:"body"`
	Prerelease      bool      `json:"prerelease"`
	TargetCommitish string    `json:"target_commitish"`
	Author          User      `json:"author"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateReleaseParams struct {
	TagName         string `json:"tag_name"`
	Name            string `json:"name"`
	Body            string `json:"body,omitempty"`
	Prerelease      bool   `json:"prerelease"`
	TargetCommitish string `json:"target_commitish,omitempty"`
}

func (c *Client) ListReleases(ctx context.Context, owner, repo string, page, perPage int) ([]Release, error) {
	q := map[string]string{}
	if page > 0 {
		q["page"] = strconv.Itoa(page)
	}
	if perPage > 0 {
		q["per_page"] = strconv.Itoa(perPage)
	}
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/releases", owner, repo), q, nil)
	if err != nil {
		return nil, err
	}
	var releases []Release
	if err := c.do(req, &releases); err != nil {
		return nil, err
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].CreatedAt.After(releases[j].CreatedAt)
	})
	return releases, nil
}

func (c *Client) ListAllReleases(ctx context.Context, owner, repo string) ([]Release, error) {
	return paginateAll(ctx, func(ctx context.Context, page, perPage int) ([]Release, error) {
		return c.ListReleases(ctx, owner, repo, page, perPage)
	})
}

func (c *Client) GetRelease(ctx context.Context, owner, repo string, id int) (*Release, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/releases/%d", owner, repo, id), nil, nil)
	if err != nil {
		return nil, err
	}
	var r Release
	return &r, c.do(req, &r)
}

func (c *Client) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/releases/tags/%s", owner, repo, tag), nil, nil)
	if err != nil {
		return nil, err
	}
	var r *Release
	if err := c.do(req, &r); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, &ErrorResponse{StatusCode: http.StatusNotFound, Message: "release not found"}
	}
	return r, nil
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo), nil, nil)
	if err != nil {
		return nil, err
	}
	var r Release
	return &r, c.do(req, &r)
}

func (c *Client) CreateRelease(ctx context.Context, owner, repo string, params *CreateReleaseParams) (*Release, error) {
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/releases", owner, repo), nil, params)
	if err != nil {
		return nil, err
	}
	var r Release
	return &r, c.do(req, &r)
}

func (c *Client) DeleteRelease(ctx context.Context, owner, repo string, id int) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/repos/%s/%s/releases/%d", owner, repo, id), nil, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
