# Brainstorm: Tags 页面搜索功能

## 背景

仓库 Tags 页面 (`/{org}/{repo}/tags`) 目前没有搜索功能。当仓库有大量标签时，用户无法快速找到特定标签。而仓库的 Branches 和 Commits 页面已经有搜索过滤功能，Tags 页面缺少一致的用户体验。

## 现有实现分析

### Tags 页面当前实现

| 组件 | 文件 | 说明 |
|------|------|------|
| 路由处理器 | `routers/web/repo/release.go:205` `TagsList` | 从数据库 `release` 表查询标签 |
| 模型层 | `repo_model.FindReleasesOptions` | 使用 XORM 查询，支持分页 |
| 模板 | `templates/repo/tag/list.tmpl` | 渲染标签列表，无搜索 UI |
| 路由注册 | `routers/web/web.go:1312-1323` | `GET /{user}/{repo}/tags` |
| JSON API | `routers/web/repo/repo.go:716` `GetTagList` | `/tags/list` 返回标签名称列表 |

**关键发现**: Tags 页面完全基于数据库 `release` 表，不直接调用 git 命令。

### Branches 页面搜索模式（参考）

| 组件 | 文件 | 说明 |
|------|------|------|
| 路由处理器 | `routers/web/repo/branch.go:38` `Branches` | 读取 `q` 参数，传递给服务层 |
| 服务层 | `services/repository/branch.go:61` `LoadBranches` | 构建 `FindBranchOptions{Keyword}` |
| 模型层 | `models/git/branch_list.go:90-106` | 使用 `builder.Like{"name", keyword}` SQL LIKE 过滤 |
| 模板 | `templates/repo/branch/list.tmpl:77-79` | 使用 `shared/search/combo` 组件 |
| 搜索参数 | `ctx.FormString("q")` → `ctx.Data["Keyword"]` | GET 参数 `q`，回填到搜索框 |

### Commits 页面搜索模式（参考）

| 组件 | 文件 | 说明 |
|------|------|------|
| 路由处理器 | `routers/web/repo/commit.go:184` `SearchCommits` | 独立的搜索端点 `/commits/<branch>/search` |
| 搜索机制 | `modules/git/repo_commit.go:111-187` | 使用 `git log --grep --fixed-strings -i` |
| 模板 | `templates/repo/commits_table.tmpl:20-29` | 使用 `shared/search/input` + `shared/search/button` |
| 搜索参数 | `q` + `all`（搜索所有分支） | GET 参数 |

## 设计决策

### 决策 1: 搜索机制选择

**问题**: Tags 搜索应该使用数据库 LIKE 还是 git 命令？

**选项**:
- **A) 数据库 LIKE 搜索**（类似 Branches）
  - 优点: 实现简单，与现有 Tags 数据流一致（已从数据库查询）
  - 优点: 性能好，支持分页
  - 缺点: 只能搜索已同步到数据库的标签
  
- **B) Git 命令搜索**（类似 Commits）
  - 优点: 可以搜索所有标签（包括未同步的）
  - 缺点: 需要额外的 git 调用，与现有架构不一致
  - 缺点: 实现复杂度高

**决策**: 选择 **A) 数据库 LIKE 搜索**。理由：
1. Tags 页面已经完全基于数据库查询，保持一致性
2. `FindReleasesOptions` 已经支持 `Keyword` 字段（需要验证）
3. 实现简单，性能好
4. 与 Branches 页面的搜索模式完全一致

### 决策 2: UI 组件选择

**问题**: 使用哪个搜索 UI 组件？

**选项**:
- **A) `shared/search/combo`**（Branches 使用）
  - 包含输入框和搜索按钮
  - 简单的单字段搜索
  
- **B) `shared/search/input` + `shared/search/button`**（Commits 使用）
  - 可以组合额外的下拉菜单
  - 更灵活

**决策**: 选择 **A) `shared/search/combo`**。理由：
1. Tags 搜索只需要按名称过滤，不需要额外选项
2. 与 Branches 页面保持一致的用户体验
3. 实现最简单

### 决策 3: 搜索参数处理

**问题**: 如何处理搜索参数？

**决策**: 遵循 Branches 模式：
1. 使用 GET 参数 `q`（表单提交到当前页面 URL）
2. 处理器读取 `ctx.FormString("q")`
3. 传递给模型层的 `FindReleasesOptions.Keyword`
4. 回填到模板 `ctx.Data["Keyword"]`

## 实现方案

### 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `routers/web/repo/release.go` | 在 `TagsList` 函数中添加搜索参数读取和传递 |
| `templates/repo/tag/list.tmpl` | 添加搜索表单（使用 `shared/search/combo`） |
| `models/repo/release.go` | 验证 `FindReleasesOptions` 是否支持 `Keyword` 字段 |

### 实现步骤

1. **验证模型层支持**
   - 检查 `FindReleasesOptions` 是否有 `Keyword` 字段
   - 如果没有，需要添加该字段和对应的 LIKE 查询逻辑

2. **修改路由处理器**
   - 在 `TagsList` 函数中读取 `q` 参数
   - 将 `Keyword` 传递给 `FindReleasesOptions`
   - 将 `Keyword` 回填到 `ctx.Data["Keyword"]`

3. **修改模板**
   - 在标签列表上方添加搜索表单
   - 使用 `shared/search/combo` 组件
   - 确保搜索框预填用户的查询关键词

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| `FindReleasesOptions` 不支持 `Keyword` | 需要修改模型层 | 先验证，必要时添加 LIKE 查询 |
| 搜索结果分页问题 | 搜索后分页可能不正确 | 确保 `NumTags` 在搜索时更新为搜索结果总数 |
| 模板位置调整 | 搜索框位置可能影响布局 | 参考 Branches 页面的布局 |

## 已确认事项

1. ❌ `FindReleasesOptions` **没有** `Keyword` 字段（仅有 `TagNames []string` 精确匹配）
   - 需要添加 `Keyword string` 字段
   - 需要在 `ToConds()` 方法中添加 `builder.Like{"tag_name", opts.Keyword}` 逻辑
2. 搜索应匹配 `tag_name` 字段（标签名称）
3. 搜索结果为空时参考 Branches 页面的处理方式

## 最终实现方案

### 需要修改的文件

| 文件 | 修改内容 |
|------|----------|
| `models/repo/release.go` | 添加 `Keyword` 字段到 `FindReleasesOptions`，在 `ToConds()` 中添加 LIKE 查询 |
| `routers/web/repo/release.go` | 在 `TagsList` 中读取 `q` 参数并传递给 `FindReleasesOptions.Keyword` |
| `templates/repo/tag/list.tmpl` | 添加搜索表单（使用 `shared/search/combo`） |

### 实现细节

**1. 模型层修改 (`models/repo/release.go`)**

```go
// FindReleasesOptions 结构体添加字段
type FindReleasesOptions struct {
    // ... 现有字段 ...
    Keyword string // 搜索关键词，用于 LIKE 匹配 tag_name
}

// ToConds() 方法添加逻辑
if opts.Keyword != "" {
    cond = cond.And(builder.Like{"tag_name", opts.Keyword})
}
```

**2. 路由处理器修改 (`routers/web/repo/release.go`)**

```go
func TagsList(ctx *context.Context) {
    // ... 现有代码 ...
    kw := ctx.FormString("q")
    opts := repo_model.FindReleasesOptions{
        // ... 现有选项 ...
        Keyword: kw,
    }
    // ... 查询逻辑 ...
    ctx.Data["Keyword"] = kw
}
```

**3. 模板修改 (`templates/repo/tag/list.tmpl`)**

```html
<!-- 在标签列表上方添加搜索表单 -->
<form class="ignore-dirty" method="get">
    {{template "shared/search/combo" dict "Value" .Keyword "Placeholder" (ctx.Locale.Tr "search.tag_kind")}}
</form>
```

**4. 国际化 (可选)**

检查 `search.tag_kind` 是否已存在，如不存在需要添加翻译。
