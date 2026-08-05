package gitee

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type GistFile struct {
	FileName string `json:"filename"`
	Size     int    `json:"size"`
	RawURL   string `json:"raw_url"`
	Content  string `json:"content,omitempty"`
}

type Gist struct {
	ID          string              `json:"id"`
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Owner       User                `json:"owner"`
	Files       map[string]GistFile `json:"files"`
	HTMLURL     string              `json:"html_url"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type CreateGistParams struct {
	Description string              `json:"description,omitempty"`
	Public      bool                `json:"public"`
	Files       map[string]GistFile `json:"files"`
}

func (c *Client) ListGists(ctx context.Context, page, perPage int) ([]Gist, error) {
	q := map[string]string{}
	if page > 0 {
		q["page"] = strconv.Itoa(page)
	}
	if perPage > 0 {
		q["per_page"] = strconv.Itoa(perPage)
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/gists", q, nil)
	if err != nil {
		return nil, err
	}
	var gists []Gist
	return gists, c.do(req, &gists)
}

func (c *Client) GetGist(ctx context.Context, id string) (*Gist, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/gists/%s", id), nil, nil)
	if err != nil {
		return nil, err
	}
	var g Gist
	return &g, c.do(req, &g)
}

func (c *Client) CreateGist(ctx context.Context, params *CreateGistParams) (*Gist, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/gists", nil, params)
	if err != nil {
		return nil, err
	}
	var g Gist
	return &g, c.do(req, &g)
}

func (c *Client) DeleteGist(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/gists/%s", id), nil, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
