---
name: gitee-pr
description: Manage the full pull request lifecycle on Gitee — create, list, view, edit, diff, review/approve, comment, merge, close, reopen, and fetch/checkout PRs locally. Use when the user says "创建 PR", "提个 PR", "看看 PR", "编辑 PR", "列出 PR", "PR diff / 看改动", "review PR", "评论 PR", "合并 PR", "关闭 PR", "重开 PR", "把 PR 拉到本地", or asks to create/list/view/edit/diff/review/merge/close/reopen/checkout a pull request. Always uses `--json` where supported and `--no-tui`; never uses `--ai`; requires explicit user confirmation before merging. If a requested PR operation has no direct command, fall back to the gitee-api skill to search the schema; API updates and deletions require a second confirmation.
metadata:
  author: gitee
  version: "1.0"
---

Pull Request 的**完整生命周期**：创建 / 列出 / 查看 / 编辑 / 看 diff / 审查 / 评论 / 合并 / 关闭 / 重开 / 拉到本地。

需要先通过 `gitee auth login` 完成认证，并在带 Gitee remote 的 git 仓库中执行或传入 `-R owner/repo`；`checkout/fetch` 必须有本地工作副本。

> PR 编号是**整数**（如 `42`），区别于 issue 编号（字母数字串如 `ICX4FO`）。

## 前置检查

1. **已认证**：`gitee auth status --no-tui`，失败则提示 `gitee auth login`。
2. **创建/checkout 需在 git 仓库内**（或用 `-R owner/repo` 指定远端仓库）。
3. **合并是可见且难以撤销的操作**：`pr merge` 执行前必须向用户明确确认。
4. **禁止 `--ai`**：`pr create --ai` 与 `pr review --ai` 都会触发交互式确认 / TUI pager 而阻塞；标题、正文、review 结论均由 Agent 自行组织后用 `-t/-b` 传入。
5. 直接命令不支持用户要求的 PR 操作时，切换到 `gitee-api` skill，先 `gitee api --search` 查 schema；通过 API 更新或删除资源前必须二次确认。

---

## 执行步骤

### Step 1：创建 PR（create）

非交互模式**必须提供 `-t` 标题**。base 分支优先用用户指定值，否则默认由 git remote 推断（兜底 `master`）：

```bash
gitee pr create -t "fix(auth): 修复 token 过期竞态" -b "## 变更\n..." --base master --json --no-tui
gitee pr create -t "feat: 支持草稿" --draft --assignees alice,bob --testers carol --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--title` / `-t` | string | — | PR 标题（非交互模式**必填**） |
| `--body` / `-b` | string | — | PR 描述正文（Markdown） |
| `--base` | string | git remote 推断 / `master` | 目标分支 |
| `--head` | string | 当前分支 | 源分支 |
| `--draft` | bool | false | 创建为草稿 PR |
| `--assignees` | string | — | 负责人用户名，逗号分隔，如 `alice,bob` |
| `--testers` | string | — | 测试者用户名，逗号分隔 |
| `--no-template` | bool | false | 跳过 PR 描述模板 |
| `--json` / `-j` | bool | false | **必须加** |
| `--no-tui` | bool | — | **必须加** |

> ⚠️ **禁止 `gitee pr create --ai`**：它会读取 diff 生成标题/正文并进入交互确认，导致阻塞。请由 Agent 自行读取 `git diff` / `git log` 生成后用 `-t`/`-b` 传入。

### Step 2：列出 PR（list）

```bash
gitee pr list --state open --json --no-tui
gitee pr list --author alice --base master --json --no-tui
gitee pr list --labels bug,performance --sort updated --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--state` / `-s` | string | `open` | `open` \| `closed` \| `merged` \| `all` |
| `--author` | string | — | 按作者用户名过滤 |
| `--assignee` | string | — | 按负责人用户名过滤 |
| `--tester` | string | — | 按测试者用户名过滤 |
| `--head` | string | — | 按源分支过滤 |
| `--base` | string | — | 按目标分支过滤 |
| `--labels` | string | — | 按标签过滤，逗号分隔 |
| `--milestone-number` | int | — | 按里程碑序号过滤 |
| `--since` | string | — | 更新时间晚于（ISO 8601） |
| `--sort` | string | `created` | `created` \| `updated` \| `popularity` \| `long-running` |
| `--direction` | string | `desc` | `asc` \| `desc` |
| `--limit` / `-l` | int | `20` | 每页数量 |
| `--page` / `-p` | int | `1` | 页码 |
| `--json` / `-j` | string | — | **必须加**；可选字段 `number,title,state,user.login,head.ref,base.ref` 等 |
| `--no-tui` | bool | — | **必须加** |

### Step 3：查看 PR（view）

```bash
gitee pr view 42 --json --no-tui
gitee pr view 42 --json=number,title,state,head.ref,base.ref --no-tui
```

**可用 flag：** `--json/-j`（**必须加**，可选字段 `number,title,body,state,draft,mergeable,head,base,user,assignees,testers,html_url,created_at,merged_at` 等）、`--no-tui`（**必须加**）。

### 编辑 PR（edit）

```bash
gitee pr edit 42 -t "新标题" -b "更新后的描述" --json --no-tui
gitee pr edit 42 --body "" --json --no-tui
gitee pr edit 42 --draft=false --json --no-tui
```

**可用 flag：** `--title/-t`、`--body/-b`、`--draft`、`--json/-j`（**必须加**）、`--no-tui`（**必须加**）。非交互模式至少提供一个编辑字段。

### Step 4：查看 diff（diff）

```bash
gitee pr diff 42 --no-tui
gitee pr diff 42 --json=filename,status,additions,deletions --no-tui
```

> 不带 `--json` 时输出 unified diff，适合代码审查；带 `--json` 时输出文件级结构化摘要，适合统计和自动化。两种模式都必须加 `--no-tui`，避免进入 TUI pager。

**可用 flag：** `--json/-j`（可选；字段包括 `filename`、`status`、`additions`、`deletions`、`patch`）、`--no-tui`（**必须加**）。

### Step 5：审查 / 通过 PR（review）

不带 `--ai` 时，`pr review` 执行的是**通过审查（approve）**动作；带 `-b` 可在通过时附评论：

```bash
gitee pr review 42 --no-tui              # 通过
gitee pr review 42 -b "LGTM" --no-tui    # 通过并附评论
gitee pr review 42 --force --no-tui      # 管理员强制通过（审查要求未满足时）
```

**可用 flag：**

| Flag | 类型 | 说明 |
|------|------|------|
| `--body` / `-b` | string | 通过后附带的评论 |
| `--force` | bool | 管理员强制通过（审查要求未满足时，仅 admin） |
| `--no-tui` | bool | **必须加** |

> ⚠️ **禁止 `gitee pr review 42 --ai`**：该 flag 会拉取 diff 做 AI review 并在 pager 中展示，导致阻塞。若需 AI 分析，请由 Agent 自行 `pr view` + `pr diff` 后组织 review 结论，再用 `pr comment` 留言或 `pr review` 通过。

### Step 6：评论 PR（comment）

**务必传 `-b`**，否则会打开编辑器等待输入而阻塞：

```bash
gitee pr comment 42 -b "注意 pkg/auth/token.go:45 的并发问题，建议加锁后合入" --json --no-tui
```

**可用 flag：** `--body/-b`（**必填，否则打开编辑器阻塞**）、`--json/-j`（**必须加**）、`--no-tui`（**必须加**）。

### Step 7：合并 PR（merge）— 可见且难撤销，需确认

**合并前必须向用户展示 PR 信息与合并方式并等待明确确认：**

```
即将合并 PR #42: fix(auth): 修复 token 过期竞态
作者: alice  |  fix/token-expiry → master  |  方式: squash  |  合并后删除源分支: 是
确认合并吗？(y/n)
```

用户确认后执行：

```bash
gitee pr merge 42 --json --no-tui
gitee pr merge 42 --method squash --delete-branch --json --no-tui
gitee pr merge 42 --method rebase --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--method` / `-m` | string | `merge` | `merge`（合并提交） \| `squash`（压缩为一次提交） \| `rebase`（变基） |
| `--delete-branch` | bool | false | 合并后删除源分支 |
| `--json` / `-j` | bool | false | **必须加** |
| `--no-tui` | bool | — | **必须加** |

### Step 8：关闭 / 重开 PR（close / reopen）

```bash
gitee pr close 42 --json --no-tui      # 不合并直接关闭
gitee pr reopen 42 --json --no-tui     # 重新打开已关闭的 PR
```

**可用 flag：** `--json/-j`（**必须加**）、`--no-tui`（**必须加**）。

### Step 9：把 PR 拉到本地（checkout / fetch）

- `pr checkout`：拉取并**切换**到该分支（一步到位，适合本地跑/测别人的 PR）。
- `pr fetch`：只拉取**不切换**（拉到本地分支后自行 `git checkout`）。

```bash
gitee pr checkout 42 --no-tui                 # 拉取并切到 pr_42
gitee pr checkout 42 --branch review --no-tui # 自定义本地分支名
gitee pr fetch 42 --no-tui                    # 只拉取到 pr_42，不切换
gitee pr checkout 42 --force --no-tui         # 已存在同名分支时重新拉取
```

**可用 flag（checkout / fetch 通用）：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--branch` / `-b` | string | `pr_<number>` | 本地分支名 |
| `--force` | bool | false | 已存在同名分支时重新拉取 / 覆盖 |
| `--remote` | string | 第一个 gitee remote | 从哪个 git remote 拉取 |
| `--no-tui` | bool | — | **必须加** |

> `pr checkout` / `pr fetch` 是本地 git 操作，**不支持 `--json`**。

---

## 判断逻辑

| 用户意图 | 执行命令 | 需确认 |
|---------|---------|--------|
| "提个 PR" / "创建 PR" | Step 1 create（记得 `-t`；先自行生成标题/正文，勿用 `--ai`） | 否 |
| "有哪些 PR" / "列出 PR" | Step 2 list | 否 |
| "看看这个 PR" | Step 3 view | 否 |
| "编辑 PR 标题/描述/草稿状态" | `pr edit` | 否 |
| "看 PR 改了啥 / diff" | Step 4 diff | 否 |
| "review / approve / 通过" | Step 5 review（先展示 PR 信息，approve 前建议确认） | 建议确认 |
| "留个评论 / 提意见" | Step 6 comment（记得 `-b`） | 否 |
| "合并 / merge" | Step 7 merge | **是（可见、难撤销）** |
| "关闭 PR" | Step 8 close | 否 |
| "重开 PR" | Step 8 reopen | 否 |
| "把 PR 拉到本地跑一下" | Step 9 checkout | 否 |
| "只拉取不切换分支" | Step 9 fetch | 否 |

---

## 完整示例

```bash
# ── 创建 PR ──（自行读 diff/log 生成 title/body，不用 --ai）
git push origin HEAD
gitee pr create \
  -t "fix(login): 修复空密码返回 500" \
  -b "## 变更\n校验空密码返回 400\n\n## 测试\n已覆盖边界用例" \
  --base master --head fix/login-empty-password \
  --assignees alice --testers bob \
  --json --no-tui

# ── review 一个 PR ──（自行分析，勿用 --ai）
gitee pr view 42 --json --no-tui
gitee pr diff 42 --no-tui
# Agent 分析后：
gitee pr comment 42 -b "整体 LGTM，pkg/auth/token.go:45 建议加锁" --json --no-tui
gitee pr review 42 --no-tui

# ── 本地跑一下别人的 PR ──
gitee pr checkout 42 --no-tui

# ── 确认后合并并清理分支 ──
gitee pr merge 42 --method squash --delete-branch --json --no-tui
```

---

## 错误处理

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `--title is required in non-interactive mode` | create 未传 `-t` | 补 `-t` 标题 |
| `failed to create PR: 422` | 同分支 PR 已存在 | 用 `gitee pr list -s open --json --no-tui` 查已有 PR |
| 命令卡住无响应 | comment 未传 `-b`，进入编辑器 | 中断并用 `-b "..."` 重跑 |
| `failed to ...: 404` | PR 号错误 | PR 号是**整数**（如 `42`），用 `gitee pr list --json --no-tui` 核对 |
| `failed to merge: 405/409` | 有冲突或未满足合并条件 | 先解决冲突/审查要求；管理员可 `pr review 42 --force` 后再合 |
| `failed to review PR: 403` | 无审查权限 | 需要仓库 developer 及以上权限 |
| `could not detect current branch` | 不在 git 仓库中 | cd 到项目目录，或用 `-R owner/repo` |
| `authentication required` | 未认证 | 提示 `gitee auth login` |
