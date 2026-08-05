# Gitee CLI

[![Go 版本](https://img.shields.io/badge/go-1.25.12+-00ADD8?logo=go)](https://golang.org)
[![许可证](https://img.shields.io/badge/license-MIT-blue)]()

Gitee CLI 是 [Gitee](https://gitee.com) 的命令行工具，让你无需离开终端即可管理 Pull Request、Issue、仓库、Release 等。

---

## 功能特性

- **核心 Gitee 工作流**：PR、Issue、仓库、Release、Gist、SSH Key 以及原始 API 调用
- **交互式 TUI 模式**：基于 Bubble Tea 的富文本表格，底部帮助栏实时展示快捷键
- **智能 PR 工作流**：`gitee pr checkout 123` 一步完成拉取并切换分支；仅拉取用 `gitee pr fetch`
- **多 Host 支持**：同时登录多个 Gitee 实例（gitee.com + 私有部署）
- **JSON 输出**：所有列表和查看命令均支持 `--json` / `-j`，方便脚本和管道处理
- **非交互友好**：核心工作流可通过显式 flag 用于 CI/脚本场景
- **AI 辅助**：`--ai` 自动生成 PR 描述、Issue 报告，或对 PR 进行 AI 代码审查
- **国际化**：界面语言支持中文（`zh_CN`）和英文（`en`），自动检测系统语言
- **缓存更新提示**：可选的新版本提醒，缓存 24 小时且在 CI 中自动关闭
- **Shell 补全**：原生支持 Bash、Zsh、Fish、PowerShell
- **命令别名**：用 `gitee alias set prs "pr list -s open"` 节省按键次数

---

## 安装

需要 Go 1.25.12 或更高版本。

```bash
go install gitee.com/oschina/gitee-cli/cmd/gitee@latest
```

### 下载二进制

从 [Releases](https://gitee.com/oschina/gitee-cli/releases) 页面下载对应平台的预编译二进制文件。

### npm / npx

```bash
npx -y @gitee/gitee-cli@latest <command>
```

或全局安装：

```bash
npm install -g @gitee/gitee-cli
gitee <command>
```

### Homebrew（计划支持）

```bash
brew install oschina/tap/gitee
```

### 通过 make 安装

需要 Go 1.25.12 或更高版本以及 `make`。

```bash
git clone https://gitee.com/oschina/gitee-cli.git
cd gitee-cli
make install
```

此命令会编译二进制文件并安装到 `$HOME/bin/gitee`。

### 从源码构建

需要 Go 1.25.12 或更高版本。

```bash
git clone https://gitee.com/oschina/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
```

---

## 快速开始

```bash
# 登录并保存 Personal Access Token
gitee auth login

# 列出当前仓库的开放 Pull Request
gitee pr list

# 拉取 PR #123 并直接切换到该分支
gitee pr checkout 123

# 列出开放的 Issue
gitee issue list -s open

# 查看当前用户信息
gitee api /user
```

---

## 认证

Gitee CLI 使用 Personal Access Token（PAT）进行认证。

1. 在 [Gitee 设置 / 私人令牌](https://gitee.com/profile/personal_access_tokens) 创建 Token
2. 运行 `gitee auth login` 并粘贴 Token
3. Token 保存在 `~/.config/gitee/credentials.yml`

```bash
gitee auth status          # 查看登录状态
gitee auth token           # 打印已保存的 Token
gitee auth logout          # 移除登录凭证
```

非交互环境可通过标准输入传递 Token：

```bash
printf '%s' "$GITEE_TOKEN" | gitee auth login --with-token
```

### 多 Host（私有部署）

同时登录多个 Gitee 实例：

```bash
# 登录私有 Gitee 实例
gitee auth login --hostname git.company.com

# 对特定 host 执行命令
gitee pr list --hostname git.company.com -R owner/repo

# 查看所有已配置 host 的状态
gitee auth status

# 登出特定 host
gitee auth logout --hostname git.company.com
```

非默认 host 的 Token 保存在 `~/.config/gitee/hosts.yml`。

---

## 命令一览

| 命令 | 子命令 | 说明 |
|---|---|---|
| `auth` | `login`, `logout`, `status`, `token` | 管理认证 |
| `config` | `get`, `set`, `list` | 管理配置 |
| `pr` | `list`, `view`, `create`, `close`, `reopen`, `merge`, `review`, `comment`, `diff`, `fetch`, `checkout` | 管理 Pull Request |
| `issue` | `list`, `view`, `create`, `close`, `reopen`, `edit`, `assign`, `comment` | 管理 Issue |
| `repo` | `list`, `view`, `clone`, `create`, `fork`, `delete` | 管理仓库 |
| `release` | `list`, `view`, `create`, `delete` | 管理 Release |
| `gist` | `list`, `view`, `create`, `delete` | 管理 Gist |
| `ssh-key` | `list`, `add`, `delete` | 管理 SSH Key |
| `user` | `search` | 搜索用户 |
| `alias` | `list`, `set`, `delete` | 管理命令别名 |
| `api` | `<endpoint>` | 发起原始 API 请求 |
| `version` | — | 显示版本信息 |
| `completion` | `bash`, `zsh`, `fish`, `powershell` | 生成 Shell 补全脚本 |

### 全局 Flag

以下 flag 对所有命令可用：

| Flag | 说明 |
|---|---|
| `--hostname <host>` | 指定 Gitee 实例地址（默认：`gitee.com`） |
| `--no-tui` | 本次命令禁用 TUI 模式 |
| `-q, --quiet` | 仅输出错误，屏蔽其他所有输出 |

以下 flag 在 `pr` 和 `issue` 子命令中可用：

| Flag | 说明 |
|---|---|
| `-R, --repo <owner/repo>` | 指定目标仓库（默认从 git remote 自动推断） |

大多数列表和查看命令还支持：

| Flag | 说明 |
|---|---|
| `-j, --json` | 以 JSON 格式输出结果 |

### 使用示例

```bash
# 以 JSON 格式列出开放的 PR
gitee pr list -s open --json

# 按分支、标签、创建者、指派人、测试人和里程碑过滤 PR
gitee pr list --base main --labels bug,performance --author alice --assignee bob --tester carol --milestone-number 7

# 拉取 PR #42 到本地分支并切换
gitee pr checkout 42

# 仅拉取，不切换分支
gitee pr fetch 42 --branch review-42

# 创建带审查人员和测试人员的 PR
gitee pr create --title "Fix login" --assignees alice,bob --testers carol

# 创建 Issue（TUI 模式下会弹出标签多选框）
gitee issue create --title "Bug report" --body "Something is wrong."

# 指派 Issue
gitee issue assign IJEE10 alice

# 通过 Tag 名查看 Release
gitee release view v1.2.0

# 原始 API 调用
gitee api /user
gitee api -X PATCH /repos/owner/repo/issues/1 -f title="Updated"
gitee api /repos/owner/repo/pulls --hostname git.company.com

# 设置别名
gitee alias set prs "pr list -s open"
gitee prs
```

---

## TUI 模式

开启 TUI 模式，以交互式表格浏览数据：

```bash
gitee config set tui true
```

开启后，列表类命令会打开交互界面，底部帮助栏实时显示可用快捷键。

### 快捷键

| 按键 | 操作 | 适用范围 |
|---|---|---|
| `enter` | 在浏览器中打开所选条目 | 所有列表 |
| `v` | 在分页器中预览正文/描述 | PR、Issue、仓库、Release、用户列表 |
| `c` | 复制条目 URL 到剪贴板 | 所有列表 |
| `d` | 在分页器中查看 diff | PR 列表 |
| `f` | 拉取 PR 到本地分支（`pr_<n>`） | PR 列表 |
| `e` | 内联编辑 Issue | Issue 列表 |
| `q` / `ctrl+c` | 退出 | 所有列表 |
| `↑` / `↓` | 上下导航 | 所有列表 |

单次禁用 TUI：

```bash
gitee pr list --no-tui
```

全局禁用：

```bash
gitee config set tui false
```

### 图片渲染

在分页器中浏览含图片的 Issue 或 PR 时，按 `i` 键打开图片查看器，支持终端内联渲染：

| 协议 | 支持终端 |
|---|---|
| Kitty Graphics | Kitty、Ghostty |
| iTerm2 Inline Images | iTerm2 |
| chafa（Unicode 字符画） | 已安装 `chafa` 的任意终端 |
| OSC 8 超链接 | 兜底（可点击链接） |

#### 在 tmux 中使用

在 tmux 会话内使用图片渲染，需在 `~/.tmux.conf` 中添加：

```
set -g allow-passthrough on
```

然后执行 `tmux source ~/.tmux.conf` 或重启 tmux。需要 tmux ≥ 3.3。

tmux 内目前支持的外层终端：**iTerm2**、**Kitty**、**Ghostty**。

---

## `pr checkout` vs `pr fetch`

| | `pr fetch` | `pr checkout` |
|---|---|---|
| 拉取 PR 到本地分支 | ✅ | ✅ |
| 切换到该分支 | ❌ | ✅ |
| 分支已存在 | 报错（加 `--force` 覆盖） | 直接切换（加 `--force` 重新拉取） |

```bash
gitee pr checkout 123                  # 拉取 + 切换
gitee pr checkout 123 --branch review  # 自定义本地分支名
gitee pr fetch 123                     # 仅拉取
gitee pr fetch 123 --force             # 强制覆盖已有分支
```

---

## `gitee api`

对任意 Gitee V5 API 端点发起鉴权请求：

```bash
gitee api /user                                      # GET（默认）
gitee api -X POST /repos/owner/repo/issues \
  -f title="Bug" -f body="Details"                 # POST，传 JSON 字段
gitee api --body '{"state":"closed"}' \
  -X PATCH /repos/owner/repo/issues/1              # 传原始 JSON body
gitee api -H "Accept: application/json" /user      # 自定义请求头
gitee api /repos/owner/repo/pulls \
  --hostname git.company.com                       # 指定其他 host
```

| Flag | 默认值 | 说明 |
|---|---|---|
| `-X, --method` | `GET` | HTTP 方法 |
| `-H, --header` | — | 添加请求头（`Key: Value`），可重复 |
| `-f, --field` | — | 添加 JSON body 字段（`key=value`），可重复 |
| `--body` | — | 原始 JSON 请求体 |
| `-p, --pretty` | `false` | 格式化输出 JSON 响应 |

---

## 配置

配置文件统一存放在 `~/.config/gitee/`：

| 文件 | 用途 |
|---|---|
| `config.yml` | 通用配置（通过 `gitee config` 管理） |
| `credentials.yml` | 默认 host 的 PAT 和 AI Token（权限为 `0600`） |
| `hosts.yml` | 多 host 场景下各 host 的 Token（权限为 `0600`） |

`gitee config get/list` 会将敏感值显示为 `<redacted>`。旧版 `config.yml` 中的 Token
会在下次写入配置时自动迁移到 `credentials.yml`。读取配置不会创建文件或修改权限，
因此帮助命令和仅使用环境变量的命令可以在只读 HOME 中运行。

### 配置项

| 键 | 默认值 | 说明 |
|---|---|---|
| `tui` | `false` | 开启交互式 TUI 模式 |
| `colorize` | `false` | 开启语法着色 |
| `update_check` | `true` | 每 24 小时至多检查一次新版本（CI 中自动关闭） |
| `locale` | 自动 | 界面语言：`en`、`zh_CN`（未设置时自动检测 `$LANG`） |
| `theme` | `default` | 配色主题：`auto`、`dark`、`light`、`dracula`、`tokyo-night`、`pink` |
| `host` | `gitee.com` | 默认 Gitee 地址 |
| `api_prefix` | `https://gitee.com/api/v5` | API 根路径 |
| `pager` | — | 分页输出所用的 pager 程序 |
| `editor` | — | 编辑器（优先级：`$GIT_EDITOR` > `$VISUAL` > `$EDITOR` > 配置值 > `vim`） |
| `default_repo` | — | 在非 Git 仓库目录下使用的默认 `owner/repo` |
| `ai.base_url` | — | OpenAI 兼容接口地址 |
| `ai.model` | `gpt-4o-mini` | 模型名称 |
| `ai.token` | — | 当前服务的 Token（可设置 `GITEE_AI_TOKEN`；仅 `api.openai.com` 可使用 `OPENAI_API_KEY`） |
| `ai.language` | `English` | AI 生成内容的语言 |

```bash
gitee config set tui true
gitee config set locale zh_CN
gitee config set theme dracula
gitee config set api_prefix https://my-gitee.example.com/api/v5
gitee config set editor nano
gitee config set default_repo myorg/myrepo
gitee config set update_check false
gitee config list
```

也可以设置 `GITEE_NO_UPDATE_NOTIFIER=1`，在不修改配置的情况下关闭更新检查。

---

## AI 功能

Gitee CLI 集成了任意 OpenAI 兼容的 LLM 接口。

### 配置

```bash
gitee config set ai.base_url https://api.openai.com/v1
gitee config set ai.model gpt-4o-mini
gitee config set ai.token              # 隐藏输入，存储在 credentials.yml
gitee config set ai.language Chinese   # 可选，默认 English

# 脚本中通过标准输入设置
printf '%s' "$AI_TOKEN" | gitee config set ai.token --stdin

# 或为任意服务使用环境变量（不写入配置文件）
export GITEE_AI_TOKEN=sk-xxx

# ai.base_url 使用 https://api.openai.com 时，也可沿用 OpenAI 标准变量
export OPENAI_API_KEY=sk-xxx
```

也可接入本地模型或任意兼容服务：

```bash
# 本地 Ollama
gitee config set ai.base_url http://localhost:11434/v1
gitee config set ai.model qwen2.5:7b
printf '%s' 'ollama' | gitee config set ai.token --stdin

# DeepSeek
gitee config set ai.base_url https://api.deepseek.com/v1
gitee config set ai.model deepseek-chat
gitee config set ai.token
```

### `pr create --ai`

读取本地 `git diff` 和 commit 日志，生成 PR 标题和描述草稿，交互式确认：

```
✦ Collecting git context...
✦ Generating PR description with gpt-4o-mini...

────────────────────────────────────────
Title: fix(auth): resolve token expiry race condition
...
────────────────────────────────────────
? 请选择操作：  [Use as-is] Edit in editor  Regenerate  Write manually
```

### `pr review --ai`

拉取 PR diff，LLM 给出结构化代码审查意见（摘要 / 阻塞问题 / 建议 / 结论），TUI 模式下在 pager 中展示：

```bash
gitee pr review 42 --ai
```

### `issue create --ai`

输入一句话描述，LLM 展开为规范的 Issue 报告：

```bash
gitee issue create --ai
# 请用一两句话描述这个 Issue：登录页面在密码为空时崩溃
```

### `gitee ai`

单次提问（默认流式输出）或开启多轮对话：

```bash
# 单次提问——流式输出
gitee ai "什么是 rebase？"

# 非流式输出（适合脚本/管道）
gitee ai -n "总结这段 diff：$(git diff HEAD~1)"

# 多轮对话（输入 exit 退出）
gitee ai --chat
```

### AI 自动生成 commit message（git hook）

通过 `prepare-commit-msg` hook 在提交时调用 `gitee ai` 根据暂存区 diff 自动生成提交信息。hook 采用**按需触发**模式：只有设置了 `USE_GITEE_AI` 环境变量时才会运行，不影响日常 `git commit` 工作流。

**1. 配置 hook**

```bash
# 将此仓库指向自定义 hooks 目录
git config core.hooksPath /path/to/custom_hooks

# 赋予执行权限
chmod +x /path/to/custom_hooks/prepare-commit-msg
```

**2. 使用**

```bash
# 普通提交——hook 跳过，不调用 AI
git commit -m "fix: 处理空指针异常"

# AI 生成 commit message——编辑器预填充 LLM 生成内容
USE_GITEE_AI=1 git commit
```

**3. 可选：添加 git alias 简化操作**

```bash
git config alias.aic '!USE_GITEE_AI=1 git commit'

# 之后只需执行：
git aic
```

---

## 命令别名

别名让你为常用命令定义快捷方式：

```bash
# 定义别名
gitee alias set prs "pr list -s open"
gitee alias set myissues "issue list -A me -s open"
gitee alias set merged "pr list -s merged -l 10"

# 使用别名——自动展开并执行完整命令
gitee prs
gitee myissues
gitee merged

# 列出所有别名
gitee alias list

# 删除别名
gitee alias delete prs
```

别名保存在 `~/.config/gitee/config.yml`，跨会话持久生效。

---

## 开发

### 环境要求

- Go 1.25.12 或更高版本
- Make（可选）
- zip（用于构建 Windows 发布包）

### 构建

```bash
make build          # 构建到 ./bin/gitee
make install        # 安装到 $HOME/bin/gitee
make test           # go test ./...
make lint           # golangci-lint run
```

或直接：

```bash
go build -o gitee ./cmd/gitee
```

### 发布

完整的预检、打 Tag、创建并上传 Gitee Release、npm 和安装验证流程见
[docs/releasing.md](docs/releasing.md)。发布流程构建 Linux/macOS/Windows × amd64/arm64，
并随归档文件一起提供校验和。

---

## 参与贡献

开发和 Pull Request 要求见 [CONTRIBUTING.md](CONTRIBUTING.md)。安全问题请按照
[SECURITY.md](SECURITY.md) 私下报告；所有参与者均需遵守
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

---

## 许可证

[MIT](LICENSE)
