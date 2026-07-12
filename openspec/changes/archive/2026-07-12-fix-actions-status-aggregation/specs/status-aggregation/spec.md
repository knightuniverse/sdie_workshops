## ADDED Requirements

### Requirement: 状态聚合逻辑

`aggregateJobStatus` 函数 SHALL 使用单次遍历收集布尔量，然后按以下顺序返回状态：

**终态判断**（当所有 job 都处于 IsDone 状态时）：
1. StatusFailure（有任何 job 失败）
2. StatusCancelled（有任何 job 被取消）
3. StatusSkipped（所有 job 都被跳过）
4. StatusSuccess（默认终态，如 success+skipped 混合）

**活跃态判断**（当存在非终态 job 时）：
5. StatusWaiting（所有非终态 job 都在等待；已完成的 job 不影响此判断，例如 success+waiting 返回 StatusWaiting）
6. StatusBlocked（所有 job 都被阻塞，且无 waiting/running）
7. StatusRunning（默认状态）

**空列表处理**：
- 当 jobs 为空时，返回 StatusRunning（防御性编程，历史上不会出现空 jobs）

#### Scenario: 所有 job 都被跳过
- **WHEN** 所有 job 的状态都是 StatusSkipped
- **THEN** 返回 StatusSkipped

#### Scenario: 有 job 被取消
- **WHEN** 任何 job 的状态是 StatusCancelled（且无 job 失败）
- **THEN** 返回 StatusCancelled

#### Scenario: 混合终态（success + skipped）
- **WHEN** 所有 job 都处于终态（Success 或 Skipped），且无 Failure 或 Cancelled
- **THEN** 返回 StatusSuccess

#### Scenario: 所有 job 都被阻塞
- **WHEN** 所有 job 的状态都是 StatusBlocked
- **THEN** 返回 StatusBlocked

#### Scenario: 混合阻塞和运行状态
- **WHEN** 有 job 状态为 StatusBlocked，也有 job 状态为 StatusRunning
- **THEN** 返回 StatusRunning

#### Scenario: 混合等待和阻塞状态
- **WHEN** 有 job 状态为 StatusWaiting，也有 job 状态为 StatusBlocked
- **THEN** 返回 StatusRunning

#### Scenario: success + waiting (done jobs do not break allWaiting)
- **WHEN** one job is StatusSuccess and another is StatusWaiting
- **THEN** returns StatusWaiting (the done job is ignored by the allWaiting check)

#### Scenario: skipped + waiting (done jobs do not break allWaiting)
- **WHEN** one job is StatusSkipped and another is StatusWaiting
- **THEN** returns StatusWaiting (the done job is ignored by the allWaiting check)

#### Scenario: 空 job 列表
- **WHEN** jobs 切片为空
- **THEN** 返回 StatusRunning

### Requirement: GetStatusInfoList 状态支持

`GetStatusInfoList` 函数 SHALL 支持所有可显示的状态（Success、Failure、Cancelled、Skipped、Waiting、Running、Blocked），共 7 种。StatusUnknown（值为 0）是内部哨兵状态，不参与 UI 筛选，聚合逻辑中未匹配的状态会 fallback 到 StatusRunning。列表排序为 UI 筛选选项顺序，与聚合优先级无关。

#### Scenario: 查询所有状态
- **WHEN** 调用 GetStatusInfoList 函数
- **THEN** 返回包含所有可显示状态的状态信息列表
