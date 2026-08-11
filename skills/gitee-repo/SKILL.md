---
name: gitee-repo
description: Manage Gitee repositories from the CLI — view details, list, clone, create, fork, and delete. Use when the user asks to create, inspect, list, fork, clone, modify, or delete a repository. Always uses `--json` where supported and `--no-tui`; never uses `--ai`; requires explicit user confirmation before deleting a repository. If a requested repository operation has no direct command, fall back to the gitee-api skill to search the schema; API updates and deletions require a second confirmation.
metadata:
  author: gitee
  version: "1.0"
---

管理 Gitee 仓库的完整生命周期：查看 / 列出 / 克隆 / 创建 / fork / 删除。

需要先通过 `gitee auth login` 完成认证；克隆、创建、fork 和删除作用于当前账号或明确指定的 `owner/repo`。

## 前置检查

1. **已认证**：`gitee auth status --no-tui`，失败则提示 `gitee auth login`。
2. **删除是不可逆操作**：`repo delete` 执行前必须向用户明确确认，`-y` 只在用户确认后才加。
3. 直接仓库命令不支持目标操作时，切换到 `gitee-api` skill，先 `gitee api --search` 查 schema；通过 API 更新或删除资源前必须二次确认。

---

## 执行步骤

### Step 1：查看仓库信息（view）

```bash
gitee repo view owner/repo --json --no-tui
# 在仓库目录内可省略 owner/repo，自动从 git remote 推断
gitee repo view --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 说明 |
|------|------|------|
| `--json` / `-j` | string | **必须加**；`--json=full_name,stargazers_count` 可只选字段 |
| `--no-tui` | bool | **必须加** |

**常用 JSON 字段：** `full_name` `description` `private` `language` `default_branch` `stargazers_count` `forks_count` `open_issues_count` `ssh_url` `html_url` `namespace` `permission`。

### Step 2：列出仓库（list）

```bash
gitee repo list --json --no-tui
gitee repo list --type owner --sort updated --json --no-tui
gitee repo list --search gitee-cli --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--type` | string | — | `owner` \| `personal` \| `member` \| `public` \| `private` |
| `--affiliation` | string | — | `owner` \| `collaborator` \| `organization_member` |
| `--search` | string | — | 关键词过滤 |
| `--sort` / `-s` | string | `full_name` | `created` \| `updated` \| `pushed` \| `full_name` |
| `--limit` / `-l` | int | `20` | 每页数量（最大 100） |
| `--page` / `-p` | int | `1` | 页码 |
| `--json` / `-j` | string | — | **必须加** |
| `--no-tui` | bool | — | **必须加** |

### Step 3：克隆仓库（clone）

```bash
gitee repo clone owner/repo --no-tui
gitee repo clone owner/repo --branch develop --depth 1 --no-tui
gitee repo clone owner/repo --ssh --no-tui
```

**可用 flag：**

| Flag | 类型 | 说明 |
|------|------|------|
| `--branch` / `-b` | string | 只克隆指定分支 |
| `--depth` | string | 浅克隆深度，如 `1` |
| `--dir` | string | 克隆到指定目录 |
| `--ssh` | bool | 用 SSH 克隆（默认 HTTPS + token） |
| `--no-tui` | bool | **必须加** |

> `clone` 支持 `owner/repo` 或完整 URL（`https://...` / `git@...`）。

### Step 4：创建仓库（create）

```bash
gitee repo create -n my-repo --json --no-tui
gitee repo create -n my-repo -d "描述" --private --auto-init --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--name` / `-n` | string | — | 仓库名（**必填**） |
| `--description` / `-d` | string | — | 仓库描述 |
| `--private` | bool | false | 创建为私有仓库 |
| `--auto-init` | bool | false | 初始化 README |
| `--has-issues` | bool | true | 启用 Issues |
| `--has-wiki` | bool | true | 启用 Wiki |
| `--homepage` | string | — | 主页 URL |
| `--json` / `-j` | bool | false | **必须加** |
| `--no-tui` | bool | — | **必须加** |

### Step 5：Fork 仓库（fork）

```bash
gitee repo fork owner/repo --json --no-tui
gitee repo fork owner/repo --org my-org --json --no-tui
```

**可用 flag：**

| Flag | 类型 | 说明 |
|------|------|------|
| `--org` | string | fork 到指定组织（默认 fork 到个人账号） |
| `--json` / `-j` | bool | **必须加** |
| `--no-tui` | bool | **必须加** |

### Step 6：删除仓库（delete）— 不可逆，需确认

**删除前必须向用户展示要删除的仓库并等待明确确认：**

```
即将删除仓库：owner/repo
此操作不可逆，仓库内容将永久丢失。
确认删除吗？(y/n)
```

用户确认后执行（此时才加 `-y` 跳过 CLI 二次确认）：

```bash
gitee repo delete owner/repo -y --no-tui
```

> ⚠️ 在用户明确确认之前，**绝不**执行 `repo delete`，更不要预先带上 `-y`。

---

## 判断逻辑

| 用户意图 | 执行命令 | 需确认 |
|---------|---------|--------|
| "看看这个仓库" / "仓库信息" | Step 1 view | 否 |
| "我有哪些仓库" / "列出仓库" | Step 2 list | 否 |
| "克隆 / clone" | Step 3 clone | 否 |
| "创建 / 新建仓库" | Step 4 create | 否 |
| "fork" | Step 5 fork | 否 |
| "删除仓库" | Step 6 delete | **是（不可逆）** |

---

## 完整示例

```bash
# 创建一个私有仓库并初始化
gitee repo create -n demo-svc -d "demo service" --private --auto-init --json --no-tui

# 查看它
gitee repo view <your-login>/demo-svc --json --no-tui

# 克隆到本地
gitee repo clone <your-login>/demo-svc --no-tui

# 列出我拥有的仓库，按最近推送排序
gitee repo list --type owner --sort pushed --json --no-tui
```

---

## 错误处理

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `--name is required` | 未传 `-n` | 补充 `-n <name>` |
| `failed to create repo: 422` | 同名仓库已存在 | 换名字，或先 `repo list --search` 确认 |
| `failed to view repo: 404` | 仓库不存在或无权限 | 核对 `owner/repo` 拼写与访问权限 |
| `authentication required` | 未认证 | 提示 `gitee auth login` |
| `403` on delete | 无删除权限 | 需要仓库 owner/admin 权限 |
