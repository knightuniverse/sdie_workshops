# services 目录代码规范

## 目录概览

`services/` 是 Gitea 的**业务逻辑层**，位于路由层（`routers/`）和数据模型层（`models/`）之间。它封装了核心业务操作，供路由器和内部 API 调用。

## 子包职责划分

| 子包               | 职责                                     |
| ------------------ | ---------------------------------------- |
| `actions/`         | GitHub Actions 工作流执行                |
| `agit/`            | AGit 流程支持（无分支 PR）               |
| `asymkey/`         | SSH/GPG 密钥管理                         |
| `attachment/`      | 附件上传与管理                           |
| `auth/`            | 认证中间件（Session、Basic Auth、OAuth） |
| `automerge/`       | 自动合并 PR                              |
| `context/`         | HTTP 请求上下文封装                      |
| `convert/`         | 模型对象 → API 响应对象转换              |
| `cron/`            | 定时任务管理                             |
| `doctor/`          | 系统健康检查                             |
| `externalaccount/` | 外部账号关联                             |
| `feed/`            | 活动动态（Activity Feed）                |
| `forms/`           | Web 表单验证结构体                       |
| `gitdiff/`         | Diff 解析与高亮                          |
| `indexer/`         | 代码/问题全文索引                        |
| `issue/`           | Issue 与 PR 业务逻辑                     |
| `lfs/`             | Git LFS 管理                             |
| `mailer/`          | 邮件发送                                 |
| `markup/`          | Markdown/Wiki 渲染                       |
| `migrations/`      | 从其他平台迁移仓库                       |
| `mirror/`          | 镜像仓库同步                             |
| `notify/`          | 事件通知分发框架                         |
| `org/`             | 组织管理                                 |
| `packages/`        | 包注册中心                               |
| `pull/`            | Pull Request 核心逻辑                    |
| `release/`         | 发布管理                                 |
| `repository/`      | 仓库生命周期管理                         |
| `secrets/`         | Actions 密钥管理                         |
| `task/`            | 后台任务管理                             |
| `uinotification/`  | UI 通知                                  |
| `user/`            | 用户管理                                 |
| `webhook/`         | Webhook 分发与投递                       |
| `webtheme/`        | Web 主题管理                             |
| `wiki/`            | Wiki 页面管理                            |

---

## 1. 文件组织规范

### 1.1 包命名

- 包名与目录名一致，使用简短小写单词，不使用下划线
- 子包按功能领域划分，每个子包是一个独立的 Go 包

```go
// 正确
package pull
package webhook
package convert

// 错误
package pull_service      // 不使用下划线
package PullService       // 不使用大写
```

### 1.2 文件结构

每个包通常包含以下文件：

```
services/issue/
├── issue.go          # 主要业务逻辑（导出函数）
├── issue_comment.go  # Issue 评论相关逻辑
├── assignee.go       # 分配人相关逻辑
├── _test.go          # 单元测试
```

### 1.3 文件头声明

每个 `.go` 文件必须包含版权和许可证头：

```go
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue
```

### 1.4 Import 分组

Import 按组排列，用空行分隔：

```go
import (
    // 标准库
    "context"
    "fmt"

    // 项目 models 层
    "code.gitea.io/gitea/models/db"
    issues_model "code.gitea.io/gitea/models/issues"
    repo_model "code.gitea.io/gitea/models/repo"
    user_model "code.gitea.io/gitea/models/user"

    // 项目 modules 层
    "code.gitea.io/gitea/modules/log"
    "code.gitea.io/gitea/modules/git"

    // 项目 services 层（跨服务调用）
    notify_service "code.gitea.io/gitea/services/notify"
    issue_service "code.gitea.io/gitea/services/issue"
)
```

---

## 2. 命名规范

### 2.1 包别名（Import Alias）

当导入路径与包名不一致，或存在同名包冲突时，使用下划线风格的别名：

```go
import (
    issues_model "code.gitea.io/gitea/models/issues"
    repo_model   "code.gitea.io/gitea/models/repo"
    user_model   "code.gitea.io/gitea/models/user"
    git_model    "code.gitea.io/gitea/models/git"
    access_model "code.gitea.io/gitea/models/perm/access"

    repo_module "code.gitea.io/gitea/modules/repository"
    repo_service "code.gitea.io/gitea/services/repository"
    notify_service "code.gitea.io/gitea/services/notify"
)
```

**命名规则**：

- `models/` 层：`{功能}_model`（如 `issues_model`、`repo_model`）
- `modules/` 层：`{功能}_module`（如 `repo_module`）
- `services/` 层：`{功能}_service`（如 `repo_service`、`notify_service`）
- `modules/structs` 统一别名为 `api`

### 2.2 函数命名

- **导出函数**：使用 PascalCase，动词开头，表达明确的业务动作

```go
func CreateRepository(ctx context.Context, doer, owner *user_model.User, opts CreateRepoOptions) (*repo_model.Repository, error)
func DeleteUser(ctx context.Context, u *user_model.User, purge bool) error
func ChangeTitle(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, title string) error
func NewPullRequest(ctx context.Context, repo *repo_model.Repository, ...) error
```

- **非导出函数**：使用 camelCase，作为内部辅助函数

```go
func deleteIssue(ctx context.Context, issue *issues_model.Issue) error
func createTag(ctx context.Context, gitRepo *git.Repository, rel *repo_model.Release, msg string) (bool, error)
func handleSignIn(resp http.ResponseWriter, req *http.Request, sess SessionStore, user *user_model.User)
```

### 2.3 结构体命名

- 业务参数结构体：使用 `XxxOptions` 或 `XxxInfo` 后缀

```go
type PackageInfo struct {
    Owner       *user_model.User
    PackageType packages_model.Type
    Name        string
    Version     string
}

type PackageCreationInfo struct {
    PackageInfo
    SemverCompatible bool
    Creator          *user_model.User
    Metadata         any
}
```

- 表单验证结构体：使用 `XxxForm` 后缀

```go
type CreateRepoForm struct {
    UID           int64  `binding:"Required"`
    RepoName      string `binding:"Required;AlphaDashDot;MaxSize(100)"`
    Private       bool
    Description   string `binding:"MaxSize(2048)"`
}
```

### 2.4 错误变量命名

- 使用 `Err` 前缀

```go
var ErrRepoNotCreated = errors.New("repository is not created yet")

var (
    ErrQuotaTypeSize   = errors.New("maximum allowed package type size exceeded")
    ErrQuotaTotalSize  = errors.New("maximum allowed package storage quota exceeded")
    ErrQuotaTotalCount = errors.New("maximum allowed package count exceeded")
)
```

- 错误判断函数使用 `IsErr` 前缀

```go
func IsRateLimitError(err error) bool {
    _, ok := err.(*github.RateLimitError)
    return ok
}
```

---

## 3. 函数签名规范

### 3.1 context.Context 参数

所有业务函数**必须**接收 `context.Context` 作为第一个参数：

```go
func CreateRepository(ctx context.Context, doer, owner *user_model.User, opts CreateRepoOptions) (*repo_model.Repository, error)
func DeleteIssue(ctx context.Context, doer *user_model.User, gitRepo *git.Repository, issue *issues_model.Issue) error
```

### 3.2 doer 参数

执行操作的用户通常命名为 `doer`，放在 `ctx` 之后：

```go
func DeleteRepository(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, notify bool) error
func ChangeTitle(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, title string) error
```

### 3.3 返回值

- 错误作为最后一个返回值
- 使用具名返回值时仅在必要时（如 defer/recover 场景）

```go
func UpdateRepository(ctx context.Context, repo *repo_model.Repository, visibilityChanged bool) (err error)
```

---

## 4. 错误处理规范

### 4.1 错误包装

使用 `fmt.Errorf` 配合 `%w` 动词包装错误，保留错误链：

```go
if err = util.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
    return fmt.Errorf("rename user directory: %w", err)
}

return fmt.Errorf("GetRepositoryCount: %w", err)
```

### 4.2 错误传播

- 不要忽略错误返回值（除了明确的 `defer Close()`）
- 使用 `_` 显式忽略必须调用但错误可忽略的函数

```go
defer committer.Close()

_ = system_model.CreateNotice(ctx, system_model.NoticeTask, fmt.Sprintf("delete user '%s': %v", u.Name, err))
```

### 4.3 自定义错误类型

在 `models/` 层定义错误类型，`services/` 层使用并返回：

```go
// models 层定义
type ErrProtectedTagName struct {
    TagName string
}

// services 层使用
if !isAllowed {
    return false, models.ErrProtectedTagName{
        TagName: rel.TagName,
    }
}
```

### 4.4 错误日志

- 使用 `log.Error` 记录不可恢复的错误
- 使用 `log.Warn` 记录可跳过的警告
- 使用 `log.Trace` 记录调试信息
- 日志格式：`"FunctionName: %v"` 或 `"描述: %v"`

```go
log.Error("PushToBaseRepo: %v", err)
log.Warn("Cannot test PR %s/%d: head_branch %s no longer exists", pr.BaseRepo.Name, pr.IssueID, pr.HeadBranch)
log.Trace("AddTestPullRequestTask [head_repo_id: %d, head_branch: %s]: finding pull requests", repoID, branch)
```

---

## 5. 数据库事务规范

### 5.1 使用 db.TxContext

标准事务模式：

```go
ctx, committer, err := db.TxContext(ctx)
if err != nil {
    return err
}
defer committer.Close()

// ... 业务逻辑 ...

return committer.Commit()
```

### 5.2 使用 db.WithTx

对于不需要手动控制 commit 的场景，使用闭包形式：

```go
if err := db.WithTx(ctx, func(ctx context.Context) error {
    if err := issues_model.NewIssue(ctx, repo, issue, labelIDs, uuids); err != nil {
        return err
    }
    // ... 更多操作 ...
    return nil
}); err != nil {
    return err
}
```

### 5.3 事务边界

- 事务应尽可能小，只包含需要原子性的操作
- 耗时操作（如文件系统操作、Git 推送）应放在事务外
- 事务失败时需要回滚文件系统操作（如有）

```go
// 事务内：数据库操作
if err := committer.Commit(); err != nil {
    u.Name = oldUserName
    // 回滚文件系统操作
    if err2 := util.Rename(newPath, oldPath); err2 != nil {
        log.Critical("Unable to rollback directory change...")
    }
    return err
}
```

---

## 6. 通知机制规范

### 6.1 Notifier 接口

事件通知通过 `services/notify` 包的 `Notifier` 接口实现：

```go
type Notifier interface {
    Run()
    CreateRepository(ctx context.Context, doer, u *user_model.User, repo *repo_model.Repository)
    NewIssue(ctx context.Context, issue *issues_model.Issue, mentions []*user_model.User)
    // ... 更多事件方法
}
```

### 6.2 触发通知

在业务操作完成后，通过 `notify_service` 包级函数触发通知：

```go
// 先执行业务逻辑
repo, err := CreateRepositoryDirectly(ctx, doer, owner, opts)
if err != nil {
    return nil, err
}

// 业务成功后触发通知
notify_service.CreateRepository(ctx, doer, owner, repo)
```

### 6.3 通知不返回错误

通知分发函数通常不返回错误（通知失败不应阻断主流程）：

```go
func NewPullRequest(ctx context.Context, pr *issues_model.PullRequest, mentions []*user_model.User) {
    if err := pr.LoadIssue(ctx); err != nil {
        log.Error("LoadIssue failed: %v", err)
        return  // 静默失败
    }
    for _, notifier := range notifiers {
        notifier.NewPullRequest(ctx, pr, mentions)
    }
}
```

---

## 7. convert 包规范

### 7.1 职责

`services/convert` 负责将 `models/` 层对象转换为 `modules/structs`（API 响应）对象。

### 7.2 函数命名

统一使用 `To{TargetType}` 命名：

```go
func ToEmail(email *user_model.EmailAddress) *api.Email
func ToBranch(ctx context.Context, repo *repo_model.Repository, ...) (*api.Branch, error)
func ToTag(repo *repo_model.Repository, t *git.Tag) *api.Tag
func ToPublicKey(apiLink string, key *asymkey_model.PublicKey) *api.PublicKey
func ToOrganization(ctx context.Context, org *organization.Organization) *api.Organization
```

### 7.3 转换函数特征

- 输入为 models 层类型指针
- 输出为 api 层类型指针
- 只做字段映射，不包含业务逻辑
- 需要上下文时才传入 `ctx`

```go
func ToEmail(email *user_model.EmailAddress) *api.Email {
    return &api.Email{
        Email:    email.Email,
        Verified: email.IsActivated,
        Primary:  email.IsPrimary,
    }
}
```

---

## 8. 表单验证规范（forms 包）

### 8.1 结构体定义

使用 `binding` 标签定义验证规则：

```go
type CreateRepoForm struct {
    UID           int64  `binding:"Required"`
    RepoName      string `binding:"Required;AlphaDashDot;MaxSize(100)"`
    Description   string `binding:"MaxSize(2048)"`
    DefaultBranch string `binding:"GitRefName;MaxSize(100)"`
}
```

### 8.2 Validate 方法

每个表单结构体实现 `Validate` 方法：

```go
func (f *CreateRepoForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
    ctx := context.GetValidateContext(req)
    return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
```

### 8.3 常用验证标签

| 标签           | 说明                           |
| -------------- | ------------------------------ |
| `Required`     | 必填                           |
| `MaxSize(n)`   | 最大长度                       |
| `AlphaDashDot` | 字母、数字、下划线、短横线、点 |
| `GitRefName`   | 合法的 Git 引用名              |
| `ValidUrl`     | 合法 URL                       |

---

## 9. Webhook 包规范

### 9.1 PayloadConvertor 接口

使用泛型接口定义 payload 转换：

```go
type payloadConvertor[T any] interface {
    Create(*api.CreatePayload) (T, error)
    Delete(*api.DeletePayload) (T, error)
    Issue(*api.IssuePayload) (T, error)
    Push(*api.PushPayload) (T, error)
    PullRequest(*api.PullRequestPayload) (T, error)
    // ...
}
```

### 9.2 Webhook 事件处理

每个平台（Slack、Discord、DingTalk 等）实现 `payloadConvertor` 接口，通过 `newPayload` 函数统一分发。

---

## 10. 并发与资源管理

### 10.1 资源关闭

使用 `defer` 确保资源释放：

```go
baseGitRepo, err := gitrepo.OpenRepository(ctx, pr.BaseRepo)
if err != nil {
    return err
}
defer baseGitRepo.Close()
```

### 10.2 Goroutine 使用

使用 `graceful.GetManager().RunWithShutdownContext` 启动后台任务：

```go
graceful.GetManager().RunWithShutdownContext(func(ctx context.Context) {
    // 后台任务
})
```

### 10.3 连接池/锁

使用 `sync.ExclusivePool` 防止并发冲突：

```go
var pullWorkingPool = sync.NewExclusivePool()

func ChangeTargetBranch(ctx context.Context, ...) error {
    pullWorkingPool.CheckIn(fmt.Sprint(pr.ID))
    defer pullWorkingPool.CheckOut(fmt.Sprint(pr.ID))
    // ...
}
```

---

## 11. 测试规范

- 测试文件与源文件同目录，以 `_test.go` 结尾
- 测试包可以是同包（白盒测试）或 `xxx_test` 包（黑盒测试）
- 集成测试放在 `tests/integration/` 目录

```go
package convert

import "testing"

func TestToEmail(t *testing.T) {
    // ...
}
```

---

## 12. 跨层调用规则

```
routers/ → services/ → models/
                ↓
            modules/
```

- `services/` 可调用 `models/` 和 `modules/`
- `services/` 子包之间可以互相调用（通过导入别名）
- `services/` **不应**调用 `routers/`
- `models/` **不应**调用 `services/` 或 `modules/`（仅依赖基础模块）

### 跨服务调用示例

```go
import (
    issue_service "code.gitea.io/gitea/services/issue"
    notify_service "code.gitea.io/gitea/services/notify"
    pull_service "code.gitea.io/gitea/services/pull"
    repo_service "code.gitea.io/gitea/services/repository"
)

// 在 repository 服务中调用 pull 服务
if err := pull_service.CloseRepoBranchesPulls(ctx, doer, repo); err != nil {
    log.Error("CloseRepoBranchesPulls failed: %v", err)
}
```

---

## 13. 常用模式总结

### 13.1 创建-通知模式

```go
func CreateXxx(ctx context.Context, doer *user_model.Xxx, ...) (*Xxx, error) {
    // 1. 前置校验
    // 2. 数据库事务
    // 3. 触发通知
    notify_service.CreateXxx(ctx, result)
    return result, nil
}
```

### 13.2 更新-通知模式

```func UpdateXxx(ctx context.Context, doer *user_model.User, xxx *Xxx) error {
    old := xxx.Field
    xxx.Field = newValue
    if err := models.UpdateXxx(ctx, xxx); err != nil {
        return err
    }
    notify_service.XxxChanged(ctx, doer, xxx, old)
    return nil
}
```

### 13.3 删除-清理模式

```go
func DeleteXxx(ctx context.Context, doer *user_model.User, xxx *Xxx) error {
    // 1. 加载关联数据
    // 2. 删除数据库记录（事务内）
    // 3. 清理文件/资源（事务外）
    // 4. 触发通知
    notify_service.DeleteXxx(ctx, doer, xxx)
    return nil
}
```

### 13.4 批量错误收集

```go
var errs errlist
for _, item := range items {
    if err := processItem(ctx, item); err != nil {
        errs = append(errs, err)
    }
}
if len(errs) > 0 {
    return errs
}
return nil
```
