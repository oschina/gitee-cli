package gitee

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) GetAuthenticatedUser(ctx context.Context) (*User, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/user", nil, nil)
	if err != nil {
		return nil, err
	}
	var u User
	return &u, c.do(req, &u)
}

func (c *Client) GetUser(ctx context.Context, login string) (*User, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/users/%s", login), nil, nil)
	if err != nil {
		return nil, err
	}
	var u User
	return &u, c.do(req, &u)
}

func (c *Client) SearchUsers(ctx context.Context, query string) ([]User, error) {
	return c.SearchUsersWithParams(ctx, &SearchUsersParams{Query: query})
}
