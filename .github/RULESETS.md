# GitHub 仓库规则集

本目录包含 yakevent 仓库的 GitHub Ruleset 导入文件，用于保护主干分支和发布标签。

## 文件列表

| 文件 | 类型 | 规则集名称 | 保护目标 |
|---|---|---|---|
| `protect-branch-main.json` | 分支规则集 | `protect-branch-main` | 默认分支（`main`） |
| `protect-semver-tags.json` | 标签规则集 | `protect-semver-tags` | 所有 `v*` 语义版本标签 |

---

## 导入方式

1. 进入仓库 **Settings → Rules → Rulesets**
2. 点击右上角 **New ruleset → Import a ruleset**
3. 上传对应 JSON 文件，点击 **Create**
4. 两个文件分别导入，各自独立生效

> GitHub 通过 JSON 中的 `"target"` 字段自动识别规则集类型（`"branch"` 或 `"tag"`），导入时无需手动选择。

---

## protect-branch-main.json — 分支规则集

**保护目标**：`~DEFAULT_BRANCH`（始终指向当前默认分支，重命名后无需修改）

| 规则 | 配置值 | 说明 |
|---|---|---|
| `deletion` | 启用 | 禁止删除 `main` 分支 |
| `non_fast_forward` | 启用 | 禁止强制推送（force push），保护提交历史 |
| `pull_request` | 需要 1 位审批者 | 所有变更必须通过 PR 合并，不可直接推送 |
| `required_status_checks` | `Test / all-pass`、`Lint / lint` | CI 全部通过后方可合并 |

**Status Check 上下文格式**：`<workflow name> / <job id>`，对应本仓库：

- `Test / all-pass` → [`.github/workflows/test.yml`](../workflows/test.yml) 中的 `all-pass` job
- `Lint / lint` → [`.github/workflows/format.yml`](../workflows/format.yml) 中的 `lint` job

**`bypass_actors`**：仓库管理员（Repository admin，`actor_id: 5`）设置为 `bypass_mode: always`，可直接推送 main 而不受 PR 和 Status Check 约束。这对单人维护的开源库是必要的——Status Check 仅在 CI 运行后才有结果，管理员直接推送时 CI 尚未运行，若无 bypass 会形成死循环。

> PR 审批人数设为 `1`：外部贡献者提交的 PR 需要仓库维护者审批后方可合并。管理员因配置了 `bypass_mode: always`，可绕过此限制直接推送 main，不受影响。

---

## protect-semver-tags.json — 标签规则集

**保护目标**：`refs/tags/v*`（匹配所有以 `v` 开头的标签，如 `v1.0.0`、`v2.3.1`）

| 规则 | 配置值 | 说明 |
|---|---|---|
| `deletion` | 启用 | 禁止删除已发布的版本标签 |
| `non_fast_forward` | 启用 | 禁止将已有标签移动到其他 commit |

**Go 模块特殊说明**：

Go module proxy（`proxy.golang.org`）和 `pkg.go.dev` 会永久缓存已发布的版本。一旦某个 `v*` tag 被推送并被用户拉取，**删除或移动该 tag 不会使旧缓存失效**，反而会造成版本内容不一致。因此标签的不可变性对 Go 公开库至关重要。

**`bypass_actors` 为空**：任何人均不可删除或移动已发布 tag。

---

## 修改规则集

如需调整规则（如增加必需审批人数、添加更多 Status Check），直接编辑对应 JSON 文件后重新导入，或在 GitHub Web UI 的 Settings → Rules → Rulesets 页面直接编辑已激活的规则集。
