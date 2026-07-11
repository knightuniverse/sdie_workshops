# modules 目录代码规范

## 目录概述

`modules/` 是 Gitea 的可复用工具包层，包含独立的、无业务依赖的基础库。每个子包职责单一，被 `models/`、`services/`、`routers/` 等上层包引用。

---

## 1. 文件头部规范

### 版权声明

每个 `.go` 文件必须包含版权头和 SPDX 许可证标识：

```go
// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT
```

- 若文件源自 Gogs 项目，保留原始版权并追加 Gitea 版权：
  ```go
  // Copyright 2015 The Gogs Authors. All rights reserved.
  // Copyright 2017 The Gitea Authors. All rights reserved.
  // SPDX-License-Identifier: MIT
  ```

### 包声明

包名与目录名一致，使用小写单词，不使用下划线：

```go
package cache
```

---

## 2. 导入规范

### 分组与排序

导入分三组，组间用空行分隔：

```go
import (
    // 标准库
    "context"
    "fmt"
    "os"

    // Gitea 内部模块
    "code.gitea.io/gitea/modules/log"
    "code.gitea.io/gitea/modules/setting"

    // 第三方库
    "github.com/stretchr/testify/assert"
)
```

### 禁止直接导入的包

- **JSON**: 使用 `modules/json`，禁止直接导入 `encoding/json`（性能优化，基于 jsoniter）
- **配置**: 使用 Gitea 的配置系统，禁止直接导入 `gopkg.in/ini.v1`
- **IO**: 使用 `os` 或 `io`，禁止使用已废弃的 `io/ioutil`

### nolint 注释

当必须导入被限制的包时，需添加 `nolint:depguard` 注释并说明原因：

```go
import (
    "encoding/json" //nolint:depguard // 需要访问标准库的 Valid 函数
)
```

---

## 3. 接口设计规范

### 小接口原则

接口应保持精简，遵循 Go 的小接口哲学：

```go
// Object 代表存储上的对象
type Object interface {
    io.ReadCloser
    io.Seeker
    Stat() (os.FileInfo, error)
}

// ObjectStorage 代表对象存储
type ObjectStorage interface {
    Open(path string) (Object, error)
    Save(path string, r io.Reader, size int64) (int64, error)
    Stat(path string) (os.FileInfo, error)
    Delete(path string) error
    URL(path, name string, reqParams url.Values) (*url.URL, error)
    IterateObjects(path string, iterator func(path string, obj Object) error) error
}
```

### 接口组合

通过嵌入已有接口构建更大接口：

```go
type Interface interface {
    Marshal(v any) ([]byte, error)
    Unmarshal(data []byte, v any) error
    NewEncoder(writer io.Writer) Encoder
    NewDecoder(reader io.Reader) Decoder
}
```

### 编译期接口检查

使用空变量赋值验证类型是否实现了接口：

```go
var (
    _ Interface = StdJSON{}
    _ Interface = JSONiter{}
)
```

---

## 4. 错误处理规范

### 错误类型定义

定义结构化的错误类型，包含上下文信息：

```go
type ErrNotExist struct {
    ID      string
    RelPath string
}

func (err ErrNotExist) Error() string {
    return fmt.Sprintf("object does not exist [id: %s, rel_path: %s]", err.ID, err.RelPath)
}
```

### 错误判断函数

每个错误类型配套一个 `IsErrXxx` 判断函数：

```go
func IsErrNotExist(err error) bool {
    _, ok := err.(ErrNotExist)
    return ok
}
```

### 错误链与 Unwrap

实现 `Unwrap()` 方法，将自定义错误关联到基础错误：

```go
func (err ErrNotExist) Unwrap() error {
    return util.ErrNotExist
}
```

### 基础错误类型

`modules/util/error.go` 定义了四个基础错误，所有业务错误应关联到其中之一：

```go
var (
    ErrInvalidArgument  = errors.New("invalid argument")
    ErrPermissionDenied = errors.New("permission denied")
    ErrAlreadyExist     = errors.New("resource already exists")
    ErrNotExist         = errors.New("resource does not exist")
)
```

### SilentWrap 包装器

当需要为错误附加人类可读消息但保留原始错误链时使用：

```go
type SilentWrap struct {
    Message string
    Err     error
}

func (w SilentWrap) Error() string { return w.Message }
func (w SilentWrap) Unwrap() error { return w.Err }
```

配套构造函数：

```go
func NewNotExistErrorf(message string, args ...any) error {
    return NewSilentWrapErrorf(ErrNotExist, message, args...)
}
```

### 错误处理原则

- 错误返回值必须被检查，禁止忽略
- 使用 `fmt.Errorf("xxx: %w", err)` 包装错误，保留上下文
- 忽略明确无害的错误时使用 `_ =` 显式标记：
  ```go
  _ = obj.Close()
  _ = defaultCache.Delete(key)
  ```

---

## 5. 命名规范

### 包级变量

包级变量使用 `var` 块集中声明，附带注释：

```go
var (
    // DefaultJSONHandler default json handler
    DefaultJSONHandler Interface = JSONiter{...}
)
```

### 导出函数

导出函数必须有 godoc 注释，格式为 `// FuncName 描述`：

```go
// GetString returns the key value from cache with callback when no key exists in cache
func GetString(key string, getFunc func() (string, error)) (string, error) {
```

### 非导出函数

非导出函数注释可选，但复杂逻辑应有注释。

### 类型命名

- 接口：动词或名词短语（`Encoder`、`ObjectStorage`）
- 结构体：名词（`Command`、`StatusTable`）
- 错误类型：`Err` 前缀（`ErrNotExist`、`ErrExecTimeout`）

### 常量

使用驼峰命名，`iota` 枚举使用块声明：

```go
type state uint8

const (
    stateInit state = iota
    stateRunning
    stateShuttingDown
    stateTerminate
)
```

---

## 6. 泛型使用规范

### 泛型容器

在 `modules/container/` 中广泛使用泛型：

```go
type Set[T comparable] map[T]struct{}

func SetOf[T comparable](values ...T) Set[T] {
    s := make(Set[T], len(values))
    s.AddMultiple(values...)
    return s
}
```

### 泛型函数

工具函数优先使用泛型而非 `any`：

```go
func SliceRemoveAll[T comparable](slice []T, target T) []T {
    return slices.DeleteFunc(slice, func(t T) bool { return t == target })
}

func SliceSortedEqual[T comparable](s1, s2 []T) bool { ... }
```

### 泛型约束

使用标准库约束或自定义约束：

```go
func Sorted[S ~[]E, E cmp.Ordered](values S) S {
    slices.Sort(values)
    return values
}
```

---

## 7. 并发规范

### 互斥锁

使用 `sync.Mutex` 或 `sync.RWMutex` 保护共享状态：

```go
type StatusTable struct {
    lock sync.RWMutex
    pool container.Set[string]
}

func (p *StatusTable) IsRunning(name string) bool {
    p.lock.RLock()
    exists := p.pool.Contains(name)
    p.lock.RUnlock()
    return exists
}
```

### 锁粒度

- 读多写少场景使用 `RWMutex`
- 锁内只做必要的读写操作，不在锁内调用外部函数
- 加锁后立即解锁，使用 `defer` 只在函数较短时使用

### 单例模式

使用 `sync.Once` 保证初始化只执行一次：

```go
var (
    manager  *Manager
    initOnce sync.Once
)

func GetManager() *Manager {
    InitManager(context.Background())
    return manager
}

func InitManager(ctx context.Context) {
    initOnce.Do(func() {
        manager = newGracefulManager(ctx)
    })
}
```

---

## 8. 初始化模式

### 包级 Init 函数

每个包提供 `Init()` 函数进行初始化，返回 `error`：

```go
func Init() error {
    if defaultCache == nil {
        c, err := NewStringCache(setting.CacheService.Cache)
        if err != nil {
            return err
        }
        defaultCache = c
    }
    return nil
}
```

### 多组件初始化

使用函数切片批量初始化：

```go
func Init() error {
    for _, f := range []func() error{
        initAttachments,
        initAvatars,
        initRepoAvatars,
        initLFS,
    } {
        if err := f(); err != nil {
            return err
        }
    }
    return nil
}
```

### 默认值与回退

初始化时提供合理的默认值和重试机制：

```go
func Init() error {
    for i := 0; i < 10; i++ {
        if err = c.Ping(); err == nil {
            break
        }
        time.Sleep(time.Second)
    }
}
```

---

## 9. 测试规范

### 测试文件命名

测试文件与源码同目录，命名为 `*_test.go`。

### 测试框架

统一使用 `github.com/stretchr/testify/assert`：

```go
func TestSliceContainsString(t *testing.T) {
    assert.True(t, SliceContainsString([]string{"c", "b", "a", "b"}, "a"))
    assert.False(t, SliceContainsString([]string{}, "a"))
}
```

### 表驱动测试

使用匿名结构体切片实现表驱动测试：

```go
func TestRepository_GetCommitBranches(t *testing.T) {
    testCases := []struct {
        CommitID         string
        ExpectedBranches []string
    }{
        {"2839944139e0de9737a044f78b0e4b40d989a9e3", []string{"branch1"}},
        {"5c80b0245c1c6f8343fa418ec374b13b5d4ee658", []string{"branch2"}},
    }
    for _, testCase := range testCases {
        commit, err := bareRepo1.GetCommit(testCase.CommitID)
        assert.NoError(t, err)
        branches, err := bareRepo1.getBranches(os.Environ(), commit.ID.String(), 2)
        assert.NoError(t, err)
        assert.Equal(t, testCase.ExpectedBranches, branches)
    }
}
```

### 测试资源清理

使用 `defer` 确保资源释放：

```go
bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
assert.NoError(t, err)
defer bareRepo1.Close()
```

---

## 10. 日志规范

### 使用 modules/log

统一使用 `modules/log` 包，禁止直接使用标准库 `log`。

### 日志级别

```go
log.Info("Initialising Avatar storage with type: %s", setting.Avatar.Storage.Type)
log.Fatal("Unable to init config provider from %q: %v", file, err)
log.Critical("PANIC during RunWithCancel: %v\nStacktrace: %s", err, log.Stack(2))
```

---

## 11. 命令执行规范

### Git 命令封装

使用 `modules/git.Command` 封装外部命令调用：

```go
cmd := NewCommand(ctx, "init")
cmd.AddOptionValues("--object-format", objectFormatName)
cmd.AddArguments("--bare")
_, _, err = cmd.RunStdString(&RunOpts{Dir: repoPath})
```

### 安全参数

用户输入必须通过 `AddDynamicArguments` 传递，不得直接拼接：

```go
// 正确
cmd.AddDynamicArguments(userInput)

// 错误 - 直接拼接参数
cmd.AddArguments(userInput)  // 仅用于可信参数
```

### 超时控制

命令执行应设置超时：

```go
var defaultCommandExecutionTimeout = 360 * time.Second
```

---

## 12. 泛型约束

### comparable 约束

用于集合、映射等需要比较的场景：

```go
type Set[T comparable] map[T]struct{}
```

### cmp.Ordered 约束

用于排序场景：

```go
func Sorted[S ~[]E, E cmp.Ordered](values S) S
```

### any 约束

用于通用容器或序列化场景：

```go
func (StdJSON) Marshal(v any) ([]byte, error)
```

---

## 13. 文档注释规范

### 包注释

包注释应说明包的用途和核心概念，位于 `package` 声明之前：

```go
// Package queue implements a specialized concurrent queue system for Gitea.
//
// Terminology:
//
//  1. Item: ...
//  2. Batch: ...
//  3. Worker: ...
package queue
```

### 函数注释

导出函数必须有注释，格式为函数名开头：

```go
// GetAllCommitsCount returns count of all commits in repository
func (repo *Repository) GetAllCommitsCount() (int64, error) {
```

### 类型注释

```go
// GPGSettings represents the default GPG settings for this repository
type GPGSettings struct {
    Sign             bool
    KeyID            string
}
```

### 字段注释

重要字段应有行内注释：

```go
var (
    // AppVer is the version of the current build of Gitea
    AppVer string
    // AppStartTime store time gitea has started
    AppStartTime time.Time
)
```

---

## 14. 总结：核心原则

| 原则       | 说明                                     |
| ---------- | ---------------------------------------- |
| 单一职责   | 每个包只做一件事                         |
| 无业务依赖 | modules 不得引用 models/services/routers |
| 接口驱动   | 通过接口定义契约，支持多实现             |
| 错误链完整 | 所有错误实现 Unwrap，支持 errors.Is/As   |
| 泛型优先   | 适用场景优先使用泛型而非 any             |
| 编译期检查 | 使用空赋值验证接口实现                   |
| 测试完备   | 每个包都有对应的测试文件                 |
| 文档齐全   | 所有导出符号都有 godoc 注释              |
