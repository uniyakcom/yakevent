# 新仓库 GitHub 手动配置清单

复制 `.github/` 目录到新仓库后，按以下顺序在 GitHub 网页完成配置。

---

## 1. 初始化 Git Hooks

克隆仓库后执行一次：

```bash
git config core.hooksPath .githooks
```

---

## 2. 创建 Fine-grained PAT

> 每个账号只需创建一次，多个仓库可共用同一个 Token（需包含对应仓库权限）。

**路径**：GitHub 头像 → Settings → Developer settings → Personal access tokens → Fine-grained tokens → **Generate new token**

| 字段 | 填写内容 |
|---|---|
| Token name | `<账号名>-bot`（如 `uniyak-bot`） |
| Expiration | 1 year |
| Repository access | Only select repositories → 选择目标仓库 |
| Permissions → Contents | **Read and write** |
| Permissions → Pull requests | **Read and write** |
| Permissions → Metadata | Read-only（自动，不可去掉） |

生成后**立即复制**，仅显示一次。

---

## 3. 添加 Repository Secret

**路径**：仓库 → Settings → Secrets and variables → Actions → **New repository secret**

| Name | Value |
|---|---|
| `RELEASE_TOKEN` | 第 2 步复制的 Token |

---

## 4. 导入 Branch Ruleset

**路径**：仓库 → Settings → Rules → Rulesets → **New ruleset** → Import a ruleset

上传文件：`.github/rulesets/protect-branch-main.json`

导入后确认以下设置（JSON 已包含，核对即可）：

- **Target**：Default branch
- **Required status checks**：`Lint / lint`、`Test / all-pass`
- **Allowed merge methods**：仅 `Squash`
- **Bypass actors**：Repository role → Admin → Always

---

## 5. 导入 Tag Ruleset

**路径**：仓库 → Settings → Rules → Rulesets → **New ruleset** → Import a ruleset

上传文件：`.github/rulesets/protect-semver-tags.json`

---

## 6. 开启仓库 Auto-merge 功能

**路径**：仓库 → Settings → General → Pull Requests → ✅ **Allow auto-merge**

---

## 7. 初次提交触发 CI

```bash
git add .
git commit -m "chore: init"
git push
```

推送后 Actions 会自动运行所有 workflow，Release workflow 将创建首个草稿 Release。

---

## 发布流程（日常使用）

```
正常开发并 push → Release workflow 自动更新草稿 Release（含 changelog）
                          ↓
     GitHub → Releases → 找到草稿 → 确认内容 → 点击 Publish release
                          ↓
                    Tag 自动创建，正式发布
```

---

## 外部贡献者 PR 流程

贡献者提交 PR 时会自动加载 PR 模板，引导填写符合 Conventional Commits 的 PR 标题。

合并条件：
- CI 全绿（`Lint / lint` + `Test / all-pass`）
- 至少 1 个 Approve
- 仅允许 Squash merge（PR 标题作为最终 commit 消息）
