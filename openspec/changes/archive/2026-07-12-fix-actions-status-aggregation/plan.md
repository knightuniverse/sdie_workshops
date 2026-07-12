# Fix Actions Status Aggregation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 Gitea Actions 状态聚合逻辑，正确处理 Cancelled、Skipped、Blocked 三种状态

**Architecture:** 使用单次遍历方案重构 `aggregateJobStatus` 函数：一次循环收集所有布尔量（allDone、hasFailure、hasCancelled、allSkipped、allWaiting、allBlocked），然后按明确的判断顺序返回状态。不新增辅助函数，不在热路径上多次扫描 jobs 切片。

**Tech Stack:** Go, XORM, testify

## Global Constraints

- 遵循 Gitea 代码规范（使用 `modules/json` 而非 `encoding/json`）
- 保持向后兼容，不改变函数签名或返回值类型
- Go 代码注释使用英文，与 Gitea 源码风格一致

---

### Task 1: 重构 aggregateJobStatus 函数

**Files:**
- Modify: `models/actions/run_job.go:155-180`

**Interfaces:**
- Produces: 重构后的 `aggregateJobStatus(jobs []*ActionRunJob) Status`

- [ ] **Step 1: 重写 aggregateJobStatus 函数**

将现有的 `aggregateJobStatus` 函数替换为：

```go
// aggregateJobStatus aggregates the statuses of all jobs into a single run-level status.
//
// Evaluation order:
//  1. Empty list guard: return StatusRunning
//  2. Terminal state (all jobs are IsDone): failure > cancelled > skipped > success
//  3. Active state: allWaiting > allBlocked > running
//
// Key semantics:
//   - "All done" is based on IsDone(), which includes Success/Failure/Cancelled/Skipped.
//   - Mixed success+skipped returns success (both are terminal and no failure/cancelled).
//   - Blocked is only returned when ALL jobs are blocked; otherwise falls back to running.
func aggregateJobStatus(jobs []*ActionRunJob) Status {
	if len(jobs) == 0 {
		return StatusRunning
	}

	allDone := true
	allWaiting := true
	allSkipped := true
	hasFailure := false
	hasCancelled := false
	allBlocked := true

	for _, job := range jobs {
		if !job.Status.IsDone() {
			allDone = false
		}
		if job.Status != StatusWaiting && !job.Status.IsDone() {
			allWaiting = false
		}
		if job.Status != StatusSkipped {
			allSkipped = false
		}
		if job.Status != StatusBlocked {
			allBlocked = false
		}
		switch job.Status {
		case StatusFailure:
			hasFailure = true
		case StatusCancelled:
			hasCancelled = true
		}
	}

	// Terminal state: all jobs have finished
	if allDone {
		if hasFailure {
			return StatusFailure
		}
		if hasCancelled {
			return StatusCancelled
		}
		if allSkipped {
			return StatusSkipped
		}
		return StatusSuccess
	}

	// Active state evaluation
	if allWaiting {
		return StatusWaiting
	}
	if allBlocked && !hasFailure && !hasCancelled {
		return StatusBlocked
	}

	return StatusRunning
}
```

- [ ] **Step 2: 删除旧的实现代码**

删除原来的实现代码（第 155-180 行）。

- [ ] **Step 3: 运行测试验证编译通过**

Run: `cd /Users/cds-dn-696/Documents/2026-07-11-AI-Coding培训/sdie_workshops && go build ./models/actions/...`
Expected: 编译通过，无错误

---

### Task 2: 更新 GetStatusInfoList 函数

**Files:**
- Modify: `models/actions/run_list.go:115-126`

**Interfaces:**
- Produces: 更新后的 `GetStatusInfoList(ctx context.Context) []StatusInfo`

- [ ] **Step 1: 更新 GetStatusInfoList 函数**

将现有的 `GetStatusInfoList` 函数替换为：

```go
// GetStatusInfoList returns a slice of StatusInfo for UI filter options.
// The order reflects display preference, not aggregation priority.
func GetStatusInfoList(ctx context.Context) []StatusInfo {
	allStatus := []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped, StatusWaiting, StatusRunning, StatusBlocked}
	statusInfoList := make([]StatusInfo, 0, len(allStatus))
	for _, s := range allStatus {
		statusInfoList = append(statusInfoList, StatusInfo{
			Status:          int(s),
			DisplayedStatus: s.String(),
		})
	}
	return statusInfoList
}
```

- [ ] **Step 2: 运行测试验证编译通过**

Run: `cd /Users/cds-dn-696/Documents/2026-07-11-AI-Coding培训/sdie_workshops && go build ./models/actions/...`
Expected: 编译通过，无错误

---

### Task 3: 添加单元测试

**Files:**
- Create: `models/actions/run_job_test.go`

**Interfaces:**
- Consumes: `aggregateJobStatus(jobs []*ActionRunJob) Status`

- [ ] **Step 1: 创建测试文件**

创建 `models/actions/run_job_test.go` 文件：

```go
// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregateJobStatus(t *testing.T) {
	tests := []struct {
		name     string
		jobs     []*ActionRunJob
		expected Status
	}{
		// Empty list
		{
			name:     "empty jobs",
			jobs:     []*ActionRunJob{},
			expected: StatusRunning,
		},
		// Terminal: failure has highest priority
		{
			name: "failure + cancelled",
			jobs: []*ActionRunJob{
				{Status: StatusFailure},
				{Status: StatusCancelled},
			},
			expected: StatusFailure,
		},
		{
			name: "failure + success",
			jobs: []*ActionRunJob{
				{Status: StatusFailure},
				{Status: StatusSuccess},
			},
			expected: StatusFailure,
		},
		// Terminal: cancelled is next priority
		{
			name: "success + cancelled",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusCancelled},
			},
			expected: StatusCancelled,
		},
		// Terminal: all skipped
		{
			name: "all skipped",
			jobs: []*ActionRunJob{
				{Status: StatusSkipped},
				{Status: StatusSkipped},
			},
			expected: StatusSkipped,
		},
		// Terminal: success + skipped mixed (all terminal, no failure/cancelled)
		{
			name: "success + skipped",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusSkipped},
			},
			expected: StatusSuccess,
		},
		// Terminal: all success
		{
			name: "all success",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusSuccess},
			},
			expected: StatusSuccess,
		},
		// Active: all waiting
		{
			name: "all waiting",
			jobs: []*ActionRunJob{
				{Status: StatusWaiting},
				{Status: StatusWaiting},
			},
			expected: StatusWaiting,
		},
		// Active: all blocked
		{
			name: "all blocked",
			jobs: []*ActionRunJob{
				{Status: StatusBlocked},
				{Status: StatusBlocked},
			},
			expected: StatusBlocked,
		},
		// Active: blocked + running (has running, should not be blocked)
		{
			name: "blocked + running",
			jobs: []*ActionRunJob{
				{Status: StatusBlocked},
				{Status: StatusRunning},
			},
			expected: StatusRunning,
		},
		// Active: waiting + blocked (has waiting, should not be blocked)
		{
			name: "waiting + blocked",
			jobs: []*ActionRunJob{
				{Status: StatusWaiting},
				{Status: StatusBlocked},
			},
			expected: StatusRunning,
		},
		// Active: running only
		{
			name: "running only",
			jobs: []*ActionRunJob{
				{Status: StatusRunning},
			},
			expected: StatusRunning,
		},
		// Single job
		{
			name:     "single success",
			jobs:     []*ActionRunJob{{Status: StatusSuccess}},
			expected: StatusSuccess,
		},
		{
			name:     "single waiting",
			jobs:     []*ActionRunJob{{Status: StatusWaiting}},
			expected: StatusWaiting,
		},
		// Active: done jobs do not break allWaiting
		{
			name: "success + waiting",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusWaiting},
			},
			expected: StatusWaiting,
		},
		{
			name: "skipped + waiting",
			jobs: []*ActionRunJob{
				{Status: StatusSkipped},
				{Status: StatusWaiting},
			},
			expected: StatusWaiting,
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

- [ ] **Step 2: 运行测试**

Run: `cd /Users/cds-dn-696/Documents/2026-07-11-AI-Coding培训/sdie_workshops && go test ./models/actions/... -run TestAggregateJobStatus -v`
Expected: 所有测试用例通过

---

### Task 4: 代码检查

**Files:**
- None

- [ ] **Step 1: 运行 lint 检查**

Run: `cd /Users/cds-dn-696/Documents/2026-07-11-AI-Coding培训/sdie_workshops && make lint`
Expected: 无 lint 错误

- [ ] **Step 2: 运行所有测试**

Run: `cd /Users/cds-dn-696/Documents/2026-07-11-AI-Coding培训/sdie_workshops && make test`
Expected: 所有测试通过

- [ ] **Step 3: Verify all changes are complete**

Review `git diff` to confirm only the intended files are modified.
