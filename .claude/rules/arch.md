# Gitea 架构设计文档

## 目录

- [1. 项目概述](#1-项目概述)
- [2. 整体架构](#2-整体架构)
- [3. 核心分层架构](#3-核心分层架构)
  - [3.1 路由层 (Routers)](#31-路由层-routers)
  - [3.2 服务层 (Services)](#32-服务层-services)
  - [3.3 模型层 (Models)](#33-模型层-models)
  - [3.4 模块层 (Modules)](#34-模块层-modules)
- [4. 请求处理流程](#4-请求处理流程)
- [5. 核心模块详解](#5-核心模块详解)
  - [5.1 数据库层](#51-数据库层)
  - [5.2 Git 操作层](#52-git-操作层)
  - [5.3 配置管理](#53-配置管理)
  - [5.4 认证与授权](#54-认证与授权)
- [6. API 设计](#6-api-设计)
- [7. Web UI 架构](#7-web-ui-架构)
- [8. 并发与异步处理](#8-并发与异步处理)
- [9. 存储架构](#9-存储架构)
- [10. 测试架构](#10-测试架构)

---

## 1. 项目概述

Gitea 是一个用 Go 语言编写的自托管 Git 服务，提供类似 GitHub/GitLab 的功能。项目采用前后端分离架构：

- **后端**: Go 语言 (1.22+)
- **前端**: JavaScript/Vue.js
- **数据库**: MySQL, PostgreSQL, SQLite3, MSSQL
- **缓存**: Redis, Memcached

### 技术栈

| 组件     | 技术                   |
| -------- | ---------------------- |
| Web 框架 | go-chi/chi             |
| ORM      | XORM                   |
| 模板引擎 | Go HTML Template       |
| 前端构建 | Webpack                |
| 前端框架 | Vue.js 3 + Fomantic UI |
| 测试框架 | testify + Playwright   |

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      客户端层 (Client)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Web UI   │  │ REST API │  │ Git CLI  │  │ SSH      │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
└───────┼─────────────┼─────────────┼─────────────┼───────────┘
        │             │             │             │
┌───────▼─────────────▼─────────────▼─────────────▼───────────┐
│                      路由层 (Routers)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ routers/web/ │  │ routers/api/ │  │ routers/     │       │
│  │ (Web UI)     │  │ (REST API)   │  │ private/     │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
└─────────┼─────────────────┼─────────────────┼───────────────┘
          │                 │                 │
┌─────────▼─────────────────▼─────────────────▼───────────────┐
│                      服务层 (Services)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ repository   │  │ issue        │  │ pull         │       │
│  │ user         │  │ actions      │  │ webhook      │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
└─────────┼─────────────────┼─────────────────┼───────────────┘
          │                 │                 │
┌─────────▼─────────────────▼─────────────────▼───────────────┐
│                      模型层 (Models)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ repo         │  │ issues       │  │ user         │       │
│  │ organization │  │ actions      │  │ auth         │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
└─────────┼─────────────────┼─────────────────┼───────────────┘
          │                 │                 │
┌─────────▼─────────────────▼─────────────────▼───────────────┐
│                   模块层 (Modules) - 基础库                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ git      │ │ setting  │ │ storage  │ │ cache    │       │
│  │ log      │ │ auth     │ │ markup   │ │ queue    │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
└─────────────────────────────────────────────────────────────┘
          │                 │                 │
┌─────────▼─────────────────▼─────────────────▼───────────────┐
│                    数据存储层 (Storage)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ 数据库       │  │ 文件系统     │  │ 外部存储     │       │
│  │ (XORM)       │  │ (Git Repos)  │  │ (S3/MinIO)   │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 核心分层架构

Gitea 采用经典的**四层架构**设计，每层职责明确，依赖关系单向：

```
routers/ → services/ → models/ → modules/
   ↓           ↓          ↓          ↓
 路由处理    业务逻辑    数据访问    基础工具
```

### 3.1 路由层 (Routers)

路由层负责处理 HTTP 请求，是应用的入口点。

#### 目录结构

```
routers/
├── api/                    # REST API 路由
│   ├── v1/                # API v1 版本
│   │   ├── repo/          # 仓库相关 API
│   │   ├── user/          # 用户相关 API
│   │   ├── org/           # 组织相关 API
│   │   ├── admin/         # 管理员 API
│   │   └── api.go         # API 路由注册
│   ├── actions/           # Actions API
│   └── packages/          # 包管理 API
├── web/                    # Web UI 路由
│   ├── repo/              # 仓库页面
│   ├── user/              # 用户页面
│   ├── org/               # 组织页面
│   ├── admin/             # 管理后台
│   └── web.go             # Web 路由注册
├── private/                # 内部 API (SSH 等)
├── install/                # 安装向导
└── init.go                 # 初始化逻辑
```

#### 路由注册示例

```go
// routers/api/v1/api.go
func Routes() *web.Route {
    r := web.NewRoute()

    // 中间件
    r.Use(context.APIContexter())
    r.Use(auth.APIMiddleware())

    // 路由组
    r.Group("/repos", func() {
        r.Get("", repo.ListMyRepos)
        r.Post("", bind(api.CreateRepoOption{}), repo.Create)
        r.Get("/{username}/{reponame}", repo.GetRepo)
    })

    return r
}
```

#### 中间件机制

```go
// 认证中间件
func APIMiddleware() func(ctx *context.APIContext) {
    return func(ctx *context.APIContext) {
        // 1. 解析 Token
        // 2. 验证用户身份
        // 3. 设置 ctx.Doer
    }
}
```

### 3.2 服务层 (Services)

服务层包含业务逻辑，是应用的核心层。

#### 目录结构

```
services/
├── repository/             # 仓库服务
│   ├── create.go          # 创建仓库
│   ├── delete.go          # 删除仓库
│   ├── fork.go            # Fork 仓库
│   ├── collaboration.go   # 协作者管理
│   └── branch.go          # 分支管理
├── issue/                  # Issue 服务
├── pull/                   # Pull Request 服务
├── actions/                # CI/CD 服务
├── webhook/                # Webhook 服务
├── auth/                   # 认证服务
├── mailer/                 # 邮件服务
└── cron/                   # 定时任务
```

#### 服务层设计原则

```go
// services/repository/create.go

// CreateRepoOptions 定义创建仓库的选项
type CreateRepoOptions struct {
    Name          string
    Description   string
    IsPrivate     bool
    AutoInit      bool
    // ...
}

// CreateRepository 创建仓库的核心业务逻辑
func CreateRepository(ctx context.Context, doer *user_model.User, owner *user_model.User, opts CreateRepoOptions) (*repo_model.Repository, error) {
    // 1. 参数验证
    if err := validateCreateRepoOptions(opts); err != nil {
        return nil, err
    }

    // 2. 权限检查
    if !owner.CanCreateRepo() {
        return nil, ErrReachLimitOfRepo{}
    }

    // 3. 调用模型层创建数据库记录
    repo := &repo_model.Repository{
        OwnerID:     owner.ID,
        Name:        opts.Name,
        Description: opts.Description,
        IsPrivate:   opts.IsPrivate,
    }
    if err := repo_model.CreateRepository(ctx, repo); err != nil {
        return nil, err
    }

    // 4. 初始化 Git 仓库
    if err := initGitRepository(ctx, repo, opts); err != nil {
        return nil, err
    }

    // 5. 触发后续操作 (Webhook, 通知等)
    notify_service.CreateRepository(ctx, doer, repo)

    return repo, nil
}
```

### 3.3 模型层 (Models)

模型层负责数据访问和数据库操作。

#### 目录结构

```
models/
├── repo/                   # 仓库模型
│   ├── repo.go            # 仓库主模型
│   ├── issue.go           # Issue 模型
│   ├── pull_request.go    # PR 模型
│   ├── branch.go          # 分支模型
│   └── attachment.go      # 附件模型
├── user/                   # 用户模型
├── organization/           # 组织模型
├── issues/                 # Issue 相关模型
├── actions/                # Actions 模型
├── auth/                   # 认证模型
├── db/                     # 数据库引擎
│   ├── engine.go          # XORM 引擎
│   ├── session.go         # 会话管理
│   └── pagination.go      # 分页支持
└── migrations/             # 数据库迁移
```

#### 模型定义示例

```go
// models/repo/repo.go

// Repository 代表一个 Git 仓库
type Repository struct {
    ID            int64              `xorm:"pk autoincr"`
    OwnerID       int64              `xorm:"UNIQUE(s)"`
    OwnerName     string             `xorm:"-"`
    Name          string             `xorm:"UNIQUE(s) INDEX NOT NULL"`
    Description   string
    Website       string
    DefaultBranch string

    // 统计字段
    NumWatches    int
    NumStars      int
    NumForks      int
    NumIssues     int
    NumClosedIssues int

    // 时间戳
    CreatedUnix   timeutil.TimeStamp `xorm:"created"`
    UpdatedUnix   timeutil.TimeStamp `xorm:"updated"`

    // 关联对象 (不存储在数据库)
    Owner         *user_model.User  `xorm:"-"`
    GitRepo       *git.Repository   `xorm:"-"`
}

func init() {
    db.RegisterModel(new(Repository))
}
```

#### 数据库操作示例

```go
// models/repo/repo.go

// GetRepositoryByID 根据 ID 获取仓库
func GetRepositoryByID(ctx context.Context, id int64) (*Repository, error) {
    repo := new(Repository)
    has, err := db.GetEngine(ctx).ID(id).Get(repo)
    if err != nil {
        return nil, err
    }
    if !has {
        return nil, ErrRepoNotExist{ID: id}
    }
    return repo, nil
}

// CreateRepository 创建仓库记录
func CreateRepository(ctx context.Context, repo *Repository) error {
    _, err := db.GetEngine(ctx).Insert(repo)
    return err
}
```

### 3.4 模块层 (Modules)

模块层提供可复用的基础工具库，不包含业务逻辑。

#### 核心模块

```
modules/
├── git/                    # Git 操作封装
│   ├── command.go         # Git 命令执行
│   ├── repository.go      # 仓库操作
│   ├── commit.go          # 提交操作
│   └── tree.go            # 树操作
├── setting/                # 配置管理
│   ├── setting.go         # 主配置
│   ├── database.go        # 数据库配置
│   └── server.go          # 服务器配置
├── storage/                # 存储抽象
│   ├── storage.go         # 存储接口
│   ├── local.go           # 本地存储
│   └── s3.go              # S3 存储
├── cache/                  # 缓存抽象
├── queue/                  # 队列抽象
├── log/                    # 日志系统
├── auth/                   # 认证工具
├── markup/                 # Markdown 渲染
└── util/                   # 通用工具
```

#### 模块设计原则

```go
// modules/git/command.go

// Command 代表一个 Git 命令
type Command struct {
    cmd    string
    args   []string
    ctx    context.Context
}

// NewCommand 创建新的 Git 命令
func NewCommand(ctx context.Context, args ...string) *Command {
    return &Command{
        cmd:  git.GitExecutable,
        args: args,
        ctx:  ctx,
    }
}

// AddArguments 添加参数
func (c *Command) AddArguments(args ...string) *Command {
    c.args = append(c.args, args...)
    return c
}

// RunStdString 执行命令并返回标准输出
func (c *Command) RunStdString(opts *RunOpts) (stdout, stderr string, err error) {
    // 实现...
}
```

---

## 4. 请求处理流程

### HTTP 请求流程

```
1. 客户端发起请求
   ↓
2. HTTP 服务器接收请求
   ↓
3. 路由匹配 (go-chi/chi)
   ↓
4. 中间件执行
   - 日志记录
   - 认证验证
   - 权限检查
   - CSRF 保护
   ↓
5. 路由处理器执行
   - 解析请求参数
   - 调用服务层
   - 返回响应
   ↓
6. 响应发送给客户端
```

### 详细流程示例：创建仓库

```
POST /api/v1/repos
    ↓
routers/api/v1/api.go: Routes()
    ↓
中间件: context.APIContexter() → auth.APIMiddleware()
    ↓
routers/api/v1/repo/repo.go: Create()
    ↓
解析请求体: bind(api.CreateRepoOption{})
    ↓
调用服务层: repo_service.CreateRepository()
    ↓
services/repository/create.go: CreateRepository()
    ↓
├── 验证参数
├── 检查权限
├── 调用模型层: repo_model.CreateRepository()
│   ↓
│   models/repo/repo.go: CreateRepository()
│   ↓
│   db.GetEngine(ctx).Insert(repo)
│   ↓
│   XORM 执行 SQL: INSERT INTO repository ...
│   ↓
│   数据库返回结果
├── 初始化 Git 仓库: git.InitRepository()
├── 触发通知: notify_service.CreateRepository()
└── 返回仓库对象
    ↓
路由处理器返回 JSON 响应
    ↓
客户端收到响应
```

---

## 5. 核心模块详解

### 5.1 数据库层

#### XORM 引擎

Gitea 使用 XORM 作为 ORM 框架，支持多种数据库：

```go
// modules/db/engine.go

// Engine 定义数据库引擎接口
type Engine interface {
    Table(tableNameOrBean any) *xorm.Session
    Get(beans ...any) (bool, error)
    Find(any, ...any) error
    Insert(...any) (int64, error)
    Update(...any) (int64, error)
    Delete(...any) (int64, error)
    // ...
}

// 全局引擎实例
var x *xorm.Engine

// GetEngine 获取数据库引擎
func GetEngine(ctx context.Context) Engine {
    // 返回绑定到上下文的会话
}
```

#### 数据库迁移

```go
// models/migrations/v1_22.go

func AddRepositoryStarsTable(x *xorm.Engine) error {
    type RepositoryStar struct {
        RepoID    int64     `xorm:"UNIQUE(s)"`
        UserID    int64     `xorm:"UNIQUE(s)"`
        CreatedAt time.Time `xorm:"created"`
    }
    return x.Sync(new(RepositoryStar))
}
```

### 5.2 Git 操作层

#### 命令执行

```go
// modules/git/command.go

// 执行 Git 命令
cmd := NewCommand(ctx, "log", "--oneline")
cmd.AddArguments("--max-count=10")
stdout, _, err := cmd.RunStdString(&RunOpts{Dir: repoPath})

// 安全参数处理
cmd.AddDynamicArguments(userInput)  // 自动转义特殊字符
```

#### 仓库操作

```go
// modules/git/repository.go

// Repository 代表一个 Git 仓库
type Repository struct {
    Path string
}

// GetBranchCommit 获取分支的最新提交
func (repo *Repository) GetBranchCommit(branch string) (*Commit, error) {
    // git rev-parse refs/heads/{branch}
}

// GetCommit 获取指定提交
func (repo *Repository) GetCommit(commitID string) (*Commit, error) {
    // git cat-file -p {commitID}
}
```

### 5.3 配置管理

#### 配置文件结构

```ini
# app.ini

[server]
ROOT_URL = https://git.example.com/
HTTP_PORT = 3000
DOMAIN = git.example.com

[database]
DB_TYPE = mysql
HOST = 127.0.0.1:3306
NAME = gitea
USER = gitea
PASSWD = password

[repository]
ROOT = /data/gitea/repositories

[security]
SECRET_KEY = secret
INTERNAL_TOKEN = token
```

#### 配置加载

```go
// modules/setting/setting.go

var (
    Server   ServerSetting
    Database DatabaseSetting
    Repository RepositorySetting
)

type ServerSetting struct {
    RootURL   string
    HTTPPort  int
    Domain    string
}

func LoadSettings() {
    Cfg, _ := ini.Load("app.ini")
    Cfg.Section("server").MapTo(&Server)
    Cfg.Section("database").MapTo(&Database)
    Cfg.Section("repository").MapTo(&Repository)
}
```

### 5.4 认证与授权

#### 认证方式

1. **Session 认证** (Web UI)
2. **Token 认证** (API)
3. **Basic Auth** (Git HTTP)
4. **SSH Key** (Git SSH)
5. **OAuth2** (第三方登录)

#### 认证中间件

```go
// services/auth/middleware.go

func AuthMiddleware() func(ctx *context.Context) {
    return func(ctx *context.Context) {
        // 1. 尝试从 Session 获取用户
        user := getSessionUser(ctx)

        // 2. 尝试从 Token 获取用户
        if user == nil {
            user = getTokenUser(ctx)
        }

        // 3. 尝试从 Basic Auth 获取用户
        if user == nil {
            user = getBasicAuthUser(ctx)
        }

        // 4. 设置当前用户
        ctx.Doer = user
    }
}
```

#### 权限检查

```go
// models/perm/access/access.go

// CheckPermission 检查用户对仓库的权限
func CheckPermission(ctx context.Context, repo *Repository, user *User, perm AccessMode) error {
    // 1. 检查是否是所有者
    if repo.OwnerID == user.ID {
        return nil
    }

    // 2. 检查是否是协作者
    if isCollaborator(repo.ID, user.ID) {
        return nil
    }

    // 3. 检查组织权限
    if repo.Owner.IsOrganization() {
        return checkOrgPermission(repo.Owner, user, perm)
    }

    // 4. 检查公开仓库
    if !repo.IsPrivate {
        return nil
    }

    return ErrPermissionDenied
}
```

---

## 6. API 设计

### RESTful API

Gitea 提供完整的 RESTful API，遵循 OpenAPI 规范。

#### API 路由结构

```
/api/v1/
├── /repos                           # 仓库相关
│   ├── GET    /                     # 列出仓库
│   ├── POST   /                     # 创建仓库
│   ├── GET    /{owner}/{repo}       # 获取仓库
│   ├── PATCH  /{owner}/{repo}       # 更新仓库
│   └── DELETE /{owner}/{repo}       # 删除仓库
├── /users                           # 用户相关
├── /orgs                            # 组织相关
├── /admin                           # 管理员相关
└── /misc                            # 杂项
```

#### Swagger 注解

```go
// swagger:operation GET /repos/{owner}/{repo} repository repoGet
//
// ---
// summary: Get a repository
// parameters:
// - name: owner
//   in: path
//   description: owner of the repo
//   type: string
//   required: true
// - name: repo
//   in: path
//   description: name of the repo
//   type: string
//   required: true
// responses:
//   "200":
//     "$ref": "#/responses/Repository"
//   "404":
//     "$ref": "#/responses/notFound"
func GetRepo(ctx *context.APIContext) {
    // 获取仓库信息
    repo := ctx.Repo.Repository
    ctx.JSON(http.StatusOK, repo)
}
```

### API 响应格式

```go
// 成功响应
{
    "id": 1,
    "name": "my-repo",
    "full_name": "user/my-repo",
    "description": "My repository"
}

// 错误响应
{
    "message": "Not Found",
    "url": "https://git.example.com/swagger"
}
```

---

## 7. Web UI 架构

### 前端技术栈

| 技术         | 用途       |
| ------------ | ---------- |
| Vue.js 3     | 前端框架   |
| Fomantic UI  | UI 组件库  |
| Webpack      | 构建工具   |
| Tailwind CSS | CSS 工具类 |

### 前端目录结构

```
web_src/
├── css/                    # 样式文件
│   ├── themes/           # 主题
│   └── features/         # 功能样式
├── js/                     # JavaScript
│   ├── components/       # Vue 组件
│   ├── features/         # 功能模块
│   ├── app.js            # 应用入口
│   └── index.js          # 入口文件
└── svg/                    # SVG 图标
```

### Vue 组件示例

```vue
<!-- web_src/js/components/RepoCard.vue -->
<template>
  <div class="repo-card">
    <h3>{{ repo.full_name }}</h3>
    <p>{{ repo.description }}</p>
    <div class="stats">
      <span>⭐ {{ repo.stars_count }}</span>
      <span>🍴 {{ repo.forks_count }}</span>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    repo: {
      type: Object,
      required: true,
    },
  },
};
</script>
```

### 模板渲染

```go
// routers/web/repo/repo.go

func ViewRepo(ctx *context.Context) {
    // 设置模板数据
    ctx.Data["Repository"] = ctx.Repo.Repository
    ctx.Data["CommitCount"] = ctx.Repo.CommitsCount

    // 渲染模板
    ctx.HTML(http.StatusOK, tplRepoHome)
}
```

```html
<!-- templates/repo/home.tmpl -->
{{template "base/head" .}}
<div class="repository">
  <h1>{{.Repository.Name}}</h1>
  <p>{{.Repository.Description}}</p>
  <div class="stats">
    <span>Commits: {{.CommitCount}}</span>
  </div>
</div>
{{template "base/foot" .}}
```

---

## 8. 并发与异步处理

### Goroutine 管理

```go
// modules/graceful/manager.go

// Manager 管理优雅关闭
type Manager struct {
    ctx       context.Context
    cancel    context.CancelFunc
    running   sync.WaitGroup
}

// RunWithCancel 在 goroutine 中运行任务
func (m *Manager) RunWithCancel(fn func(ctx context.Context)) {
    m.running.Add(1)
    go func() {
        defer m.running.Done()
        fn(m.ctx)
    }()
}
```

### 队列系统

```go
// modules/queue/queue.go

// Queue 定义队列接口
type Queue interface {
    Push(items ...any) error
    Pop() (any, error)
    Close() error
}

// WorkerQueue 工作队列实现
type WorkerQueue struct {
    queue     Queue
    workers   int
    handler   func(items ...any)
}

// 启动工作队列
func (q *WorkerQueue) Start() {
    for i := 0; i < q.workers; i++ {
        go q.worker()
    }
}
```

### 定时任务

```go
// services/cron/cron.go

// 定时任务配置
var tasks = []struct {
    Name     string
    Schedule string
    Handler  func(ctx context.Context)
}{
    {"UpdateMirrors", "0 */1 * * *", updateMirrors},
    {"CleanupPackages", "0 0 * * *", cleanupPackages},
    {"SyncExternalUsers", "0 */6 * * *", syncExternalUsers},
}
```

---

## 9. 存储架构

### 存储抽象

```go
// modules/storage/storage.go

// ObjectStorage 定义对象存储接口
type ObjectStorage interface {
    Open(path string) (Object, error)
    Save(path string, r io.Reader, size int64) (int64, error)
    Stat(path string) (FileInfo, error)
    Delete(path string) error
    URL(path, name string, reqParams url.Values) (*url.URL, error)
    IterateObjects(path string, iterator func(path string, obj Object) error) error
}
```

### 存储实现

1. **本地存储** (默认)
2. **MinIO/S3**
3. **Azure Blob**
4. **Google Cloud Storage**
5. **腾讯云 COS**

### 存储配置

```ini
[storage]
STORAGE_TYPE = local
PATH = /data/gitea/attachments

[storage.minio]
STORAGE_TYPE = minio
MINIO_ENDPOINT = minio:9000
MINIO_ACCESS_KEY_ID = minio
MINIO_SECRET_ACCESS_KEY = minio123
MINIO_BUCKET = gitea
```

---

## 10. 测试架构

### 测试类型

1. **单元测试** - 测试单个函数/方法
2. **集成测试** - 测试模块间交互
3. **端到端测试** - 测试完整流程

### 测试框架

```go
// 使用 testify 进行断言
import "github.com/stretchr/testify/assert"

func TestCreateRepository(t *testing.T) {
    repo, err := CreateRepository(ctx, opts)
    assert.NoError(t, err)
    assert.NotNil(t, repo)
    assert.Equal(t, "test-repo", repo.Name)
}
```

### 表驱动测试

```go
func TestRepository_GetCommitBranches(t *testing.T) {
    tests := []struct {
        name     string
        commitID string
        want     []string
    }{
        {"main branch", "abc123", []string{"main"}},
        {"feature branch", "def456", []string{"feature"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            branches, err := repo.GetBranches(tt.commitID)
            assert.NoError(t, err)
            assert.Equal(t, tt.want, branches)
        })
    }
}
```

### 集成测试

```go
// tests/integration/repo_test.go

func TestCreateRepo(t *testing.T) {
    // 准备测试环境
    defer tests.PrepareTestEnv(t)()

    // 登录用户
    session := loginUser(t, "user1")

    // 创建仓库
    req := NewRequestWithValues(t, "POST", "/api/v1/repos", map[string]string{
        "name": "test-repo",
    })
    session.MakeRequest(t, req, http.StatusCreated)

    // 验证结果
    repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
        Name: "test-repo",
    })
    assert.NotNil(t, repo)
}
```

---

## 总结

Gitea 的架构设计遵循以下原则：

1. **分层清晰** - 每层职责明确，依赖单向
2. **模块化** - 功能模块独立，可复用
3. **可扩展** - 接口驱动，支持多种实现
4. **高性能** - 并发优化，缓存机制
5. **可测试** - 依赖注入，测试友好

这种架构使得 Gitea 易于维护、扩展和部署，是一个优秀的 Go 语言项目实践。
