# Gitee CLI Agent Skills

> 一套面向 AI Agent 的 [Gitee CLI](https://gitee.com/oschina/gitee-cli) 技能包 (Agent Skills)，让 Agent 能安全、可靠地在 Gitee 上完成日常研发操作。
>
> A collection of Agent Skills for the [Gitee CLI](https://gitee.com/oschina/gitee-cli), enabling AI coding agents to perform everyday Gitee workflows reliably and safely.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](../LICENSE)

---

## 🤖 让 Agent 一键自主安装 / One-shot install by your Agent

把下面整段话**原样复制**发给你的 AI 编程助手（Claude Code / Codex / OpenCode 等），它就会使用当前 CLI 内置资源离线安装：

```text
请帮我安装 Gitee CLI Agent Skills：
1. 运行 `gitee skills install`，把当前 Gitee CLI 内置的技能安装到 ~/.agents/skills/。
2. 运行 `gitee skills list --json`，确认 gitee-pr、gitee-issue、gitee-release、gitee-repo、gitee-search、gitee-api 均为已安装状态。
3. 告诉我安装结果，并提示我重启 / 重新加载 Agent 以使技能生效。
```

> 更新 Gitee CLI 后，把同一段话再发一次即可。安装命令采用镜像语义，会自动反映内置 Skill 的新增、修改、删除和重命名。

---

## 这是什么 / What is this

**Agent Skills** 是一种「渐进式披露 (progressive disclosure)」的能力包：每个技能是一个 `SKILL.md` 文件，包含 YAML frontmatter（`name` / `description` / 触发词）和正文操作手册。当用户的请求命中某个技能的触发词时，Agent 会加载该技能并按其中的步骤执行。

本仓库提供 **6 个 Gitee CLI 技能**，覆盖官方 CLI 的核心能力面。每个技能都遵循同一套安全约定：

- 所有命令强制 `--no-tui`（禁用交互式 TUI，避免 Agent 阻塞）；
- 支持结构化输出的命令一律加 `--json`；
- **绝不使用 `--ai` 系列 flag**（会触发交互式 pager / 确认，导致阻塞）；
- 直接命令不支持时，通过 `gitee-api` 的 `--search` 发现 schema 后再调用；
- API 更新和删除在执行前必须向用户二次确认，删除等不可逆操作明确提示风险。

---

## 包含的技能 / Skills

| 技能 | 作用 | 触发示例 |
|------|------|---------|
| **gitee-pr** | Pull Request 全生命周期：创建 / 列出 / 查看 / 编辑 / diff / 审查(approve) / 评论 / 合并 / 关闭 / 重开 / 拉到本地(checkout/fetch)。合并需确认。 | "提个 PR"、"看看 PR"、"编辑 PR"、"review / 通过 PR"、"合并 PR"、"把 PR 拉到本地跑一下" |
| **gitee-release** | Release 生命周期：创建 / 列出 / 查看 / 编辑 / 删除。删除需确认。 | "发布 Release"、"编辑 Release"、"查看版本"、"删除 Release" |
| **gitee-api** | 直接调用 Gitee V5 REST API 的「万能逃生舱」：用 `--search` 发现 endpoint，再发起请求。解锁里程碑、标签、协作者、Webhook、文件内容等没有一级命令的能力。 | "调用 gitee api"、"加个里程碑 / label"、"管理 webhook / 协作者"、"没有现成命令怎么办" |
| **gitee-repo** | 仓库全生命周期：查看 / 列出 / 克隆 / 创建 / fork / 删除（删除需确认）。 | "创建仓库"、"fork 这个仓库"、"克隆仓库"、"删除仓库"、"列出我的仓库" |
| **gitee-search** | 全站搜索仓库 / issue / 用户，支持语言、owner、状态、标签等过滤。 | "搜索仓库"、"找一下 Go 的项目"、"搜 issue"、"搜索用户" |
| **gitee-issue** | Issue 全生命周期：列出 / 查看 / 创建 / 编辑 / 评论 / 关闭 / 重开 / 指派 / 认领并创建工作分支。 | "看看 issue"、"提个 issue"、"评论 issue"、"关闭 issue"、"认领这个 issue" |

> 各技能 `description` 中标注了触发词与相互边界，Agent 依据用户措辞自动加载对应技能。

---

## 前置条件 / Prerequisites

1. **安装 Gitee CLI**：参见 <https://gitee.com/oschina/gitee-cli>（`gitee version` 可验证）。
2. **完成认证**：
   ```sh
   gitee auth login
   gitee auth status
   ```
3. 一个支持 Agent Skills 的 Agent（技能默认安装到 `~/.agents/skills/`）。

---

## 安装 / Install

### 方式一：CLI 离线安装（推荐）

安装 Gitee CLI 后直接运行，不需要克隆仓库或访问网络：

```sh
gitee skills install
gitee skills list
```

命令会把当前 CLI 二进制内置的 Skill 写入 `~/.agents/skills/<name>/`。更新时还会清理本项目已经废弃的旧名称，避免新旧 Skill 同时生效。

### 更新 / Update

更新 Gitee CLI 后直接重跑 `gitee skills install`，无需先卸载。命令采用「镜像式」安装：每次运行时会把本项目管理的每个技能目录**整体替换**为当前 CLI 内置版本，因此内容修改、文件新增、文件删除、文件重命名都会被准确反映。命令只会额外清理本项目发布过的废弃名称；`~/.agents/skills/` 下由其他来源安装的技能**不会被触碰**。

自定义安装目录：

```sh
gitee skills install --dir "$HOME/.config/agents/skills"
# 或
AGENTS_SKILLS_DIR="$HOME/.config/agents/skills" gitee skills install
```

### 方式二：源码检出兼容入口

仅在从源码工作且尚未构建 CLI 时使用：

```sh
sh skills/install.sh
```

### 方式三：手动复制

```sh
mkdir -p ~/.agents/skills
cp -R skills/gitee-pr            ~/.agents/skills/
cp -R skills/gitee-release       ~/.agents/skills/
cp -R skills/gitee-api          ~/.agents/skills/
cp -R skills/gitee-repo         ~/.agents/skills/
cp -R skills/gitee-search       ~/.agents/skills/
cp -R skills/gitee-issue        ~/.agents/skills/
```

安装后重启 / 重新加载 Agent 即可让技能生效。

### 卸载 / Uninstall

```sh
gitee skills uninstall
# 非交互环境：
gitee skills uninstall --yes
```

（同样支持 `--dir` 或 `AGENTS_SKILLS_DIR` 覆盖目标目录；源码检出兼容入口为 `sh skills/uninstall.sh`。）

---

## 目录结构 / Layout

```
gitee-cli/
├── LICENSE
├── README.md
└── skills/
    ├── README.md
    ├── install.sh
    ├── uninstall.sh
    ├── gitee-pr/SKILL.md
    ├── gitee-release/SKILL.md
    ├── gitee-api/SKILL.md
    ├── gitee-repo/SKILL.md
    ├── gitee-search/SKILL.md
    └── gitee-issue/SKILL.md
```

---

## 贡献 / Contributing

欢迎 PR！新增或修改技能时请遵循既有约定：

- 每个技能至少包含 `skills/<name>/SKILL.md`，可选 `agents/openai.yaml` 等标准资源；
- frontmatter 必须包含 `name`、`description`（含中英文触发词）；可选字段必须符合 Skill 校验器规范；
- 正文（中文）按 `前置检查 → 执行步骤 → 判断逻辑 → 完整示例 → 错误处理` 组织；
- 所有命令示例必须使用**真实存在的 flag**、强制 `--no-tui`、支持时加 `--json`、**禁止实际执行 `--ai`**；
- 不可逆操作必须包含用户确认步骤。

提交前可自查：

```sh
# 不应有实际执行 `gitee ... --ai` 的命令行；说明文字中可以提及该 flag
grep -Enr '^[[:space:]]*gitee .*--ai' skills/ && echo "FAIL" || echo "OK: no executable --ai command"
# 安装脚本语法检查
sh -n skills/install.sh
sh -n skills/uninstall.sh
```

---

## License

[MIT](../LICENSE) © 2026 gitee
