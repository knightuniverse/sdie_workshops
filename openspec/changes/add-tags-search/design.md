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

## Decisions

### D1: 搜索机制选择
- **选择**: 数据库 LIKE 搜索（类似 Branches 页面）
- **理由**:
  - Tags 页面已经完全基于数据库查询，保持一致性
  - 实现简单，性能好
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

## Risks / Trade-offs

- [Risk] `FindReleasesOptions` 不支持 `Keyword` 字段 → Mitigation: 添加 `Keyword string` 字段和对应的 LIKE 查询逻辑
- [Risk] 搜索结果分页问题 → Mitigation: 确保 `NumTags` 在搜索时更新为搜索结果总数
- [Trade-off] 只搜索已同步到数据库的标签 → 接受理由: 标签通常都会同步到数据库，且搜索主要是为了快速过滤已显示的标签

## Migration Plan

N/A — 本 change 不涉及部署变更，仅添加搜索功能。

## Open Questions

1. 搜索是否应该同时匹配标签名称和描述（如果有）？→ 初步决定只匹配 `tag_name`
2. 是否需要在搜索结果为空时显示提示信息？→ 参考 Branches 页面的处理方式
