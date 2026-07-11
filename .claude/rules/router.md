# routers 目录代码规范

## 目录概述

`routers/` 是 Gitea 的**路由层**，负责处理所有 HTTP 请求，是应用的入口点。它接收客户端请求，解析参数，调用 `services/` 层完成业务逻辑，并返回响应。

---

## 目录

- [1. 目录结构](#1-目录结构)
- [2. 整体架构与请求流程](#2-整体架构与请求流程)
- [3. 初始化机制](#3-初始化机制)
- [4. REST API 路由层 (api/)](#4-rest-api-路由层-api)
- [5. Web UI 路由层 (web/)](#5-web-ui-路由层-web)
- [6. 内部 API 路由层 (private/)](#6-内部-api-路由层-private)
- [7. 安装路由层 (install/)](#7-安装路由层-install)
- [8. 公共模块 (common/ 和 utils/)](#8-公共模块-common-和-utils)
- [9. 导入别名规范](#9-导入别名规范)
- [10. 中间件规范](#10-中间件规范)
- [11. 表单绑定规范](#11-表单绑定规范)
- [12. 错误处理规范](#12-错误处理规范)
- [13. 测试规范](#13-测试规范)
- [14. 总结：核心原则](#14-总结核心原则)

---

## 1. 目录结构

```
routers/
├── init.go                 # 顶层初始化入口，子路由挂载
├── api/                    # REST API 路由
│   └── v1/                # API v1 版本
│       ├── api.go         # 路由注册 + 中间件定义 + swagger:meta
│       ├── repo/          # 仓库相关 API
│       ├── user/          # 用户相关 API
│       ├── org/           # 组织相关 API
│       ├── admin/         # 管理员 API
│       ├── packages/      # 包管理 API
│       ├── misc/          # 杂项 API
│       ├── notify/        # 通知 API
│       ├── settings/      # 设置 API
│       ├── shared/        # 共享工具
│       ├── swagger/       # Swagger 响应类型定义
│       ├── activitypub/   # ActivityPub API
│       └── utils/         # API 工具函数
├── web/                    # Web UI 路由
│   ├── web.go             # 路由注册 + 全局中间件
│   ├── base.go            # 基础中间件
│   ├── repo/              # 仓库页面
│   ├── user/              # 用户页面
│   ├── org/               # 组织页面
│   ├── admin/             # 管理后台
│   ├── auth/              # 认证页面
│   ├── explore/           # 探索页面
│   ├── feed/              # 活动动态
│   ├── misc/              # 杂项页面
│   ├── shared/            # 共享组件
│   ├── events/            # SSE 事件流
│   ├── healthcheck/       # 健康检查
│   ├── home.go            # 首页处理
│   ├── githttp.go         # Git HTTP 协议
│   ├── goget.go           # go get 支持
│   └── ...                # 其他顶层处理文件
├── private/                # 内部 API（SSH、Git Hooks 等）
│   ├── internal.go        # 路由注册 + authInternal 中间件
│   ├── hook_pre_receive.go
│   ├── hook_post_receive.go
│   ├── hook_proc_receive.go
│   ├── key.go             # SSH 密钥管理
│   ├── serv.go            # Git 服务命令
│   ├── manager.go         # 进程管理
│   └── ...
├── install/                # 安装向导
│   ├── routes.go          # 路由注册
│   └── install.go         # 处理逻辑 + Contexter
├── common/                 # 跨路由层共享代码
│   ├── middleware.go      # ProtocolMiddlewares、Sessioner
│   ├── auth.go            # 认证结果、VerifyOptions
│   ├── errpage.go         # 500 错误页面渲染
│   ├── serve.go           # Blob/内容服务
│   ├── redirect.go        # Fetch 重定向委托
│   ├── db.go              # 数据库初始化
│   └── markup.go          # Markup 渲染辅助
└── utils/                  # 路由层工具函数
    └── utils.go           # SanitizeFlashErrorString
```

---

## 2. 整体架构与请求流程

### 路由挂载关系

```
routers/init.go: NormalRoutes()
├── "/"                  → web.Routes()        (Web UI)
├── "/api/v1"            → apiv1.Routes()      (REST API)
├── "/api/internal"      → private.Routes()    (内部 API)
├── "/api/packages"      → packages.CommonRoutes()  (条件挂载)
├── "/v2"                → packages.ContainerRoutes() (条件挂载)
└── "/api/actions"       → actions.Routes()    (条件挂载)
```

### 请求处理流程

```
HTTP 请求
    ↓
routers/init.go: NormalRoutes()
    ↓
common.ProtocolMiddlewares()     ← 全局中间件（panic恢复、日志、代理头等）
    ↓
子路由中间件链
    ├── API:    securityHeaders → CORS → APIContexter → auth → verifyAuth
    ├── Web:    Sessioner → Contexter → webAuth → goGet
    └── Private: PrivateContexter → authInternal → RealIP
    ↓
路由处理器 (Handler)
    ↓
调用 services/ 层业务逻辑
    ↓
返回响应 (HTML / JSON / 重定向)
```

### 跨层调用规则

```
routers/ → services/ → models/
    ↓
  modules/
```

- `routers/` 可调用 `services/` 和 `modules/`
- `routers/` **不应**调用 `models/` 直接进行业务操作（读取简单数据可接受）
- `routers/` **不应**被其他层调用

---

## 3. 初始化机制

### 初始化入口

`routers/init.go` 提供两个初始化函数：

```go
// 轻量级初始化（安装页面，无数据库）
func InitWebInstallPage(ctx context.Context)

// 完整初始化（已安装状态，按序启动约 20 个子系统）
func InitWebInstalled(ctx context.Context)
```

### 初始化顺序（InitWebInstalled）

按以下顺序初始化，顺序不可随意调整：

1. `git.InitFull` — Git 操作层
2. `translation.InitLocales` — 国际化
3. `setting.LoadSettings` — 配置加载
4. `storage.Init` — 存储层
5. `mailer.NewContext` — 邮件服务
6. `cache.Init` — 缓存
7. `feed_service.Init` / `uinotification.Init` — 动态与通知
8. `archiver.Init` — 归档服务
9. `highlight.NewContext` / `markup.Init` — Markdown 渲染
10. `common.InitDBEngine` — 数据库连接
11. `system.Init` / `oauth2.Init` / `release_service.Init`
12. `models.Init` / `repo_service.Init` — 模型与仓库服务
13. `indexer_service.Init` — 全文索引
14. Mirror / Webhook / Pull / Automerge / Task / Migrations — 各业务服务
15. `ssh.Init` — SSH 服务
16. `auth.Init` — 认证
17. `actions_service.Init` — CI/CD
18. `cron.NewContext(ctx)` — 定时任务（最后启动）

### mustInit 辅助函数

```go
// 初始化失败则 Fatal，自动记录失败函数名
func mustInit(fn func() error)
func mustInitCtx(ctx context.Context, fn func(ctx context.Context) error)
```

---

## 4. REST API 路由层 (api/)

### 路由注册

所有 API 路由在 `routers/api/v1/api.go` 的 `Routes()` 函数中注册：

```go
func Routes() *web.Route {
    m := web.NewRoute()

    // 全局中间件
    m.Use(securityHeaders())
    m.Use(cors.Handler(cors.Options{...}))
    m.Use(context.APIContexter())
    m.Use(checkDeprecatedAuthMethods)
    m.Use(apiAuth(buildAuthGroup()))
    m.Use(verifyAuthWithOptions(&common.VerifyOptions{...}))

    m.Group("", func() {
        m.Get("/version", misc.Version)
        m.Group("/repos", func() {
            m.Get("", repo.Search)
            m.Post("", bind(api.CreateRepoOption{}), repo.Create)
            m.Get("/{username}/{reponame}", repo.Get)
        }, context.UserAssignmentAPI())
    })
    return r
}
```

### 路由注册模式

| 模式 | 语法 | 说明 |
|------|------|------|
| 单方法 | `m.Get("/path", handler)` | GET 请求 |
| 多方法 | `m.Combo("/path").Get(h1).Post(h2)` | 同路径不同方法 |
| 路由组 | `m.Group("/prefix", func(){...}, middleware...)` | 前缀分组 + 尾部中间件 |
| 表单绑定 | `m.Post("/path", web.Bind(T{}), handler)` | 自动绑定请求体 |

### 处理器签名

所有 API 处理器统一使用 `*context.APIContext`：

```go
func HandlerName(ctx *context.APIContext)
```

`APIContext` 核心字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `ctx.Doer` | `*user_model.User` | 当前认证用户 |
| `ctx.IsSigned` | `bool` | 是否已登录 |
| `ctx.IsBasicAuth` | `bool` | 是否 Basic Auth |
| `ctx.ContextUser` | `*user_model.User` | 被访问的用户 |
| `ctx.Repo` | `*Repository` | 当前仓库上下文 |
| `ctx.Org` | `*APIOrganization` | 当前组织上下文 |
| `ctx.PublicOnly` | `bool` | 是否仅公开数据 |

### Swagger 注解

每个 API 处理器函数体内必须包含 Swagger 注解：

```go
func GetRepo(ctx *context.APIContext) {
    // swagger:operation GET /repos/{owner}/{repo} repository repoGet
    // ---
    // summary: Get a repository
    // produces:
    // - application/json
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

    repo := ctx.Repo.Repository
    ctx.JSON(http.StatusOK, repo)
}
```

### 常用 Context 方法

```go
// 读取参数
ctx.FormString("q")          // 查询参数/表单
ctx.FormBool("private")      // 布尔参数
ctx.FormInt64("uid")         // 整数参数
ctx.FormTrim("q")            // 去空白
ctx.Params("username")       // URL 路径参数

// 分页
utils.GetListOptions(ctx)    // 返回 db.ListOptions{Page, PageSize}
ctx.SetTotalCountHeader(count)
ctx.SetLinkHeader(total, pageSize)

// 响应
ctx.JSON(http.StatusOK, data)
ctx.Error(http.StatusBadRequest, "message", err)
ctx.NotFound()
ctx.InternalServerError(err)
```

### 中间件模式

```go
// 闭包式中间件（需要配置时）
func reqToken() func(ctx *context.APIContext) {
    return func(ctx *context.APIContext) {
        if ctx.IsSigned {
            return
        }
        ctx.Error(http.StatusUnauthorized, "reqToken", "token is required")
    }
}

// 直接式中间件（无配置）
func mustEnableIssues(ctx *context.APIContext) {
    if !ctx.Repo.CanRead(unit.TypeIssues) {
        ctx.NotFound()
        return
    }
}
```

---

## 5. Web UI 路由层 (web/)

### 路由注册

Web 路由在 `routers/web/web.go` 中分两阶段注册：

**阶段 1：`Routes()` 构建中间件栈**

```go
func Routes() *web.Route {
    routes := web.NewRoute()
    // 静态资源、gzip、验证码、指标...
    mid = append(mid, common.Sessioner(), context.Contexter())
    mid = append(mid, webAuth(buildAuthGroup()))
    mid = append(mid, chi_middleware.GetHead)
    mid = append(mid, user.GetNotificationCount)
    mid = append(mid, repo.GetActiveStopwatch)
    mid = append(mid, goGet)

    others := web.NewRoute()
    others.Use(mid...)
    registerRoutes(others)
    routes.Mount("", others)
    return routes
}
```

**阶段 2：`registerRoutes()` 注册所有端点**

```go
func registerRoutes(m *web.Route) {
    // 简单路由
    m.Get("/", Home)

    // 带中间件链
    m.Get("/sitemap.xml", sitemapEnabled, ignExploreSignIn, HomeSitemap)

    // 路由组 + 嵌套
    m.Group("/{username}/{reponame}", func() {
        m.Get("", repo.Home)
        m.Get("/issues", repo.Issues)
        m.Group("/settings", func() {
            m.Get("", repo_setting.Settings)
            m.Post("", web.Bind(forms.RepoSettingForm{}), repo_setting.SettingsPost)
        })
    }, ignSignIn, context.RepoAssignment)

    // Combo（同路径 GET + POST）
    m.Combo("/compare/*").
        Get(repo.SetDiffViewStyle, repo.CompareDiff).
        Post(reqSignIn, reqToken, bind(forms.CreateIssueForm{}), repo.CompareAndPullRequestPost)
}
```

### 路由注册模式

| 模式 | 语法 | 说明 |
|------|------|------|
| 简单路由 | `m.Get("/path", handler)` | 单个处理器 |
| 中间件链 | `m.Get("/path", mid1, mid2, handler)` | 多个前置中间件 |
| 路由组 | `m.Group("/prefix", func(){...}, mid...)` | 前缀分组 + 尾部中间件 |
| Combo | `m.Combo("/path").Get(h1).Post(h2)` | 同路径多方法 |
| 表单绑定 | `m.Post("/path", web.Bind(T{}), handler)` | 自动绑定表单 |
| 方法指定 | `m.Methods("GET, HEAD", "/assets/*", handler)` | 指定 HTTP 方法 |

### 处理器签名

所有 Web 处理器统一使用 `*context.Context`：

```go
func HandlerName(ctx *context.Context)
```

> **注意**：这里的 `context` 是 `code.gitea.io/gitea/services/context`，不是标准库的 `context.Context`。

`Context` 核心字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `ctx.Doer` | `*user_model.User` | 当前登录用户 |
| `ctx.IsSigned` | `bool` | 是否已登录 |
| `ctx.Repo` | `*Repository` | 当前仓库上下文 |
| `ctx.Org` | `*Organization` | 当前组织上下文 |
| `ctx.Csrf` | `CSRFProtector` | CSRF 保护 |
| `ctx.Flash` | `*middleware.Flash` | Flash 消息 |
| `ctx.Session` | `session.Store` | Session 存储 |
| `ctx.Locale` | `Locale` | 国际化 |
| `ctx.Data` | `ContextData` | 模板数据 map |

### 模板渲染

```go
// 1. 定义模板常量（文件顶部）
const (
    tplRepoHome  base.TplName = "repo/home"
    tplWatchers  base.TplName = "repo/watchers"
    tplForks     base.TplName = "repo/forks"
)

// 2. 填充模板数据
ctx.Data["Repository"] = repo
ctx.Data["Page"] = pager
ctx.Data["Cards"] = items

// 3. 渲染模板
ctx.HTML(http.StatusOK, tplRepoHome)
```

### 常用 Context 方法

```go
// 读取参数
ctx.FormInt("page")
ctx.FormString("q")
ctx.FormBool("private")
ctx.Params(":reponame")       // URL 路径参数（注意冒号前缀）

// 模板数据
ctx.Data["Key"] = value

// 国际化
ctx.Tr("repo.settings")

// 响应
ctx.HTML(http.StatusOK, tplName)
ctx.JSON(http.StatusOK, data)
ctx.Redirect(url)
ctx.ServerError("description", err)   // 500 错误页
ctx.NotFound("description", err)      // 404 错误页
ctx.RenderWithErr(message, tpl, form) // 带错误消息的表单回显

// 状态检查
ctx.Written()    // 响应是否已发送（防止重复写入）
```

### 典型处理器流程

```go
func Issues(ctx *context.Context) {
    // 1. 读取输入
    page := ctx.FormInt("page")
    if page <= 0 {
        page = 1
    }

    // 2. 调用服务层
    issues, err := issue_model.Issues(ctx, &issue_model.IssuesOptions{...})
    if err != nil {
        ctx.ServerError("Issues", err)
        return
    }

    // 3. 检查子调用是否已写入响应
    if ctx.Written() {
        return
    }

    // 4. 填充模板数据
    ctx.Data["Issues"] = issues
    ctx.Data["Page"] = context.NewPagination(total, setting.ItemsPerPage, page, 5)

    // 5. 渲染模板
    ctx.HTML(http.StatusOK, tplIssues)
}
```

---

## 6. 内部 API 路由层 (private/)

### 路由注册

所有内部 API 路由在 `routers/private/internal.go` 的 `Routes()` 函数中注册：

```go
func Routes() *web.Route {
    r := web.NewRoute()
    r.Use(context.PrivateContexter())
    r.Use(authInternal)
    r.Use(chi_middleware.RealIP)

    r.Post("/ssh/authorized_keys", AuthorizedPublicKeyByContent)
    r.Post("/hook/pre-receive/{owner}/{repo}", RepoAssignment, bind(private.HookOptions{}), HookPreReceive)
    r.Post("/hook/post-receive/{owner}/{repo}", context.OverrideContext, bind(private.HookOptions{}), HookPostReceive)
    r.Get("/serv/command/{keyid}/{owner}/{repo}", ServCommand)
    r.Post("/manager/shutdown", Shutdown)
    // ...
    return r
}
```

### 认证机制

内部 API 使用 `authInternal` 中间件，验证 `X-Gitea-Internal-Auth` 请求头中的 Bearer Token：

```go
func authInternal(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        token := req.Header.Get("X-Gitea-Internal-Auth")
        if subtle.ConstantTimeCompare([]byte(token), []byte(setting.InternalToken)) != 1 {
            // 拒绝访问
            return
        }
        next.ServeHTTP(w, req)
    })
}
```

### 处理器签名

```go
func HandlerName(ctx *context.PrivateContext)
```

### 泛型绑定辅助函数

```go
// 传入零值，中间件创建新实例并绑定请求体
func bind[T any](_ T) any {
    return func(ctx *context.PrivateContext) {
        theObj := new(T)
        binding.Bind(ctx.Req, theObj)
        web.SetForm(ctx, theObj)
    }
}

// 在处理器中获取绑定对象
opts := web.GetForm(ctx).(*private.HookOptions)
```

### 响应格式

```go
// JSON 响应（错误）
ctx.JSON(http.StatusInternalServerError, private.Response{
    Err: err.Error(),
})

// JSON 响应（成功）
ctx.JSON(http.StatusOK, &results)

// 纯文本响应
ctx.PlainText(http.StatusOK, "success")
```

`private.Response` 结构体：

| 字段 | 说明 |
|------|------|
| `Err` | 内部/技术错误消息 |
| `UserMsg` | 用户可见的错误消息 |

### 路由特点

- 绝大多数为 `POST` 方法，仅少量只读端点使用 `GET`
- 路径参数使用 `{name}` 语法（chi 风格）
- 中间件可链式内联：`RepoAssignment, bind(private.HookOptions{}), HookPreReceive`

---

## 7. 安装路由层 (install/)

### 路由注册

```go
func Routes() *web.Route {
    base := web.NewRoute()
    base.Use(common.ProtocolMiddlewares()...)
    base.Methods("GET, HEAD", "/assets/*", public.FileHandlerFunc())

    r := web.NewRoute()
    r.Use(common.Sessioner(), Contexter())
    r.Get("/", Install)
    r.Post("/", web.Bind(forms.InstallForm{}), SubmitInstall)
    r.Get("/post-install", InstallDone)
    r.Get("/api/healthz", healthcheck.Check)
    r.NotFound(installNotFound)

    base.Mount("", r)
    return base
}
```

### 两层路由结构

- `base` 层：处理协议中间件和静态资源
- `r` 层：处理安装特定路由（Session + Contexter）

### Contexter 中间件

安装路由使用独立的 `Contexter` 中间件，创建轻量级 `WebContext`：

```go
func Contexter() func(next http.Handler) http.Handler {
    rnd := templates.HTMLRenderer()
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
            base, baseCleanUp := context.NewBaseContext(resp, req)
            defer baseCleanUp()

            ctx := context.NewWebContext(base, rnd, session.GetSession(req))
            ctx.Data.MergeFrom(middleware.CommonTemplateContextData())
            ctx.Data.MergeFrom(middleware.ContextData{
                "PageIsInstall": true,
                // ...
            })
            next.ServeHTTP(resp, ctx.Req)
        })
    }
}
```

---

## 8. 公共模块 (common/ 和 utils/)

### routers/common/

跨路由层共享的代码，主要包含：

| 文件 | 导出函数/类型 | 说明 |
|------|--------------|------|
| `middleware.go` | `ProtocolMiddlewares()` | 全局协议中间件链（panic 恢复、日志、代理头等） |
| `middleware.go` | `Sessioner()` | Session 中间件工厂 |
| `auth.go` | `AuthShared()` | 共享认证验证逻辑 |
| `auth.go` | `VerifyOptions` | 认证选项（SignInRequired、AdminRequired 等） |
| `errpage.go` | `RenderPanicErrorPage()` | 500 错误页面渲染（含双重 panic 保护） |
| `serve.go` | `ServeBlob()` / `ServeContentByReader()` | 内容服务（ETag/缓存支持） |
| `redirect.go` | `FetchRedirectDelegate()` | 前端 fetch 重定向委托 |
| `db.go` | `InitDBEngine()` | 数据库引擎初始化 |
| `markup.go` | Markup 相关辅助 | Markdown 渲染辅助 |

### ProtocolMiddlewares 中间件链

```go
func ProtocolMiddlewares() (handlers []any) {
    // 1. stripSlashes — URL 路径规范化（合并连续斜杠）
    // 2. panic recovery + ContextData 初始化
    // 3. 进程管理器注册
    // 4. 反向代理转发头（条件：setting.ReverseProxyLimit > 0）
    // 5. 路由日志（条件：setting.IsRouteLogEnabled()）
    // 6. 访问日志（条件：setting.IsAccessLogEnabled()）
}
```

### routers/utils/

仅包含一个工具函数：

```go
// SanitizeFlashErrorString HTML 转义并将换行替换为 <br>
func SanitizeFlashErrorString(x string) string
```

---

## 9. 导入别名规范

### 标准别名规则

| 导入路径 | 别名 | 命名规则 |
|----------|------|----------|
| `models/*` | `{domain}_model` | `repo_model`, `user_model`, `issue_model` |
| `services/*` | `{domain}_service` | `repo_service`, `issue_service`, `files_service` |
| `modules/repository` | `repo_module` | 避免与 `repo_model`/`repo_service` 冲突 |
| `modules/structs` | `api` | API 响应结构体 |
| `services/context` | `context` | 无需别名（Web/API 专用） |
| `context` (标准库) | `gocontext` | 避免与 `services/context` 冲突 |

### 导入分组

```go
import (
    // 1. 标准库
    "fmt"
    "net/http"

    // 2. models 层
    repo_model "code.gitea.io/gitea/models/repo"
    user_model "code.gitea.io/gitea/models/user"

    // 3. modules 层
    "code.gitea.io/gitea/modules/log"
    "code.gitea.io/gitea/modules/setting"
    api "code.gitea.io/gitea/modules/structs"
    "code.gitea.io/gitea/modules/web"

    // 4. services 层
    "code.gitea.io/gitea/services/auth"
    "code.gitea.io/gitea/services/context"
    repo_service "code.gitea.io/gitea/services/repository"

    // 5. routers 层内部
    "code.gitea.io/gitea/routers/common"

    // 6. 第三方库
    "gitea.com/go-chi/binding"
    "github.com/go-chi/cors"

    // 7. 空导入（副作用）
    _ "code.gitea.io/gitea/routers/api/v1/swagger"
)
```

### 冲突解决

当标准库 `context` 和 `services/context` 同时需要时：

```go
import (
    gocontext "context"  // 标准库使用别名
    "code.gitea.io/gitea/services/context"  // Gitea context 无别名
)
```

---

## 10. 中间件规范

### 中间件签名

中间件必须匹配以下签名之一：

```go
// 直接式（无配置）
func MyMiddleware(ctx *context.APIContext)

// 闭包式（需要配置）
func MyMiddleware(cfg Config) func(ctx *context.APIContext) {
    return func(ctx *context.APIContext) {
        // ...
    }
}

// HTTP Handler 式（用于 common/ 包）
func MyMiddleware(next http.Handler) http.Handler
```

### 中间件职责

| 类型 | 示例 | 说明 |
|------|------|------|
| 认证 | `reqToken()`, `authInternal` | 验证用户身份 |
| 授权 | `reqOwner()`, `reqAdmin()` | 检查权限 |
| 仓库 | `context.RepoAssignment`, `RepoAssignment` | 加载仓库上下文 |
| 组织 | `context.OrgAssignment` | 加载组织上下文 |
| 功能开关 | `mustEnableIssues`, `mustAllowPulls` | 检查功能是否启用 |
| 表单绑定 | `web.Bind(T{})` | 绑定请求体到结构体 |
| 安全 | `securityHeaders()`, `cors.Handler()` | 安全头、CORS |

### Guard 模式

Guard 函数检查前置条件，不满足时提前返回：

```go
func MustBeNotEmpty(ctx *context.Context) {
    if ctx.Repo.Repository.IsEmpty {
        ctx.NotFound("MustBeNotEmpty", nil)
        return
    }
}

func MustBeEditable(ctx *context.Context) {
    if !ctx.Repo.Permission.CanWrite(unit.TypeCode) {
        ctx.NotFound("MustBeEditable", nil)
        return
    }
}
```

Guard 函数可直接作为路由中间件使用：

```go
m.Get("/issues", MustBeNotEmpty, repo.Issues)
```

---

## 11. 表单绑定规范

### 绑定方式

```go
// 路由注册时绑定
m.Post("/create", web.Bind(forms.CreateRepoForm{}), repo.CreatePost)

// private API 使用泛型 bind
r.Post("/hook/pre-receive/{owner}/{repo}",
    RepoAssignment,
    bind(private.HookOptions{}),
    HookPreReceive,
)
```

### 获取绑定对象

```go
// Web UI
form := web.GetForm(ctx).(*forms.CreateRepoForm)

// Private API
opts := web.GetForm(ctx).(*private.HookOptions)
```

### 表单验证

表单结构体使用 `binding` 标签定义验证规则：

```go
type CreateRepoForm struct {
    RepoName    string `binding:"Required;AlphaDashDot;MaxSize(100)"`
    Description string `binding:"MaxSize(2048)"`
    Private     bool
}
```

---

## 12. 错误处理规范

### API 错误响应

```go
// 标准错误
ctx.Error(http.StatusUnprocessableEntity, "message", err)

// 特定错误
ctx.NotFound()
ctx.InternalServerError(err)
ctx.Error(http.StatusForbidden, "message", "not allowed")
```

### Web 错误响应

```go
// 500 错误页
ctx.ServerError("functionName", err)

// 404 错误页
ctx.NotFound("functionName", err)

// 带错误消息的表单回显
ctx.RenderWithErr(err.Error(), tplCreate, form)

// Flash 消息（下次请求显示）
ctx.Flash.Error(err.Error())
ctx.Redirect(url)
```

### 错误处理原则

- 处理器不返回 error，错误通过 Context 方法直接写入响应
- 调用子函数后检查 `ctx.Written()` 防止重复写入
- 日志使用 `log.Error("FunctionName: %v", err)` 格式
- 不要忽略错误返回值（除了明确的 `defer Close()`）

---

## 13. 测试规范

### 测试文件位置

- 路由层测试：`tests/integration/` 目录
- 与源码同目录的 `*_test.go` 用于单元测试

### 集成测试模式

```go
func TestCreateRepo(t *testing.T) {
    defer tests.PrepareTestEnv(t)()

    session := loginUser(t, "user1")

    req := NewRequestWithValues(t, "POST", "/api/v1/repos", map[string]string{
        "name": "test-repo",
    })
    session.MakeRequest(t, req, http.StatusCreated)

    repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
        Name: "test-repo",
    })
    assert.NotNil(t, repo)
}
```

---

## 14. 总结：核心原则

| 原则 | 说明 |
|------|------|
| 单一入口 | 每个子路由层有唯一的 `Routes()` 函数注册所有路由 |
| 统一签名 | API 用 `*context.APIContext`，Web 用 `*context.Context`，Private 用 `*context.PrivateContext` |
| 中间件链 | 路由组尾部中间件 → 路由级中间件 → 处理器 |
| 表单绑定 | 使用 `web.Bind(T{})` 或泛型 `bind[T]()` 自动绑定 |
| 不返回错误 | 处理器通过 Context 方法直接写入错误响应 |
| 检查 Written | 调用子函数后检查 `ctx.Written()` 防止重复写入 |
| 导入别名 | `*_model`、`*_service`、`*_module`、`api` |
| Swagger 注解 | API 处理器必须包含完整的 Swagger 注解 |
| 模板常量 | Web 处理器使用 `base.TplName` 常量定义模板名 |
