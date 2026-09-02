---
name: gitee-cli
description: Bootstrap and use the Gitee CLI (`gitee`) from the terminal or an AI agent — detect and install the CLI when missing (curl script / npm / source), authenticate with a personal access token (`gitee auth login`, `--with-token`, `GITEE_TOKEN`), discover commands via `--help` and `gitee api --search`, run core pr / issue / repo / release / search / pipeline workflows with agent-safe flags (`--json`, `--no-tui`, `-R`, `--hostname`), and install / update / uninstall the bundled Agent Skills (`gitee skills`). Use when the user asks "gitee cli 怎么用", "安装 gitee cli", "gitee 登录", "gitee 授权", "查找 gitee 命令", "安装/卸载 gitee skills", "gitee 核心命令", "gitee cli 入门", or when a Gitee workflow needs first-time setup before any deeper gitee-pr / gitee-issue / gitee-api skill runs.
metadata:
  author: gitee
  version: "1.0"
---

Gitee CLI（`gitee`）的**引导与总览技能**：检测/安装 CLI → 授权 → 指令发现 → 核心指令 → 内置 Skills 管理。

本技能是其他 Gitee 工作流技能（`gitee-pr`、`gitee-issue`、`gitee-repo`、`gitee-release`、`gitee-search`、`gitee-api`、`gitee-go`）的前置入口：先按本技能把环境和授权弄好，深度操作再交给对应技能（见 Step 5 安装）。

## 全局安全约定（Agent 必须遵守）

- **所有命令加 `--no-tui`**，支持结构化输出的命令**一律加 `--json`**，避免交互式表格阻塞。
- **绝不执行 `--ai` 系列 flag**（仅存在于 `pr create` / `pr review` / `issue create`，会触发交互式编辑/pager 导致阻塞）；标题、正文由 Agent 自行组织后用 `-t` / `-b` 传入。
- 诊断信息走 stderr，stdout 只输出结果，可安全管道解析。
- 不在仓库目录内时用 `-R owner/repo` 显式指定目标；私有化部署用 `--hostname <host>`。
- **绝不打印、回显或落盘明文 token**（`gitee auth token` 会输出 token，除非用户明确要求，不要运行）。
- 不可逆操作（merge / delete / force 类）执行前必须向用户展示目标并获得明确确认。

---

## Step 1：检测是否已安装

```bash
gitee --version || echo "GITEE_CLI_NOT_INSTALLED"
```

Windows 上可用 `where gitee`。**未安装则按下面任一方式安装后回到本步验证。**

### 安装方式 A：官方安装脚本（macOS / Linux）

```bash
curl -fsSL https://gitee.com/oschina/gitee-cli/raw/main/scripts/install.sh | sh
```

脚本自动识别平台、校验 SHA-256、安装到 `$HOME/.local/bin`（root 运行时为 `/usr/local/bin`）。可固定版本或改安装目录：

```bash
GITEE_CLI_VERSION=1.2.3 INSTALL_DIR="$HOME/bin" \
  sh -c "$(curl -fsSL https://gitee.com/oschina/gitee-cli/raw/main/scripts/install.sh)"
```

> 装到 `$HOME/.local/bin` 时确认该目录在 `PATH` 中，否则提示用户追加。

### 安装方式 B：npm（全平台，含 Windows）

```bash
npm install -g @gitee/gitee-cli
gitee --version
```

按平台自动拉取对应二进制（darwin / linux / windows × amd64 / arm64）。Windows 用户走这条路径。

### 后续升级

```bash
gitee update --check   # 只检查
gitee update --yes     # 非交互升级（自动识别 npm / Homebrew / 独立二进制来源）
```

---

## Step 2：授权（PAT 登录）

Gitee CLI 使用个人访问令牌（Personal Access Token）认证。令牌在
<https://gitee.com/profile/personal_access_tokens> 创建。

### 人工使用

```bash
gitee auth login            # 交互式：打开浏览器或粘贴 token
gitee auth status           # 查看登录状态与当前主机
gitee auth logout           # 删除已存凭据
```

### Agent / CI / 非交互环境

非 TTY 下 `gitee auth login` 必须走 `--with-token`，从 stdin 读取：

```bash
printf '%s' "$GITEE_TOKEN" | gitee auth login --with-token
gitee auth status --json --no-tui
```

`$GITEE_TOKEN` 未设置时，提示用户先提供 token（或先在本机交互登录一次），**不要**替用户编造或索要后回显。

### 凭据存储

| 文件（默认 `~/.config/gitee/`，可用 `GITEE_CONFIG_DIR` 覆盖） | 内容 |
|---|---|
| `credentials.yml` | 默认主机 PAT 与 AI token（0600） |
| `hosts.yml` | 其他主机（私有化部署）的 token |
| `config.yml` | 通用配置（`gitee config` 管理） |

### 多主机 / 私有化部署

```bash
printf '%s' "$GITEE_TOKEN" | gitee auth login --hostname git.company.com --with-token
gitee pr list -R owner/repo --no-tui        # 登录成功的主机成为默认 host
gitee <command> --hostname git.company.com  # 单次调用显式指定其他主机
```

> 私有化实例的 token 创建页可能打不开：让用户在对应实例手工创建 PAT 后直接粘贴。

---

## Step 3：查找 / 发现指令

三层发现机制，Agent 遇到不确定的命令**先看 help，不要凭记忆猜 flag**：

```bash
gitee --help                        # 一级命令总览 + 全局 flag
gitee pr --help                     # 子命令列表
gitee pr create --help              # 具体 flag（--json 可选字段也会列出）
```

1. **`--help` 链**：每个层级都支持；`gitee config set locale zh_CN` 可切换中文帮助。
2. **`gitee api --search "<关键词>"`**：一级命令不覆盖某操作时，用它在 Gitee V5 OpenAPI（swagger）中检索 endpoint——中英文关键词均可，返回 method / path / summary / 参数表（必填项带 `*`）。分页用 `--limit`（默认 20）与 `--page`。
   ```bash
   gitee api --search "milestone" --no-tui
   gitee api --search "创建 label" --limit 50 --no-tui
   ```
   拿到 endpoint 后替换 `{owner}/{repo}/{number}` 占位符再发请求（见 Step 4 `api`）。
3. **Shell 补全与别名**：
   ```bash
   gitee completion zsh > /usr/local/share/zsh/site-functions/_gitee   # bash / zsh / fish / powershell，输出重定向到对应补全目录
   gitee alias set prs 'pr list -s open'                  # 别名存于 config.yml，支持 $1 位置参数
   ```

---

## Step 4：核心指令总览

16 个功能命令（另有内置 `help`）：

| 命令 | 子命令 | 作用 |
|---|---|---|
| `auth` | `login` `logout` `status` `token` | 认证管理 |
| `config` | `get` `set` `list` | 配置管理（`set` 支持 `--stdin`） |
| `pr` | `list` `view` `create` `edit` `close` `reopen` `merge` `review` `comment` `diff` `fetch` `checkout` | Pull Request 全生命周期 |
| `issue` | `list` `view` `create` `edit` `close` `reopen` `assign` `comment` | Issue 管理（编号形如 `IJEE10`） |
| `repo` | `list` `view` `clone` `create` `fork` `delete` | 仓库管理 |
| `release` | `list` `view` `create` `edit` `delete` | Release 管理（按 ID 或 tag） |
| `search` | `repos` `issues` `users` | 全站搜索（支持 `--language` `--sort` 等过滤） |
| `api` | `<endpoint>`（+ `--search` 模式） | 直连 Gitee V5 REST API |
| `skills` | `list` `install` `uninstall` | 管理内置 Agent Skills（见 Step 5） |
| `pipeline` | `list` `view` `example` `run` `commit` `build`（含 `view/status/last/list/cancel/rebuild/stage/job`）`plugin` `request` `program` | Gitee-Go 流水线（仓库级必须 `-R owner/repo`；`program` 用 `-E/-P` 定位企业项目） |
| `update` | `--check` `--yes` | CLI 自升级 |
| `alias` | `list` `set` `delete` | 命令别名 |
| `ai` | `[prompt]` `-c/--chat` | 对话式 AI（需配置 `ai.base_url` / `ai.model` / `ai.token`） |
| `ssh-key` | `list` `add` `delete` | SSH 公钥管理（支持 stdin） |
| `version` | — | 版本信息（`gitee --version` 亦可） |
| `completion` | `bash` `zsh` `fish` `powershell` | 生成补全脚本 |

**全局 flag**：`--no-tui`、`-q/--quiet`、`-V/--verbose`（重试与限流诊断）、`--hostname`；仓库类命令另有 `-R/--repo`；多数 list/view 支持 `--json`（可 `--json number,title` 选字段）。

常见用法：

```bash
gitee pr list -R owner/repo -s open --json --no-tui
gitee pr view 42 -R owner/repo --json --no-tui
gitee issue list -s open --sort updated --direction asc --no-tui
gitee search repos "cli" --language Go --sort stars_count --no-tui
gitee api /user --no-tui
gitee api -X POST /repos/owner/repo/issues -f title="Bug" -f body="Details" --no-tui
```

---

## Step 5：安装 / 卸载内置 Agent Skills

CLI 二进制内置 **7 个官方 Agent Skills**（离线分发）：
`gitee-pr` · `gitee-issue` · `gitee-repo` · `gitee-release` · `gitee-search` · `gitee-api` · `gitee-go`。
安装到 `~/.agents/skills/`，供 Claude Code / Codex / OpenCode 等支持 Agent Skills 的助手加载。

```bash
gitee skills list --json --no-tui     # NAME / STATUS / PATH
gitee skills install                  # 安装或更新（可反复执行）
gitee skills uninstall --yes          # 卸载（非交互必须 --yes）
```

- **镜像式安装**：每次 `install` 把本 CLI 管理的技能目录整体替换为当前内置版本，反映新增 / 修改 / 删除 / 重命名；**其他来源的技能不受影响**。升级 CLI 后直接重跑 `gitee skills install` 即可，无需先卸载。
- 自定义目录：`gitee skills install --dir "$HOME/.config/agents/skills"`，或设环境变量 `AGENTS_SKILLS_DIR`（对 `uninstall` 同样生效）。
- 安装/更新后**重启或重新加载 Agent** 才能生效。
- 给用户的「一键安装」引导词（可原样转发）：

  ```text
  请帮我安装 Gitee CLI Agent Skills：
  1. 运行 `gitee skills install`；
  2. 运行 `gitee skills list --json`，确认 7 个 gitee-* 技能均为已安装；
  3. 告诉我结果，并提示我重新加载 Agent。
  ```

---

## 环境变量速查

| 变量 | 作用 |
|---|---|
| `GITEE_TOKEN` | 登录 token 来源（`auth login --with-token` 时经 stdin 管道传入） |
| `GITEE_CONFIG_DIR` | 覆盖配置目录（默认 `~/.config/gitee`） |
| `GITEE_API_PREFIX` | 覆盖 API base URL |
| `GITEE_AI_TOKEN` / `OPENAI_API_KEY` | AI 功能凭据（后者仅当 `ai.base_url` 为 api.openai.com 时生效） |
| `GITEE_NO_UPDATE_NOTIFIER` / `CI` | 关闭后台更新检查（CI 环境自动关闭） |
| `AGENTS_SKILLS_DIR` | `gitee skills` 安装目标目录 |

---

## 错误处理

| 现象 | 原因 | 处理 |
|---|---|---|
| `command not found: gitee` | 未安装 / 不在 PATH | 回到 Step 1 安装；脚本安装时确认 `~/.local/bin` 在 PATH |
| `401 Unauthorized` / 提示未登录 | 未认证或 token 失效 | `gitee auth status`；重新 `gitee auth login`（非交互用 `--with-token`） |
| 非交互登录报错 | 非 TTY 环境走了交互式登录 | `printf '%s' "$GITEE_TOKEN" \| gitee auth login --with-token` |
| 命令卡在表格/分页 | 命中 TUI 或 pager | 补 `--no-tui`；必要时 `gitee config set tui false` |
| `404 Not Found`（`gitee api`） | 路径拼错 / 占位符未替换 | 重新 `gitee api --search` 核对 endpoint |
| `422 Unprocessable`（`gitee api`） | 缺必填字段 | 对照 `--search` 输出中带 `*` 的参数补齐 |
| 权限不足 | token scope 不够 | 让用户在 Gitee 设置里重建含所需权限的 PAT |
| `--search` 无结果 | 关键词过窄 | 换更通用的中/英关键词重试 |

---

## 参考

- 仓库与文档：<https://gitee.com/oschina/gitee-cli>（`docs/usage.md` 为实践指南）
- 创建 PAT：<https://gitee.com/profile/personal_access_tokens>
- Gitee V5 API：<https://gitee.com/api/v5/swagger_doc.json>
