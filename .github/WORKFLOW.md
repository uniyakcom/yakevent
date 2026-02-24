# 工作流说明

## 文件清单

| 文件 | 作用 |
|---|---|
| `.githooks/pre-commit` | 提交时自动格式化 Go 代码 |
| `.github/workflows/format.yml` | lint 门禁（format + vet），push + PR 触发 |
| `.github/workflows/test.yml` | 单元测试 + race 检测（4 平台矩阵），push + PR 触发 |
| `.github/workflows/bench.yml` | 性能基准测试（4 平台并发）+ benchstat 对比（PR）+ 结果归档到 `bench` 分支 |
| `.github/workflows/release.yml` | push 到 main 时自动维护草稿 Release（Conventional Commits 驱动） |

## 初始化

克隆仓库后执行一次：

```bash
git config core.hooksPath .githooks
```

## 日常开发

直接向 main push 即可，**无需开 PR**：

```bash
git add .
git commit -m "feat: add batch mode"   # pre-commit 自动格式化
git push                               # release workflow 自动更新草稿 Release
```

## Conventional Commits 规范

提交信息格式决定版本递增和 Changelog 分类：

| 提交前缀 | Changelog 分类 | 版本变化 |
|---|---|---|
| `feat:` / `feat(scope):` | 🚀 Features | minor +1 |
| `feat!:` / `BREAKING CHANGE:` | ⚠ Breaking Changes | major +1 |
| `fix:` / `fix(scope):` | 🐛 Bug Fixes | patch +1 |
| `perf:` / `opt:` | 🛩 Enhancements | patch +1 |
| `refactor:` | 🛩 Enhancements | patch +1 |
| `docs:` | 📚 Documentation | — |
| `chore:` / `ci:` / `dep:` | 🗃 Misc | — |

> **Breaking Changes**：在提交信息末尾加 `!`，或在 body/footer 中写 `BREAKING CHANGE: 说明`

## 发布流程

1. 向 main 推送遵循 Conventional Commits 的提交
2. release workflow 自动在 GitHub 上更新一个**草稿 Release**，内含完整 Changelog
3. 确认版本和变更内容后，在 GitHub **Releases** 页面点击 **Edit → Publish release** 公开发布
4. Publish 时自动创建 Git Tag（`vX.Y.Z`）

## 性能基准数据

基准结果存储在独立孤立分支 **`bench`**（不污染 main 历史），目录结构：

```
bench 分支/
├── main/                          ← 每次 push to main 后自动更新
│   ├── ubuntu-latest-amd64.txt    # Ubuntu 24.04, x86_64
│   ├── windows-latest-amd64.txt   # Windows Server 2025, x86_64
│   ├── ubuntu-24.04-arm-arm64.txt # Ubuntu 24.04, aarch64
│   └── macos-latest-arm64.txt     # macOS 15, Apple M 系列
├── pr-{N}/                        ← PR #{N} 触发时写入
│   ├── {os}-{arch}.txt           # HEAD 结果
│   ├── {os}-{arch}-base.txt      # base commit 结果
│   └── {os}-{arch}-diff.txt      # benchstat 差异对比
└── manual/YYYYMMDD/               ← 手动触发时写入
    └── {os}-{arch}.txt
```

每个文件头部包含环境元数据：

```
# os:     ubuntu-latest
# arch:   amd64
# kernel: Linux 6.8.0-1021-azure x86_64
# go:     go1.25.x linux/amd64
# sha:    3204245bd5ba5acb...
# ref:    refs/heads/main
# date:   2026-02-24T10:30:00Z
BenchmarkImplSync-4    ...
```

**查看历史基线：**

```bash
git fetch origin bench
git checkout bench
cat main/ubuntu-latest-amd64.txt
```

**本地与 CI 跨版本对比：**

```bash
benchstat bench/main/ubuntu-latest-amd64.txt local-result.txt
```

## 格式化

- **本地**：`pre-commit` 钩子在每次 commit 时自动运行 `gofmt`
- **远程**：`format.yml` 检查 PR 格式，未通过则阻止合并

## Release 产物

每次发布后自动生成：

- Git Tag（`vX.Y.Z`）
- GitHub Release 页面（含 Changelog）
- 分类 Changelog（Breaking Changes / Features / Enhancements / Bug Fixes / Documentation / Misc）
- Full Changelog 对比链接
