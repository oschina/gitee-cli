package giteego

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// Program (program / 流水线程序) support implements the enterprise-scoped
// pipeline (pipelineOps), scoped by enterprise id + program id (the project
// id in the gitee-go DB scope) and routed via the /multi-source family (e.g.
// /rest/v5/multi-source/pipelines). This is distinct from the repo-scoped
// pipeline (GitOps) exposed in pipeline.go.
//
// Files are named program.go to avoid the ambiguous "project" term (in gitee-go
// a "project" is really the repo, so "program pipeline" was misleading).
//
// Calling convention (design, locked in .gitee-go/support-ledger.md §3.1/§4.8/§5.1):
//   - pathBase = "{enterpriseId}/{programId}" (e.g. "2/423"), passed as the first
//     arg of Factory.GoAPIClient; the gateway routes
//     {host}/{enterpriseId}/{programId}/gitee-go/ipipe/rest/v5/multi-source/...
//   - every method uses the "multi-source" route prefix (/rest/v5/multi-source/...)
//     and identifies pipeline builds by the PIPELINE_OPS identifier
//     pipeline.ops.pipeline.{pipelineId} (Magic.PIPELINE_OPS_PIPELINE_IDENTIFIER).
//   - CLI command family is `gitee pipeline program ...` with --enterprise/--program
//     (flags -E/-P), distinct from the repo pipeline family which requires -R.

// Program list query parameters shared by the pipeline list endpoint.
type ProgramListQuery struct {
	Current  int // page number, 1-based; 0 → default
	PageSize int // page size; 0 → default
	Statuses []string
	LabelIDs []int64
	GroupID  int64
	Order    string // "create_time"/"id"/...
	Sort     string // "asc"/"desc"
	Search   string
}

// toQuery drops zero values so the backend uses its own defaults.
func (q ProgramListQuery) toQuery() map[string]string {
	m := map[string]string{}
	if q.Current > 0 {
		m["current"] = strconv.Itoa(q.Current)
	}
	if q.PageSize > 0 {
		m["pageSize"] = strconv.Itoa(q.PageSize)
	}
	if len(q.Statuses) > 0 {
		m["statuses"] = strings.Join(q.Statuses, ",")
	}
	if len(q.LabelIDs) > 0 {
		ids := make([]string, len(q.LabelIDs))
		for i, id := range q.LabelIDs {
			ids[i] = strconv.FormatInt(id, 10)
		}
		m["labelIds"] = strings.Join(ids, ",")
	}
	if q.GroupID > 0 {
		m["groupId"] = strconv.FormatInt(q.GroupID, 10)
	}
	if q.Order != "" {
		m["order"] = q.Order
	}
	if q.Sort != "" {
		m["sort"] = q.Sort
	}
	if q.Search != "" {
		m["search"] = q.Search
	}
	return m
}

// ListProgramPipelines returns the program pipeline summary list (paged).
func (c *Client) ListProgramPipelines(ctx context.Context, q ProgramListQuery) (PageVO[PipelineSummaryVO], error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines", q.toQuery(), nil)
	if err != nil {
		return PageVO[PipelineSummaryVO]{}, err
	}
	var out ResultVO[PageVO[PipelineSummaryVO]]
	if err := c.do(req, &out); err != nil {
		return PageVO[PipelineSummaryVO]{}, err
	}
	return out.Data, nil
}

// GetProgramPipeline returns a program pipeline config detail by id.
func (c *Client) GetProgramPipeline(ctx context.Context, id int64) (*PipelineVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CreateProgramPipeline creates a program pipeline and returns the created
// config.
func (c *Client) CreateProgramPipeline(ctx context.Context, reqBody PipelineRequest) (*PipelineVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines", nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// UpdateProgramPipeline updates a program pipeline config.
func (c *Client) UpdateProgramPipeline(ctx context.Context, id int64, reqBody PipelineRequest) (*PipelineVO, error) {
	req, err := c.newRequest(ctx, http.MethodPut, "/rest/v5/multi-source/pipelines/"+strconv.FormatInt(id, 10), nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CloneProgramPipeline clones a program pipeline and returns the new config.
func (c *Client) CloneProgramPipeline(ctx context.Context, id int64) (*PipelineVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/"+strconv.FormatInt(id, 10)+"/clone", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// DeleteProgramPipeline deletes a program pipeline config.
func (c *Client) DeleteProgramPipeline(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/rest/v5/multi-source/pipelines/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[bool]
	return c.do(req, &out)
}

// ListProgramPipelineHistory lists the config version history of a program
// pipeline.
func (c *Client) ListProgramPipelineHistory(ctx context.Context, id int64, current, pageSize int) (PageVO[PipelineHistoryVO], error) {
	q := map[string]string{}
	if current > 0 {
		q["current"] = strconv.Itoa(current)
	}
	if pageSize > 0 {
		q["pageSize"] = strconv.Itoa(pageSize)
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/"+strconv.FormatInt(id, 10)+"/history", q, nil)
	if err != nil {
		return PageVO[PipelineHistoryVO]{}, err
	}
	var out ResultVO[PageVO[PipelineHistoryVO]]
	if err := c.do(req, &out); err != nil {
		return PageVO[PipelineHistoryVO]{}, err
	}
	return out.Data, nil
}

// GetProgramPipelineHistory returns the config of a history version by its own
// id (GET /multi-source/pipelines/history/{historyId}). The backend returns the
// PipelineHistoryVO whose Data is the config at that version.
func (c *Client) GetProgramPipelineHistory(ctx context.Context, historyID int64) (*PipelineHistoryVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/history/"+strconv.FormatInt(historyID, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineHistoryVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ApplyProgramPipelineHistory applies (activates) a config history version.
func (c *Client) ApplyProgramPipelineHistory(ctx context.Context, historyID int64) (*PipelineVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/history/"+strconv.FormatInt(historyID, 10)+"/apply", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// TriggerProgramBuild triggers a program pipeline build. The request carries
// only the pipeline (DB) scope fields — pipelineId, params and optional source
// overrides; there is no fileName/ref/commit (unlike the GitOps BuildRequest).
func (c *Client) TriggerProgramBuild(ctx context.Context, reqBody PipelineOpsBuildRequest) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/builds", nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ListProgramBuildHistory lists builds of a program pipeline (paged), keyed by
// the "pipeline.ops.pipeline.{pipelineId}" identifier.
func (c *Client) ListProgramBuildHistory(ctx context.Context, pipelineID int64, q ProgramListQuery) (PageVO[PipelineBuildSimpleVO], error) {
	query := q.toQuery()
	query["identifier"] = PipelineOpsPipelineIdentifier(pipelineID)
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/builds/history", query, nil)
	if err != nil {
		return PageVO[PipelineBuildSimpleVO]{}, err
	}
	var out ResultVO[PageVO[PipelineBuildSimpleVO]]
	if err := c.do(req, &out); err != nil {
		return PageVO[PipelineBuildSimpleVO]{}, err
	}
	return out.Data, nil
}

// GetProgramLastBuild returns the most recent build of a program pipeline (null
// if none yet).
func (c *Client) GetProgramLastBuild(ctx context.Context, pipelineID int64) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/builds/last", map[string]string{
		"identifier": PipelineOpsPipelineIdentifier(pipelineID),
	}, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetProgramBuild returns a program pipeline build detail by build id.
func (c *Client) GetProgramBuild(ctx context.Context, id int64) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/builds/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetProgramBuildStatus returns the build status tree of a program pipeline
// build.
func (c *Client) GetProgramBuildStatus(ctx context.Context, id int64) (*PipelineBuildStatusVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/builds/"+strconv.FormatInt(id, 10)+"/status", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildStatusVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CancelProgramBuild cancels a program pipeline build by id.
func (c *Client) CancelProgramBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/builds/"+strconv.FormatInt(id, 10)+"/cancel", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// RebuildProgramBuild re-runs a previous program pipeline build by id.
func (c *Client) RebuildProgramBuild(ctx context.Context, id int64) (*PipelineBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/builds/"+strconv.FormatInt(id, 10)+"/rebuild", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetProgramStageBuild returns a program pipeline stage build detail by id.
func (c *Client) GetProgramStageBuild(ctx context.Context, id int64) (*PipelineStageBuildVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/stages/builds/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*PipelineStageBuildVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CancelProgramStageBuild cancels a program pipeline stage build by id.
func (c *Client) CancelProgramStageBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/builds/"+strconv.FormatInt(id, 10)+"/cancel", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// RetryProgramStageBuild retries a program pipeline stage build by id.
func (c *Client) RetryProgramStageBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/builds/"+strconv.FormatInt(id, 10)+"/retry", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// ContinueProgramStageBuild continues a paused program pipeline stage build.
func (c *Client) ContinueProgramStageBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/builds/"+strconv.FormatInt(id, 10)+"/continue", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// CancelProgramJobBuild cancels a program pipeline job build by id.
func (c *Client) CancelProgramJobBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/cancel", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// SkipProgramJobBuild skips a program pipeline job build by id.
func (c *Client) SkipProgramJobBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/skip", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// RetryProgramJobBuild retries a program pipeline job build by id.
func (c *Client) RetryProgramJobBuild(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/retry", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// MarkProgramJobAsSuccess marks a program pipeline job build as success with an
// optional reason.
func (c *Client) MarkProgramJobAsSuccess(ctx context.Context, id int64, reason string) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/stages/jobs/builds/"+strconv.FormatInt(id, 10)+"/mark-as-success", nil, JobMarkSuccessRequest{Reason: reason})
	if err != nil {
		return err
	}
	var out ResultVO[string]
	return c.do(req, &out)
}

// ListProgramParams lists the program-scoped parameter templates.
func (c *Client) ListProgramParams(ctx context.Context) ([]ParamVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/params", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]ParamVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetProgramParam returns a program parameter template by id.
func (c *Client) GetProgramParam(ctx context.Context, id int64) (*ParamVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/params/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ParamVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CreateProgramParam creates a program parameter template.
func (c *Client) CreateProgramParam(ctx context.Context, reqBody ParamRequest) (*ParamVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/params", nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ParamVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// UpdateProgramParam updates a program parameter template.
func (c *Client) UpdateProgramParam(ctx context.Context, id int64, reqBody ParamRequest) (*ParamVO, error) {
	req, err := c.newRequest(ctx, http.MethodPut, "/rest/v5/multi-source/params/"+strconv.FormatInt(id, 10), nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ParamVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CloneProgramParam clones a program parameter template.
func (c *Client) CloneProgramParam(ctx context.Context, id int64) (*ParamVO, error) {
	req, err := c.newRequest(ctx, http.MethodPut, "/rest/v5/multi-source/params/"+strconv.FormatInt(id, 10)+"/clone", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ParamVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// DeleteProgramParam deletes a program parameter template.
func (c *Client) DeleteProgramParam(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/rest/v5/multi-source/params/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[bool]
	return c.do(req, &out)
}

// ListProgramTemplates lists the program-scoped pipeline templates.
func (c *Client) ListProgramTemplates(ctx context.Context) ([]ProgramPipelineTemplateVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/templates", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]ProgramPipelineTemplateVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetProgramTemplate returns a program pipeline template by id.
func (c *Client) GetProgramTemplate(ctx context.Context, id int64) (*ProgramPipelineTemplateVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/templates/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ProgramPipelineTemplateVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CreateProgramTemplate creates a program pipeline template.
func (c *Client) CreateProgramTemplate(ctx context.Context, reqBody ProgramPipelineTemplateRequest) (*ProgramPipelineTemplateVO, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/templates", nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ProgramPipelineTemplateVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// UpdateProgramTemplate updates a program pipeline template.
func (c *Client) UpdateProgramTemplate(ctx context.Context, id int64, reqBody ProgramPipelineTemplateRequest) (*ProgramPipelineTemplateVO, error) {
	req, err := c.newRequest(ctx, http.MethodPut, "/rest/v5/multi-source/pipelines/templates/"+strconv.FormatInt(id, 10), nil, reqBody)
	if err != nil {
		return nil, err
	}
	var out ResultVO[*ProgramPipelineTemplateVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// DeleteProgramTemplate deletes a program pipeline template.
func (c *Client) DeleteProgramTemplate(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/rest/v5/multi-source/pipelines/templates/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[bool]
	return c.do(req, &out)
}

// DisableProgramTemplate disables a program pipeline template.
func (c *Client) DisableProgramTemplate(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/templates/"+strconv.FormatInt(id, 10)+"/disable", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[bool]
	return c.do(req, &out)
}

// EnableProgramTemplate enables a program pipeline template.
func (c *Client) EnableProgramTemplate(ctx context.Context, id int64) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/rest/v5/multi-source/pipelines/templates/"+strconv.FormatInt(id, 10)+"/enable", nil, nil)
	if err != nil {
		return err
	}
	var out ResultVO[bool]
	return c.do(req, &out)
}

// ListProgramTemplateCategories returns the program template categories.
func (c *Client) ListProgramTemplateCategories(ctx context.Context) ([]PipelineTemplateCategoryVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/pipelines/templates/categories", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]PipelineTemplateCategoryVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ListProgramPlugins returns the available project-scoped plugins grouped by
// category (routed via /rest/v5/multi-source/plugins).
func (c *Client) ListProgramPlugins(ctx context.Context) ([]CategoryPluginVO, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/plugins", nil, nil)
	if err != nil {
		return nil, err
	}
	var out ResultVO[[]CategoryPluginVO]
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// RemoteSelectURLVO.PipelineOps is the variant used by program pipelines; this
// helper serves the route for a project-scoped remote-select scheme component.
// When the scheme's url only carries the gitOps form, the project route falls
// back to the gitOps path.

// GetProgramPluginSchemes returns the plugin scheme(s) for a job type on the
// project (multi-source) route.
func (c *Client) GetProgramPluginSchemes(ctx context.Context, jobType string) (map[string]*PluginSchemeVO, error) {
	return c.getPluginSchemes(ctx, "/rest/v5/multi-source/plugins/scheme", jobType)
}

// GenerateProgramPluginExampleYaml returns the YAML example for a job type on
// the project (multi-source) route.
func (c *Client) GenerateProgramPluginExampleYaml(ctx context.Context, jobType string) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/rest/v5/multi-source/plugins/example", map[string]string{"jobType": jobType}, nil)
	if err != nil {
		return "", err
	}
	var out ResultVO[string]
	if err := c.do(req, &out); err != nil {
		return "", err
	}
	return out.Data, nil
}
