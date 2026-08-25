---
name: gitee-go
description: Manage Gitee-Go pipelines (GitOps) from the CLI — repository pipelines (list/view/commit YAML, trigger builds, inspect build/(stage/job) runs, plugins & Schemes) use `-R owner/repo`; project pipelines (pipelineOps, enterprise-scoped) use `-E <entId> -P <projectId>`. Use when the user asks to run, list, view, commit, cancel, inspect, create/assemble a Gitee-Go repository or project pipeline or its plugins (e.g. "跑一下流水线", "看流水线", "触发构建", "提交流水线", "创建项目流水线", "组装 job json", "插件", "项目流水线"). Always uses `--json` where supported and `--no-tui`; never uses `--ai`; requires explicit user confirmation before commit/cancel/rebuild/delete and other side-effect operations. If a requested gitee-go operation has no direct command, fall back to the gitee-api skill to search the schema.
metadata:
  author: gitee
  version: "1.0"
---
管理 Gitee-Go 流水线（GitOps）的命令，分两类：
- **仓库流水线**（`gitee pipeline`，下面前置检查/Step 1–5）：**必须 `-R owner/repo`**，不依赖 git remote 推断。
- **项目流水线 pipelineOps**（`gitee pipeline program`，见「项目流水线」章节）：**必须 `-E <企业id> -P <项目id>`**，走
  `/rest/v5/multi-source/` 网关段，按企业+项目定位，不依赖 `-R`、不做 gitee-go 开通检查。
两者都需要先通过 `gitee auth login` 完成认证（gitee-go 网关复用同一 host 凭证，临时也可用 `.gitee-go/cookie`）。
## 前置检查
1. **已认证**：`gitee auth status --no-tui`，失败则提示 `gitee auth login`。
2. **仓库参数必填**：所有 `gitee pipeline ...` 命令都必须带 `-R owner/repo`；缺省会直接报错
   「repo is required for pipeline commands」。不要用 `default_repo` 或 git remote 猜测。
3. **副作用操作需确认**：`commit / run / build cancel / build rebuild` 属于写入/触发/改变运行状态，
   执行前必须向用户明确确认；非交互环境下确认后才执行。
4. **gitee-go 未开启**：每个命令前会检查 gitee-go 是否已开通；**未开通直接报错**，报错信息里带
   开通页面地址（`{host}/{owner}/{repo}/gitee_go/open?qt=path`），让用户**自行打开页面开通**后重试。
   **不存在 `--with-open` 自动代开逻辑**（open URL 无法由 CLI 驱动，不要尝试帮用户打开）。
## 执行步骤
### Step 1：查看仓库流水线（pipeline list / view）
```bash
# 列出某个分支/标签下的流水线 YAML 文件
gitee pipeline list -R owner/repo --ref master --json --no-tui
gitee pipeline list -R owner/repo --ref develop --json --no-tui

# 查看某个流水线文件的 YAML 详情
gitee pipeline view -R owner/repo --ref master --file .gitee/pipelines/ci.yml --json --no-tui
```
**可用 flag：**
| Flag | 类型 | 说明 |
|------|------|------|
| `-R` / `--repo` | string | `owner/repo`，**必填** |
| `--ref` | string | 分支或标签名（list/view **必填**） |
| `--file` | string | 流水线 YAML 文件名（view 必填） |
| `--json` / `-j` | string | **必须加**；`--json=*` 完整输出，`--json=file_name,ref` 选字段 |
| `--no-tui` | bool | **必须加** |
> 列表只扫描仓库根目录 YAML；`view` 的 `--file` 用列表中的文件名。
### Step 1.5：提交 / 更新仓库流水线 YAML（commit）— 副作用，需确认
把本地 YAML 文件内容提交到仓库某个分支（后端 `POST /sources/gitee-code/yaml/commits`，
**必须提供 YAML 内容**，即本地 YAML 文件；`--yaml -` 可从 stdin 读取）。这是写操作，会
在仓库中产生一次提交：执行前必须确认，非交互必须 `--yes`。
**fileName 只带文件名、不带目录前缀**（如 `流水线-202608131716.yml`）：省略 `--file` 时
默认取本地 `--yaml` 文件的基本名；显式传 `--file` 带路径也会被归一到基本名。
```bash
gitee pipeline commit -R owner/repo --ref master --yaml ./ci.yml -m "feat: 新增 CI 流水线" --yes --no-tui   # fileName 默认 = ci.yml
gitee pipeline commit -R owner/repo --ref develop --file 流水线.yml --yaml release.yml --yes --no-tui        # 未传 -m 用默认提交信息
```
**可用 flag：**
| Flag | 类型 | 说明 |
|------|------|------|
| `-R` / `--repo` | string | `owner/repo`，**必填** |
| `--ref` | string | 提交到的分支（**必填**） |
| `--file` | string | 仓库侧的流水线 YAML **文件名（不带目录前缀）**；缺省取 `--yaml` 基本名 |
| `--yaml` | string | 本地 YAML 文件路径（**必填**；`-` 表示从 stdin 读取） |
| `-m` / `--message` | string | 提交信息（默认 `feat: update pipeline configuration`） |
| `-y` / `--yes` | bool | 跳过确认（非交互**必须加**） |
| `--json` | - | 不支持（后端无结构化返回） |
| `--no-tui` | bool | **必须加** |
> 提交后可用 `pipeline list/view --ref <分支>` 核对（列表只扫仓库根目录 YAML），再用 `pipeline run` 触发。
### Step 1.6：生成整条仓库流水线 YAML 示例（example）
从后端拉一份**完整仓库流水线 YAML 示例**（`GET /rest/v5/yaml/example`：version/name/displayName、
stages+steps、triggers（push/pr/schedule）、notify、strategy、variables）。与 `pipeline plugin example`
（单个插件 job 片段）不同，它是整条流水线，可直接改造成自己的配置再 `pipeline commit` 提交。
```bash
gitee pipeline example -R owner/repo --no-tui
```
**可用 flag：** `-R/--repo` 必填；`--no-tui` 必加；无 `--json`（输出为 YAML 文本，直接 stdout）。
> 先 `example` 拿模板 → 按需改 → `commit` 提交 → `list/view` 核对 → `run` 触发。
### Step 2：触发仓库流水线构建（run）
```bash
gitee pipeline run -R owner/repo --ref master --file .gitee/pipelines/ci.yml --json --no-tui
gitee pipeline run -R owner/repo --ref master --file ci.yml --params KEY=VALUE,KEY2=VALUE2 --json --no-tui
```
**可用 flag：**
| Flag | 类型 | 说明 |
|------|------|------|
| `-R` / `--repo` | string | `owner/repo`，**必填** |
| `--ref` | string | 分支/标签（**必填**） |
| `--file` | string | 流水线 YAML 文件名（**必填**） |
| `--params` | string | 逗号分隔 `KEY=VALUE` 构建参数 |
| `--json` / `-j` | string | **必须加** |
| `--no-tui` | bool | **必须加** |
> `run` 是**触发操作**：执行前需向用户确认。
### Step 3：查看构建运行（build view / last / status）
```bash
gitee pipeline build view 123 --json --no-tui
gitee pipeline build view 123 -w --no-tui          # 每 2s 刷新至终态（SUCCESS/FAILED/CANCELLED/SKIPPED/TIMEOUT）
gitee pipeline build list -R owner/repo --json --no-tui         # 构建历史（分页，可 --file/--ref/--status/--order/--sort）
gitee pipeline build last -R owner/repo --file ci.yml --ref master --json --no-tui
gitee pipeline build status 123 --json --no-tui
```
**可用 flag：**
| Flag | 类型 | 说明 |
|------|------|------|
| `-R` / `--repo` | string | `owner/repo`，**必填** |
| `--ref` | string | 分支/标签（last 必填；list 可过滤） |
| `--file` | string | 流水线 YAML 文件名（last 必填；list 可过滤） |
| `--status` | string | list 过滤状态，逗号分隔（如 `FAILED,SUCCESS`） |
| `--order` | string | list 排序字段：`create_time`/`id`/`build_number`（默认 build_number） |
| `--sort` | string | list 排序方向：`asc`/`desc` |
| `-p` / `--page` | int | list 页码（默认 1） |
| `-L` / `--page-size` | int | list 页大小（默认 10） |
| `-w` / `--watch` | bool | build view 每 2s 刷新状态直至终态 |
| `--json` / `-j` | string | **必须加** |
| `--no-tui` | bool | **必须加** |
> `view <id>` / `status <id>` 以构建 ID 为参数；`last` 需 `--file --ref`。
> `build view` 默认输出含触发方式、耗时、流水线/阶段/任务出入参与日志（record.loggers[].logger URL）。
> `build list` 表格含 SOURCE/COMMIT 列：PR 触发显示 `PR #N`，push 显示分支；COMMIT 取
> `sources[].source.message`（PR 构建时为 PR 标题，对齐前端行为）。
> `status` 的 `data.stages[].jobs` 是二维数组（并行/串行）。
### Step 4：取消 / 重新构建（build cancel / rebuild）— 副作用，需确认
```bash
# 先确认再执行
gitee pipeline build cancel 123 --no-tui
gitee pipeline build rebuild 123 --json --no-tui
```
> ⚠️ `cancel`、`rebuild` 会改变运行状态，执行前必须向用户明确确认。
### Step 4.5：构建的阶段 / 任务运行操作（build stage / build job）— 副作用，需确认
阶段（stage）与任务（job）是 `pipeline build` 下的子命令，属于**构建运行期的实体**，
没有管理 / 配置的概念：不存在 list / create / update / delete / 配置类命令，也不能脱离
构建单独存在。它们的 ID 从 `build view` / `build status` 输出中取，只能对**仍在运行的
构建**做运行时操作（build stage 支持 `view/cancel/retry/continue`；build job 支持
`cancel/skip/retry/mark-success`），终态构建会直接报错。副作用操作
（cancel / retry / continue / skip / mark-success）非交互下必须 `--yes`，执行前需确认。
```bash
# 仓库流水线（-R 定位）
gitee pipeline build stage view 2278 -R owner/repo --json --no-tui
gitee pipeline build stage cancel 2278 -R owner/repo --yes --no-tui
gitee pipeline build stage retry 2278 -R owner/repo --yes --no-tui
gitee pipeline build stage continue 2278 -R owner/repo --yes --no-tui        # 继续暂停的阶段
gitee pipeline build job cancel 2730 -R owner/repo --yes --no-tui
gitee pipeline build job skip 2730 -R owner/repo --yes --no-tui
gitee pipeline build job retry 2730 -R owner/repo --yes --no-tui
gitee pipeline build job mark-success 2730 -R owner/repo --reason "manual ok" --yes --no-tui

# 项目流水线（-E/-P 定位，命令同构）
gitee pipeline program build stage view 2278 -E 2 -P 423 --json --no-tui
gitee pipeline program build stage cancel 2278 -E 2 -P 423 --yes --no-tui
gitee pipeline program build job skip 2730 -E 2 -P 423 --yes --no-tui
```
> 只有 `build stage view` 支持 `--json`；`build job` 系列仅输出确认信息。`mark-success` 可带 `--reason`。
### Step 5：插件（plugin list / scheme / example）与通用请求（pipeline request）
```bash
# 插件列表（按分类分组，全局但走仓库段网关）
gitee pipeline plugin list -R owner/repo --json --no-tui

# 查看某插件的参数 Scheme（--type 用 plugin list 输出的 type，即 json type）
gitee pipeline plugin list -R owner/repo --json --no-tui   # 先拿 type
gitee pipeline plugin scheme -R owner/repo --type maven-build@v1.0.0 --json --no-tui

# 生成示例 YAML 片段
gitee pipeline plugin example -R owner/repo --type maven-build@v1.0.0 --no-tui

# 触发 RemoteSelect 组件的远端下拉（如 GCC 版本 / Jenkins 任务选项）。
# 组件的 url.gitOps 自带 query（如 ?type=gcc），无需 -p；-p 仅用于覆盖/替换 ${identifier} 参数
gitee pipeline request PLUGIN_GCC_VERSION -R owner/repo --type gcc-build@v1.0.0 --json --no-tui

# 已登录 GET 一个 http(s) URL（如 build view 输出的 record.loggers[].logger 日志地址），无需 -R
gitee pipeline request "https://premium-k8s.gitee.cn/2/426/gitee-go/log-server/.../logs?recordUuid=..." --no-tui
```
**可用 flag：**
| Flag | 类型 | 说明 |
|------|------|------|
| `-R` / `--repo` | string | `owner/repo`，**必填** |
| `--type` | string | 插件 json job type（scheme/example/request **必填**） |
| `-p` / `--param` | string | request 端点参数 `key=value`（可重复，**可选**：组件 url 自带 query；`-p` 仅覆盖/替换 `${identifier}`） |
| `--json` / `-j` | string | **必须加**（example 为 YAML 文本，无 json） |
| `--no-tui` | bool | **必须加** |
> 插件 `type` 是 **json type**（如 `maven-build@v1.0.0`、`JENKINS_JOB`）。`request <component>` 的
> `<component>` 是 scheme 中 `RemoteSelect` 组件的 `identifier`（如 `PLUGIN_GCC_VERSION`），
> `plugin scheme` 会列出 `remote:` 端点路径与 `params:`。url.gitOps 是完整网关路由，CLI 会剥掉
> `/gitee-go/{service}` 前导段，避免重复拼接（否则网关 403）。
> **凭证 / 主机组件**：若插件 scheme 含 `Certification`（凭证）或 `HostSelect`（主机），CLI 目前
> **没有可用 API** 代为选择；提交（`pipeline commit`）后需**提示用户到页面二次编辑**，在凭证/主机
> 下拉中选中真实值，之后该流水线才能正常使用。
## 项目流水线（pipelineOps，`gitee pipeline program`）
**定位：按企业+项目**。所有命令必须以 `-E <企业id>`（enterprise）和 `-P <项目id>`（project）定位，
缺任一直接报错：`enterprise id is required: use --enterprise/-E <id>` / `program id is required: use --program/-P <id>`。
请求走 `/rest/v5/multi-source/` 网关段（如 `/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines`）。
与仓库流水线不同：**不做 gitee-go 开通检查、不需要 `--ref/--file`**；每个请求恰好一次调用。
> 项目流水线 ID 是**数字**：`program list` 输出即数字 id；也可以接受完整 identifier（`pipeline.ops.pipeline.706`）。
> 副作用操作（`delete` / `param delete` / `build cancel` / `history-apply` / `template delete|disable`）在
> 非交互下**必须 `--yes`/`-y`**，否则报 `--yes is required in non-interactive mode`。
> 注意：`build rebuild` 与 `template enable` 在 CLI 层**没有** `--yes`（直接执行），仍需按「副作用需确认」原则先向用户确认。

```bash
# 流水线：列表 / 查看 / 创建 / 更新 / 克隆 / 删除 / 版本历史 / 应用历史版本
gitee pipeline program list -E 2 -P 423 --json --no-tui    # 可加 --search 名称搜索 / --order asc|desc / --sort 字段
gitee pipeline program view 706 -E 2 -P 423 --json --no-tui
gitee pipeline program create -E 2 -P 423 --name build-all --json --no-tui          # 或 --body ./pipeline.json（job data 按下面「convertor 规则」组装）
gitee pipeline program edit 706 -E 2 -P 423 --name new-name --json --no-tui        # 或 --body ./pipeline.json（同上；update 为兼容别名）
gitee pipeline program clone 706 -E 2 -P 423 --json --no-tui
gitee pipeline program delete 706 -E 2 -P 423 --yes --no-tui
gitee pipeline program history 706 -E 2 -P 423 --json --no-tui
gitee pipeline program history-apply 706 -E 2 -P 423 --history 3 --yes --no-tui     # 应用历史版本

# 触发构建 / 构建管理（run 走 --pipeline，其余 build 子命令按构建 ID）
gitee pipeline program run --pipeline 706 -E 2 -P 423 --params "GITEE_BRANCH=master,FOO=bar" --json --no-tui
gitee pipeline program build list --pipeline 706 -E 2 -P 423 --json --no-tui
gitee pipeline program build view 9001 -E 2 -P 423 --json --no-tui                    # 也可 -w 每 2s 刷新至终态
gitee pipeline program build status 9001 -E 2 -P 423 --json --no-tui
gitee pipeline program build cancel 9001 -E 2 -P 423 --yes --no-tui
gitee pipeline program build rebuild 9001 -E 2 -P 423 --json --no-tui
gitee pipeline program build last --pipeline 706 -E 2 -P 423 --json --no-tui

# 模板：列表 / 查看 / 创建 / 更新 / 删除 / 启用 / 停用 / 分类
gitee pipeline program template list -E 2 -P 423 --json --no-tui
gitee pipeline program template view 12 -E 2 -P 423 --json --no-tui
gitee pipeline program template create -E 2 -P 423 --name deploy --description "build and deploy" --json --no-tui
gitee pipeline program template edit 12 -E 2 -P 423 --name new-name --json --no-tui   # 或 --description / --config（update 为兼容别名）
gitee pipeline program template delete 12 -E 2 -P 423 --yes --no-tui
gitee pipeline program template enable 12 -E 2 -P 423 --no-tui
gitee pipeline program template disable 12 -E 2 -P 423 --yes --no-tui
gitee pipeline program template categories -E 2 -P 423 --json --no-tui

# 插件 / 参数模板（与企业 + 项目定位，不取全局插件）
gitee pipeline program plugin list -E 2 -P 423 --json --no-tui
gitee pipeline program plugin scheme -E 2 -P 423 --type maven-build --json --no-tui
gitee pipeline program plugin example -E 2 -P 423 --type maven-build --no-tui
gitee pipeline program param list -E 2 -P 423 --json --no-tui
gitee pipeline program param view 12 -E 2 -P 423 --json --no-tui
gitee pipeline program param create -E 2 -P 423 --name build-params --json --no-tui   # 或 --description / --body ./params.json
gitee pipeline program param edit 12 -E 2 -P 423 --name new-name --json --no-tui     # 或 --description / --body（update 为兼容别名）
gitee pipeline program param clone 12 -E 2 -P 423 --json --no-tui
gitee pipeline program param delete 12 -E 2 -P 423 --yes --no-tui
```
### 子步骤：项目流水线 create/edit 的 JSON 组装（插件组件 convertor 规则）
`program create/edit --body <json>`（或 `--config`，值为 `-` 时读 stdin）提交 `PipelineRequest` JSON。
**每个 job 的 `data` 按插件 scheme 的组件 `convertor` 规则组装**（镜像 gitee-go
`Converter.toFormGeneric` 与前端 `PluginTypeRelation.ts`）。组装前先拿 scheme——
`--json=*` 输出完整 `config`，每项组件含 `type / identifier / options / children` 与
`convertor{parameters, yamlFiled, yamlArray}`：
```bash
gitee pipeline program plugin list -E 2 -P 423 --json --no-tui               # 先拿 json type（如 maven-build@v1.0.0）
gitee pipeline program plugin scheme -E 2 -P 423 --type maven-build@v1.0.0 --json=* --no-tui   # 看 convertor 与字段规则
```
**Job `data` 组装规则（按 scheme `config` 顺序逐项处理）：**
| 规则 | 说明 |
|------|------|
| 默认归属 | 每个顶层组件 → `data.parameters` 数组的一项 `{"key": <scheme identifier>, "value": <值>}`，按 scheme 顺序 |
| `convertor.parameters: false` | 该组件 → `data.<identifier>` **顶层字段**（不进 parameters）；`parameters` 缺省/为 true 都进 parameters |
| key 取值 | 项目流水线 create JSON 的 key **永远是 scheme `identifier`**；`convertor.yamlFiled` / `yamlArray` 只用于仓库流水线 YAML↔JSON 映射（YAML key 用 yamlFiled、yamlArray 表示数组），**组装 create JSON 时不参与** |
| Compose | 子组件不单独成项，值挂在父项 value 内并 **JSON 字符串化**（如 artifacts `"[{\"name\":\"BUILD_ARTIFACT\",\"path\":[\"./target\"]}]"`）；Compose array / ObjectArray → JSON 字符串化数组 |
| 重复 identifier | 最后出现者胜（如同一字段的 RefValue 变体 + 真实 Input） |
| Switch / 无 options 的 Checkbox | `"true"` / `"false"` 字符串 |
| Select / Radio | 单个值（取 options 中一项的 key） |
| Checkbox（有 options）/ Select.multiple | 数组 |
| **Password** | 标量字符串，且 parameters 条目**额外带 `"type": "PASSWORD"`**（如 `{"key":"PLUGIN_TOKEN","value":"s3cr3t","type":"PASSWORD"}`，供表单按密文渲染） |
| Certification（凭证） / HostSelect（主机） | **目前没有可用 API 拉取可选凭证/主机列表，也无法替用户选择**；组装时可临时填凭证名称/引用或主机 ID/名称（向导即此行为），**提交后必须提示用户到页面完成二次编辑**（在网页端下拉里选中真实凭证/主机）后方可使用 |
| Input / Textarea / Number / UserSelect / RemoteSelect / Command | 标量字符串（Command 多行；空值填 `""`） |
| RefValue | 用兄弟字段的值替换 `${sibling}` 解析（同字段另有真实 Input 时该 Input 胜出），不单独成键 |
| RefExport | 上游任务导出，CLI 不设置 |

**`PipelineRequest` JSON 骨架（`--body` 提交，job 混合两种归属的示例）：**
```json
{
  "name": "build-all",
  "stages": [
    {
      "name": "build",
      "jobs": [
        [
          {
            "name": "compile",
            "type": "maven-build@v1.0.0",
            "data": {
              "parameters": [
                {"key": "PLUGIN_JAVA_VERSION", "value": "8"},
                {"key": "PLUGIN_TOKEN", "value": "s3cr3t", "type": "PASSWORD"},
                {"key": "PLUGIN_ARTIFACTS", "value": "[{\"name\":\"BUILD_ARTIFACT\",\"path\":[\"./target\"]}]"}
              ],
              "PLUGIN_SKIP": "false"
            }
          }
        ]
      ]
    }
  ]
}
```
> `stages[].jobs` 是**二维数组**（并行组）；`sources`（代码源）、`triggers`（源触发/定时）、
> `parameters`（流水线级参数模板）、`labels` / `group`（分组）均可选。`job.identifier` 创建时由后端
> 分配，**CLI 不主动设置**；`job.type` 必须用 scheme 的 json type。示例中 `PLUGIN_SKIP` 即
> `convertor.parameters: false` 的组件（Switch），放 `data` 顶层。`program plugin example` 给出的是
> YAML 片段，字段名可对照，但**组装 JSON 一律按上表规则**。
> **凭证 / 主机组件（务必告知用户）**：若插件的 scheme 含 `Certification`（凭证）或
> `HostSelect`（主机）组件，**目前没有可用 API** 让 CLI/组装去拉取并选择（也不能用
> `pipeline request` 拿选项）——这些字段只能先填占位文本。**提交/更新后必须提示用户**：
> 去页面对该任务做二次编辑，在凭证/主机下拉里选中真实值，之后该流水线才能正常使用。

**判断方式：** 用户提到「项目流水线 / pipelineOps / program」或提供 `-E/-P` 数值（如「2/423」）→ 用上面的
`program` 命令；否则按仓库流水线（`-R`）处理。

## 判断逻辑
| 用户意图 | 执行命令 | 需确认 |
|---------|---------|--------|
| "看仓库流水线 / 流水线列表" | Step 1 list | 否 |
| "看某条流水线配置" | Step 1 view | 否 |
| "要个流水线 YAML 示例 / 生成示例流水线（整条）" | Step 1.6 example | 否 |
| "触发构建 / 跑一下流水线" | Step 2 run | **是（触发）** |
| "看构建 / 最后一次构建" | Step 3 build view/last | 否 |
| "构建状态" | Step 3 build status | 否 |
| "取消构建" | Step 4 build cancel | **是（改变状态）** |
| "重新构建" | Step 4 build rebuild | **是（改变状态）** |
| "阶段/任务运行操作 / 跳过任务（运行时实体）" | Step 4.5 `build stage/job`（cancel/retry/continue/skip/mark-success） | **是（改变状态）** |
| "继续暂停的阶段" | Step 4.5 `build stage continue` | **是（改变状态）** |
| "插件 / 有哪些插件" | Step 5 plugin list | 否 |
| "插件参数/怎么配这插件" | Step 5 plugin scheme | 否 |
| "插件示例 / 示例 yaml" | Step 5 plugin example | 否 |
| "拉取插件下拉项 / 版本列表" | Step 5 pipeline request（组件模式） | 否 |
| "抓取日志 URL / 已登录 GET 某地址" | Step 5 pipeline request（URL 模式，无需 -R） | 否 |
| "项目流水线 / pipelineOps" | `pipeline program ...`（-E 企业id -P 项目id） | 否 |
| "项目流水线列表/配置" | `pipeline program list / view`（-E -P） | 否 |
| "创建/编辑项目流水线 JSON / 组装 job data（组件 convertor）" | `pipeline program create/edit --body <json>`（-E -P，job data 按「convertor 规则」子步骤组装） | **是（提交配置）** |
| "触发项目流水线构建" | `pipeline program run --pipeline <id>`（-E -P） | **是（触发）** |
| "项目流水线构建查看/状态" | `pipeline program build view/last/status`（-E -P） | 否 |
| "取消/重建项目流水线构建" | `pipeline program build cancel / rebuild`（-E -P） | **是（改变状态）** |
| "项目流水线阶段/任务运行操作（运行时实体）" | `pipeline program build stage/job ...`（-E -P） | **是（改变状态）** |
| "项目流水线模板/参数模板" | `pipeline program template list / param list`（-E -P） | 否 |
## 完整示例
```bash
# 罗列 master 上的流水线并查看 ci.yml
gitee pipeline list -R autodeploy/java-maven-example --ref master --json --no-tui
gitee pipeline view -R autodeploy/java-maven-example --ref master --file .gitee/pipelines/ci.yml --json --no-tui

# 拉一份整条仓库流水线 YAML 示例作模板
gitee pipeline example -R autodeploy/java-maven-example --no-tui

# 触发构建并确认
gitee pipeline run -R autodeploy/java-maven-example --ref master --file ci.yml --params BRANCH=main --json --no-tui

# 查看最近一次构建与状态
gitee pipeline build last -R autodeploy/java-maven-example --file ci.yml --ref master --json --no-tui
gitee pipeline build view 123 --json --no-tui

# 插件：找 Maven 构建的类型，看参数，拉版本下拉
gitee pipeline plugin list -R autodeploy/java-maven-example --json --no-tui
gitee pipeline plugin scheme -R autodeploy/java-maven-example --type maven-build@v1.0.0 --json --no-tui
gitee pipeline request PLUGIN_JAVA_VERSION -R autodeploy/java-maven-example --type maven-build@v1.0.0 --json --no-tui
```
## 错误处理
| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `repo is required for pipeline commands` | 缺 `-R owner/repo` | 补 `-R owner/repo` |
| `no scheme for job type` / 400 | `--type` 不是该插件的 json type | 先 `plugin list --json` 拿 `type` |
| `no RemoteSelect component` | `request` 的组件 identifier 不对 | `plugin scheme` 看 config 里的 identifier |
| `gitee-go is not enabled for ...` | gitee-go 未开通 | 报错信息带开通页面地址，告诉用户自行打开页面开通后重试（无 `--with-open` 自动代开） |
| `gitee-go: HTTP 404` | 仓库路径 / host 解析不对 | 确认 `-R` 拼写与 `go_api_host` 配置 |
| `enterprise id is required: use --enterprise/-E <id>` | 项目流水线缺 `-E` | 补 `-E <企业id>` |
| `program id is required: use --program/-P <id>` | 项目流水线缺 `-P` | 补 `-P <项目id>` |
| `required flag(s) "pipeline" not set` | `program run` / `build list` 缺 `--pipeline` | 补 `--pipeline <id>`（先 `program list` 拿 id） |
| `--pipeline <pipeline-id> is required` | `build last` 缺 `--pipeline` | 补 `--pipeline <id>` |
| `--yes is required in non-interactive mode` | 项目副作用操作缺确认 | 加 `--yes`/`-y` |
| `belongs to build ... already finished` | stage/job 操作要求所属构建仍在运行 | 构建已终态，无法再 cancel/retry/skip，改用 `build view` 查看结果 |
| `authentication required` | 未认证 | `gitee auth login`（或检查 `.gitee-go/cookie`） |