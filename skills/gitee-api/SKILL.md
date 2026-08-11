---
name: gitee-api
description: Call any Gitee V5 REST API endpoint through the gitee CLI when no first-class command supports the requested operation. Discover the endpoint and schema with `gitee api --search`, then issue the request. Use when the user says "调用 Gitee API", "调接口", "gitee api", "没有现成命令怎么办", asks for milestones, labels, collaborators, webhooks, members, or requests an operation unsupported by a direct gitee command. Always uses `--no-tui`; never uses `--ai`; requires a second explicit user confirmation before resource updates (PUT/PATCH) and deletions (DELETE).
metadata:
  author: gitee
  version: "1.0"
---

`gitee api` 是一级命令不支持某项操作时的降级通道。优先使用 `gitee pr/issue/repo/release/...`；缺少对应能力时，再用 `--search` 查 OpenAPI schema 并调用底层接口。

需要先通过 `gitee auth login` 完成认证；仓库级 endpoint 还需要明确目标 `owner/repo`。

## 前置检查

1. **已认证**：`gitee auth status --no-tui`，失败则提示 `gitee auth login`。
2. **先检查直接命令**：运行目标资源的 `--help` 或依据对应资源 skill 判断是否已有直接子命令；已有则使用直接命令，不绕到 API。
3. **先发现，再调用**：直接命令不支持时，必须先用 `--search` 从 OpenAPI 规范中检索准确的 endpoint、HTTP 方法和参数，不凭记忆猜路径或 schema。
4. **区分风险**：`GET` 可直接执行；创建操作必须来自用户明确请求；`PUT` / `PATCH` / `DELETE` 即使用户已经提出，也必须在执行前再次展示目标和完整请求并等待确认。

---

## 执行步骤

### Step 1：检查直接命令

先确认一级命令是否支持目标操作，例如：

```bash
gitee pr --help --no-tui
gitee issue edit --help --no-tui
gitee release --help --no-tui
```

有对应能力时直接使用；没有时才进入 Step 2。

### Step 2：发现 endpoint（`--search`）

用关键词从 Gitee OpenAPI 规范中检索匹配的 endpoint，输出包含路径、方法和参数说明：

```bash
gitee api --search "milestone" --no-tui
gitee api --search "创建 label" --limit 50 --no-tui
gitee api --search "webhook" --page 2 --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--search` | string | — | 按关键词检索 endpoint（中英文均可） |
| `--limit` | int | `20` | 每页结果数 |
| `--page` | int | `1` | 页码 |
| `--no-tui` | bool | — | **必须加**，禁用 TUI |

**输出示例：**
```
Found 5 matching endpoint(s):

  GET    /repos/{owner}/{repo}/milestones
        获取仓库所有里程碑
        [Milestones]
        * owner (string, path) - 仓库所属空间地址(path)
        * repo  (string, path) - 仓库路径(path)
          state (string, query) one of: open|closed|all - 里程碑状态。默认: open

  POST   /repos/{owner}/{repo}/milestones
        创建仓库里程碑
        [Milestones]
        * owner (string, path)
        * repo  (string, path)
        * title (string, form) - 里程碑标题
        * due_on (string, form) - 截止日期
          state (string, form) - open|closed|all
```

> 参数标注含义：`* ` 前缀 = **必填**；`path` = 拼进 URL 路径；`query` = 拼进 query string；`form` = 作为请求体字段（用 `-f key=value` 传）。

### Step 3：解析 schema 并发起请求

把 `--search` 得到的 endpoint 里的 `{owner}` / `{repo}` / `{number}` 等占位符替换为真实值后调用。

**只读（GET，可直接执行）：**
```bash
# path 参数直接拼进路径；query 参数拼在 ? 后面
gitee api "/repos/oschina/gitee/milestones?state=open" -p --no-tui
```

**写操作（更新和删除必须先二次确认，见下方协议）：**
```bash
# 用 -f 逐个传 form 字段（可重复），-X 指定方法
gitee api "/repos/oschina/gitee/milestones" -X POST \
  -f title="v1.2.0" \
  -f due_on="2026-12-31T00:00:00+08:00" \
  -p --no-tui

# 或用 --body 传原始 JSON（与 -f 二选一）
gitee api "/repos/oschina/gitee/issues/ICX4FO/labels" -X POST \
  --body '["bug","urgent"]' \
  -p --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-X` / `--method` | string | `GET` | HTTP 方法：`GET` \| `POST` \| `PUT` \| `PATCH` \| `DELETE` |
| `-f` / `--field` | string(可重复) | — | 请求体字段 `key=value`，多个字段重复传 |
| `--body` | string | — | 原始 JSON 请求体（与 `-f` 二选一，适合数组/嵌套结构） |
| `-H` / `--header` | string(可重复) | — | 自定义请求头 `Key: Value` |
| `-p` / `--pretty` | bool | false | 美化输出 JSON（**建议加**，便于解析） |
| `--no-tui` | bool | — | **必须加** |

> `--search`、`--limit`、`--page` 三个 flag 仅在**检索模式**（带 `--search`）下生效；发起真实请求时不要用它们做分页，分页请拼在 endpoint 的 query string 里，如 `?page=2&per_page=50`。

---

## 判断逻辑

| 用户意图 | HTTP 方法 | 是否需确认 |
|---------|-----------|-----------|
| "看看有哪些接口" / "有没有 X 的接口" | 仅 `--search` | 否 |
| "查询 / 列出 / 获取 X" | `GET` | 否，直接执行 |
| "创建 / 新增 X" | `POST` | 用户已明确要求且目标完整时可执行 |
| "修改 / 更新 X" | `PUT` / `PATCH` | **是，必须二次确认** |
| "删除 X" | `DELETE` | **是**（不可逆，强确认） |

### 更新和删除确认协议

`gitee api` 绕过一级命令的保护逻辑。执行 `PUT`、`PATCH` 或 `DELETE` 前：

1. 尽可能先用对应 GET endpoint 读取目标资源，确认 ID、名称和当前状态。
2. 展示方法、完整路径、请求体以及会改变或删除的资源。
3. 等待用户再次明确确认；此前的“帮我更新/删除”请求不代替这次确认。
4. 用户确认后执行；拒绝或没有答复时停止。

```
即将发起写操作：
  方法: DELETE
  路径: /repos/oschina/gitee/milestones/12
  说明: 删除仓库里程碑 #12（不可逆）
确认执行吗？(y/n)
```

删除类操作必须明确标注「不可逆」。创建操作不需要机械地二次确认，但目标、权限范围或字段不明确时仍应先询问。

---

## 完整示例

```bash
# ── 场景 1：给仓库加一个里程碑 ──────────────────────
# 1. 发现 endpoint
gitee api --search "创建里程碑" --no-tui
#    → POST /repos/{owner}/{repo}/milestones  (title*, due_on*)

# 2. 用户明确要求后创建
gitee api "/repos/oschina/gitee/milestones" -X POST \
  -f title="2026 Q1" \
  -f due_on="2026-03-31T00:00:00+08:00" \
  -p --no-tui

# ── 场景 2：查看仓库协作者（只读，直接执行）──────────
gitee api --search "collaborators" --no-tui
gitee api "/repos/oschina/gitee/collaborators" -p --no-tui

# ── 场景 3：给 issue 打标签 ─────────────────────────
gitee api "/repos/oschina/gitee/issues/ICX4FO/labels" -X POST \
  --body '["bug","help wanted"]' \
  -p --no-tui
```

---

## 错误处理

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `401 Unauthorized` | token 未认证或已失效 | 提示执行 `gitee auth login` |
| `404 Not Found` | endpoint 路径拼错或占位符没替换 | 重新 `--search` 核对路径，检查 `{owner}/{repo}/{number}` 是否都替换成真实值 |
| `403 Forbidden` | 无该操作权限 | 告知用户需要更高的仓库/组织权限 |
| `422 Unprocessable` | 缺少必填字段或字段格式错 | 对照 `--search` 输出中带 `*` 的必填参数补齐 |
| `--search` 无结果 | 关键词太具体或用错语言 | 换更通用的关键词（中/英）重试，如 "issue" 而非 "issue 标签管理" |

> ⚠️ **禁止使用任何 `--ai` flag**；本 skill 不涉及 AI 生成，所有分析由 Agent 基于返回的 JSON 自行完成。
