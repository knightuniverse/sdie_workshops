## Context

仓库 Tags 页面 (`/{org}/{repo}/tags`) 目前没有搜索功能。当仓库有大量标签时，用户无法快速找到特定标签。而仓库的 Branches 和 Commits 页面已经有搜索过滤功能，Tags 页面缺少一致的用户体验。

**当前状态**:
- Tags 页面完全基于数据库 `release` 表查询（不直接调用 git 命令）
- `FindReleasesOptions` 结构体没有 `Keyword` 字段，无法进行 LIKE 搜索
- Branches 页面已经实现了基于数据库 LIKE 的搜索模式，可作为参考

**约束条件**:
- 遵循 Gitea 分层架构：routers → services → models
- 使用现有共享搜索组件 `shared/search/combo`
- 搜索结果需要支持分页

**利益相关者**:
- 用户：需要快速查找标签
- 开发者：需要一致的代码模式

## Goals / Non-Goals

**Goals:**
- 在 Tags 页面添加按名称搜索标签的功能
- 与 Branches 页面的搜索体验保持一致
- 使用数据库 LIKE 搜索，保持与现有架构一致

**Non-Goals:**
- 不实现高级搜索（如按日期、作者过滤）
- 不修改标签的创建或删除逻辑
- 不改变标签的数据存储方式
- 搜索仅匹配 Git tag name，不匹配 release title、note、commit SHA、作者
- 删除/创建操作后不保留搜索态（q 和 page 参数）

## Decisions

### D1: 搜索机制选择
- **选择**: 数据库 LIKE 搜索（类似 Branches 页面）
- **理由**:
  - Tags 页面已经完全基于数据库查询，保持一致性
  - 实现简单；`repo_id` 索引先收敛查询范围到当前仓库，LIKE 再在仓库范围内过滤，是可接受的 trade-off
  - 与 Branches 页面的搜索模式完全一致
- **已考虑 alternative**:
  - Git 命令搜索（类似 Commits 页面）：需要额外的 git 调用，与现有架构不一致，实现复杂度高

### D2: UI 组件选择
- **选择**: 使用 `shared/search/combo` 组件
- **理由**:
  - Tags 搜索只需要按名称过滤，不需要额外选项
  - 与 Branches 页面保持一致的用户体验
  - 实现最简单
- **已考虑 alternative**:
  - `shared/search/input` + `shared/search/button` 组合：更灵活，但 Tags 搜索不需要额外下拉菜单

### D3: 搜索参数处理
- **选择**: 使用 GET 参数 `q`，表单提交到当前页面 URL
- **理由**:
  - 遵循 Branches 页面的模式
  - 支持书签和分享搜索结果
  - 实现简单
- **已考虑 alternative**:
  - 独立搜索端点（类似 Commits 的 `/commits/<branch>/search`）：增加复杂度，不必要
- **服务端规范化**: `ctx.FormTrim("q")` 读取参数（`FormTrim` 执行 `strings.TrimSpace`，而 `FormString` 不 trim），trim 后若超过 255 字符则按 rune 安全截断（与 HTML `maxlength=255` 对齐）。截断必须使用 `[]rune(keyword)[:255]` 等 UTF-8 安全方式，禁止按字节切片，避免多字节字符被截断成非法字符串

### D4: 搜索大小写策略
- **选择**: 大小写不敏感搜索
- **理由**:
  - Release 模型已有 `LowerTagName` 字段，系统本身依赖规范化名称处理
  - 与 `GetRelease` 等现有函数的标签查找模式一致
  - 跨数据库（MySQL、PostgreSQL、SQLite）行为一致
- **实现**: 使用 `builder.Expr("lower_tag_name LIKE ? ESCAPE '!'", "%"+escapeLike(strings.ToLower(keyword))+"%")`，其中 `escapeLike` 按以下顺序转义：先 `!` → `!!`，再 `%` → `!%`，再 `_` → `!_`（使用 `!` 作为 escape 字符，避免反斜杠在不同 SQL 方言中转义语义不一致的问题）。不使用 `builder.Like`，因为 `builder.Like` 不支持 `ESCAPE` 子句

### D5: 搜索范围
- **选择**: 仅匹配 Git tag name（`lower_tag_name` 列）
- **理由**:
  - 简单明确，与用户对"搜索标签"的直觉一致
  - 避免匹配 release title、note 等可能产生歧义的结果
- **不匹配**: release title、note、commit SHA、作者

## Risks / Trade-offs

- [Risk] `FindReleasesOptions` 不支持 `Keyword` 字段 → Mitigation: 添加 `Keyword string` 字段和对应的 LIKE 查询逻辑；空 Keyword 不影响现有调用方
- [Risk] 搜索结果分页问题 → Mitigation: `TagsList` 使用 `db.Count` 以相同 `FindReleasesOptions` 重新计算过滤后总数，替换中间件预计算的全量 `NumTags`
- [Risk] `%keyword%` LIKE 无法利用 B-tree 索引 → Mitigation: `repo_id` 索引先收敛查询范围到当前仓库记录，LIKE 在有限集上操作；对于绝大多数仓库（< 10K 标签）性能可接受
- [Risk] LIKE 通配符 `%` 和 `_` 的特殊语义 → Mitigation: 用户输入中的 `!`、`%`、`_` 按字面转义（`!` → `!!`、`%` → `!%`、`_` → `!_`），使用 `builder.Expr` 配合 `ESCAPE '!'` 子句，确保搜索行为等价于"名称包含该字符串"。使用 `!` 而非 `\` 作为 escape 字符，避免反斜杠在 MySQL/PostgreSQL/SQLite 之间转义语义不一致。不使用 `builder.Like`（该类型不支持 `ESCAPE` 子句）
- [Trade-off] 只搜索已同步到数据库的标签 → 接受理由: 标签通常都会同步到数据库，且搜索主要是为了快速过滤已显示的标签
- [Trade-off] 删除/创建操作后不保留搜索态 → 接受理由: 属于后续优化项，本次实现保持简单

## Migration Plan

N/A — 本 change 不涉及部署变更，仅添加搜索功能。

## Open Questions

1. ~~搜索是否应该同时匹配标签名称和描述（如果有）？~~ → 已决定：仅匹配 Git tag name（使用 `lower_tag_name`），不匹配 release title、note
2. 是否需要在搜索结果为空时显示提示信息？→ 参考 Branches 页面的处理方式
