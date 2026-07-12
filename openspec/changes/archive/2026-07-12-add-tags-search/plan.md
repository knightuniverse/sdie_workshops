# Tags 页面搜索功能实现计划

> **For agentic workers:** Use superpowers:subagent-driven-development
> to implement this plan task-by-task.

**Goal:** 在 Tags 页面添加按名称搜索标签的功能，与 Branches 页面的搜索体验保持一致

**Architecture:** 遵循 Gitea 分层架构，在模型层添加 Keyword 字段支持 LIKE 查询，路由层读取搜索参数并传递给模型层，模板层添加搜索表单

**Tech Stack:** Go, XORM, Go HTML Template

---

## Task 1: 模型层修改

- [ ] **Step 1:** 在 `models/repo/release.go` 的 `FindReleasesOptions` 结构体中添加 `Keyword string` 字段
  - 文件: `models/repo/release.go`
  - 位置: 第 229-238 行的 `FindReleasesOptions` 结构体
  - 添加: `Keyword string` 字段

- [ ] **Step 2:** 在 `FindReleasesOptions.ToConds()` 方法中添加 `Keyword` 的 LIKE 查询逻辑
  - 文件: `models/repo/release.go`
  - 位置: 第 240-266 行的 `ToConds()` 方法
  - 添加: 在方法末尾添加 `if opts.Keyword != "" { cond = cond.And(builder.Like{"tag_name", opts.Keyword}) }`

- [ ] **Step 3:** 运行测试验证模型层修改
  - 命令: `go test ./models/repo/... -run TestFindReleases`

## Task 2: 路由处理器修改

- [ ] **Step 1:** 在 `routers/web/repo/release.go` 的 `TagsList` 函数中读取 `q` 参数
  - 文件: `routers/web/repo/release.go`
  - 位置: 第 205 行开始的 `TagsList` 函数
  - 添加: `kw := ctx.FormString("q")`

- [ ] **Step 2:** 将 `Keyword` 传递给 `FindReleasesOptions`
  - 文件: `routers/web/repo/release.go`
  - 位置: 第 226-234 行的 `opts` 构建
  - 修改: 添加 `Keyword: kw` 到 `FindReleasesOptions` 初始化

- [ ] **Step 3:** 将 `Keyword` 回填到 `ctx.Data["Keyword"]`
  - 文件: `routers/web/repo/release.go`
  - 位置: 查询结果处理后
  - 添加: `ctx.Data["Keyword"] = kw`

## Task 3: 模板修改

- [ ] **Step 1:** 在 `templates/repo/tag/list.tmpl` 中添加搜索表单
  - 文件: `templates/repo/tag/list.tmpl`
  - 位置: 标签列表上方
  - 添加: 使用 `shared/search/combo` 组件的搜索表单

- [ ] **Step 2:** 设置正确的 Placeholder
  - 使用 `ctx.Locale.Tr "search.tag_kind"` 或类似翻译

## Task 4: 国际化

- [ ] **Step 1:** 检查 `search.tag_kind` 是否已存在
  - 命令: `grep -r "search.tag_kind" options/locale/`

- [ ] **Step 2:** 如不存在，添加翻译
  - 文件: `options/locale/locale_en-US.ini`
  - 添加: `search.tag_kind = tags`
