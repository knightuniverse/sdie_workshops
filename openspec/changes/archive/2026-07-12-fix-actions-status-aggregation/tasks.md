## 1. 重构 aggregateJobStatus 函数

- [x] 1.1 在 `models/actions/run_job.go` 中重写 `aggregateJobStatus` 函数，使用单次遍历收集布尔量
- [x] 1.2 实现终态判断逻辑（allDone 分支：failure > cancelled > skipped > success）
- [x] 1.3 实现活跃态判断逻辑（allWaiting、allBlocked、running）
- [x] 1.4 添加空列表防御性检查

## 2. 更新 GetStatusInfoList 函数

- [x] 2.1 更新 `GetStatusInfoList` 函数支持所有 7 种可显示状态
- [x] 2.2 更新注释说明排序为 UI 筛选选项顺序，与聚合优先级无关

## 3. 单元测试

- [x] 3.1 创建 `models/actions/run_job_test.go` 文件（使用 `package actions` 包内测试）
- [x] 3.2 添加空 job 列表测试用例
- [x] 3.3 添加 all skipped 测试用例
- [x] 3.4 添加 all failure 测试用例
- [x] 3.5 添加 has cancelled（success + cancelled）测试用例
- [x] 3.6 添加 failure + cancelled 测试用例（failure 优先）
- [x] 3.7 添加 failure + success 测试用例（failure 优先）
- [x] 3.8 添加 success + cancelled 测试用例（cancelled 优先）
- [x] 3.9 添加 success + skipped 测试用例（返回 success）
- [x] 3.10 添加 all blocked 测试用例
- [x] 3.11 添加 blocked + running 测试用例（返回 running）
- [x] 3.12 添加 waiting + blocked 测试用例（返回 running）
- [x] 3.13 添加 all waiting 测试用例
- [x] 3.14 添加 running only 测试用例
- [x] 3.15 添加 single success 测试用例
- [x] 3.16 添加 single waiting 测试用例
- [x] 3.17 运行测试确保通过

## 4. 代码检查

- [x] 4.1 运行 `go build ./models/actions/...` 确保编译通过
- [x] 4.2 运行 `go test ./models/actions/... -run TestAggregateJobStatus -v` 确保测试通过
- [x] 4.3 运行 `make lint` 检查代码规范
