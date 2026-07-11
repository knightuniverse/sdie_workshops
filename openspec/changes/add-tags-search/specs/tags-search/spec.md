## ADDED Requirements

### Requirement: 标签搜索功能
系统 SHALL 在 Tags 页面提供按名称搜索标签的功能，允许用户通过关键词快速过滤标签列表。

#### Scenario: 用户搜索标签
- **WHEN** 用户在 Tags 页面的搜索框中输入关键词并提交表单
- **THEN** 系统 SHALL 显示标签名称包含该关键词的标签列表

#### Scenario: 搜索结果为空
- **WHEN** 用户搜索的关键词不匹配任何标签名称
- **THEN** 系统 SHALL 显示空列表，并保持搜索框中的关键词

#### Scenario: 清除搜索条件
- **WHEN** 用户清空搜索框并提交表单
- **THEN** 系统 SHALL 显示所有标签（恢复默认列表）

### Requirement: 搜索参数处理
系统 SHALL 使用 GET 参数 `q` 传递搜索关键词，支持书签和分享搜索结果。

#### Scenario: 搜索参数传递
- **WHEN** 用户提交搜索表单
- **THEN** 系统 SHALL 将搜索关键词作为 `q` 参数添加到 URL 中

#### Scenario: 搜索参数回填
- **WHEN** 页面加载时 URL 包含 `q` 参数
- **THEN** 系统 SHALL 将 `q` 参数的值预填到搜索框中

### Requirement: 搜索结果分页
系统 SHALL 在搜索结果中支持分页，分页总数 SHALL 反映搜索结果的实际数量。

#### Scenario: 搜索结果分页
- **WHEN** 用户搜索的关键词匹配多个标签
- **THEN** 系统 SHALL 按分页显示搜索结果，分页总数 SHALL 等于匹配的标签总数
