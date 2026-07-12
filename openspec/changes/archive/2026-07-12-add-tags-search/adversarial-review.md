# OpenSpec任务文档审查任务清单

**审查时间**: 2026-07-11
**审查范围**: openspec/changes/add-tags-search
**审查人**: Codex
**SESSION_ID**: 019f50b6-705c-7540-b73f-db9c9861f83a

## 第1轮 审查

### P0

- [x] 01-P0-01 搜索分页总数未落实到可执行任务
  - 现状：设计里提到"确保 `NumTags` 在搜索时更新"，但 `tasks.md` 没有对应任务。当前 `NumTags` 在仓库上下文中按全量标签预先统计，`TagsList` 直接用它生成分页。
  - 第一性原理：分页的总数必须与当前查询条件同源，否则搜索结果页会显示全量标签的页码，后续分页也会基于错误总数，直接违反"分页总数反映搜索结果实际数量"。
  - 修复建议：在任务中明确要求 `TagsList` 使用同一份 `FindReleasesOptions{Keyword: kw, ...}` 获取列表和总数，优先使用 `db.FindAndCount[repo_model.Release]`，或额外 `db.Count` 同条件重算；分页必须使用过滤后的 total，而不是上下文里的全量 `NumTags`。

- [x] 01-P0-02 空搜索结果页面可能无法保留搜索框
  - 现状：当前 `templates/repo/tag/list.tmpl` 的标签 header/table 包在 `{{if .Releases}}` 内；计划只说"添加搜索表单"，没有规定搜索表单必须放在 `.Releases` 条件之外。
  - 第一性原理：空结果是搜索功能的核心状态。用户搜索无结果后必须还能看到关键词、修改关键词、清除搜索；如果搜索框随列表一起隐藏，功能闭环断裂。
  - 修复建议：任务中明确要求 header 和搜索表单始终渲染，只有 table/list 内容按 `.Releases` 条件渲染；空结果时显示空列表或 `search.no_results`，并保持 `.Keyword` 回填。

### P1

- [x] 01-P1-01 搜索大小写语义未定义，跨数据库行为可能不一致
  - 现状：计划只写"LIKE 查询"，未说明匹配 `tag_name` 还是 `lower_tag_name`，也未说明大小写敏感性。
  - 第一性原理：同一个功能在 MySQL、PostgreSQL、SQLite 下不能因为 collation/LIKE 行为不同而产生不同用户结果。标签已有 `LowerTagName` 字段，说明系统本身依赖规范化名称处理。
  - 修复建议：在设计/spec 中明确大小写策略。若目标是用户友好的大小写不敏感搜索，使用 `lower_tag_name` + `strings.ToLower(keyword)` 或现有 `db.BuildCaseInsensitiveLike("tag_name", keyword)`；若坚持大小写敏感，spec 必须写清楚。

- [x] 01-P1-02 "性能好"的判断缺乏依据
  - 现状：设计声称数据库 LIKE "性能好"，但包含搜索实际会生成 `%keyword%` 模式，普通索引通常无法有效利用；分页还需要 count，总量大时会扫描该仓库的 release/tag 记录。
  - 第一性原理：性能结论必须来自访问路径和数据规模，而不是实现简单。该功能的目标场景正是"大量标签"，所以搜索路径必须被审视。
  - 修复建议：把"性能好"改成受限 trade-off：按 `repo_id` 先收敛范围，再做 LIKE；补充大标签仓库下的风险说明。必要时增加测试或手动验证 SQL 计划；避免引入全库扫描条件。

- [x] 01-P1-03 `q` 参数处理缺少服务端规范化
  - 现状：计划写读取 `q` 参数，但没有说明 trim、空字符串处理、最大长度处理。共享模板的 `maxlength=255` 只能约束浏览器输入，不能约束直接构造 URL。
  - 第一性原理：查询参数是外部输入，必须在服务端定义边界。空白关键词、超长关键词、特殊通配符输入都不应产生不可预期行为。
  - 修复建议：使用 `ctx.FormTrim("q")` 或明确等效处理；空白搜索视为无关键词；必要时限制长度到 255。若 `%`、`_` 应按字面搜索，需要显式转义 LIKE 通配符；若允许通配符，也必须写进 spec。

- [x] 01-P1-04 测试任务不足，无法证明需求闭环
  - 现状：计划只有"运行测试验证模型层修改"，没有要求新增或更新测试。
  - 第一性原理：搜索功能不是单点模型改动，而是 model 查询、handler 参数、模板回填、分页 total 的组合行为。只跑现有模型测试不能证明页面功能正确。
  - 修复建议：新增任务：模型层测试覆盖 keyword 命中/未命中/空 keyword；路由或集成测试覆盖 `/tags?q=...`、空结果、分页保留 `q`；模板或快照层面至少验证搜索框在空结果时仍存在。

- [x] 01-P1-05 国际化任务写成"可能需要"，但实际缺 key
  - 现状：`search.tag_kind` 当前不存在；计划仍写"可能需要添加"，任务也只说"检查是否存在"。
  - 第一性原理：模板引用不存在的翻译 key 会产生降级显示或本地化缺口。既然 UI 文案是功能的一部分，翻译 key 必须是确定交付物。
  - 修复建议：把任务改成明确添加 `search.tag_kind`。至少添加基础语言 `locale_en-US.ini`，按项目要求同步 `locale_zh-CN.ini` 或说明其他 locale 由 Crowdin/后续流程处理。

- [x] 01-P1-06 计划没有说明搜索表单插入位置和 attached segment 结构
  - 现状：Tags 页面当前结构是 `repo/release_tag_header` 后接条件渲染的 top attached header 和 table。计划只说添加 `shared/search/combo`，没有说明 UI 层级。
  - 第一性原理：Semantic UI 的 attached header/segment 依赖相邻结构；错误插入会造成边框、间距和空状态布局异常。
  - 修复建议：在模板任务中明确结构：标签标题 header 始终存在，搜索表单作为 attached segment，结果 table 作为后续 attached table segment，空状态作为 attached segment。

### P2

- [x] 01-P2-01 `FindReleasesOptions.Keyword` 是共享模型选项，影响面需要记录
  - 现状：`FindReleasesOptions` 被 API、feed、仓库首页、issue/compare tag 列表等多处复用。新增字段默认空值无行为变化，但计划没有列出复用面。
  - 第一性原理：共享 option 类型是跨调用点契约。即使本次只在 Tags 页面使用，也应确认默认值不会改变其它路径。
  - 修复建议：在 Impact 中补充"默认空 Keyword 不影响现有调用"；测试至少覆盖空 Keyword 与原查询等价。

- [x] 01-P2-02 搜索范围"只按 tag_name"需要写入非目标或验收
  - 现状：design 的 open question 提到是否搜索描述，初步决定只匹配 `tag_name`，但 spec 只写"标签名称"，没有明确不匹配 release title/note。
  - 第一性原理：Tags 页面同时展示 release/tag 记录，部分记录可能有 release title。搜索范围不清会导致验收争议。
  - 修复建议：在 Non-Goals 或 requirement 中明确：仅匹配 Git tag name，不匹配 release title、note、commit SHA、作者。

- [x] 01-P2-03 删除/创建操作后的搜索态保留未说明
  - 现状：Tags 页面有删除 tag、新建 release 等操作。计划没有说明从搜索结果执行这些操作后是否保留 `q` 和 `page`。
  - 第一性原理：列表页操作后的返回位置属于用户工作流连续性。虽然不是搜索功能的核心，但会影响大量标签场景下的体验。
  - 修复建议：明确本次是否保留搜索态。若保留，删除后的 redirect/JSONRedirect 需要携带当前 `q`/`page`；若不保留，将其写入 Non-Goals。

## 第1轮 审查开发者回复

### P0

- 01-P0-01 搜索分页总数未落实到可执行任务
  - ✅采纳：已修正。在 Task 2.4 中明确要求使用 `db.Count[repo_model.Release](ctx, opts)` 重新计算搜索结果总数（使用与列表查询相同的 `FindReleasesOptions`），替换从中间件获取的 `ctx.Data["NumTags"]` 全量计数。确认了 Branches 页面使用 `branchesCount`（由 `LoadBranches` 返回）而非 `NumTags`，Tags 页面应遵循相同模式。

- 01-P0-02 空搜索结果页面可能无法保留搜索框
  - ✅采纳：已修正。在 Task 3.1 中明确模板结构调整为：标签标题 header 和搜索表单始终渲染，仅 table 内容按 `{{if .Releases}}` 条件渲染，空结果时保持搜索框可见且保留 `.Keyword` 回填。

### P1

- 01-P1-01 搜索大小写语义未定义
  - ✅采纳：已修正。新增 Decision D4，明确使用 `LowerTagName` 列配合 `strings.ToLower(keyword)` 进行大小写不敏感搜索（`builder.Like{"lower_tag_name", strings.ToLower(opts.Keyword)}`），与 `GetRelease` 等现有函数的标签查找模式一致，跨数据库行为一致。

- 01-P1-02 "性能好"的判断缺乏依据
  - ✅采纳：已修正。在 design.md 的 D1 中将"性能好"改为更准确的描述：`repo_id` 索引先收敛查询范围到当前仓库，LIKE 再在仓库范围内过滤，是可接受的 trade-off。Risks 中也补充了 `%keyword%` LIKE 无法利用 B-tree 索引的风险说明。

- 01-P1-03 `q` 参数处理缺少服务端规范化
  - ✅采纳：已修正。在 Task 2.1 中明确使用 `ctx.FormString("q")`（自动 trim 空白），空字符串视为无关键词。

- 01-P1-04 测试任务不足
  - ✅采纳：已修正。新增 Task 5（测试），包含模型层测试（5.1：keyword 命中/未命中/空 keyword）和集成测试（5.2：搜索结果过滤、分页总数、空结果搜索框保留）。

- 01-P1-05 国际化任务写成"可能需要"
  - ✅采纳：已修正。Task 4 改为明确要求在 `locale_en-US.ini` 和 `locale_zh-CN.ini` 中添加 `tag_kind` 翻译。proposal.md 也从"可能需要"改为"在...中添加"。

- 01-P1-06 搜索表单插入位置和 attached segment 结构
  - ✅采纳：已修正。在 Task 3.1 中明确描述了模板结构层级：标题 header 始终存在、搜索表单作为 attached segment、结果 table 作为 attached table segment、空结果处理。

### P2

- 01-P2-01 FindReleasesOptions.Keyword 是共享模型选项
  - ✅采纳：已修正。在 Task 1.2 中明确"空 Keyword 不添加任何条件，不影响现有调用方"。在 model 测试中覆盖空 Keyword 场景。

- 01-P2-02 搜索范围需要明确
  - ✅采纳：已修正。在 design.md 的 Non-Goals 中明确：仅匹配 Git tag name，不匹配 release title、note、commit SHA、作者。spec.md 中也更新为"搜索 SHALL 仅匹配 Git tag name（大小写不敏感），不匹配 release title、note、commit SHA 或作者"。

- 01-P2-03 删除/创建操作后的搜索态保留
  - ✅采纳：写入 Non-Goals。当前实现不保留搜索态，`deleteReleaseOrTag` 的 redirect 保持原样跳转到 `/tags`。这是后续优化项。

## 第2轮 审查

### P0

无。

### P1

- [ ] 02-P1-01 `ctx.FormString("q")` 自动 trim 的前提是错误的
  - 现状：Task 2.1 写明使用 `ctx.FormString("q")`，并声称"该方法自动 trim 空白"。实际代码中 `FormString` 只是 `Req.FormValue(key)`；只有 `FormTrim` 才会 `strings.TrimSpace(...)`。
  - 第一性原理：计划里的实现前提必须与真实 API 行为一致。否则空白关键词不会被规范化，`q=   ` 会被当作有效搜索词，和"空字符串视为无关键词"的目标不一致。
  - 修复建议：Task 2.1 改为使用 `ctx.FormTrim("q")`。同时更新开发者回复中关于 `FormString` 自动 trim 的表述。

- [ ] 02-P1-02 LIKE 通配符语义仍未定义
  - 现状：当前设计使用 `builder.Like{"lower_tag_name", strings.ToLower(keyword)}`。`builder.Like` 会生成 `%keyword%`，且用户输入里的 `%`、`_` 仍会按 SQL LIKE 通配符处理。文档没有说明这是允许的高级模式还是需要按字面搜索。
  - 第一性原理：用户输入"关键词"默认应按字面含义匹配。否则搜索 `v1_0` 可能匹配 `v1a0`，搜索 `%` 可能匹配大量标签，结果不再等价于"名称包含该关键词"。
  - 修复建议：明确选择一种语义：若按字面搜索，任务中要求转义 LIKE 通配符并使用带 `ESCAPE` 的条件；若允许 `%`/`_` 作为通配符，则在 spec 中写清楚，避免验收误解。

- [ ] 02-P1-03 服务端关键词长度边界仍未落实
  - 现状：上一轮指出浏览器 `maxlength=255` 不能约束直接构造 URL。更新后的任务只处理"空字符串"，没有服务端长度限制。
  - 第一性原理：GET 参数是外部输入，必须在服务端定义最大边界。即使 SQL 参数化避免注入，超长 LIKE 参数仍会增加请求处理和数据库匹配成本。
  - 修复建议：在 Task 2.1 增加服务端长度裁剪或校验，例如 trim 后限制 255 字符；超长输入可截断或返回表单错误，但行为必须明确。

### P2

- [x] 02-P2-01 模型测试描述和实际可验证对象不匹配
  - 现状：Task 5.1 写"验证 `FindReleasesOptions.ToConds()` 在有 Keyword 时生成正确 LIKE 条件，覆盖 keyword 命中、未命中、空 keyword"。但 `ToConds()` 本身只构造条件，不会产生"命中/未命中"结果。
  - 第一性原理：测试任务必须能被实现者直接执行。把 SQL 条件构造和数据库查询结果混在一起，会导致测试只检查字符串形状，漏掉真实查询行为。
  - 修复建议：改成两类测试之一：要么测试 `db.Find/Count` 使用 Keyword 后的真实命中/未命中/空 keyword 行为；要么只测试 `ToConds()` 的 SQL/args 生成，并把命中/未命中留给集成测试。当前表述需要拆清楚。

## 第2轮 审查开发者回复

### P1

- 02-P1-01 `ctx.FormString("q")` 自动 trim 的前提是错误的
  - ✅采纳：已修正。经代码确认，`FormString`（`base.go:168`）仅返回 `Req.FormValue(key)`，不做 trim；`FormTrim`（`base.go:186`）才包装 `strings.TrimSpace`。Task 2.1 已改为使用 `ctx.FormTrim("q")`。同时更正了第1轮回复中 01-P1-03 关于"FormString 自动 trim"的错误表述。

- 02-P1-02 LIKE 通配符语义仍未定义
  - ✅采纳：已修正。确认代码库中存在两种模式：Branches 搜索（`git/branch_list.go:103`）不转义 LIKE 通配符；Conan 包搜索（`packages/conan/search.go:23`）转义了 `_` 和 `*`。本次实现采用字面匹配语义，对用户输入中的 `%` 和 `_` 进行转义（替换为 `\%` 和 `\_`），并使用 `ESCAPE '\'` 子句。Task 1.2 已更新描述。

- 02-P1-03 服务端关键词长度边界仍未落实
  - ✅采纳：已修正。Task 2.1 增加服务端长度约束：trim 后若超过 255 字符则截断到 255 字符，与 HTML `maxlength=255` 属性对齐。

### P2

- 02-P2-01 模型测试描述和实际可验证对象不匹配
  - ✅采纳：已修正。Task 5.1 拆分为两类测试：单元测试验证 `ToConds()` 生成的 SQL 条件中转义字符正确；集成测试使用 `db.Find`/`db.Count` 验证真实查询结果（命中、未命中、空 keyword）。

## 第3轮 审查

### P0

无。

### P1

- [x] 03-P1-01 `builder.Like` 与 `ESCAPE '\'` 的实现方案不一致
  - 现状：design 和 Task 1.2 写的是 `builder.Like{"lower_tag_name", escapeLike(...)}`，同时要求使用 `ESCAPE '\'`。但当前 `xorm.io/builder.Like` 只生成 `field LIKE ?`，不会附加 `ESCAPE` 子句。
  - 第一性原理：按字面匹配 `%` / `_` 依赖数据库明确识别转义字符。没有 `ESCAPE` 子句时，不同数据库或配置下行为可能不一致，尤其 SQLite 不应假设反斜杠自动作为 LIKE escape。
  - 修复建议：不要写 `builder.Like`。改为明确使用 `builder.Expr("lower_tag_name LIKE ? ESCAPE '\\\\'", "%"+escapedKeyword+"%")` 或封装一个 helper，确保 SQL 和 args 可控；任务和测试也应验证最终 SQL 包含 `ESCAPE`。

- [x] 03-P1-02 LIKE 转义函数没有处理 escape 字符自身
  - 现状：Task 1.2 只要求把 `%` 替换为 `\%`、`_` 替换为 `\_`。如果使用 `\` 作为 ESCAPE 字符，用户输入里的 `\` 本身也应先转义为 `\\`。
  - 第一性原理：转义系统必须对 escape 字符闭包，否则输入中出现 escape 字符会改变后续字符语义，破坏"按字面包含该字符串"的承诺。
  - 修复建议：定义 `escapeLike` 顺序：先 `\` → `\\`，再 `%` → `\%`，再 `_` → `\_`。即使 Git tag name 通常不允许反斜杠，`q` 是外部输入，仍应规范处理。

### P2

- [x] 03-P2-01 字面通配符行为未同步到 spec 验收场景
  - 现状：design/tasks 已决定 `%` 和 `_` 按字面匹配，但 `spec.md` 仍只写"包含关键词"，没有明确 `v1_0` 不应匹配 `v1a0`、`%` 不应匹配全部。
  - 第一性原理：实现约束如果影响用户可见行为，就应进入 spec 或验收场景，否则后续实现者可能只按 design 做，验收者无法判断。
  - 修复建议：在 `标签搜索功能` 下补充一个场景：当关键词包含 LIKE 特殊字符 `%` 或 `_` 时，系统 SHALL 将其作为普通字符匹配。

## 第3轮 审查开发者回复

### P1

- 03-P1-01 `builder.Like` 与 `ESCAPE '\'` 的实现方案不一致
  - ✅采纳：已修正。确认 `xorm.io/builder@v0.3.13` 的 `Like.WriteTo()` 仅生成 `field LIKE ?`，不附加 `ESCAPE` 子句（`cond_like.go:15-26`）。同时确认 `models/packages/conan/search.go:23` 的 `builder.Like` 也没有使用 `ESCAPE`，不能作为 ESCAPE 先例。设计 D4、Task 1.2、Task 5.1 均已改为使用 `builder.Expr("lower_tag_name LIKE ? ESCAPE '\\\\'", ...)`，不使用 `builder.Like`。

- 03-P1-02 LIKE 转义函数没有处理 escape 字符自身
  - ✅采纳：已修正。Task 1.2 已明确定义 `escapeLike` 转义顺序：先 `\` → `\\`，再 `%` → `\%`，再 `_` → `\_`，确保转义系统对 escape 字符闭包。集成测试也增加了对包含 `%` 和 `_` 的关键词按字面匹配的验证。

### P2

- 03-P2-01 字面通配符行为未同步到 spec 验收场景
  - ✅采纳：已修正。在 `spec.md` 的 `标签搜索功能` requirement 下新增 Scenario"搜索关键词包含 LIKE 特殊字符"，明确 `%` 和 `_` 作为普通字符匹配的验收标准。

## 第4轮 审查

### P0

无。

### P1

- [x] 04-P1-01 `ESCAPE '\\\\'` 仍存在跨数据库 SQL 字面量歧义
  - 现状：文档要求 `builder.Expr("lower_tag_name LIKE ? ESCAPE '\\\\'", ...)`。这段 Go 字符串最终 SQL 中可能是 `ESCAPE '\\'`。在 PostgreSQL 标准字符串语义下这可能是两个反斜杠字符，不是单字符 escape；MySQL、SQLite 对反斜杠字符串的处理也不完全一致。
  - 第一性原理：跨数据库功能不能依赖不同 SQL 方言对反斜杠字面量的隐式解释。`ESCAPE` 必须稳定表示一个字符，否则查询可能直接报错或匹配语义漂移。
  - 修复建议：避免使用反斜杠作为 escape 字符。改用普通 ASCII 字符如 `!`：`builder.Expr("lower_tag_name LIKE ? ESCAPE '!'", pattern)`，并定义转义顺序为 `!` → `!!`、`%` → `!%`、`_` → `!_`。或者使用 `ESCAPE ?` 参数化 escape 字符，但必须确认 Gitea 支持的 MySQL/PostgreSQL/SQLite 均可用并有测试覆盖。优先建议 `!`，实现最少、SQL 字面量无歧义。

### P2

- [x] 04-P2-01 255 长度截断需要明确按字符而不是字节
  - 现状：Task 2.1 写"超过 255 字符则截断到 255 字符"，但没有规定实现方式。若实现者用 `keyword[:255]`，中文或 emoji 等多字节 UTF-8 输入可能被截断成非法字符串。
  - 第一性原理：用户输入边界处理不能破坏编码合法性。搜索框的 `maxlength=255` 是字符概念，服务端应保持一致。
  - 修复建议：任务中补一句：截断必须使用 UTF-8/rune 安全方式，或复用项目已有字符串截断 helper，避免按字节切片。

## 第4轮 审查开发者回复

### P1

- 04-P1-01 `ESCAPE '\\\\'` 仍存在跨数据库 SQL 字面量歧义
  - ✅采纳：已修正。Go 字符串 `"\\\\\\\\"` 最终 SQL 中变成 `ESCAPE '\\'`，PostgreSQL 会解释为两个反斜杠字符，MySQL 解释为一个，语义不一致。已将 ESCAPE 字符改为 `!`（普通 ASCII 字符），转义序列更新为 `!` → `!!`、`%` → `!%`、`_` → `!_`。`builder.Expr` 改为 `ESCAPE '!'`。设计 D4、Task 1.2、Task 5.1、spec.md 均已同步更新。

### P2

- 04-P2-01 255 长度截断需要明确按字符而不是字节
  - ✅采纳：已修正。Task 2.1 增加明确要求：截断必须使用 UTF-8/rune 安全方式（如 `[]rune(keyword)[:255]`），禁止按字节切片 `keyword[:255]`，避免中文/emoji 等多字节字符被截断成非法 UTF-8 字符串。spec.md 也新增了"超长搜索关键词处理"场景。
