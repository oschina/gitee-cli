package giteego

import (
	"context"
	"encoding/json"
	"net/http"
)

// CheckServiceStatus queries the gitee-go service availability endpoint:
//
//	{baseURL}/ipipe/rest/v1/billing/gitee-go-service/status
//
// Despite the "billing" route segment, this is an "is gitee-go enabled/opened
// for this repo" check, not a billing query. The response carries a
// gitee_go_status field: "active" (已开通) / "closed" (开通后关闭) / "not_open"
// (未开通). Enabled is derived leniently from common shapes.
func (c *Client) CheckServiceStatus(ctx context.Context) (*ServiceStatusVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v1/billing/gitee-go-service/status", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[json.RawMessage]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return parseServiceStatus(out.Data, out.Msg), nil
}

// parseServiceStatus leniently inspects a service-status payload for an
// "enabled" signal. It accepts the frontend shape (data.gitee_go_status being
// "active"/"not_open"/"closed") as well as common boolean/status shapes.
func parseServiceStatus(data json.RawMessage, msg string) *ServiceStatusVO {
	vo := &ServiceStatusVO{Raw: data, Enabled: true}
	if len(data) == 0 || string(data) == "null" {
		vo.Enabled = msg == ""
		return vo
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return vo
	}
	// Frontend shape: { gitee_go_status: "active" | "not_open" | "closed" }.
	if v, ok := m["gitee_go_status"]; ok {
		if s, ok := v.(string); ok {
			vo.Status = s
			vo.Enabled = s == "active"
		}
		return vo
	}
	if v, ok := m["status"]; ok {
		if s, ok := v.(string); ok {
			vo.Status = s
			vo.Enabled = s == "active" || s == "OPENED" || s == "OPEN" || s == "ENABLED" || s == "SUCCEED"
		}
		return vo
	}
	if v, ok := m["enabled"]; ok {
		switch t := v.(type) {
		case bool:
			vo.Enabled = t
		case string:
			vo.Enabled = t == "true" || t == "1" || t == "OPENED" || t == "OPEN"
		case float64:
			vo.Enabled = t != 0
		}
		return vo
	}
	if v, ok := m["opened"]; ok {
		if b, ok := v.(bool); ok {
			vo.Enabled = b
		}
		return vo
	}
	return vo
}
