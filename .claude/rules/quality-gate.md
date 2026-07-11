# 代码质量保证体系

本文档梳理 Gitea 项目支持的所有代码质量保证方式，涵盖静态分析、格式化、测试、安全检查等维度。

---

## 目录

- [1. 总览](#1-总览)
- [2. 静态分析 (Lint)](#2-静态分析-lint)
- [3. 代码格式化 (Format)](#3-代码格式化-format)
- [4. 一致性检查 (Checks)](#4-一致性检查-checks)
- [5. 测试体系](#5-测试体系)
- [6. 安全检查](#6-安全检查)
- [7. 聚合命令速查](#7-聚合命令速查)

---

## 1. 总览

Gitea 的质量保证体系覆盖 **Go 后端**、**JavaScript/Vue 前端**、**CSS**、**模板**、**文档**、**配置文件** 六大领域，共 20+ 个质量工具。

```
质量保证
├── 静态分析 (Lint)
│   ├── Go:       golangci-lint, gitea-vet
│   ├── JS/Vue:   ESLint
│   ├── CSS:      Stylelint
│   ├── 模板:     djlint, SVG lint
│   ├── 文档:     markdownlint
│   ├── 配置:     yamllint, actionlint
│   ├── 拼写:     misspell
│   ├── 编辑器:   editorconfig-checker
│   └── API:      Spectral (Swagger)
├── 代码格式化 (Format)
│   ├── Go:       gofumpt
│   └── 模板:     whitespace cleanup
├── 一致性检查 (Checks)
│   ├── go mod tidy-check
│   ├── swagger-check / swagger-validate
│   ├── fmt-check
│   ├── lockfile-check
│   └── svg-check
├── 测试
│   ├── 单元测试:   Go unit tests, Vitest
│   ├── 集成测试:   SQLite/MySQL/PostgreSQL/MSSQL
│   ├── E2E 测试:   Playwright
│   ├── 迁移测试:   database migration tests
│   ├── 模糊测试:   Go fuzz tests
│   └── 覆盖率:     coverage reports
└── 安全检查
    └── govulncheck
```

---

## 2. 静态分析 (Lint)

### 2.1 Go 后端

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **golangci-lint** v1.57.2 | 18+ 子检查器的静态分析聚合（errcheck, govet, staticcheck, revive, gocritic, depguard, unused 等） | `make lint-go` | `.golangci.yml` |
| **gitea-vet** | 自定义 Go vet 规则，检查 Gitea 特有的代码问题 | `make lint-go-vet` | `code.gitea.io/gitea-vet` |

**golangci-lint 启用的子检查器**：bidichk, depguard, dupl, errcheck, forbidigo, gocritic, gofmt, gofumpt, gosimple, govet, ineffassign, nakedret, nolintlint, revive, staticcheck, stylecheck, typecheck, unconvert, unused, wastedassign

**depguard 限制导入**：禁止直接使用 `encoding/json`、`io/ioutil`、`golang.org/x/exp`、`gopkg.in/ini.v1`，需使用 Gitea 内部替代品。

### 2.2 JavaScript / Vue

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **ESLint** 8.57.0 | JS/Vue 代码质量、最佳实践、导入规范、jQuery 使用、正则表达式、可访问性、Web Components、Vitest 规则 | `make lint-js` | `.eslintrc.yaml` |

ESLint 插件：eslint-plugin-github, eslint-plugin-vue, eslint-plugin-unicorn, eslint-plugin-sonarjs, eslint-plugin-regexp, eslint-plugin-no-jquery, eslint-plugin-wc, eslint-plugin-vitest, eslint-plugin-array-func, eslint-plugin-i, @stylistic/eslint-plugin-js, eslint-plugin-no-use-extend-native 等 17 个。

### 2.3 CSS

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **Stylelint** 16.4.0 | CSS 代码质量、厂商前缀、无效属性、颜色变量强制使用、CSS 自定义属性验证 | `make lint-css` | `stylelint.config.js` |

### 2.4 模板

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **djlint** 1.34.1 | Go HTML 模板格式和代码质量 | `make lint-templates` | `pyproject.toml` (`[tool.djlint]`) |
| **SVG lint** (自定义) | 检查模板中引用的 SVG 文件是否存在于输出目录 | `make lint-templates` | `tools/lint-templates-svg.js` |

### 2.5 文档与配置

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **markdownlint** 0.39.0 | Markdown 格式：标题结构、代码块、尾部空格、行内 HTML | `make lint-md` | `.markdownlint.yaml` |
| **yamllint** 1.35.1 | YAML 格式：缩进、注释、文档结构、truthy 值 | `make lint-yaml` | `.yamllint.yaml` |
| **actionlint** v1 | GitHub Actions 工作流文件语法和正确性 | `make lint-actions` | — |

### 2.6 拼写检查

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **misspell** v0.5.1 | Go、JS、Markdown、YAML 等源文件中的拼写错误 | `make lint-spell` | `tools/misspellings.csv`（21 个自定义词条） |

### 2.7 编辑器规范

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **editorconfig-checker** v2.7.0 | 文件是否符合 `.editorconfig` 规则（缩进、尾部空格、最终换行） | `make lint-editorconfig` | `.editorconfig` |

### 2.8 API 规范

| 工具 | 检查内容 | 命令 | 配置文件 |
|------|----------|------|----------|
| **Spectral** 6.11.1 | OpenAPI/Swagger 规范有效性和最佳实践 | `make lint-swagger` | — |

---

## 3. 代码格式化 (Format)

| 工具 | 作用 | 命令 | 配置文件 |
|------|------|------|----------|
| **gofumpt** v0.6.0 | Go 代码格式化（比 gofmt 更严格，启用 extra-rules） | `make fmt` | `.golangci.yml` |
| **模板空白清理** | 清理模板文件中的尾部空白 | `make fmt` | — |

**格式化检查**（确保代码已格式化）：

```bash
make fmt-check   # 运行 make fmt 后检查 git diff，有变更则失败
```

---

## 4. 一致性检查 (Checks)

确保生成的文件与源码保持同步。

### 4.1 后端检查

| 检查项 | 命令 | 说明 |
|--------|------|------|
| **Go 依赖** | `make tidy-check` | 运行 `go mod tidy` 后检查 `git diff`，确保 `go.mod`/`go.sum` 一致 |
| **Swagger 规范** | `make swagger-check` | 从 Go 注解生成 Swagger 后检查 `git diff`，确保规范与代码同步 |
| **Swagger 验证** | `make swagger-validate` | 验证生成的 Swagger 规范是否为合法 OpenAPI |
| **格式检查** | `make fmt-check` | 确保所有 Go 和模板文件已格式化 |

### 4.2 前端检查

| 检查项 | 命令 | 说明 |
|--------|------|------|
| **Lockfile** | `make lockfile-check` | 确保 `package-lock.json` 与 `package.json` 一致 |
| **SVG 文件** | `make svg-check` | 确保 `public/assets/img/svg/` 中的 SVG 文件是最新的 |

---

## 5. 测试体系

### 5.1 单元测试

| 语言 | 框架 | 命令 | 测试文件位置 |
|------|------|------|-------------|
| **Go** | `go test` + testify | `make test-backend` | 与源码同目录的 `*_test.go`（约 492 个文件） |
| **JS/Vue** | Vitest 1.5.2 | `make test-frontend` | `web_src/**/*.test.js` |

**Go 单元测试分布**：models/ (131), modules/ (223), routers/ (29), services/ (102), cmd/ (6)

**竞态检测**：`RACE_ENABLED=true make test-backend`

**覆盖率报告**：`make unit-test-coverage`（输出 `coverage.out`）

### 5.2 集成测试

| 数据库 | 命令 | 说明 |
|--------|------|------|
| **SQLite** | `make test-sqlite` | 本地开发推荐，无需额外数据库 |
| **MySQL** | `make test-mysql` | 需要 MySQL 服务 |
| **PostgreSQL** | `make test-pgsql` | 需要 PostgreSQL 服务 |
| **MSSQL** | `make test-mssql` | 需要 MSSQL 服务 |

- 测试文件：`tests/integration/`（约 200 个文件）
- 配置模板：`tests/{sqlite,mysql,pgsql,mssql}.ini.tmpl`
- 覆盖率：`make integration-test-coverage`

**运行单个集成测试**：`make test-sqlite#TestSpecificName`

### 5.3 数据库迁移测试

| 命令 | 说明 |
|------|------|
| `make test-sqlite-migration` | SQLite 迁移测试 |
| `make test-mysql-migration` | MySQL 迁移测试 |
| `make test-pgsql-migration` | PostgreSQL 迁移测试 |

**运行单个迁移测试**：`make migrations.individual.sqlite.test#v1_23`

### 5.4 E2E 测试

| 框架 | 命令 | 配置文件 |
|------|------|----------|
| **Playwright** 1.43.1 | `make test-e2e` | `playwright.config.js` |

- 测试文件：`tests/e2e/*.test.e2e.js`
- 浏览器引擎：Chromium, WebKit, Mobile Chrome, Mobile Safari
- 超时：30 秒
- 失败时生成 HTML 报告和截图

### 5.5 模糊测试 (Fuzz)

| 目标 | 命令 |
|------|------|
| Markdown 渲染 | `go test -fuzz FuzzMarkdownRenderRaw ./tests/fuzz/` |
| Markup 后处理 | `go test -fuzz FuzzMarkupPostProcess ./tests/fuzz/` |

测试文件：`tests/fuzz/fuzz_test.go`

### 5.6 覆盖率

| 命令 | 说明 |
|------|------|
| `make unit-test-coverage` | 单元测试覆盖率 → `coverage.out` |
| `make integration-test-coverage` | 集成测试覆盖率 → `integration.coverage.out` |
| `make coverage` | 合并所有覆盖率（通过 `gocovmerge.go`） |

---

## 6. 安全检查

| 工具 | 检查内容 | 命令 |
|------|----------|------|
| **govulncheck** v1 | Go 依赖中的已知安全漏洞 | `make security-check` |

集成在 `make checks-backend` 中自动运行。

---

## 7. 聚合命令速查

### Lint 命令

| 命令 | 范围 |
|------|------|
| `make lint` | 全部 Lint（前端 + 后端 + 拼写） |
| `make lint-fix` | 全部 Lint 并自动修复 |
| `make lint-backend` | 后端 Lint（golangci-lint + gitea-vet + editorconfig-checker） |
| `make lint-frontend` | 前端 Lint（ESLint + Stylelint） |
| `make lint-go` | 仅 Go Lint |
| `make lint-go-vet` | 仅 gitea-vet |
| `make lint-js` | 仅 ESLint |
| `make lint-css` | 仅 Stylelint |
| `make lint-md` | 仅 Markdown |
| `make lint-yaml` | 仅 YAML |
| `make lint-spell` | 仅拼写检查 |
| `make lint-swagger` | 仅 Swagger 规范 |
| `make lint-templates` | 仅模板（djlint + SVG lint） |
| `make lint-editorconfig` | 仅 EditorConfig |
| `make lint-actions` | 仅 GitHub Actions |

### Check 命令

| 命令 | 范围 |
|------|------|
| `make checks` | 全部一致性检查 |
| `make checks-backend` | 后端检查（tidy + swagger + fmt + security） |
| `make checks-frontend` | 前端检查（lockfile + svg） |

### Test 命令

| 命令 | 范围 |
|------|------|
| `make test` | 全部测试（前端 + 后端） |
| `make test-backend` | Go 单元测试 |
| `make test-frontend` | Vitest 前端测试 |
| `make test-sqlite` | SQLite 集成测试 |
| `make test-mysql` | MySQL 集成测试 |
| `make test-pgsql` | PostgreSQL 集成测试 |
| `make test-mssql` | MSSQL 集成测试 |
| `make test-e2e` | Playwright E2E 测试 |
| `make test-e2e-sqlite` | E2E + SQLite |
| `make test-*-migration` | 数据库迁移测试 |

### 依赖安装

| 命令 | 说明 |
|------|------|
| `make deps` | 安装所有依赖 |
| `make deps-backend` | 安装 Go 模块 |
| `make deps-frontend` | 安装 npm 包 |
| `make deps-tools` | 安装 Go 质量工具（golangci-lint, gofumpt, misspell 等） |
| `make deps-py` | 安装 Python 工具（djlint, yamllint） |

---

## 8. 工具版本清单

| 工具 | 版本 |
|------|------|
| golangci-lint | v1.57.2 |
| gofumpt | v0.6.0 |
| misspell | v0.5.1 |
| editorconfig-checker | v2.7.0 |
| govulncheck | v1 |
| actionlint | v1 |
| ESLint | 8.57.0 |
| Stylelint | 16.4.0 |
| Vitest | 1.5.2 |
| Playwright | 1.43.1 |
| Spectral CLI | 6.11.1 |
| markdownlint-cli | 0.39.0 |
| yamllint | 1.35.1 |
| djlint | 1.34.1 |
