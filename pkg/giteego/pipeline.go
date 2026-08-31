package giteego

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// Repository pipeline (GitOps) methods. The versioned controller prefix is part
// of each path; the service segment (ServiceIPipe) is prepended by newRequest.

// ListRepositoryPipelines returns the pipeline YAML list for a ref.
func (c *Client) ListRepositoryPipelines(ctx context.Context, ref string) ([]PipelineYamlSummaryVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines", map[string]string{"ref": ref}, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]PipelineYamlSummaryVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetRepositoryPipeline returns a single pipeline YAML detail for ref+fileName.
func (c *Client) GetRepositoryPipeline(ctx context.Context, ref, fileName string) (*PipelineYamlSummaryVO, error) {
	q := map[string]string{"ref": ref, "fileName": fileName}
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/detail", q, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineYamlSummaryVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CommitRepositoryPipelineYaml commits a pipeline YAML file to a repository
// branch via the gitee-code external commit endpoint. The YAML pipeline commit
// requires the YAML content itself (req.Content); the backend writes it to
// req.FileName on req.Branch with req.CommitMessage.
func (c *Client) CommitRepositoryPipelineYaml(ctx context.Context, req GiteeCodeCommitYamlRequest) error {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/external/sources/gitee-code/yaml/commits", nil, req)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(httpReq, &out)
}

// GetPipelineYamlExample returns the generated example YAML for a whole
// repository pipeline (GET /rest/v5/yaml/example). The data is a plain YAML
// text blob covering version/name/stages/triggers/notify/strategy/variables.
func (c *Client) GetPipelineYamlExample(ctx context.Context) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/yaml/example", nil, nil)
	if err != nil {
		return "", err
	}
	var out ResultVO[string]
	if err := c.do(req, &out); err != nil {
		return "", err
	}
	return out.Data, nil
}

// ListPipelineFileBranches returns all pipeline files and their branches.
func (c *Client) ListPipelineFileBranches(ctx context.Context) ([]PipelineFileBranchVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/file-branches", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]PipelineFileBranchVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// TriggerBuild starts a repository pipeline build.
func (c *Client) TriggerBuild(ctx context.Context, reqBody BuildRequest) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/builds", nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetBuild returns a pipeline build detail by id.
func (c *Client) GetBuild(ctx context.Context, id int64) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/builds/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ListBuildHistory lists pipeline build history with paging.
//
// order is the sort field ("create_time"/"id"/"build_number") and sort is the
// direction ("asc"/"desc"); the backend builds `order + "_" + sort` and defaults
// to buildNumber desc when unknown. Pass "statuses" to filter by BuildStatus.
func (c *Client) ListBuildHistory(ctx context.Context, fileName, ref, order, sort string, current, pageSize int, statuses []string) (PageVO[PipelineBuildSimpleVO], error) {
	q := map[string]string{
		"fileName": fileName,
		"ref":      ref,
		"current":  strconv.Itoa(current),
		"pageSize": strconv.Itoa(pageSize),
		"order":    order,
		"sort":     sort,
	}
	if len(statuses) > 0 {
		q["statuses"] = strings.Join(statuses, ",")
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/builds/history", q, nil)
	if err != nil {
		return PageVO[PipelineBuildSimpleVO]{}, err
	}
	var out ResultVO[PageVO[PipelineBuildSimpleVO]]
	if err := c.do(req, &out); err != nil {
		return PageVO[PipelineBuildSimpleVO]{}, err
	}
	return out.Data, nil
}

// GetLastBuild returns the most recent build for ref+fileName (null if none).
func (c *Client) GetLastBuild(ctx context.Context, fileName, ref string) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/builds/last", map[string]string{"fileName": fileName, "ref": ref}, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetBuildStatus returns the build status tree (pipeline → stages → jobs).
func (c *Client) GetBuildStatus(ctx context.Context, id int64) (*PipelineBuildStatusVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/builds/"+strconv.FormatInt(id, 10)+"/status", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildStatusVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CancelBuild cancels a pipeline build by id.
func (c *Client) CancelBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/builds/"+strconv.FormatInt(id, 10)+"/cancel", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// RebuildBuild re-runs a previous pipeline build by id.
func (c *Client) RebuildBuild(ctx context.Context, id int64) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/builds/"+strconv.FormatInt(id, 10)+"/rebuild", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetStageBuild returns a stage build detail by id.
func (c *Client) GetStageBuild(ctx context.Context, id int64) (*PipelineStageBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/pipelines/stages/builds/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineStageBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CancelStageBuild cancels a stage build by id.
func (c *Client) CancelStageBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/builds/"+strconv.FormatInt(id, 10)+"/cancel", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// RetryStageBuild retries a stage build by id.
func (c *Client) RetryStageBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/builds/"+strconv.FormatInt(id, 10)+"/retry", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// ContinueStageBuild continues a paused stage build by id.
func (c *Client) ContinueStageBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/builds/"+strconv.FormatInt(id, 10)+"/continue", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// CancelJobBuild cancels a job build by id.
func (c *Client) CancelJobBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/cancel", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// SkipJobBuild skips a job build by id.
func (c *Client) SkipJobBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/skip", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// RetryJobBuild retries a job build by id.
func (c *Client) RetryJobBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/retry", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// MarkJobAsSuccess marks a job build as success with an optional reason.
func (c *Client) MarkJobAsSuccess(ctx context.Context, id int64, reason string) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/mark-as-success", nil, JobMarkSuccessRequest{Reason: reason})
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}
