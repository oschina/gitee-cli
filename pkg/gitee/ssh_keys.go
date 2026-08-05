package gitee

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) ListSSHKeys(ctx context.Context, login string) ([]SSHKey, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/users/%s/keys", login), nil, nil)
	if err != nil {
		return nil, err
	}
	var keys []SSHKey
	return keys, c.do(req, &keys)
}

func (c *Client) AddSSHKey(ctx context.Context, title, key string) (*SSHKey, error) {
	payload := map[string]string{"title": title, "key": key}
	req, err := c.newRequest(ctx, http.MethodPost, "/user/keys", nil, payload)
	if err != nil {
		return nil, err
	}
	var result SSHKey
	return &result, c.do(req, &result)
}

func (c *Client) DeleteSSHKey(ctx context.Context, id int) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/user/keys/%d", id), nil, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
