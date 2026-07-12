## Why

在 Gitea Actions 中，`aggregateJobStatus` 函数负责聚合所有 job 的状态以显示整体运行结果。当前实现存在三个问题：

1. **Cancelled 被当作 Failure**: 函数将 `StatusCancelled` 归类为 `hasFailure`，导致 Cancelled 状态被显示为 Failure
2. **All-Skipped 被显示为 Success**: 函数没有检查 Skipped 状态，导致所有 job 都被跳过时显示为 Success
3. **Blocked 被显示为 Running**: 函数没有检查 Blocked 状态，导致 Blocked 状态被显示为 Running

这些问题导致用户看到的状态与实际状态不一致，影响用户体验。前端已正确支持所有 8 种状态，问题完全在后端聚合逻辑。

## What Changes

**aggregateJobStatus 函数重构**
- From: 只处理 4 种状态（Failure、Success、Waiting、Running），使用三个布尔变量进行状态判断
- To: 使用单次遍历收集布尔量，按分支规则处理所有 8 种状态（终态: failure > cancelled > skipped > success; 活跃态: waiting > blocked > running）
- Reason: 修复状态聚合逻辑，正确处理 Cancelled、Skipped、Blocked 状态
- Impact: 非破坏性变更，函数签名和返回值类型保持不变

**GetStatusInfoList 函数更新**
- From: 硬编码 4 种状态（Success、Failure、Waiting、Running）
- To: 支持所有 7 种可显示状态（Success、Failure、Cancelled、Skipped、Waiting、Running、Blocked），StatusUnknown 是内部哨兵状态不参与 UI 筛选
- Reason: 保持状态处理的一致性
- Impact: 非破坏性变更，函数签名保持不变

**单元测试添加**
- From: 无单元测试
- To: 添加单元测试覆盖所有状态组合
- Reason: 确保修复正确，防止回归
- Impact: 无影响，只添加测试代码

## Capabilities

### New Capabilities
- `status-aggregation`: 修复 Gitea Actions 状态聚合逻辑，正确处理所有 8 种状态

### Modified Capabilities
无。本次修改不涉及现有 capability 的需求变更。

## Impact

**受影响的代码**:
- `models/actions/run_job.go`: 重构 `aggregateJobStatus` 函数
- `models/actions/run_list.go`: 更新 `GetStatusInfoList` 函数
- `models/actions/run_job_test.go`: 添加单元测试

**受影响的 API**:
无。本次修改不影响任何 API。

**受影响的依赖**:
无。本次修改不引入新的依赖。

**受影响的系统**:
- Gitea Actions 状态显示：用户将看到正确的状态（Cancelled、Skipped、Blocked）
