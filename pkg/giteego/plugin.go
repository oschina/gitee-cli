package giteego

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Plugin methods (GitOps). The plugins endpoint is global (not repo-bound) but
// is reached through the same repo-scoped gateway prefix as the other pipeline
// endpoints.

// ListPlugins returns the available gitee-go plugins grouped by category.
func (c *Client) ListPlugins(ctx context.Context) ([]CategoryPluginVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/plugins", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]CategoryPluginVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetPluginSchemes returns the plugin scheme(s) for a job type. jobType is the
// JSON type (e.g. "JENKINS_JOB", "maven-build@v1.0.0") as returned by
// ListPlugins' PluginVO.Type.
//
// The backend (/plugins/scheme) returns the data as an object keyed by the json
// type (e.g. {"gcc-build@v1.0.0": {type:{...}, config:[...]}}). Some versions
// return the single scheme directly; resultKey returns the value for jobType
// when present, else the single/first entry, so both forms are handled.
func (c *Client) GetPluginSchemes(ctx context.Context, jobType string) (map[string]*PluginSchemeVO, error) {
	return c.getPluginSchemes(ctx, "/rest/v5/plugins/scheme", jobType)
}

// getPluginSchemes implements plugin scheme resolution for an arbitrary scheme
// route (repo GitOps and project multi-source variants share the same payload
// shape). path is the full versioned route (newRequest prepends baseURL/service).
func (c *Client) getPluginSchemes(ctx context.Context, path, jobType string) (map[string]*PluginSchemeVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, map[string]string{"jobType": jobType}, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[json.RawMessage]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	schemes := map[string]*PluginSchemeVO{}
	// data may be an object keyed by type, or a bare single scheme.
	raw := out.Data
	if len(raw) == 0 || string(raw) == "null" {
		return schemes, nil
	}
	// Try map form first.
	var asMap map[string]*PluginSchemeVO
	if err := json.Unmarshal(raw, &asMap); err == nil {
		for k, v := range asMap {
			if v != nil {
				schemes[k] = v
			}
		}
		if len(schemes) > 0 {
			return schemes, nil
		}
	}
	// Fallback: single scheme object.
	var one PluginSchemeVO
	if err := json.Unmarshal(raw, &one); err == nil {
		key := one.Type.JSON
		if key == "" {
			key = jobType
		}
		schemes[key] = &one
	}
	return schemes, nil
}

// GetPluginScheme resolves a single plugin scheme for a job type. It returns
// the scheme whose json type matches jobType, else the only (first) one.
func (c *Client) GetPluginScheme(ctx context.Context, jobType string) (*PluginSchemeVO, error) {
	schemes, err := c.GetPluginSchemes(ctx, jobType)
	if err != nil {
		return nil, err
	}
	if v, ok := schemes[jobType]; ok {
		return v, nil
	}
	for _, v := range schemes {
		return v, nil
	}
	return nil, nil
}

// RemoteSelectOptions fetches the selectable options for a RemoteSelect scheme
// component. path is the component's url.gitOps/pipelineOps value, which is the
// FULL gateway route from the host root and may already carry its own query
// string (e.g. "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=gcc").
// The leading /gitee-go/{service} is reduced so newRequest does not double it
// (baseURL ends in /gitee-go and servicePath is /ipipe). The URL's own query is
// kept, then caller query params override it.
//
// The response data is a {total, list:[{key,label,value,description}]} object,
// but can be a bare array when empty; 401 is returned on a mismatched uuid.
func (c *Client) RemoteSelectOptions(ctx context.Context, path string, query map[string]string) (*RemoteSelectResponse, error) {
	route := remoteSelectRoute(path, c.servicePath)
	merged, err := remoteSelectQuery(path, query)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, route, merged, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[json.RawMessage]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	resp := &RemoteSelectResponse{}
	if len(out.Data) == 0 || string(out.Data) == "null" {
		return resp, nil
	}
	// Prefer object form {total, list}.
	if err := json.Unmarshal(out.Data, resp); err == nil && resp.List != nil {
		return resp, nil
	}
	// Fallback: bare array.
	var list []ComponentOptionVO
	if err := json.Unmarshal(out.Data, &list); err == nil {
		resp.List = list
		resp.Total = int64(len(list))
	}
	return resp, nil
}

// remoteSelectRoute reduces a RemoteSelect scheme URL to the relative versioned
// route for newRequest. The scheme value resembles
// "/gitee-go/ipipe/rest/v5/external/...": strip the leading /gitee-go and the
// following service segment (serv) so the route begins at /rest/v5/... (newRequest
// then prepends baseURL + /serv). Handles "/gitee-go/ipipe", "/gitee-go/sa", etc.
func remoteSelectRoute(path, serv string) string {
	p := strings.TrimLeft(path, "/")
	// Strip any query string.
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	seg := strings.Trim(serv, "/")
	if strings.HasPrefix(p, "gitee-go/") {
		p = p[len("gitee-go/"):]
		// Drop the service segment if present (ipipe/sa/...).
		if seg != "" && strings.HasPrefix(p, seg) {
			rest := p[len(seg):]
			p = strings.TrimLeft(rest, "/")
		}
	}
	return "/" + p
}

// remoteSelectQuery parses the scheme URL's own query string and merges the
// caller query params on top (caller wins).
func remoteSelectQuery(path string, query map[string]string) (map[string]string, error) {
	merged := map[string]string{}
	if i := strings.IndexByte(path, '?'); i >= 0 {
		u, err := url.ParseQuery(path[i+1:])
		if err != nil {
			return nil, fmt.Errorf("giteego: parse remote-select query: %w", err)
		}
		for k, vs := range u {
			if len(vs) > 0 {
				merged[k] = vs[len(vs)-1]
			}
		}
	}
	for k, v := range query {
		if v != "" {
			merged[k] = v
		}
	}
	return merged, nil
}

// GeneratePluginExampleYaml returns the generated example YAML snippet for a
// plugin job type. The data is a plain YAML string (step-level snippet, possibly
// with multiple commented variants).
func (c *Client) GeneratePluginExampleYaml(ctx context.Context, jobType string) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/plugins/example", map[string]string{"jobType": jobType}, nil)
	if err != nil {
		return "", err
	}
	var out ResultVO[string]
	if err := c.do(req, &out); err != nil {
		return "", err
	}
	return out.Data, nil
}
