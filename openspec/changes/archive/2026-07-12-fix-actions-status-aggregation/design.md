## Context

在 Gitea Actions 中，`aggregateJobStatus` 函数负责聚合所有 job 的状态以显示整体运行结果。当前实现存在三个问题：

1. **Cancelled 被当作 Failure**: 函数将 `StatusCancelled` 归类为 `hasFailure`，导致 Cancelled 状态被显示为 Failure
2. **All-Skipped 被显示为 Success**: 函数没有检查 Skipped 状态，导致所有 job 都被跳过时显示为 Success
3. **Blocked 被显示为 Running**: 函数没有检查 Blocked 状态，导致 Blocked 状态被显示为 Running

**当前实现**（`models/actions/run_job.go:155-180`）：
- 只处理了 4 种状态（Failure、Success、Waiting、Running）
- 使用三个布尔变量（allDone、allWaiting、hasFailure）进行状态判断
- 忽略了 Cancelled、Skipped、Blocked 三种状态

**Status 类型定义**（`models/actions/status.go:13-24`）：
- Status 是 int 类型，定义了 8 种状态
- `IsDone()` 方法只包含 Success、Failure、Cancelled、Skipped

**调用链**：
```
UpdateRunJob → aggregateJobStatus(jobs) → Status
```

**前端已支持所有 8 种状态**，问题完全在后端聚合逻辑。

## Goals / Non-Goals

**Goals:**
- 修复 `aggregateJobStatus` 函数，正确处理所有 8 种状态
- 按分支规则返回状态（非线性优先级）：终态分支（failure > cancelled > skipped > success）和活跃态分支（waiting > blocked > running），两个分支互斥
- 更新 `GetStatusInfoList` 函数以保持一致性
- 添加单元测试确保修复正确

**Non-Goals:**
- 不修改前端代码（前端已正确支持所有状态）
- 不改变函数签名或返回值类型（保持向后兼容）
- 不重构整个 Actions 模块（只修复聚合逻辑）

## Decisions

### D1: 状态判断顺序

- **选择**: 使用分支规则而非线性优先级。终态分支（所有 job 都 IsDone 时）按 failure > cancelled > skipped > success 返回；活跃态分支按 waiting > blocked > running 返回
- **理由**: 两个分支互斥（allDone 为 true 时不会进入活跃态分支），比线性优先级更精确地表达业务语义
- **已考虑 alternative**: 线性全局优先级（Failure > Cancelled > Skipped > Blocked > Success > Waiting > Running），但 blocked 和 waiting 的判断需要区分"是否所有 job 都已终止"

### D2: 修复范围

- **选择**: 同时修复 `aggregateJobStatus` 和 `GetStatusInfoList` 函数
- **理由**: `GetStatusInfoList` 也硬编码了 4 种状态，需要保持一致性
- **已考虑 alternative**: 只修复 `aggregateJobStatus`，但会导致状态不一致

### D3: 实现方案

- **选择**: 单次遍历 + 条件判断方案（基于根因分析文档的方案）
- **理由**: 一次遍历收集所有布尔量，按明确顺序判断，逻辑清晰且无性能浪费
- **已考虑 alternative**:
  - 方案 A（优先级表驱动）：可扩展性强，但 jobs 数量通常很小（1-10），且表驱动方案下 blocked/skipped/success 的条件语义容易出错（如 success+skipped 被误判为 running）
  - 方案 B（辅助函数重构）：可复用性好，但增加函数数量，且不解决核心逻辑问题

### D4: 辅助函数设计

- **选择**: 不新增辅助函数，直接在 `aggregateJobStatus` 中使用单次遍历收集布尔量
- **理由**: 聚合逻辑本质上是一次归约，一次遍历更直观；辅助函数导致多次扫描 jobs 切片，且 blocked/skipped 等条件的语义需要上下文信息（如 allDone），辅助函数无法优雅获取
- **已考虑 alternative**: 使用 `hasStatus` 和 `allHaveStatus` 辅助函数，但 blocked 规则需要复合条件，辅助函数无法简洁表达

### D5: 测试策略

- **选择**: 添加单元测试，覆盖根因分析文档中的测试用例
- **理由**: 确保修复正确，防止回归
- **已考虑 alternative**: 不添加测试，但风险较高

## Risks / Trade-offs

**[Risk] 边界情况处理**:
- 空切片：如果 `jobs` 为空，函数开头显式返回 `StatusRunning`（历史上不会出现空 jobs，这是防御性编程）
- 单个 job：单次遍历正确处理单个 job 的情况
- 所有状态相同：所有 job 终态时按 failure > cancelled > skipped > success 的子优先级返回
- 混合终态（如 success+skipped）：通过 `allDone` 判断确认所有 job 已终止，然后按子优先级返回
- 混合活跃态（如 waiting+blocked）：blocked 不单独触发返回，只有当没有 waiting/running 时才可能返回 blocked
- StatusUnknown：StatusUnknown（值为 0）是内部哨兵状态，不参与 UI 筛选（GetStatusInfoList 返回 7 种可显示状态），聚合逻辑中未匹配的状态 fallback 到 StatusRunning

**[Risk] 向后兼容性**:
- 函数签名保持不变
- 返回值类型保持不变
- 调用方无需修改

**[Trade-off] 单次遍历 vs 辅助函数多次扫描**:
- 选择单次遍历：性能更优，语义更清晰，一次收集所有布尔量后按序判断
- 代价：代码稍长，但状态机逻辑本身需要明确的判断顺序，辅助函数无法简化核心逻辑

## Migration Plan

N/A — 本 change 不涉及部署变更。修改仅限于模型层代码，不影响 API 或数据库。

## Open Questions

无。所有设计决策已在 brainstorm 阶段确认。
