## 1. 模型层修改

- [ ] 1.1 在 `models/repo/release.go` 的 `FindReleasesOptions` 结构体中添加 `Keyword string` 字段
- [ ] 1.2 在 `FindReleasesOptions.ToConds()` 方法中添加 `Keyword` 的 LIKE 查询逻辑

## 2. 路由处理器修改

- [ ] 2.1 在 `routers/web/repo/release.go` 的 `TagsList` 函数中读取 `q` 参数
- [ ] 2.2 将 `Keyword` 传递给 `FindReleasesOptions`
- [ ] 2.3 将 `Keyword` 回填到 `ctx.Data["Keyword"]`

## 3. 模板修改

- [ ] 3.1 在 `templates/repo/tag/list.tmpl` 中添加搜索表单
- [ ] 3.2 使用 `shared/search/combo` 组件，设置正确的 Placeholder

## 4. 国际化

- [ ] 4.1 检查 `search.tag_kind` 是否已存在，如不存在添加翻译
