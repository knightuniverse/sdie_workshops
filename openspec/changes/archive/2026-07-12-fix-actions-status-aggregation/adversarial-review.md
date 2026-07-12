# OpenSpec 对抗式审查文档

**SESSION_ID**: 019f543d-9ebe-7391-9bfe-1445f645632a
**审查范围**: openspec/changes/fix-actions-status-aggregation

---

# OpenSpec任务文档审查任务清单

**审查时间**: 2026-07-12
**审查范围**: openspec/changes/fix-actions-status-aggregation
**审查人**: Codex

## 第1轮 审查

### P0

- [x] 01-P0-01 聚合规则会把 `success + skipped` 误判为 `running`
  - 现状：计划只在 `allHaveStatus(jobs, StatusSkipped)` 时返回 `StatusSkipped`，只在 `allHaveStatus(jobs, StatusSuccess)` 时返回 `StatusSuccess`。因此 `[success, skipped]` 不匹配任何终态规则，最终返回 `StatusRunning`。
  - 第一性原理：run 的终态应由所有 job 是否已终止决定；`StatusSkipped.IsDone() == true`。所有 job 都是 done 时返回 running 会破坏生命周期：`run.Stopped` 不会设置，仓库 closed run 计数不会更新，列表会持续显示未完成。
  - 修复建议：保留"所有 done"的总判断：若存在 failure 返回 failure；否则存在 cancelled 返回 cancelled；否则若所有 skipped 返回 skipped；否则返回 success。不要用 `all success` 代替"所有 done 且无失败/取消"的语义。

- [x] 01-P0-02 `StatusBlocked` 规则与测试用例自相矛盾
  - 现状：计划中的规则是 `hasStatus(blocked) && !hasStatus(failure) && !hasStatus(cancelled)`，这会让 `[blocked, running]` 返回 `StatusBlocked`；但测试用例 `"mixed blocked and running"` 期望 `StatusRunning`。
  - 第一性原理：方案必须是可执行、可验证的。同一计划中实现与测试不一致，说明状态优先级没有被精确定义，落地后必然失败或被随意改动。
  - 修复建议：明确 blocked 的条件。若目标只是修复"全部 blocked 显示 running"，则使用 `allHaveStatus(jobs, StatusBlocked)`。若目标是"无 active job 且存在 blocked"，则规则应排除 running/waiting，并覆盖 skipped/success 混合场景，测试同步补齐。

- [x] 01-P0-03 任意 blocked 优先会把已有可调度 job 的 run 错标为 blocked
  - 现状：Gitea 创建 run 时，带 `needs` 的 job 初始为 `StatusBlocked`，无依赖 job 初始为 `StatusWaiting`。计划会把 `[waiting, blocked]` 聚合为 `StatusBlocked`。
  - 第一性原理：run 级状态应表达整体可执行性。只要存在 waiting/running job，系统仍可调度或正在执行，整体不应被标成 blocked；否则 UI/API 会误导用户，且筛选结果不再代表真实运行状态。
  - 修复建议：blocked 只应在没有 waiting/running 且确实存在 blocked 时返回，或者更保守地只在所有 job 都 blocked 时返回。至少补测试：`waiting + blocked => waiting/running`、`running + blocked => running`、`success + blocked => blocked 或 running`，并在设计中说明语义。

### P1

- [x] 01-P1-01 计划没有覆盖空 job 列表的语义
  - 现状：`allHaveStatus` 对空切片会返回 `true`，因此空 jobs 会命中第一个 `allHaveStatus` 类规则的风险；当前计划的顺序下 `StatusSkipped` 的 `allHaveStatus` 会让空列表返回 skipped。
  - 第一性原理：空集合上的 all 判断是真值，但业务状态不能直接继承数学真值。run 没有 job 是异常、unknown 还是 success，必须显式定义。
  - 修复建议：在 `aggregateJobStatus` 开头增加 `if len(jobs) == 0 { return StatusUnknown /* 或保持现有约定 */ }`，并补测试。若历史上不会出现空 jobs，也应在注释中写明不变量。

- [x] 01-P1-02 `GetStatusInfoList` 扩展状态后未说明筛选语义和排序依据
  - 现状：计划把 cancelled/skipped/blocked 加入运行列表筛选，但排序为 `Success, Failure, Cancelled, Skipped, Waiting, Running, Blocked`，与文档里的优先级 `Failure > Cancelled > Skipped > Blocked > Success > Waiting > Running` 不一致。
  - 第一性原理：列表筛选是用户面对的查询契约，不只是 UI 枚举。排序和可选状态应稳定、可解释，并与聚合状态保持一致。
  - 修复建议：明确 `GetStatusInfoList` 的排序是 UI 排序还是聚合优先级排序。若注释仍写 "same as those in aggregateJobStatus"，应与实际可返回状态完全一致，并避免让读者误以为顺序就是优先级。

- [x] 01-P1-03 测试覆盖不足，无法防止核心回归
  - 现状：测试只覆盖 all skipped、has cancelled、all blocked、blocked+running。缺少 failure 优先级、success+skipped、cancelled+failure、waiting+blocked、waiting-only、running-only、空列表等关键组合。
  - 第一性原理：这是状态机逻辑，风险来自组合爆炸。只测单点 happy path 不能证明优先级表正确。
  - 修复建议：用表驱动补全状态矩阵，至少覆盖：`failure+cancelled => failure`、`success+skipped => success`、`all waiting => waiting`、`waiting+blocked`、`running+blocked`、`all blocked`、`empty jobs`。

### P2

- [ ] 01-P2-01 辅助函数会导致多次遍历，且 blocked 规则重复扫描
  - 现状：优先级表每条规则都扫描 jobs，blocked 规则还扫描多次。job 数量通常不大，但这是热路径上的状态更新逻辑。
  - 第一性原理：状态聚合本质上是一次归约，应该一次遍历得到必要布尔量，降低认知成本和性能浪费。
  - 修复建议：优先使用单次循环统计 `allDone/allWaiting/allSkipped/allBlocked/hasFailure/hasCancelled/hasRunning/hasWaiting`，然后按明确顺序返回。若坚持表驱动，先预计算 summary，再让规则读取 summary。

- [ ] 01-P2-02 新增 `run_job_test.go` 需要确认包名和依赖风格
  - 现状：`models/actions` 现有测试文件很少，计划直接使用 `assert.Equal`，但没有说明导入和包内私有函数测试策略。
  - 第一性原理：测试必须能直接运行，并符合仓库现有风格。`aggregateJobStatus` 是未导出函数，应使用 `package actions` 包内测试。
  - 修复建议：新增测试文件使用 `package actions`，导入 `testing` 和 `github.com/stretchr/testify/assert`，并运行 `go test ./models/actions -run TestAggregateJobStatus` 验证。

---

# 第1轮 审查开发者回复

## P0

- 01-P0-01 聚合规则会把 `success + skipped` 误判为 `running`
  - **采纳**：优先级表方案确实存在此 bug。`[success, skipped]` 不匹配任何 `allHaveStatus` 规则，最终返回 `StatusRunning`。根因是用 `allHaveStatus(jobs, StatusSkipped)` 代替了"所有 job 都 done 且无 failure/cancelled"的语义。采纳修复建议：在 `aggregateJobStatus` 中首先判断 `allDone`（所有 job 都处于终态），然后在终态分支内按 failure > cancelled > skipped > success 的顺序返回。具体修改：
    - 优先级表中 Skipped 和 Success 的检查从 `allHaveStatus(jobs, StatusSkipped)` / `allHaveStatus(jobs, StatusSuccess)` 改为基于 `allDone` 总判断后按子优先级返回
    - 更新 design.md Decisions、spec.md、tasks.md 中的相关代码
    - 补充测试用例 `success+skipped => success`（两个 job 都是终态，无 failure/cancelled，应返回 success）

- 01-P0-02 `StatusBlocked` 规则与测试用例自相矛盾
  - **采纳**：优先级表中 Blocked 的条件 `hasStatus(jobs, StatusBlocked) && !hasStatus(jobs, StatusFailure) && !hasStatus(jobs, StatusCancelled)` 与测试用例 `"mixed blocked and running"` 期望 `StatusRunning` 直接矛盾。这是同一文档内的不一致，必须修复。采纳修复建议：将 Blocked 条件收紧为 `allHaveStatus(jobs, StatusBlocked)`（即只有所有 job 都 blocked 时才返回 blocked），并在测试用例中补充 `blocked+running => running`。具体修改：
    - Blocked 规则改为 `allHaveStatus(jobs, StatusBlocked)`
    - 更新 spec.md 中 Scenario "混合阻塞和运行状态" 的表述
    - 更新 tasks.md 中 2.5 的描述

- 01-P0-03 任意 blocked 优先会把已有可调度 job 的 run 错标为 blocked
  - **采纳**：这是 01-P0-02 的逻辑延伸。Gitea 创建 run 时，带 `needs` 的 job 初始为 `StatusBlocked`，无依赖 job 初始为 `StatusWaiting`。如果任意 blocked 优先，`[waiting, blocked]` 会被聚合为 `StatusBlocked`，但此时系统仍有可调度 job（waiting），整体不应标为 blocked。采纳更保守的方案：Blocked 只在 `allHaveStatus(jobs, StatusBlocked)` 时返回，即只有所有 job 都 blocked 时才标为 blocked。这样 `[waiting, blocked]` 返回 `StatusWaiting`（因为 allWaiting 不成立，allDone 不成立，最终 fallback 到 running，但由于有 waiting job，应由 allWaiting 判断处理——实际上 `[waiting, blocked]` 中 allWaiting 为 false（因为 blocked 不是 waiting），allDone 为 false（因为 blocked 和 waiting 都不是 done），blocked 条件不满足（因为不是 all blocked），最终返回 StatusRunning，这是正确的）。补充测试用例：`waiting+blocked => running`。

## P1

- 01-P1-01 计划没有覆盖空 job 列表的语义
  - **部分采纳**：审查指出 `allHaveStatus` 对空切片返回 `true`，这在数学上正确但业务上需显式处理。当前设计文档 Risks 中已说明"空切片：如果 `jobs` 为空...返回 `StatusRunning`（默认状态）"。但优先级表方案下，空 jobs 实际会命中 Skipped 的 `allHaveStatus` 检查并返回 `StatusSkipped`（因为 `allHaveStatus([] , StatusSkipped)` 返回 true），与设计文档的意图不符。采纳修复建议：在 `aggregateJobStatus` 开头增加空列表检查。鉴于历史上不会出现空 jobs（`GetRunJobsByRunID` 总是返回至少一个 job），采用防御性编程：`if len(jobs) == 0 { return StatusRunning }`（与设计文档 Risk 中的约定保持一致），并补测试用例。具体修改：
    - 在 `aggregateJobStatus` 函数开头添加 `if len(jobs) == 0 { return StatusRunning }`
    - 更新 design.md Risks 部分的描述以明确此不变量
    - 补充测试用例 `empty jobs => StatusRunning`

- 01-P1-02 `GetStatusInfoList` 扩展状态后未说明筛选语义和排序依据
  - **部分采纳**：审查指出 `GetStatusInfoList` 的排序与聚合优先级不一致。实际上 `GetStatusInfoList` 用于前端筛选下拉框的选项列表，其排序应以 UI 用户体验为准（按常见程度排列），而非聚合优先级。当前计划中的排序 `Success, Failure, Cancelled, Skipped, Waiting, Running, Blocked` 是合理的 UI 排序。但注释 `"same as those in aggregateJobStatus"` 确实容易误导。采纳修复建议：更新注释，明确说明这是 UI 筛选选项列表，排序与聚合优先级无关。具体修改：
    - 更新 plan.md 中 Task 4 的注释为 `// UI filter options for run status, order reflects display preference, not aggregation priority`
    - 在 design.md 中补充说明 `GetStatusInfoList` 的排序是 UI 排序，与聚合优先级独立

- 01-P1-03 测试覆盖不足，无法防止核心回归
  - **采纳**：当前计划只覆盖 4 个测试用例，对于状态机逻辑而言覆盖不足。采纳修复建议，补全状态矩阵。具体修改：
    - 在 tasks.md 和 plan.md 的 Task 5 中补充以下测试用例：
      - `failure+cancelled => failure`（failure 最高优先级）
      - `failure+success => failure`
      - `success+skipped => success`（所有终态，无 failure/cancelled）
      - `success+cancelled => cancelled`
      - `all waiting => waiting`
      - `waiting+blocked => running`（有 waiting 且 blocked 不全，fallback to running）
      - `running+blocked => running`
      - `running only => running`
      - `empty jobs => running`
      - `single success => success`
      - `single waiting => waiting`

## P2

- 01-P2-01 辅助函数会导致多次遍历，且 blocked 规则重复扫描
  - **不采纳**：审查指出优先级表每条规则都扫描 jobs，性能不佳。虽然从理论上讲"一次归约"更优，但实际场景中 jobs 数量通常为 1-10 个（一个 workflow 一般不超过 10 个 job），且此函数在 `UpdateRunJob` 中调用，不是热路径上的性能瓶颈。优先级表方案的可读性和可扩展性比微优化更重要。此外，审查本身也承认"job 数量通常不大"。当前方案的代码清晰度足以支撑未来维护，不做提前优化。

- 01-P2-02 新增 `run_job_test.go` 需要确认包名和依赖风格
  - **不采纳**：审查指出需要确认包名和依赖风格。但 plan.md Task 5 Step 1 中已明确指定 `package actions`（包内测试），导入 `testing` 和 `github.com/stretchr/testify/assert`，这与仓库中已有的 `models/actions/utils_test.go` 风格完全一致。此条审查项的信息已在计划中覆盖，无需额外修改。

---

## 第2轮 审查

### P0

- [x] 02-P0-01 计划中的目标代码无法编译
  - 现状：`plan.md` 的 `aggregateJobStatus` 示例代码声明了 `hasBlocked := false`，循环里也只设置 `hasBlocked = true`，但后面判断使用的是 `allBlocked`：`if allBlocked && !hasFailure && !hasCancelled`。`allBlocked` 未声明，`hasBlocked` 未使用，Go 会直接编译失败。
  - 第一性原理：技术计划里的可复制实现必须至少能编译。否则后续所有语义讨论都没有落地基础。
  - 修复建议：删除 `hasBlocked`，改为声明并维护 `allBlocked := true`；循环中加入 `if job.Status != StatusBlocked { allBlocked = false }`。然后保留空列表提前返回，避免空列表下 `allBlocked` 的真值问题。

### P1

- [x] 02-P1-01 多个 OpenSpec 文档仍然互相矛盾
  - 现状：`design.md` 已改为"单次遍历"，但 `proposal.md` 仍写"使用优先级表驱动"；`design.md` 的 Goals/D1 仍写简单优先级 `Failure > Cancelled > Skipped > Blocked > Success > Waiting > Running`，但实际规则是"只有 all skipped 才返回 skipped，success+skipped 返回 success"；`adversarial-review.md` 里 P2-01 仍写"不采纳"，但当前设计实际已经采纳了单次遍历。
  - 第一性原理：OpenSpec 是给实现者执行的契约。契约互相冲突时，实现者会按不同文件做出不同实现，审查无法证明方案稳定。
  - 修复建议：统一所有 artifact：明确实现方式是单次遍历；把"优先级"改写成分支规则，而不是线性全局优先级；将 P2-01 的开发者回复更新为"已由后续设计采纳"或移除过期表述。

- [x] 02-P1-02 `allWaiting` 的真实语义没有被文档和测试锁住
  - 现状：计划代码沿用现有逻辑：done 状态不会破坏 `allWaiting`，所以 `success + waiting` 会返回 `StatusWaiting`。但 spec 写的是"所有 job 都在等待"，这与代码真实语义不一致。
  - 第一性原理：状态机条件必须按真实业务含义命名和测试。这里真实含义更像"所有未完成 job 都在 waiting，且不存在 running/blocked"。
  - 修复建议：把文档改为"所有未完成 job 均为 waiting"或修改代码为严格 all waiting。建议补测试：`success + waiting => waiting`、`skipped + waiting => waiting`，以固定现有兼容语义。

- [x] 02-P1-03 plan 中的提交步骤存在误提交风险
  - 现状：`plan.md` 多处要求 `git add ... && git commit ...`，最后还有 `git add -A`。在可能存在用户未提交改动的工作区中，`git add -A` 会把无关文件一起提交。
  - 第一性原理：实现计划不能要求破坏性或宽泛的版本控制操作。提交边界必须只包含本 change 的文件。
  - 修复建议：从任务步骤中移除强制 commit，或改为"检查 git diff 后仅 stage 本 change 文件"。禁止使用 `git add -A`。

### P2

- [x] 02-P2-01 `StatusUnknown` 的范围说明仍不完整
  - 现状：文档多处写"所有 8 种状态"，但 `GetStatusInfoList` 只返回 7 种，不包含 `StatusUnknown`；聚合逻辑也没有显式定义 unknown，实际会 fallback 到 running。
  - 第一性原理：如果宣称覆盖所有状态，就必须定义每个状态的聚合和展示行为。
  - 修复建议：明确 `StatusUnknown` 是非筛选状态且聚合时按 running fallback，或将其纳入测试和 UI 筛选。文档中避免同时说"8 种状态"和"7 种可显示状态"而不解释差异。

- [x] 02-P2-02 计划要求 Go 代码使用中文注释，与 Gitea 源码风格不一致
  - 现状：`plan.md` 的 Global Constraints 写"使用中文注释和文档"，示例 Go 注释也是中文；但 Gitea 源码注释基本是英文。
  - 第一性原理：代码注释应服从代码库长期维护语言，而不是任务文档语言。
  - 修复建议：文档可以中文，Go 代码注释改为英文，保持与现有 Gitea 代码风格一致。

---

# 第2轮 审查开发者回复

## P0

- 02-P0-01 计划中的目标代码无法编译
  - **采纳**：plan.md 中 `aggregateJobStatus` 示例代码声明了 `hasBlocked := false`，循环中只设置 `hasBlocked = true`，但判断条件使用了未声明的 `allBlocked`。这是代码级错误，会导致 Go 编译失败。采纳修复建议：将 `hasBlocked` 改为 `allBlocked := true`，循环中加入 `if job.Status != StatusBlocked { allBlocked = false }`，同时删除 switch 中的 StatusBlocked case。空列表提前返回已保留，避免空列表下 `allBlocked` 的真值问题。具体修改：
    - plan.md Task 1 Step 1：`hasBlocked := false` → `allBlocked := true`
    - plan.md Task 1 Step 1：循环中新增 `if job.Status != StatusBlocked { allBlocked = false }`
    - plan.md Task 1 Step 1：删除 switch 中的 `case StatusBlocked: hasBlocked = true`

## P1

- 02-P1-01 多个 OpenSpec 文档仍然互相矛盾
  - **采纳**：design.md 已改为"单次遍历"，但 proposal.md 仍写"使用优先级表驱动"；design.md Goals/D1 仍写线性优先级 `Failure > Cancelled > Skipped > Blocked > Success > Waiting > Running`，但实际规则是分支判断（终态分支和活跃态分支互斥）。采纳修复建议：统一所有 artifact。具体修改：
    - proposal.md：`使用优先级表驱动` → `使用单次遍历收集布尔量，按分支规则处理`
    - design.md Goals：`按优先级顺序返回状态` → `按分支规则返回状态（非线性优先级）`
    - design.md D1：标题从"优先级顺序"改为"状态判断顺序"，描述改为分支规则（终态分支和活跃态分支互斥）
    - 保留第1轮审查中 P2-01 的"不采纳"回复不变，因为那是第1轮审查的记录，反映当时的决策过程

- 02-P1-02 `allWaiting` 的真实语义没有被文档和测试锁住
  - **采纳**：代码中 `allWaiting` 的判断条件是 `job.Status != StatusWaiting && !job.Status.IsDone()`，即已完成的 job（IsDone 为 true）不会破坏 allWaiting。这意味着 `success + waiting` 会返回 StatusWaiting。但 spec 写的是"所有 job 都在等待"，语义不一致。采纳修复建议：将文档改为"所有非终态 job 都在等待"，并补充测试用例固定现有兼容语义。具体修改：
    - spec.md：活跃态判断第 5 条改为 `StatusWaiting（所有非终态 job 都在等待；已完成的 job 不影响此判断，例如 success+waiting 返回 StatusWaiting）`
    - spec.md：新增 Scenario "success + waiting" 和 "skipped + waiting"
    - plan.md Task 3：新增测试用例 `success + waiting => StatusWaiting` 和 `skipped + waiting => StatusWaiting`

- 02-P1-03 plan 中的提交步骤存在误提交风险
  - **采纳**：plan.md 中 Task 1/2/3 都有 `git add ... && git commit ...` 步骤，Task 4 还有 `git add -A`。在可能存在用户未提交改动的工作区中，`git add -A` 会把无关文件一起提交。采纳修复建议：移除所有强制 commit 步骤，改为仅验证编译和测试通过。具体修改：
    - 删除 Task 1 Step 4（git add + commit）
    - 删除 Task 2 Step 3（git add + commit）
    - 删除 Task 3 Step 3（git add + commit）
    - Task 4 Step 3 从 `git add -A && git commit` 改为 `Review git diff to confirm only the intended files are modified`

## P2

- 02-P2-01 `StatusUnknown` 的范围说明仍不完整
  - **采纳**：文档多处写"所有 8 种状态"，但 GetStatusInfoList 只返回 7 种，不包含 StatusUnknown；聚合逻辑也没有显式定义 unknown，实际会 fallback 到 running。采纳修复建议：明确 StatusUnknown 是内部哨兵状态，不参与 UI 筛选。具体修改：
    - spec.md：GetStatusInfoList Requirement 补充说明"共 7 种。StatusUnknown（值为 0）是内部哨兵状态，不参与 UI 筛选，聚合逻辑中未匹配的状态会 fallback 到 StatusRunning"
    - design.md Risks：新增 StatusUnknown 说明

- 02-P2-02 计划要求 Go 代码使用中文注释，与 Gitea 源码风格不一致
  - **采纳**：plan.md 的 Global Constraints 写"使用中文注释和文档"，示例 Go 注释也是中文；但 Gitea 源码注释基本是英文。采纳修复建议：Go 代码注释改为英文，文档保持中文。具体修改：
    - plan.md Global Constraints：删除"使用中文注释和文档"
    - plan.md Task 1 Step 1：所有 Go 代码块中的中文注释改为英文（函数注释、行内注释）
    - plan.md Task 3：所有测试代码中的中文注释改为英文

---

## 第3轮 审查

### P0

无。

### P1

无。

### P2

- [x] 03-P2-01 `proposal.md` 仍残留 "GetStatusInfoList 支持所有 8 种状态" 的错误表述
  - 现状：`spec.md` 已明确 `GetStatusInfoList` 返回 7 种可显示状态，`StatusUnknown` 是内部哨兵状态，不参与 UI 筛选。但 `proposal.md` 的 `GetStatusInfoList 函数更新` 仍写：`To: 支持所有 8 种状态`。
  - 第一性原理：proposal 是变更入口摘要，不能和 spec 的可执行契约冲突。
  - 修复建议：将 `proposal.md` 中该句改为：`To: 支持所有 7 种可显示状态（Success、Failure、Cancelled、Skipped、Waiting、Running、Blocked），StatusUnknown 不参与 UI 筛选`。

---

# 第3轮 审查开发者回复

## P2

- 03-P2-01 `proposal.md` 残留 "8 种状态" 表述
  - **采纳**：已将 `proposal.md` 第 21 行从 `To: 支持所有 8 种状态` 改为 `To: 支持所有 7 种可显示状态（Success、Failure、Cancelled、Skipped、Waiting、Running、Blocked），StatusUnknown 是内部哨兵状态不参与 UI 筛选`。所有文档现在一致。
