---
name: gitee-issue
description: >-
  Manage the full Gitee issue lifecycle: list, view, create, edit, comment,
  close, reopen, assign, and start work on an issue. Use when the user says
  "查看 issue", "列出 issue", "创建 issue", "提个 issue", "编辑 issue",
  "评论 issue", "关闭 issue", "重新打开 issue", "把 issue 指派给 X",
  "认领 issue", "开始处理 issue", or
  "list/view/create/edit/comment/close/reopen/assign/start an issue". This
  skill uses direct issue commands first and falls back to the gitee-api
  skill when an operation is unsupported. Always uses `--json` and
  `--no-tui`; never uses `--ai`. API updates and deletions require a second
  explicit user confirmation.
metadata:
  author: gitee
  version: "1.0"
---

Issue 的完整生命周期：列出 / 查看 / 创建 / 编辑 / 评论 / 关闭 / 重开 / 指派 / 认领并开始处理。

需要先通过 `gitee auth login` 完成认证，并在带 Gitee remote 的 git 仓库中执行或传入 `-R owner/repo`。

## 前置检查

1. **已认证**：`gitee auth status --no-tui`，失败则提示 `gitee auth login`。
2. **Issue 编号是字母数字串**（如 `ICX4FO`），**不是整数**，别当成 `#42` 那样的 PR 编号。
3. 优先使用直接 Issue 命令；不支持目标操作时切换到 `gitee-api` skill，先 `gitee api --search` 查 schema。通过 API 更新或删除资源前必须二次确认。

---

## 执行步骤

### Step 1：列出 / 查看 issue（list / view）

使用仓库级 `issue list`，不要用全站 `search issues` 代替：

```bash
gitee issue list --state open --json=number,title,state,assignee,labels --no-tui
gitee issue list --query "登录" --state all --json --no-tui
gitee issue view ICX4FO --json --no-tui
```

`list` 常用筛选：`--state/-s`、`--assignee/-A`、`--labels`、`--query`、`--sort`、`--direction`、`--limit/-l`、`--page/-p`。`list` 和 `view` 都必须加 `--json`、`--no-tui`。

### Step 2：创建 issue（create）

非交互模式**必须提供 `-t` 标题**：

```bash
gitee issue create -t "登录页空密码崩溃" -b "复现步骤：..." --json --no-tui
gitee issue create -t "支持 OAuth 登录" -a alice --labels feature,auth --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 说明 |
|------|------|------|
| `--title` / `-t` | string | 标题（非交互模式**必填**） |
| `--body` / `-b` | string | 正文描述 |
| `--assignee` / `-a` | string | 指派给某用户（**登录名**，非昵称） |
| `--labels` | string | 标签，逗号分隔，如 `bug,urgent` |
| `--json` / `-j` | bool | **必须加** |
| `--no-tui` | bool | **必须加** |

> ⚠️ **禁止使用 `gitee issue create --ai`**：该 flag 会把一句话扩写成结构化 issue 并进入交互确认，导致阻塞。issue 正文由 Agent 自行组织后用 `-b` 传入。

创建成功后，以返回 JSON 中的 `number` 为准。若创建时设置了 `--labels` 或 `--assignee`，再执行一次 `view` 核验最终状态；创建命令的即时响应可能暂未包含这些关联字段：

```bash
gitee issue view ICX4FO --json=number,title,state,assignee,labels --no-tui
```

### Step 3：编辑 issue（edit）

非交互模式下至少提供一个编辑字段：

```bash
gitee issue edit ICX4FO -t "新标题" -b "更新后的描述" --json --no-tui
gitee issue edit ICX4FO -a bob --json --no-tui
gitee issue edit ICX4FO --labels bug,urgent --milestone 12 --json --no-tui
gitee issue edit ICX4FO --body "" --assignee "" --labels "" --json --no-tui
```

**可用 flag：** `--title/-t`、`--body/-b`、`--assignee/-a`、`--labels`、`--milestone`、`--json/-j`（**必须加**）、`--no-tui`（**必须加**）。空正文、负责人或标签表示清空该字段。

### Step 4：评论 issue（comment）

**务必传 `-b`**，否则会打开编辑器等待输入而阻塞：

```bash
gitee issue comment ICX4FO -b "已在最新版本修复，请验证" --json --no-tui
```

**可用 flag：** `--body/-b`（**必填，否则打开编辑器阻塞**）、`--json/-j`（**必须加**）、`--no-tui`（**必须加**）。

### Step 5：指派 issue（assign）

assignee 是 Gitee **登录名**，不是显示昵称：

```bash
gitee issue assign ICX4FO alice --json --no-tui
```

**可用 flag：** `--json/-j`（**必须加**）、`--no-tui`（**必须加**）。

### Step 6：关闭 / 重开 issue（close / reopen）

```bash
gitee issue close ICX4FO --json --no-tui
gitee issue reopen ICX4FO --json --no-tui
```

**可用 flag：** `--json/-j`（**必须加**）、`--no-tui`（**必须加**）。

### Step 7：认领并开始处理 issue（start）

先查看 Issue 和当前登录用户，再创建工作分支并自指派。分支名使用 Issue 编号，方便追踪：

```bash
gitee auth status --no-tui
gitee issue view ICX4FO --json=number,title,state,assignee --no-tui
git status --short
git switch -c issue/ICX4FO-login-empty-password
gitee issue assign ICX4FO alice --json --no-tui
```

其中 `alice` 必须替换为 `auth status` 显示的当前登录名。若工作区存在未提交改动，创建分支前先确认不会影响用户现有工作；不要擅自清理或覆盖改动。

> 所有 `gitee issue` 命令均支持 `-R owner/repo` 指定仓库（不在 git 目录时使用）。

---

## 判断逻辑

| 用户意图 | 执行命令 |
|---------|---------|
| "看看有哪些 issue" / "查看这个 issue" | Step 1 list / view |
| "提个 issue" / "创建问题" | Step 2 create（记得 `-t`） |
| "改一下 issue 标题/描述/负责人" | Step 3 edit |
| "给 issue 留言/评论" | Step 4 comment（记得 `-b`） |
| "把 issue 指派给 X" | Step 5 assign（X 用登录名） |
| "关闭这个 issue" | Step 6 close |
| "重新打开 issue" | Step 6 reopen |
| "我来做这个 issue" / "认领并开始处理" | Step 7 start |

---

## 完整示例

```bash
# 创建一个带标签和负责人的 issue
gitee issue create \
  -t "fix: 登录页空密码返回 500" \
  -b "## 现象\n空密码提交返回 500\n\n## 期望\n返回 400 并提示" \
  -a alice \
  --labels bug,urgent \
  --json --no-tui

# 使用 create 返回的 Issue 编号核验标签和负责人
gitee issue view ICX4FO --json=number,title,state,assignee,labels --no-tui

# 补一条评论
gitee issue comment ICX4FO -b "已定位到 auth 模块，PR #42 修复中" --json --no-tui

# 修复合入后关闭
gitee issue close ICX4FO --json --no-tui
```

---

## 错误处理

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `--title is required` | 创建时未传 `-t` | 补 `-t` 标题 |
| `at least one of ... is required` | edit 未传任何字段 | 至少提供一个待改字段 |
| create 返回的 `labels` / `assignee` 为空 | 即时响应尚未包含关联字段 | 用返回的 Issue 编号执行 `issue view --json --no-tui` 核验最终状态 |
| 命令卡住无响应 | comment 未传 `-b`，进入编辑器 | 中断并用 `-b "..."` 重跑 |
| `failed to ...: 404` | issue 编号错误 | issue 号是字母数字串（如 `ICX4FO`），用 `gitee issue list --json --no-tui` 核对 |
| 指派后负责人不对 | 用了昵称而非登录名 | 用 `gitee search users <关键词> --json --no-tui` 查准确 `login` |
| `401` | 未认证 | 提示 `gitee auth login` |
