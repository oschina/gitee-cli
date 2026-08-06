<div align="center">

<img src="logo.png" alt="Gitee CLI Logo" width="360">

# Gitee CLI

**无需离开终端，即可完成 Gitee 日常工作。**

Pull Request · Issue · 仓库 · Release · AI 工作流 · Gitee API

[![Go 版本](https://img.shields.io/badge/Go-1.25.12+-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![许可证](https://img.shields.io/badge/License-MIT-2F6FEB)](LICENSE)
![支持平台](https://img.shields.io/badge/平台-macOS%20%7C%20Linux%20%7C%20Windows-4C566A)

[English](README.md) · **简体中文**

[快速开始](#快速开始) · [命令一览](#命令一览) · [Agent 工作流](#agent-工作流) · [实践指南](#实践指南) · [参与贡献](#参与贡献)

</div>

---

## 为什么选择 Gitee CLI

| 核心能力 |
|---|
| **日常工作流** · 管理 PR、Issue、仓库、Release、Gist、SSH Key，并直接调用 API。 |
| **终端交互** · 在交互式表格中浏览和操作数据，快捷键随场景动态展示。 |
| **自动化** · 使用 JSON 输出、非交互参数、命令别名与 Shell 补全接入脚本和 CI。 |
| **Agent 友好** · 通过结构化 JSON、显式的 `--repo` 与 `--hostname`、非交互参数及原始 `gitee api` 入口，方便 Agent 组合工具工作流。 |
| **AI 辅助** · 起草 PR 和 Issue、审查代码、连续对话，并接入任意 OpenAI 兼容服务。 |
| **多 Host** · 在同一份配置中使用 `gitee.com` 和私有部署的 Gitee。 |
| **稳定执行** · 自动重试临时故障、跟踪限额、干净取消，并通过 `--verbose` 排查请求。 |

## 安装

通过 npm 全局安装最新版本：

```bash
npm install -g @gitee/gitee-cli
```

包会通过 npm 的可选依赖（optional dependencies）自动安装与你平台（macOS、Linux 或 Windows）匹配的二进制。验证是否可用：

```bash
gitee --version
```

### 从源码构建

需要 Go 1.25.12 或更高版本。

```bash
git clone https://gitee.com/oschina/gitee-cli.git
cd gitee-cli
make install
```

`make install` 会完成构建，并将 CLI 安装到 `$HOME/bin/gitee`。

<details>
<summary><strong>不使用 make，直接构建</strong></summary>

```bash
git clone https://gitee.com/oschina/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
```

</details>

## 快速开始

```console
$ gitee auth login
$ gitee pr list
$ gitee pr checkout 123
$ gitee issue list -s open
$ gitee search repos gitee-cli
$ gitee api /user
```

| 目标 | 命令 |
|---|---|
| 登录 Gitee | `gitee auth login` |
| 搜索仓库、Issue 或用户 | `gitee search repos <query>` |
| 在仓库目录之外执行命令 | `gitee pr list -R owner/repo` |
| 开启交互式表格 | `gitee config set tui true` |
| 将界面切换为中文 | `gitee config set locale zh_CN` |
| 查看命令帮助 | `gitee <command> --help` |

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
gitee auth login --hostname git.company.com
gitee pr list --hostname git.company.com -R owner/repo
gitee auth status
gitee auth logout --hostname git.company.com
```

非默认 Host 的 Token 保存在 `~/.config/gitee/hosts.yml`。

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
| `search` | `repos`, `issues`, `users` | 在 Gitee 全站搜索 |
| `update` | `--check`, `--yes` | 检查并安装 CLI 更新 |
| `alias` | `list`, `set`, `delete` | 管理命令别名 |
| `ai` | `[prompt]`, `--chat` | 与 OpenAI 兼容模型对话 |
| `api` | `<endpoint>` | 发起原始 API 请求 |
| `version` | — | 显示版本信息 |
| `completion` | `bash`, `zsh`, `fish`, `powershell` | 生成 Shell 补全脚本 |

## Agent 工作流

Gitee CLI 同时面向开发者和编码 Agent。结构化输出、显式目标与非交互输入，
让每一步都可以稳定组合：

```bash
gitee pr list -R owner/repo -s open --json
gitee pr view 42 -R owner/repo --json
gitee api /repos/owner/repo/pulls --hostname gitee.com
```

诊断信息写入 stderr，不会污染 stdout 中供 Agent 解析的 JSON。脚本、CI、别名和
原始接口的组合方式见 [Agent 与自动化指南](docs/usage_zh.md#agent-与自动化工作流)。

## 实践指南

README 只保留上手所需内容，详细用法集中在一份可按章节跳转的指南中：

| 指南 | 内容 |
|---|---|
| [Agent 与自动化](docs/usage_zh.md#agent-与自动化工作流) | JSON 输出、显式目标、非交互调用、别名与常用参数 |
| [TUI 模式](docs/usage_zh.md#tui-模式) | 交互式表格、快捷键、图片渲染与 tmux |
| [Pull Request 本地分支](docs/usage_zh.md#pull-request-本地分支) | 如何选择 `pr checkout` 与 `pr fetch` |
| [AI 工作流](docs/usage_zh.md#ai-工作流) | 服务配置、PR 与 Issue 辅助、对话和 Commit Message |
| [原始 API 请求](docs/usage_zh.md#原始-api-请求) | 直接调用 Gitee V5 接口 |
| [配置](docs/usage_zh.md#配置) | 配置文件、键、主题、语言、编辑器与凭证 |
| [稳定性与调试](docs/usage_zh.md#稳定性与调试) | 重试、取消、限额与详细诊断 |

---

## 参与贡献

- 开发和 Pull Request 要求见 [CONTRIBUTING.md](CONTRIBUTING.md)。
- 安全问题请按照 [SECURITY.md](SECURITY.md) 私下报告；
- 所有参与者均需遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

---

## 许可证

[MIT](LICENSE)
