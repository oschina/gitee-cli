---
name: gitee-release
description: Manage Gitee releases from the CLI, including listing, viewing, creating, editing, and deleting releases by numeric ID or tag. Use when the user says "创建 Release", "发布版本", "查看 Release", "编辑 Release", "修改版本说明", "删除 Release", "create/edit/delete a release", or asks to manage release names, notes, tags, targets, or pre-release status. Always uses `--json` where supported and `--no-tui`; never uses `--ai`; requires explicit confirmation before deleting. If a requested release operation has no direct command, fall back to the gitee-api skill to search the schema; API updates and deletions require a second explicit confirmation.
---

# Gitee Release

使用一级命令管理 Release 的创建、查询、编辑和删除。需要先通过 `gitee auth login` 完成认证，并在带 Gitee remote 的仓库中执行或传入 `-R owner/repo`。

## 前置检查

1. 运行 `gitee auth status --no-tui` 检查认证，失败则提示执行 `gitee auth login`。
2. 优先使用 `gitee release` 的直接子命令。
3. 直接命令不支持目标操作时，切换到 `gitee-api` skill，先执行 `gitee api --search` 获取 endpoint 和 schema。通过 API 更新或删除资源前必须二次确认。
4. 删除 Release 前必须展示仓库、Release ID 或 tag，并等待用户明确确认。

## 列出和查看

```bash
gitee release list -R owner/repo --json=id,tag_name,name,prerelease --no-tui
gitee release view v1.2.0 -R owner/repo --json --no-tui
gitee release view 42 -R owner/repo --json --no-tui
```

`view` 同时支持数字 ID 和 tag。需要解析后续操作时优先请求结构化字段。

## 创建

```bash
gitee release create -R owner/repo \
  --tag v1.2.0 \
  --name "Version 1.2" \
  --body "Release notes" \
  --no-tui

gitee release create -R owner/repo \
  --tag v1.3.0-rc.1 \
  --prerelease \
  --target develop \
  --no-tui
```

- `--tag` 为必填项。
- 省略 `--name` 时使用 tag 作为名称。
- 省略 `--target` 时使用仓库默认分支。
- `create` 暂不支持 `--json`，不要添加不存在的 flag。

## 编辑

```bash
gitee release edit v1.2.0 -R owner/repo \
  --name "Version 1.2 updated" \
  --body "Updated release notes" \
  --json --no-tui

gitee release edit 42 -R owner/repo --prerelease=false --json --no-tui
gitee release edit 42 -R owner/repo --body "" --json --no-tui
```

可编辑 `--tag`、`--name/-n`、`--body/-b` 和 `--prerelease`。非交互模式至少提供一个编辑字段；空正文表示清空 Release notes。

## 删除

先读取并展示目标：

```bash
gitee release view v1.2.0 -R owner/repo --json=id,tag_name,name --no-tui
```

等待用户明确确认后再执行：

```bash
gitee release delete v1.2.0 -R owner/repo --yes --no-tui
```

不要在用户确认前添加 `--yes`。删除 Release 不等同于删除其 Git tag；以命令返回结果为准。

## 判断逻辑

| 用户意图 | 命令 | 是否确认 |
|---|---|---|
| 列出或查看 Release | `release list/view` | 否 |
| 创建或发布 Release | `release create` | 用户明确要求后执行 |
| 修改名称、正文、tag 或预发布状态 | `release edit` | 否 |
| 删除 Release | `release delete` | 是 |
| 直接命令不支持的 Release 操作 | `gitee-api` fallback | API 更新和删除需二次确认 |

## 错误处理

| 错误 | 处理方式 |
|---|---|
| `--tag is required` | 补充 `--tag <tag>` |
| `target_commitish is missing` | 升级 CLI，或临时显式传入 `--target <branch>` |
| `404 Not Found` | 核对 `owner/repo`、Release ID 或 tag，并检查访问权限 |
| `at least one of --tag, --name, --body, --prerelease is required` | 为 `release edit` 提供至少一个编辑字段 |
| 删除时要求 `--yes` | 先取得用户确认，再带 `--yes` 重试 |

禁止使用任何 `--ai` flag。
