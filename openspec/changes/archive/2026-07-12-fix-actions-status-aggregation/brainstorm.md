# Brainstorm: Fix Gitea Actions Status Aggregation

## 背景

在 Gitea Actions 中，`aggregateJobStatus` 函数负责聚合所有 job 的状态以显示整体运行结果。当前实现存在三个问题：

1. **Cancelled 被当作 Failure**: 函数将 `StatusCancelled` 归类为 `hasFailure`，导致 Cancelled 状态被显示为 Failure
2. **All-Skipped 被显示为 Success**: 函数没有检查 Skipped 状态，导致所有 job 都被跳过时显示为 Success
3. **Blocked 被显示为 Running**: 函数没有检查 Blocked 状态，导致 Blocked 状态被显示为 Running

## 决策链

### Q1: 修复方案的优先级顺序

**问题**: 根因分析文档建议的优先级顺序是：Failure > Cancelled > Skipped > Blocked > Success > Waiting > Running。这个优先级顺序是否符合预期？

**决策**: 是，符合预期。按根因分析文档建议的优先级顺序实现。

### Q2: 修复范围

**问题**: 除了修复 `aggregateJobStatus` 函数，是否还需要更新 `run_list.go` 中的 `GetStatusInfoList` 函数？

**决策**: 同时更新 `GetStatusInfoList`。该函数目前也只硬编码了 4 种状态（Success, Failure, Waiting, Running），需要保持一致性。

### Q3: 实现方案

**问题**: 有 3 种实现方案：
- **方案 A: 直接实现** - 直接更新函数，添加多个变量按优先级返回状态
- **方案 B: 辅助函数重构** - 提取辅助函数使主函数更简洁
- **方案 C: 优先级表驱动** - 定义优先级表，按优先级顺序检查状态

**决策**: 选择方案 C（优先级表驱动）。可扩展性强，逻辑清晰。

## 设计取舍

### 架构设计

**整体思路**: 使用优先级表驱动的方式重构 `aggregateJobStatus` 函数，按优先级顺序检查状态。

**修改范围**:
1. `models/actions/run_job.go` - 重构 `aggregateJobStatus` 函数
2. `models/actions/run_list.go` - 更新 `GetStatusInfoList` 函数
3. `models/actions/run_job_test.go` - 添加单元测试

**设计原则**:
- 优先级表定义清晰，易于扩展
- 每个状态检查逻辑独立，可复用
- 保持向后兼容，不改变现有接口

### 组件设计

**新增辅助函数**:
```go
// hasStatus 检查是否有任何 job 具有指定状态
func hasStatus(jobs []*ActionRunJob, status Status) bool

// allHaveStatus 检查是否所有 job 都具有指定状态
func allHaveStatus(jobs []*ActionRunJob, status Status) bool
```

**优先级表定义**:
```go
var statusPriority = []struct {
    status Status
    check  func([]*ActionRunJob) bool
}{
    {StatusFailure, func(jobs []*ActionRunJob) bool {
        return hasStatus(jobs, StatusFailure)
    }},
    {StatusCancelled, func(jobs []*ActionRunJob) bool {
        return hasStatus(jobs, StatusCancelled)
    }},
    {StatusSkipped, func(jobs []*ActionRunJob) bool {
        return allHaveStatus(jobs, StatusSkipped)
    }},
    {StatusBlocked, func(jobs []*ActionRunJob) bool {
        return hasStatus(jobs, StatusBlocked) &&
               !hasStatus(jobs, StatusFailure) &&
               !hasStatus(jobs, StatusCancelled)
    }},
    {StatusSuccess, func(jobs []*ActionRunJob) bool {
        return allHaveStatus(jobs, StatusSuccess)
    }},
    {StatusWaiting, func(jobs []*ActionRunJob) bool {
        return allHaveStatus(jobs, StatusWaiting)
    }},
}
```

**重构后的 `aggregateJobStatus` 函数**:
```go
func aggregateJobStatus(jobs []*ActionRunJob) Status {
    for _, p := range statusPriority {
        if p.check(jobs) {
            return p.status
        }
    }
    return StatusRunning
}
```

### 数据流设计

**状态聚合流程**:
1. 接收 jobs 切片作为输入
2. 遍历优先级表（按优先级从高到低）
3. 对每个优先级项，检查对应的条件函数
4. 如果条件满足，立即返回对应状态
5. 如果所有条件都不满足，返回 StatusRunning（默认状态）

**优先级检查顺序**:
- Priority 1: StatusFailure → 有任何 job 失败
- Priority 2: StatusCancelled → 有任何 job 被取消
- Priority 3: StatusSkipped → 所有 job 都被跳过
- Priority 4: StatusBlocked → 有 job 被阻塞且无失败/取消
- Priority 5: StatusSuccess → 所有 job 都成功
- Priority 6: StatusWaiting → 所有 job 都在等待
- Default: StatusRunning → 其他情况

**调用链保持不变**:
```
UpdateRunJob → aggregateJobStatus(jobs) → Status
```

### 测试设计

**单元测试用例**:
```go
func TestAggregateJobStatus(t *testing.T) {
    tests := []struct {
        name     string
        jobs     []*ActionRunJob
        expected Status
    }{
        {
            name: "all skipped",
            jobs: []*ActionRunJob{
                {Status: StatusSkipped},
                {Status: StatusSkipped},
            },
            expected: StatusSkipped,
        },
        {
            name: "has cancelled",
            jobs: []*ActionRunJob{
                {Status: StatusSuccess},
                {Status: StatusCancelled},
            },
            expected: StatusCancelled,
        },
        {
            name: "all blocked",
            jobs: []*ActionRunJob{
                {Status: StatusBlocked},
                {Status: StatusBlocked},
            },
            expected: StatusBlocked,
        },
        {
            name: "mixed blocked and running",
            jobs: []*ActionRunJob{
                {Status: StatusBlocked},
                {Status: StatusRunning},
            },
            expected: StatusRunning,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := aggregateJobStatus(tt.jobs)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**测试文件位置**: `models/actions/run_job_test.go`

### 错误处理设计

**边界情况处理**:
1. **空切片**: 如果 `jobs` 为空，优先级表中的所有条件都不满足，返回 `StatusRunning`（默认状态）
2. **单个 job**: 优先级表正确处理单个 job 的情况
3. **所有状态相同**: 优先级表正确处理所有 job 状态相同的情况
4. **混合状态**: 优先级表按优先级顺序检查，确保高优先级状态优先返回

**向后兼容性**:
- 函数签名保持不变
- 返回值类型保持不变
- 调用方无需修改

## 参考资料

- Status 类型定义: `models/actions/status.go`
- 聚合函数: `models/actions/run_job.go:155-180`
- 前端组件: `web_src/js/components/ActionRunStatus.vue`
- 模板文件: `templates/repo/actions/view.tmpl`
- 根因分析文档: `docs/root-cause-analysis.md`
