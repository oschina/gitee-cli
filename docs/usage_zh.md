# Gitee CLI 实践指南

[返回中文 README](../README_zh.md) · [English](usage.md)

本文档收录从主 README 拆出的详细工作流与参考信息。所有命令和参数的最终说明，
请以 `gitee <command> --help` 为准。

- [Agent 与自动化工作流](#agent-与自动化工作流)
- [TUI 模式](#tui-模式)
- [Pull Request 本地分支](#pull-request-本地分支)
- [AI 工作流](#ai-工作流)
- [原始 API 请求](#原始-api-请求)
- [配置](#配置)
- [稳定性与调试](#稳定性与调试)

## Agent 与自动化工作流

Gitee CLI 可直接组合到编码 Agent、Shell 脚本和 CI 任务中：

- 对支持的列表和查看命令使用 `--json` 获取结构化输出。
- 通过 `--repo` 和 `--hostname` 显式指定仓库与 Host。
- 通过参数或 stdin 提供变更输入，避免触发交互提示。
- 当 Agent 需要的端点暂无专用命令时，使用 `gitee api`。
- 诊断信息写入 stderr，保证 stdout 中的 JSON 可被机器稳定读取。

```bash
# 查看指定仓库的开放 PR。
gitee pr list --repo owner/repo --json

# 提取 PR 编号并用于下一次工具调用。
PR_NUMBER=$(gitee pr list -R owner/repo -s open --json | jq -r '.[0].number')
gitee pr view "$PR_NUMBER" -R owner/repo --json

# 显式指定私有部署。
gitee api /user --hostname git.company.com
```

### 常用参数

| 参数 | 范围 | 说明 |
|---|---|---|
| `--hostname <host>` | 所有命令 | 选择 Gitee Host，默认为 `gitee.com`。 |
| `-R, --repo <owner/repo>` | PR 与 Issue 命令 | 覆盖从 git remote 推断的仓库。 |
| `-j, --json` | 支持的列表和查看命令 | 输出结构化 JSON。 |
| `--no-tui` | 所有命令 | 单次禁用交互式 TUI。 |
| `-q, --quiet` | 所有命令 | 除错误外不输出其他内容。 |
| `-V, --verbose` | 所有命令 | 显示请求、重试和限额诊断信息。 |

### 命令实践

```bash
# 按分支、标签、人员和里程碑过滤 PR。
gitee pr list --base main --labels bug,performance \
  --author alice --assignee bob --tester carol --milestone-number 7

# 优先查看最久未更新的 Issue。
gitee issue list --sort updated --direction asc

# 在 Gitee 全站搜索仓库和 Issue。
gitee search repos cli --language Go --sort stars_count
gitee search issues timeout --repo owner/repo --state open

# 仅拉取 PR，不切换分支。
gitee pr fetch 42 --branch review-42

# 创建带审查人员和测试人员的 PR。
gitee pr create --title "Fix login" --assignees alice,bob --testers carol

# 创建或指派 Issue。
gitee issue create --ai
gitee issue assign IJEE10 alice

# 从文件或 stdin 添加 SSH Key。
gitee ssh-key add --title "My laptop" --file ~/.ssh/id_rsa.pub
cat ~/.ssh/id_rsa.pub | gitee ssh-key add --title "CI server" --file -

# 在脚本中屏蔽普通输出。
gitee issue close ICX4FO --quiet
```

### 命令别名

别名保存在 `~/.config/gitee/config.yml` 中，可跨会话使用：

```bash
gitee alias set prs "pr list -s open"
gitee alias set myissues "issue list -A me -s open"
gitee prs
gitee myissues
gitee alias list
gitee alias delete prs
```

在展开内容前添加 `!` 可以组合 Shell 工作流，并支持 `$1` 等位置变量和额外参数。

```bash
gitee alias set deploy "!PR=\$(gitee pr create --base \$1 --json | jq -r .number) && gitee pr comment \$PR -b ci_deploy"
gitee deploy main
```

Shell 别名在 Unix/macOS 上通过 `sh -c` 执行，在 Windows 上通过 `cmd /c` 执行。

## TUI 模式

全局开启交互式表格：

```bash
gitee config set tui true
```

单次禁用可添加 `--no-tui`，全局关闭可执行 `gitee config set tui false`。

### 快捷键

| 按键 | 操作 | 适用范围 |
|---|---|---|
| `enter` | 在浏览器中打开所选条目 | 所有列表 |
| `v` | 预览正文或描述 | PR、Issue、仓库、Release、用户列表 |
| `c` | 复制条目 URL | 所有列表 |
| `d` | 查看 diff | PR 列表 |
| `f` | 拉取 PR 到 `pr_<n>` | PR 列表 |
| `e` | 内联编辑 Issue | Issue 列表 |
| `i` | 打开图片查看器 | 支持的分页内容 |
| `q` / `ctrl+c` | 退出 | 所有列表 |
| `↑` / `↓` | 上下导航 | 所有列表 |

### 图片渲染

Issue 或 PR 包含图片时，在分页器中按 `i` 打开图片查看器。

| 协议 | 支持终端 |
|---|---|
| Kitty Graphics | Kitty、Ghostty |
| iTerm2 Inline Images | iTerm2 |
| chafa | 已安装 `chafa` 的任意终端 |
| OSC 8 超链接 | 兜底的可点击链接 |

使用 tmux 3.3 或更高版本时，在 `~/.tmux.conf` 中添加以下配置，然后重新加载
配置文件或重启 tmux：

```tmux
set -g allow-passthrough on
```

tmux 内目前支持的外层终端包括 iTerm2、Kitty 和 Ghostty。

## Pull Request 本地分支

### `pr checkout` 与 `pr fetch`

| 行为 | `pr fetch` | `pr checkout` |
|---|---|---|
| 拉取 PR 到本地分支 | 是 | 是 |
| 切换到该分支 | 否 | 是 |
| 分支已存在 | 报错，除非使用 `--force` | 直接切换；`--force` 会重新拉取 |

```bash
gitee pr checkout 123                  # 拉取并切换
gitee pr checkout 123 --branch review  # 指定本地分支名
gitee pr fetch 123                     # 仅拉取
gitee pr fetch 123 --force             # 覆盖已有分支
```

## AI 工作流

Gitee CLI 支持 OpenAI 及其兼容的 LLM API。

### 配置

```bash
gitee config set ai.base_url https://api.openai.com/v1
gitee config set ai.model gpt-4o-mini
gitee config set ai.token              # 隐藏输入
gitee config set ai.language Chinese   # 可选，默认为 English

# 非交互配置。
printf '%s' "$AI_TOKEN" | gitee config set ai.token --stdin

# 适用于任意服务且不写入配置文件。
export GITEE_AI_TOKEN=sk-xxx

# api.openai.com 也支持此变量。
export OPENAI_API_KEY=sk-xxx
```

本地模型和其他兼容服务使用相同的配置方式：

```bash
# Ollama
gitee config set ai.base_url http://localhost:11434/v1
gitee config set ai.model qwen2.5:7b
printf '%s' 'ollama' | gitee config set ai.token --stdin

# DeepSeek
gitee config set ai.base_url https://api.deepseek.com/v1
gitee config set ai.model deepseek-chat
gitee config set ai.token
```

### PR 与 Issue 辅助

`pr create --ai` 读取本地 diff 和 commit 日志，生成可编辑的标题与描述草稿；
`pr review --ai` 拉取 PR diff 并返回结构化审查；`issue create --ai` 将简短描述
扩展为格式化 Issue。

```bash
gitee pr create --ai
gitee pr review 42 --ai
gitee issue create --ai
```

### 直接对话

```bash
gitee ai "什么是 rebase？"                         # 流式输出
gitee ai -n "总结这段 diff：$(git diff HEAD~1)"    # 非流式输出
gitee ai --chat                                    # 多轮对话
```

输入 `exit` 可退出多轮对话。

### AI Commit Message

内置的 `prepare-commit-msg` 工作流按需启用，仅在设置 `USE_GITEE_AI` 时调用
`gitee ai`。

```bash
git config core.hooksPath /path/to/custom_hooks
chmod +x /path/to/custom_hooks/prepare-commit-msg

git commit -m "fix: 处理空指针异常"  # 普通提交，不调用 AI
USE_GITEE_AI=1 git commit              # 在编辑器中打开 AI 草稿

git config alias.aic '!USE_GITEE_AI=1 git commit'
git aic
```

## 原始 API 请求

`gitee api` 可向任意 Gitee V5 端点发送鉴权请求。

```bash
gitee api /user
gitee api -X POST /repos/owner/repo/issues -f title="Bug" -f body="Details"
gitee api --body '{"state":"closed"}' -X PATCH /repos/owner/repo/issues/1
gitee api -H "Accept: application/json" /user
gitee api /repos/owner/repo/pulls --hostname git.company.com
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-X, --method` | `GET` | HTTP 方法 |
| `-H, --header` | — | 添加可重复的 `Key: Value` 请求头。 |
| `-f, --field` | — | 添加可重复的 `key=value` JSON 字段。 |
| `--body` | — | 发送原始 JSON Body。 |
| `-p, --pretty` | `false` | 格式化 JSON 响应。 |

## 配置

配置文件统一存放在 `~/.config/gitee/`：

| 文件 | 用途 |
|---|---|
| `config.yml` | 通过 `gitee config` 管理的通用配置 |
| `credentials.yml` | 默认 Host 的 PAT 和 AI Token，权限为 `0600` |
| `hosts.yml` | 其他 Host 的 Token，权限为 `0600` |

敏感值显示为 `<redacted>`。旧版 `config.yml` 中的 Token 会在下次写入时迁移到
`credentials.yml`。读取配置不会创建文件或修改权限。

### 配置项

| 键 | 默认值 | 说明 |
|---|---|---|
| `tui` | `false` | 开启交互式 TUI。 |
| `colorize` | `false` | 开启语法着色。 |
| `update_check` | `true` | 每 24 小时至多检查一次版本，CI 中自动关闭。 |
| `locale` | 自动 | `en` 或 `zh_CN`，未设置时从 `$LANG` 推断。 |
| `theme` | `default` | `auto`、`dark`、`light`、`dracula`、`tokyo-night` 或 `pink`。 |
| `host` | `gitee.com` | 默认 Gitee Host。 |
| `api_prefix` | `https://gitee.com/api/v5` | API 根路径。 |
| `pager` | — | 长内容使用的分页器。 |
| `editor` | — | 交互式编辑器，按 Git 与 Shell 编辑器变量回退。 |
| `default_repo` | — | 在非 Git 仓库中使用的默认 `owner/repo`。 |
| `ai.base_url` | — | OpenAI 兼容接口地址。 |
| `ai.model` | `gpt-4o-mini` | 模型名称。 |
| `ai.token` | — | 服务 Token，支持 `GITEE_AI_TOKEN`。 |
| `ai.language` | `English` | AI 生成内容的语言。 |

```bash
gitee config set tui true
gitee config set locale zh_CN
gitee config set theme dracula
gitee config set default_repo myorg/myrepo
gitee config set editor nano
gitee config set update_check false
gitee config list
```

设置 `GITEE_NO_UPDATE_NOTIFIER=1` 可在不修改配置的情况下关闭版本检查。

### CLI 更新

后台更新检查只输出通知，不会自动安装，也不会弹出交互确认。可永久关闭：

```bash
gitee config set update_check false
```

关闭后台检查后，仍可主动执行更新：

```bash
gitee update --check       # 只检查，不安装
gitee update               # 显示识别到的安装方式并确认
gitee update --yes         # 非交互更新
```

全局 npm 安装会通过 npm 更新；Release、带版本标签的源码安装和独立二进制会从
Gitee Release 下载对应平台的附件，校验 `checksums.txt` 后原子替换。本地 npm
依赖必须由所属项目更新，更新器不会调用 `sudo`。

## 稳定性与调试

HTTP 429、5xx、超时和临时网络错误会通过指数退避自动重试。长时间运行的操作可
通过 `Ctrl+C` 取消。

使用 `--verbose` 或 `-V` 查看 HTTP 请求、重试和限额信息：

```bash
gitee pr list --verbose
# [DEBUG] GET /api/v5/repos/owner/repo/pulls
# [DEBUG] Rate limit: 4998/5000 (resets in 3598s)
# [DEBUG] Response 200 OK (0.234s)
```

诊断信息写入 stderr，不会影响 JSON 管道：

```bash
gitee pr list --verbose --json | jq '.[] | .number'
```

客户端会自动跟踪 `X-RateLimit-*` 响应头。
