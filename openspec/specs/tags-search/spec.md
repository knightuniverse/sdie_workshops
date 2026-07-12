## ADDED Requirements

### Requirement: 标签搜索功能
系统 SHALL 在 Tags 页面提供按名称搜索标签的功能，允许用户通过关键词快速过滤标签列表。搜索 SHALL 仅匹配 Git tag name（大小写不敏感），不匹配 release title、note、commit SHA 或作者。

#### Scenario: 用户搜索标签
- **WHEN** 用户在 Tags 页面的搜索框中输入关键词并提交表单
- **THEN** 系统 SHALL 显示标签名称（tag_name）包含该关键词的标签列表（大小写不敏感匹配）

#### Scenario: 搜索结果为空
- **WHEN** 用户搜索的关键词不匹配任何标签名称
- **THEN** 系统 SHALL 显示空列表，并保持搜索框中的关键词

#### Scenario: 清除搜索条件
- **WHEN** 用户清空搜索框并提交表单
- **THEN** 系统 SHALL 显示所有标签（恢复默认列表）

#### Scenario: 搜索关键词包含 LIKE 特殊字符
- **WHEN** 用户输入的搜索关键词包含 `%` 或 `_` 等 SQL LIKE 通配符字符
- **THEN** 系统 SHALL 将这些字符作为普通字符进行字面匹配，而非作为通配符处理。例如搜索 `v1_0` SHALL 仅匹配包含字面 `v1_0` 的标签名称，不匹配 `v1a0`；搜索 `%` SHALL 仅匹配包含字面 `%` 的标签名称，不匹配所有标签。系统 SHALL 使用 `!` 作为 SQL LIKE escape 字符（`ESCAPE '!'`），确保跨数据库（MySQL、PostgreSQL、SQLite）行为一致

### Requirement: 搜索参数处理
系统 SHALL 使用 GET 参数 `q` 传递搜索关键词，支持书签和分享搜索结果。

#### Scenario: 搜索参数传递
- **WHEN** 用户提交搜索表单
- **THEN** 系统 SHALL 将搜索关键词作为 `q` 参数添加到 URL 中

#### Scenario: 搜索参数回填
- **WHEN** 页面加载时 URL 包含 `q` 参数
- **THEN** 系统 SHALL 将 `q` 参数的值预填到搜索框中

#### Scenario: 超长搜索关键词处理
- **WHEN** 用户通过 URL 参数传递超过 255 个 Unicode 字符的搜索关键词
- **THEN** 系统 SHALL 在服务端按 Unicode 字符（rune）截断到 255 字符，而非按字节截断，确保多字节字符（中文、emoji 等）不会被截断成非法字符串

### Requirement: 搜索结果分页
系统 SHALL 在搜索结果中支持分页，分页总数 SHALL 反映搜索结果的实际数量。

#### Scenario: 搜索结果分页
- **WHEN** 用户搜索的关键词匹配多个标签
- **THEN** 系统 SHALL 按分页显示搜索结果，分页总数 SHALL 等于匹配的标签总数
