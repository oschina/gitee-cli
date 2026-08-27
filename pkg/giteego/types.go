package giteego

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ResultVO mirrors the gitee-go ResultVO<T> wrapper: {code, msg, data}.
type ResultVO[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// PageVO mirrors gitee-go PageVO<T>.
type PageVO[T any] struct {
	Current  int `json:"current"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
	Data     []T `json:"data"`
}

// PipelineYamlSummaryVO is a repository pipeline YAML list item
// (also carries the raw yaml when detail is requested).
type PipelineYamlSummaryVO struct {
	FileName string `json:"fileName"`
	Yaml     string `json:"yaml"`
	Message  string `json:"message"`

	Identifier string `json:"identifier,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Name       string `json:"name,omitempty"`
}

// PipelineBuildVO covers GET /pipelines/builds/{id} and /pipelines/builds/last.
type PipelineBuildVO struct {
	ID               int64                  `json:"id"`
	BelongType       string                 `json:"belongType"`
	BelongIdentifier string                 `json:"belongIdentifier"`
	BuildNumber      int64                  `json:"buildNumber"`
	StartTime        *FlexTime              `json:"startTime"`
	EndTime          *FlexTime              `json:"endTime"`
	Status           string                 `json:"status"`
	Message          string                 `json:"message"`
	FileName         string                 `json:"fileName"`
	Ref              string                 `json:"ref"`
	Stages           []PipelineStageBuildVO `json:"stages,omitempty"`
	Param            *BuildParamSetVO       `json:"param,omitempty"`
	Trigger          *BuildTriggerVO        `json:"trigger,omitempty"`
	Strategy         *BuildStrategyVO       `json:"strategy,omitempty"`
	Sources          []BuildSourceVO        `json:"sources,omitempty"`
}

// PipelineBuildSimpleVO is a build history list item (the data of
// GET /pipelines/builds/history). Mirrors PipelineBuildSimpleYamlWrapperVO which
// extends BasePipelineBuildVO and adds fileName/ref.
type PipelineBuildSimpleVO struct {
	ID               int64           `json:"id"`
	BelongType       string          `json:"belongType"`
	BelongIdentifier string          `json:"belongIdentifier"`
	BuildNumber      int64           `json:"buildNumber"`
	StartTime        *FlexTime       `json:"startTime"`
	EndTime          *FlexTime       `json:"endTime"`
	Status           string          `json:"status"`
	Message          string          `json:"message"`
	FileName         string          `json:"fileName,omitempty"`
	Ref              string          `json:"ref,omitempty"`
	Trigger          *BuildTriggerVO `json:"trigger,omitempty"`
	Sources          []BuildSourceVO `json:"sources,omitempty"`
}

// PipelineStageBuildVO is a stage build inside a PipelineBuildVO.
type PipelineStageBuildVO struct {
	ID              int64                  `json:"id"`
	PipelineBuildID int64                  `json:"pipelineBuildId"`
	Name            string                 `json:"name"`
	UpstreamStageID int64                  `json:"upstreamStageBuildId"`
	Latest          bool                   `json:"latest"`
	Status          string                 `json:"status,omitempty"`
	StartTime       *FlexTime              `json:"startTime,omitempty"`
	EndTime         *FlexTime              `json:"endTime,omitempty"`
	Jobs            [][]PipelineJobBuildVO `json:"jobs,omitempty"`
	Param           *BuildParamSetVO       `json:"param,omitempty"`
	Strategy        *BuildStrategyVO       `json:"strategy,omitempty"`
}

// PipelineJobBuildVO is a job build inside a stage build.
type PipelineJobBuildVO struct {
	ID           int64            `json:"id"`
	Name         string           `json:"name"`
	Identifier   string           `json:"identifier"`
	Type         string           `json:"type"`
	StageBuildID int64            `json:"stageBuildId"`
	Status       string           `json:"status,omitempty"`
	Latest       bool             `json:"latest,omitempty"`
	StartTime    *FlexTime        `json:"startTime,omitempty"`
	EndTime      *FlexTime        `json:"endTime,omitempty"`
	Data         *JobRunDataVO    `json:"data,omitempty"`
	Record       *JobRecordVO     `json:"record,omitempty"`
	Param        *BuildParamSetVO `json:"param,omitempty"`
	Strategy     *BuildStrategyVO `json:"strategy,omitempty"`
	Message      string           `json:"message,omitempty"`
	Sources      []string         `json:"sources,omitempty"`
}

// BuildParamSetVO is the in/out param set of a pipeline/stage/job.
type BuildParamSetVO struct {
	ID         int64             `json:"id"`
	BelongType string            `json:"belongType"`
	BelongID   int64             `json:"belongId"`
	InParams   []ParamValueVO    `json:"inParams,omitempty"`
	OutParams  []ParamValueVO    `json:"outParams,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
}

// ParamValueVO is a single parameter entry.
type ParamValueVO struct {
	Key          string      `json:"key"`
	Value        FlexString  `json:"value"`
	Type         string      `json:"type,omitempty"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Description  string      `json:"description,omitempty"`
}

// JobRunDataVO is the job execution data (parameters/caches).
type JobRunDataVO struct {
	Parameters     []ParamValueVO `json:"parameters,omitempty"`
	Caches         []string       `json:"caches,omitempty"`
	OutParamsAlias interface{}    `json:"outParamsAlias,omitempty"`
	Resource       interface{}    `json:"resource,omitempty"`
}

// JobRecordVO is a job's execution record (JOB_CENTER) with its loggers.
type JobRecordVO struct {
	Type          string             `json:"type,omitempty"`
	UUID          string             `json:"uuid,omitempty"`
	Status        string             `json:"status,omitempty"`
	Loggers       []JobLoggerVO      `json:"loggers,omitempty"`
	Stages        []JobRecordStageVO `json:"stages,omitempty"`
	FailureReason string             `json:"failureReason,omitempty"`
	Message       string             `json:"message,omitempty"`
}

// JobLoggerVO is one log channel inside a job record.
type JobLoggerVO struct {
	Name      string `json:"name,omitempty"`
	UUID      string `json:"uuid,omitempty"`
	Logger    string `json:"logger,omitempty"`
	Message   string `json:"message,omitempty"`
	Status    string `json:"status,omitempty"`
	StartTime int64  `json:"startTime,omitempty"`
	EndTime   int64  `json:"endTime,omitempty"`
}

// JobRecordStageVO is a step/stage inside a job record.
type JobRecordStageVO struct {
	UUID       string `json:"uuid,omitempty"`
	Plugin     string `json:"plugin,omitempty"`
	Logger     string `json:"logger,omitempty"`
	Status     string `json:"status,omitempty"`
	StartTime  int64  `json:"startTime,omitempty"`
	FinishTime int64  `json:"finishTime,omitempty"`
}

// BuildTriggerVO is the build trigger info.
type BuildTriggerVO struct {
	Username string       `json:"username,omitempty"`
	UserID   string       `json:"userId,omitempty"`
	Data     *TriggerData `json:"data,omitempty"`
}

// TriggerData is the trigger.type payload (e.g. WEBHOOK).
type TriggerData struct {
	Type string `json:"type,omitempty"`
}

// BuildStrategyVO is the strategy envelope with triggerMode.
type BuildStrategyVO struct {
	Strategy *StrategyDetail `json:"strategy,omitempty"`
}

// StrategyDetail holds the trigger mode / timeout / retry.
type StrategyDetail struct {
	TriggerMode string `json:"triggerMode,omitempty"`
	Blocked     bool   `json:"blocked,omitempty"`
	Timeout     int64  `json:"timeout,omitempty"`
	Retry       int64  `json:"retry,omitempty"`
	FailFast    bool   `json:"failFast,omitempty"`
}

// BuildSourceVO is a build source (code source).
type BuildSourceVO struct {
	Name       string             `json:"name,omitempty"`
	Type       string             `json:"type,omitempty"`
	Identifier string             `json:"identifier,omitempty"`
	Source     *BuildSourceDetail `json:"source,omitempty"`
}

// BuildSourceDetail is the runtime source info of a build (frontend displays
// commit message and PR-vs-branch from it). Mirrors gitee-go's GitCodeSourceStruct.
type BuildSourceDetail struct {
	PathWithNamespace string    `json:"pathWithNamespace,omitempty"`
	Event             string    `json:"event,omitempty"`
	Branch            string    `json:"branch,omitempty"`
	RefPath           string    `json:"refPath,omitempty"`
	Revision          string    `json:"revision,omitempty"`
	PrSourceBranch    string    `json:"prSourceBranch,omitempty"`
	PrIID             FlexInt64 `json:"prIid,omitempty"`
	PrTitle           string    `json:"prTitle,omitempty"`
	Message           string    `json:"message,omitempty"`
}

// PipelineBuildStatusVO covers GET /pipelines/builds/{id}/status.
type PipelineBuildStatusVO struct {
	ID               int64                   `json:"id"`
	BelongType       string                  `json:"belongType"`
	BelongIdentifier string                  `json:"belongIdentifier"`
	StartTime        *FlexTime               `json:"startTime"`
	EndTime          *FlexTime               `json:"endTime"`
	Status           string                  `json:"status"`
	Stages           []PipelineStageStatusVO `json:"stages"`
}

// PipelineStageStatusVO is a stage status node.
type PipelineStageStatusVO struct {
	ID        int64                   `json:"id"`
	Name      string                  `json:"name"`
	StartTime *FlexTime               `json:"startTime"`
	EndTime   *FlexTime               `json:"endTime"`
	Status    string                  `json:"status"`
	Jobs      [][]PipelineJobStatusVO `json:"jobs"`
}

// PipelineJobStatusVO is a job status node.
type PipelineJobStatusVO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	StartTime *FlexTime `json:"startTime"`
	EndTime   *FlexTime `json:"endTime"`
	Status    string    `json:"status"`
}

// PipelineFileBranchVO is the file→branches mapping.
type PipelineFileBranchVO struct {
	FileName string   `json:"fileName"`
	Branches []string `json:"branches"`
}

// PipelineTemplateVO is a repository pipeline template.
type PipelineTemplateVO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// BuildRequest is the request body for triggering a build.
type BuildRequest struct {
	FileName string      `json:"fileName"`
	Ref      string      `json:"ref"`
	Params   []ParamItem `json:"params,omitempty"`
	Commit   string      `json:"commit,omitempty"`
}

// ParamItem is a single build/param entry.
type ParamItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// JobMarkSuccessRequest is the body for marking a job build as success.
type JobMarkSuccessRequest struct {
	Reason string `json:"reason"`
}

// GiteeCodeCommitYamlRequest is the body for committing a repository pipeline
// YAML file: POST /rest/v5/external/sources/gitee-code/yaml/commits. Mirrors
// gitee-go's GiteeCodeCommitYamlRequest (branch, fileName, content,
// commitMessage). The yaml pipeline commit requires the YAML content itself.
type GiteeCodeCommitYamlRequest struct {
	Branch        string `json:"branch"`
	FileName      string `json:"fileName"`
	Content       string `json:"content"`
	CommitMessage string `json:"commitMessage"`
}

// CategoryPluginVO is a gitee-go plugin category with its plugins.
type CategoryPluginVO struct {
	Name    string     `json:"name"`
	Plugins []PluginVO `json:"plugins"`
}

// PluginVO is a single available gitee-go plugin.
type PluginVO struct {
	Name        string `json:"name"`
	Doc         string `json:"doc"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// PluginSchemeVO is the parameter scheme for one plugin job type (the data of
// GET /rest/v5/plugins/scheme). Mirrors gitee-go's PluginScheme.java.
type PluginSchemeVO struct {
	Type         PluginSchemeTypeVO      `json:"type"`
	Name         string                  `json:"name"`
	Doc          string                  `json:"doc"`
	Icon         string                  `json:"icon"`
	Description  string                  `json:"description"`
	Advance      *PluginAdvanceSchemeVO  `json:"advance,omitempty"`
	Config       []PluginComponentScheme `json:"config,omitempty"`
	Source       *PluginSourceSchemeVO   `json:"source,omitempty"`
	Cache        *PluginCacheSchemeVO    `json:"cache,omitempty"`
	OutputParams []string                `json:"outputParams,omitempty"`
}

// PluginSchemeTypeVO is the {json,yaml} type pair of a plugin scheme.
type PluginSchemeTypeVO struct {
	JSON string `json:"json"`
	YAML string `json:"yaml"`
}

// PluginAdvanceSchemeVO lists the per-plugin advance (进阶) capabilities.
type PluginAdvanceSchemeVO struct {
	Notification bool `json:"notification,omitempty"`
	Timeout      bool `json:"timeout,omitempty"`
	Skip         bool `json:"skip,omitempty"`
	Retry        bool `json:"retry,omitempty"`
}

// PluginSourceSchemeVO describes whether the plugin uses a source.
type PluginSourceSchemeVO struct {
	Enabled  bool `json:"enabled,omitempty"`
	Multiple bool `json:"multiple,omitempty"`
}

// PluginCacheSchemeVO is the plugin cache setting.
type PluginCacheSchemeVO struct {
	Enabled      bool   `json:"enabled,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

// PluginComponentScheme is one form-field/component definition in a plugin
// scheme. The gitee-go backend is polymorphic per `type` (Input/Select/Compose/
// Certification/HostSelect/Command/RemoteSelect/...); this mirrors the common
// fields, with the type-specific extras (options/children/url) kept as generic
// JSON to stay forward-compatible.
type PluginComponentScheme struct {
	Identifier        string                  `json:"identifier,omitempty"`
	Name              string                  `json:"name,omitempty"`
	Type              string                  `json:"type,omitempty"`
	Tooltip           string                  `json:"tooltip,omitempty"`
	DefaultValue      interface{}             `json:"defaultValue,omitempty"`
	Hidden            interface{}             `json:"hidden,omitempty"`
	HTML              bool                    `json:"html,omitempty"`
	Array             bool                    `json:"array,omitempty"`
	Disabled          bool                    `json:"disabled,omitempty"`
	GuideTips         string                  `json:"guideTips,omitempty"`
	Export            string                  `json:"export,omitempty"`
	Placeholder       string                  `json:"placeholder,omitempty"`
	Convertor         *PluginFieldConvertorVO `json:"convertor,omitempty"`
	Rules             []PluginValidateRuleVO  `json:"rules,omitempty"`
	Options           []ComponentOptionVO     `json:"options,omitempty"`
	Multiple          bool                    `json:"multiple,omitempty"`
	Children          []PluginComponentScheme `json:"children,omitempty"`
	CertificationType string                  `json:"certificationType,omitempty"`
	// RemoteSelect-specific extras (see RemoteSelectComponent.java).
	ExtraOptions []ComponentOptionVO `json:"extraOptions,omitempty"`
	Search       bool                `json:"search,omitempty"`
	URL          *RemoteSelectURLVO  `json:"url,omitempty"`
	Params       map[string]string   `json:"params,omitempty"`
}

// RemoteSelectURLVO holds the remote-select endpoint paths of a RemoteSelect
// scheme component (gitOps for repo pipelines, pipelineOps for project
// pipelines).
type RemoteSelectURLVO struct {
	GitOps      string `json:"gitOps"`
	PipelineOps string `json:"pipelineOps"`
}

// RemoteSelectResponse is the data of a remote-select endpoint ({total, list});
// empty responses may be a bare array instead of the object form.
type RemoteSelectResponse struct {
	Total int64               `json:"total"`
	List  []ComponentOptionVO `json:"list"`
}

// PluginFieldConvertorVO is the YAML<->JSON mapping rule of a scheme component.
type PluginFieldConvertorVO struct {
	Parameters bool   `json:"parameters,omitempty"`
	YAMLField  string `json:"yamlFiled,omitempty"`
	YAMLArray  bool   `json:"yamlArray,omitempty"`
}

// PluginValidateRuleVO is a validation rule (regex + error message) of a
// scheme component.
type PluginValidateRuleVO struct {
	Required bool   `json:"required,omitempty"`
	Regex    string `json:"regex,omitempty"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

// ComponentOptionVO is an option of a Select/Radio-style scheme component.
type ComponentOptionVO struct {
	Key         string      `json:"key,omitempty"`
	Label       string      `json:"label,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ServiceStatusVO is the response data of the gitee-go service status endpoint
// (an "is gitee-go enabled" check, not billing). Enabled is derived leniently
// from Raw; Status carries the raw gitee_go_status value when present.
type ServiceStatusVO struct {
	Enabled bool            `json:"-"`
	Status  string          `json:"-"`
	Raw     json.RawMessage `json:"-"`
}

// Service path constants exposed to callers. The versioned path (e.g.
// /rest/v5) lives in the gitee-go backend controller mappings and is included
// in each request path.
const (
	ServiceIPipe = "/ipipe"
	ServiceSA    = "/sa"
)

// PipelineOpsPipelineIDFmt is the pipelineOps (project pipeline) build-identifier
// format. Builds of a project pipeline carry belongIdentifier
// "pipeline.ops.pipeline.{pipelineId}" (backend Magic.PIPELINE_OPS_PIPELINE_IDENTIFIER)
// and are queried by this identifier on the /builds/history and /builds/last
// endpoints.
const PipelineOpsPipelineIDFmt = "pipeline.ops.pipeline.%d"

// PipelineOpsPipelineIdentifier returns the build identifier for a project
// pipeline id (e.g. 706 → "pipeline.ops.pipeline.706").
func PipelineOpsPipelineIdentifier(id int64) string {
	return fmt.Sprintf(PipelineOpsPipelineIDFmt, id)
}

// Program (pipelineOps / 项目流水线) response and request types. These mirror
// the pipelineOps controllers routed via /rest/v5/multi-source/... and are
// distinct from the repository (GitOps) types above. Polymorphic struct fields
// (GiteeStrategyStruct, NotificationStruct, SourceSettingStruct, TriggerRulesStruct,
// BaseJobData) are kept as interface{} to stay forward-compatible.

// PipelineSummaryVO is a project pipeline list item (GET /multi-source/pipelines).
type PipelineSummaryVO struct {
	Identifier string                 `json:"identifier"`
	Ref        string                 `json:"ref,omitempty"`
	UUID       string                 `json:"uuid,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Build      *PipelineBuildSimpleVO `json:"build,omitempty"`
	Group      *BasePipelineGroupVO   `json:"group,omitempty"`
	Labels     []LabelVO              `json:"labels,omitempty"`
}

// BasePipelineGroupVO is a pipeline group (分组) used to organize project
// pipelines.
type BasePipelineGroupVO struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Sort        int64     `json:"sort,omitempty"`
	ParentID    int64     `json:"parentId,omitempty"`
	CreateTime  *FlexTime `json:"createTime,omitempty"`
	UpdateTime  *FlexTime `json:"updateTime,omitempty"`
}

// LabelVO is a label (标签) attached to a pipeline/parameter.
type LabelVO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name,omitempty"`
	Color      string    `json:"color,omitempty"`
	Sort       int64     `json:"sort,omitempty"`
	BelongType string    `json:"belongType,omitempty"`
	BelongID   int64     `json:"belongId,omitempty"`
	CreateTime *FlexTime `json:"createTime,omitempty"`
	UpdateTime *FlexTime `json:"updateTime,omitempty"`
	Deleted    bool      `json:"deleted,omitempty"`
}

// PipelineVO is a project pipeline config (GET/POST/PUT /multi-source/pipelines).
type PipelineVO struct {
	Identifier    string            `json:"identifier"`
	Ref           string            `json:"ref,omitempty"`
	UUID          string            `json:"uuid,omitempty"`
	Name          string            `json:"name,omitempty"`
	Stages        []StageVO         `json:"stages,omitempty"`
	Sources       []SourceVO        `json:"sources,omitempty"`
	Triggers      []TriggerVO       `json:"triggers,omitempty"`
	Strategy      interface{}       `json:"strategy,omitempty"`
	Parameters    []ParameterStruct `json:"parameters,omitempty"`
	Notifications interface{}       `json:"notifications,omitempty"`
	Labels        []LabelVO         `json:"labels,omitempty"`
	Group         int64             `json:"group,omitempty"`
	Extra         map[string]string `json:"extra,omitempty"`
}

// StageVO is a stage definition inside a PipelineVO.
type StageVO struct {
	ID            int64       `json:"id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Jobs          [][]JobVO   `json:"jobs,omitempty"`
	Strategy      interface{} `json:"strategy,omitempty"`
	Notifications interface{} `json:"notifications,omitempty"`
}

// SourceVO is a source definition inside a PipelineVO.
type SourceVO struct {
	Name       string      `json:"name,omitempty"`
	Type       string      `json:"type,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Setting    interface{} `json:"setting,omitempty"`
	Source     interface{} `json:"source,omitempty"`
}

// TriggerVO is a trigger definition inside a PipelineVO.
type TriggerVO struct {
	Type  string      `json:"type,omitempty"`
	Name  string      `json:"name,omitempty"`
	Auto  bool        `json:"auto,omitempty"`
	Rules interface{} `json:"rules,omitempty"`
}

// JobVO is a job definition inside a StageVO.
type JobVO struct {
	ID            int64       `json:"id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Identifier    string      `json:"identifier,omitempty"`
	Type          string      `json:"type,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	Sources       []string    `json:"sources,omitempty"`
	Notifications interface{} `json:"notifications,omitempty"`
	Strategy      interface{} `json:"strategy,omitempty"`
}

// ParameterStruct is a parameter definition (ParameterType enum: STRING, REPORT,
// DOCKER, RADIO, TEXT, CHECKBOX, PASSWORD, ARTIFACT, REF_PROGRAM, REF_PROJECT,
// EXPORT). defaultValue/value/options are polymorphic per type.
type ParameterStruct struct {
	Key          string        `json:"key"`
	DefaultValue interface{}   `json:"defaultValue,omitempty"`
	Value        interface{}   `json:"value,omitempty"`
	Type         string        `json:"type,omitempty"`
	Options      []interface{} `json:"options,omitempty"`
	Description  string        `json:"description,omitempty"`
}

// PipelineHistoryVO is a config version history item
// (GET /multi-source/pipelines/{id}/history).
type PipelineHistoryVO struct {
	ID         int64       `json:"id"`
	UUID       string      `json:"uuid,omitempty"`
	Version    int64       `json:"version,omitempty"`
	Online     bool        `json:"online,omitempty"`
	Hits       int64       `json:"hits,omitempty"`
	Creator    string      `json:"creator,omitempty"`
	Data       *PipelineVO `json:"data,omitempty"`
	CreateTime *FlexTime   `json:"createTime,omitempty"`
}

// PipelineRequest is the create/update body for a project pipeline.
type PipelineRequest struct {
	Name          string            `json:"name,omitempty"`
	Stages        []StageRequest    `json:"stages,omitempty"`
	Sources       []SourceRequest   `json:"sources,omitempty"`
	Triggers      []TriggerRequest  `json:"triggers,omitempty"`
	Notifications interface{}       `json:"notifications,omitempty"`
	Parameters    []ParameterStruct `json:"parameters,omitempty"`
	Strategy      interface{}       `json:"strategy,omitempty"`
	Labels        []int64           `json:"labels,omitempty"`
	Group         int64             `json:"group,omitempty"`
	UUID          string            `json:"uuid,omitempty"`
}

// StageRequest is one stage of a PipelineRequest.
type StageRequest struct {
	Name          string         `json:"name,omitempty"`
	Strategy      interface{}    `json:"strategy,omitempty"`
	Notifications interface{}    `json:"notifications,omitempty"`
	Jobs          [][]JobRequest `json:"jobs,omitempty"`
}

// JobRequest is one job (or job group item) of a StageRequest.
type JobRequest struct {
	Name          string      `json:"name"`
	Identifier    string      `json:"identifier,omitempty"`
	Type          string      `json:"type"`
	Data          interface{} `json:"data"`
	Sources       []string    `json:"sources,omitempty"`
	Notifications interface{} `json:"notifications,omitempty"`
	Strategy      interface{} `json:"strategy,omitempty"`
}

// SourceRequest is a source of a PipelineRequest.
type SourceRequest struct {
	Name       string      `json:"name,omitempty"`
	Type       string      `json:"type,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Setting    interface{} `json:"setting,omitempty"`
}

// TriggerRequest is a trigger of a PipelineRequest.
type TriggerRequest struct {
	Type  string      `json:"type,omitempty"`
	Name  string      `json:"name,omitempty"`
	Auto  bool        `json:"auto,omitempty"`
	Rules interface{} `json:"rules,omitempty"`
}

// PipelineOpsBuildRequest is the trigger-build body for a project pipeline
// (POST /multi-source/pipelines/builds). Mirrors PipelineBuildPipelineOpsRequest:
// it carries only the pipeline (DB) scope fields — no fileName/ref/commit like
// the GitOps BuildRequest.
type PipelineOpsBuildRequest struct {
	PipelineID int64                        `json:"pipelineId"`
	Params     []ParameterStruct            `json:"params,omitempty"`
	Sources    []PipelineBuildSourceRequest `json:"sources,omitempty"`
}

// PipelineBuildSourceRequest is a build source override of
// PipelineOpsBuildRequest (name + type, polymorphic sub-types kept generic).
type PipelineBuildSourceRequest struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// ParamVO is a project-scoped parameter template
// (GET /multi-source/params, /multi-source/params/{id}).
type ParamVO struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	BelongType  string            `json:"belongType,omitempty"`
	BelongID    int64             `json:"belongId,omitempty"`
	Parameters  []ParameterStruct `json:"parameters,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
	Labels      []LabelVO         `json:"labels,omitempty"`
	Pipelines   []PipelineVO      `json:"pipelines,omitempty"`
	CreateTime  *FlexTime         `json:"createTime,omitempty"`
	UpdateTime  *FlexTime         `json:"updateTime,omitempty"`
}

// ParamRequest is the create/update body for a project parameter.
type ParamRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Parameters  []ParameterStruct `json:"parameters,omitempty"`
	Labels      []int64           `json:"labels,omitempty"`
}

// ProgramPipelineTemplateVO is a project pipeline template
// (GET /multi-source/pipelines/templates). Distinct from the repo-scoped
// PipelineTemplateVO.
type ProgramPipelineTemplateVO struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name,omitempty"`
	Description string                      `json:"description,omitempty"`
	Config      *PipelineVO                 `json:"config,omitempty"`
	CreateTime  *FlexTime                   `json:"createTime,omitempty"`
	UpdateTime  *FlexTime                   `json:"updateTime,omitempty"`
	CategoryID  int64                       `json:"categoryId,omitempty"`
	Category    *PipelineTemplateCategoryVO `json:"category,omitempty"`
	Disabled    bool                        `json:"disabled,omitempty"`
}

// PipelineTemplateCategoryVO is a template category.
type PipelineTemplateCategoryVO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name,omitempty"`
	Icon       string    `json:"icon,omitempty"`
	Sort       int64     `json:"sort,omitempty"`
	CreateTime *FlexTime `json:"createTime,omitempty"`
	UpdateTime *FlexTime `json:"updateTime,omitempty"`
}

// ProgramPipelineTemplateRequest is the create/update body for a project
// template.
type ProgramPipelineTemplateRequest struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Config      *PipelineVO `json:"config,omitempty"`
}

// FlexString is a string wrapper tolerant of the gitee-go backend emitting a
// scalar parameter value as a JSON array (e.g. inParams.value may come back as
// ["master"] instead of "master"). It accepts a plain string, a string array
// (joined), a number, or null.
type FlexString string

// UnmarshalJSON accepts a JSON string, a JSON array of strings, a JSON number,
// or null.
func (f *FlexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*f = FlexString(str)
		return nil
	}
	if b[0] == '[' {
		var arr []string
		if err := json.Unmarshal(b, &arr); err != nil {
			// Fall back to raw JSON text if the array has non-string items.
			*f = FlexString(string(b))
			return nil
		}
		*f = FlexString(strings.Join(arr, ","))
		return nil
	}
	// Number, bool, or other scalar: keep the raw JSON text.
	*f = FlexString(string(b))
	return nil
}

// FlexInt64 is an int64 wrapper tolerant of the gitee-go backend emitting an
// integer field as a JSON string (e.g. prIid may come back as "12" or 12).
type FlexInt64 int64

// UnmarshalJSON accepts either a JSON number or a JSON string containing an
// integer, or null.
func (f *FlexInt64) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	// Strip surrounding quotes when the backend sends a string.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("giteego: parse int %s: %w", string(b), err)
	}
	*f = FlexInt64(n)
	return nil
}

// FlexTime is a time.Time wrapper tolerant of the gitee-go backend's date
// serialization: with `write-dates-as-timestamps=true`, java.util.Date fields
// are emitted as epoch-millis JSON numbers (or null). It also accepts RFC3339
// and "2006-01-02 15:04:05" strings so it survives format drift.
type FlexTime struct {
	time.Time
}

// UnmarshalJSON accepts a JSON number (epoch millis), an RFC3339 string, a
// "2006-01-02 15:04:05" string, or null.
func (ft *FlexTime) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*ft = FlexTime{}
		return nil
	}
	if b[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		return ft.parseString(str)
	}
	// Number: epoch millis (or seconds, small values).
	var num int64
	if err := json.Unmarshal(b, &num); err != nil {
		return fmt.Errorf("giteego: parse time %s: %w", s, err)
	}
	sec := num / 1000
	nsec := (num % 1000) * int64(time.Millisecond)
	ft.Time = time.Unix(sec, nsec)
	return nil
}

func (ft *FlexTime) parseString(str string) error {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
	} {
		if t, err := time.ParseInLocation(layout, str, time.Local); err == nil {
			ft.Time = t
			return nil
		}
	}
	// Fall back to RFC3339 to produce a useful error.
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		ft.Time = t
		return nil
	}
	return fmt.Errorf("giteego: unsupported time layout %q", str)
}

// MarshalJSON serializes as RFC3339.
func (ft FlexTime) MarshalJSON() ([]byte, error) {
	if ft.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ft.Time.Format(time.RFC3339))
}
