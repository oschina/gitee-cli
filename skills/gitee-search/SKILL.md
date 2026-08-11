---
name: gitee-search
description: Search across all of Gitee for repositories, issues, and users with rich filters (language, owner, state, author, assignee, label, sort). Use when the user says "搜索仓库", "找一下 Go 的项目", "搜 issue", "查一下有没有相关 issue", "搜索用户", "search gitee for X", "find repos in a language", or "search issues/users". This is GLOBAL discovery across Gitee; use gitee-issue to list or view issues in a specific repository. Always uses `--json` and `--no-tui`; read-only, never mutates; never uses `--ai`.
metadata:
  author: gitee
  version: "1.0"
---

在**整个 Gitee 平台**范围内搜索仓库、issue 和用户。

> 边界说明：本 skill 是**全站搜索**。若只想列出或查看「当前仓库自己的 issue」，用 **gitee-issue**；若想搜索特定接口/endpoint，用 **gitee-api --search**。

## 前置检查

- **已认证**：`gitee auth status --no-tui`，失败则提示 `gitee auth login`。

---

## 执行步骤

### Step 1：搜索仓库（search repos）

```bash
gitee search repos gitee-cli --json --no-tui
gitee search repos sdk --owner oschina --language Go --json --no-tui
gitee search repos database --sort stars_count --order desc --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--language` | string | — | 按编程语言过滤，如 `Go` |
| `--owner` | string | — | 按 namespace path 过滤 |
| `--fork` | bool | false | 包含 fork 仓库 |
| `--sort` | string | 最佳匹配 | `last_push_at` \| `stars_count` \| `forks_count` \| `watches_count` |
| `--order` | string | `desc` | `asc` \| `desc` |
| `--limit` / `-l` | int | `20` | 每页数量（最大 100） |
| `--page` / `-p` | int | `1` | 页码 |
| `--json` / `-j` | string | — | **必须加** |
| `--no-tui` | bool | — | **必须加** |

**常用 JSON 字段：** `full_name` `description` `language` `stargazers_count` `forks_count` `html_url` `namespace`。

### Step 2：搜索 issue（search issues）

```bash
gitee search issues timeout --json --no-tui
gitee search issues bug --repo oschina/gitee --state open --json --no-tui
gitee search issues crash --label bug --assignee alice --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--repo` / `-R` | string | — | 限定仓库 `OWNER/REPO` |
| `--state` / `-s` | string | — | `open` \| `progressing` \| `closed` \| `rejected` |
| `--author` | string | — | 按作者用户名过滤 |
| `--assignee` / `-A` | string | — | 按负责人用户名过滤（**大写 -A**） |
| `--label` | string | — | 按标签过滤 |
| `--language` | string | — | 按仓库语言过滤 |
| `--sort` | string | 最佳匹配 | `created_at` \| `updated_at` \| `notes_count` |
| `--order` | string | `desc` | `asc` \| `desc` |
| `--limit` / `-l` | int | `20` | 每页数量（最大 100） |
| `--page` / `-p` | int | `1` | 页码 |
| `--json` / `-j` | string | — | **必须加** |
| `--no-tui` | bool | — | **必须加** |

### Step 3：搜索用户（search users）

```bash
gitee search users alice --json --no-tui
gitee search users "alice bob" --sort joined_at --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--sort` | string | 最佳匹配 | `joined_at` |
| `--order` | string | `desc` | `asc` \| `desc` |
| `--limit` / `-l` | int | `20` | 每页数量（最大 100） |
| `--page` / `-p` | int | `1` | 页码 |
| `--json` / `-j` | string | — | **必须加** |
| `--no-tui` | bool | — | **必须加** |

**常用 JSON 字段：** `login` `name` `html_url` `type`。

---

## 判断逻辑

| 用户意图 | 执行命令 |
|---------|---------|
| "搜仓库" / "找 X 语言的项目" | Step 1 search repos（可加 `--language` / `--owner` / `--sort`） |
| "搜 issue" / "有没有类似的 issue" | Step 2 search issues（可加 `--repo` / `--state` / `--label`） |
| "搜用户" / "找某个人的账号" | Step 3 search users |
| "列出**当前仓库**的 issue" | ❌ 不属于本 skill → 用 **gitee-issue** |

---

## 完整示例

```bash
# 找 Go 语言、按 star 排序的数据库相关仓库
gitee search repos database --language Go --sort stars_count --order desc --limit 10 --json --no-tui

# 在指定仓库里搜 open 状态、带 bug 标签的 issue
gitee search issues 崩溃 --repo oschina/gitee --state open --label bug --json --no-tui

# 搜用户
gitee search users oschina --json --no-tui
```

---

## 错误处理

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `401` | 未认证 | 提示 `gitee auth login` |
| 返回空数组 `[]` | 没有匹配结果 | 放宽关键词或去掉部分过滤条件重试 |
| `422` | 过滤参数取值非法 | 对照上表校正 `--sort` / `--state` / `--order` 的可选值 |
| 结果太多不精准 | 关键词太宽泛 | 增加 `--language` / `--owner` / `--repo` 等过滤条件缩小范围 |
