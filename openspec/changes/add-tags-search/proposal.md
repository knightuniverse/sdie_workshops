## Why

仓库 Tags 页面目前没有搜索功能。当仓库有大量标签时，用户无法快速找到特定标签，只能手动翻页查找。而仓库的 Branches 和 Commits 页面已经有搜索过滤功能，Tags 页面缺少一致的用户体验。添加搜索功能可以提高用户查找标签的效率，保持与其他页面的一致性。

## What Changes

**Tags 页面搜索功能**
- From: Tags 页面只显示分页列表，无搜索功能
- To: Tags 页面添加按名称搜索标签的功能，与 Branches 页面的搜索体验一致
- Impact: 非破坏性变更，仅添加新功能

## Capabilities

### New Capabilities
- `tags-search`: 在 Tags 页面添加按名称搜索标签的功能，包括模型层 Keyword 字段支持、路由处理器搜索参数处理、模板搜索表单添加

### Modified Capabilities
- 无

## Impact

- **模型层**: `models/repo/release.go` - 添加 `Keyword` 字段到 `FindReleasesOptions`，在 `ToConds()` 中添加 LIKE 查询
- **路由层**: `routers/web/repo/release.go` - 在 `TagsList` 中读取 `q` 参数并传递给查询
- **模板层**: `templates/repo/tag/list.tmpl` - 添加搜索表单（使用 `shared/search/combo`）
- **国际化**: 可能需要添加 `search.tag_kind` 翻译
