## 1. 模型层修改

- [ ] 1.1 在 `models/repo/release.go` 的 `FindReleasesOptions` 结构体中添加 `Keyword string` 字段
- [ ] 1.2 在 `FindReleasesOptions.ToConds()` 方法中添加 `Keyword` 的 LIKE 查询逻辑：使用 `LowerTagName` 列配合 `strings.ToLower(keyword)` 进行大小写不敏感搜索。LIKE 通配符需按字面匹配：对用户输入中的特殊字符按以下顺序转义——先 `!` → `!!`，再 `%` → `!%`，再 `_` → `!_`（使用 `!` 作为 escape 字符，避免反斜杠在不同 SQL 方言中转义语义不一致的问题）。使用 `builder.Expr("lower_tag_name LIKE ? ESCAPE '!'", "%"+escapedKeyword+"%")` 构造条件，不使用 `builder.Like`（该类型不支持 `ESCAPE` 子句）。空 Keyword 不添加任何条件，不影响现有调用方

## 2. 路由处理器修改

- [ ] 2.1 在 `routers/web/repo/release.go` 的 `TagsList` 函数中使用 `ctx.FormTrim("q")` 读取搜索关键词（`FormTrim` 会 `strings.TrimSpace`，而 `FormString` 不会 trim），空字符串视为无关键词；trim 后若超过 255 字符则按 rune（Unicode 字符）截断到 255 字符（与 HTML maxlength 属性一致，防止直接构造超长 URL 参数）。截断必须使用 UTF-8/rune 安全方式（如 `[]rune(keyword)[:255]`），禁止按字节切片（`keyword[:255]`），避免中文或 emoji 等多字节字符被截断成非法字符串
- [ ] 2.2 将 `Keyword` 传递给 `FindReleasesOptions`
- [ ] 2.3 将 `Keyword` 回填到 `ctx.Data["Keyword"]`
- [ ] 2.4 使用 `db.Count[repo_model.Release](ctx, opts)` 重新计算搜索结果总数（使用与列表查询相同的 `FindReleasesOptions`），替换从中间件获取的 `ctx.Data["NumTags"]` 全量计数，确保分页总数反映搜索结果的实际数量

## 3. 模板修改

- [ ] 3.1 在 `templates/repo/tag/list.tmpl` 中添加搜索表单。模板结构调整为：
  1. 标签标题 header（`h4.ui.top attached header`）始终渲染
  2. 搜索表单作为 attached segment（使用 `shared/search/combo`），始终渲染
  3. 结果 table 作为 attached table segment，仅在 `{{if .Releases}}` 时渲染
  4. 空结果时显示空的 attached segment（或隐藏 table），搜索框始终可见且保留 `.Keyword` 回填
- [ ] 3.2 使用 `shared/search/combo` 组件，设置 Placeholder 为 `search.tag_kind`

## 4. 国际化

- [ ] 4.1 在 `options/locale/locale_en-US.ini` 中添加 `tag_kind = Search tags...` 翻译
- [ ] 4.2 在 `options/locale/locale_zh-CN.ini` 中添加对应的中文翻译

## 5. 测试

- [ ] 5.1 在 `models/repo/release_test.go` 中添加两类测试：
  - **单元测试**：验证 `FindReleasesOptions.ToConds()` 在有 Keyword 时生成正确的 SQL 条件（检查 SQL 包含 `ESCAPE '!'` 子句，且转义字符 `!%`、`!_`、`!!` 正确）；验证空 Keyword 不添加任何 LIKE 条件
  - **集成测试**：使用 `db.Find` / `db.Count` 配合带 Keyword 的 `FindReleasesOptions`，验证真实查询结果：keyword 命中返回正确标签、keyword 未命中返回空、空 keyword 返回全部标签、关键词包含 `%` 或 `_` 时按字面匹配
- [ ] 5.2 添加集成测试覆盖 `GET /{user}/{repo}/tags?q=...` 场景，验证搜索结果正确过滤、分页总数正确、空结果时搜索框保留关键词
