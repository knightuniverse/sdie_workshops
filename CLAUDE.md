# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 提供在 Gitea 代码库中工作的指导。

## 项目概述

Gitea 是一个用 Go 语言编写的自托管 Git 服务，前端使用 JavaScript/Vue.js。代码采用分层架构：模型层（models）→ 服务层（services）→ 路由层（routers）。

## ⚠️ 修改前必读文档

**在修改任何模块代码之前，请先阅读对应的规范文档：**

| 修改模块 | 必读文档 | 文档路径 |
|----------|----------|----------|
| **整体架构** / 跨模块修改 | Gitea 架构设计文档 | `.claude/rules/arch.md` |
| **modules/** 目录 | modules 目录代码规范 | `.claude/rules/model.md` |
| **routers/** 目录 | routers 目录代码规范 | `.claude/rules/router.md` |
| **services/** 目录 | services 目录代码规范 | `.claude/rules/service.md` |
| **代码质量** / Lint / 测试 | 代码质量保证体系 | `.claude/rules/quality-gate.md` |

这些文档包含了各模块的代码规范、命名约定、错误处理模式、导入别名规则等关键信息，遵循这些规范能确保代码一致性和可维护性。

## 核心目录

- **cmd/** - CLI 命令实现（admin, doctor, dump 等）
- **models/** - 数据库模型和 ORM 逻辑（基于 XORM）
- **modules/** - 可复用的包和工具
  - `modules/git/` - Git 操作封装
  - `modules/setting/` - 配置管理
  - `modules/markup/` - Markdown/HTML 渲染
- **routers/** - HTTP 请求处理器
  - `routers/api/v1/` - REST API 端点（带 Swagger 注解）
  - `routers/web/` - Web UI 处理器
  - `routers/private/` - 内部 API（用于 SSH）
- **services/** - 业务逻辑层
- **templates/** - Go HTML 模板（Web UI）
- **web_src/** - 前端源码（JavaScript, CSS, Vue.js）
- **tests/** - 集成测试和 E2E 测试

## 请求流程

```
HTTP 请求 → routers/ → services/ → models/ → 数据库
```

## 常用命令

### 构建项目

```bash
# 完整构建（推荐）
TAGS="bindata" make build

# 仅构建后端
make backend

# 仅构建前端
make frontend

# 支持 SQLite 的构建
TAGS="bindata sqlite sqlite_unlock_notify" make build
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行后端单元测试
make test-backend

# 运行前端测试
make test-frontend

# 运行特定测试
make test#TestSpecificName

# 运行 SQLite 集成测试
make test-sqlite

# 运行特定集成测试
make test-sqlite#TestSpecificName
```

### 代码检查和格式化

```bash
# 检查所有代码
make lint

# 检查后端代码
make lint-backend

# 检查前端代码
make lint-frontend

# 自动修复问题
make lint-fix

# 格式化代码（提交前必做）
make fmt
```

### 开发模式

```bash
# 监听模式（自动重建）
make watch          # 监听所有文件
make watch-frontend # 仅监听前端
make watch-backend  # 仅监听后端（使用 air）

# 安装依赖
make deps           # 安装所有依赖
make deps-frontend  # 安装前端依赖
make deps-backend   # 安装后端依赖
```

## 关键约定

### Go 代码规范

- 使用 `modules/json` 而非 `encoding/json`
- 使用 `os` 或 `io` 而非 `io/ioutil`
- 使用 Gitea 的配置系统而非 `gopkg.in/ini.v1`
- 提交前必须运行 `make fmt`

### 数据库迁移

- 迁移文件位于 `models/migrations/`
- 每个迁移都有对应的测试文件
- 运行迁移测试：`make migrations.individual.sqlite.test#v1_23`

### API 开发

- API 路由位于 `routers/api/v1/`
- 添加 Swagger 注解用于自动文档
- API 修改后运行：`make generate-swagger`
- 验证 Swagger：`make swagger-validate`

### 测试文件位置

- 单元测试：与源码同目录的 `*_test.go` 文件
- 集成测试：`tests/integration/`
- E2E 测试：`tests/e2e/`（使用 Playwright）
- 测试数据：`models/fixtures/`（YAML 格式）

## 常见问题

### 构建失败

- 确保 Go 版本符合 `go.mod` 要求（当前为 Go 1.22）
- 运行 `make deps-backend` 下载 Go 模块
- 运行 `make deps-frontend` 安装 npm 包

### 测试失败

- 集成测试需要支持 LFS 的 git
- 某些测试需要特定数据库后端（MySQL, PostgreSQL, SQLite）
- 本地测试建议使用 `make test-sqlite`

### 前端问题

- 如果 `node_modules` 不存在，运行 `npm install`
- Node.js 版本要求：>= 18.0.0
