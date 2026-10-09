# P03-04 真实 Token 权限最小验收报告

## 结果

**状态：`BLOCKED`。** 本次没有配置 `TUSHARE_TOKEN`，也没有单独授权真实 provider 请求；真实调用次数为 0。没有对 Tushare 发出请求，因此所有账户权限都保持 `NOT_TESTED`。本报告不把 fixture 或官方文档门槛当成账户实测结果。

## 已完成的本地检查

- 官方公开最低积分门槛已整理至 [`permission-matrix.md`](permission-matrix.md)，并与账户实测列分开维护。
- `stock_st` 在权限元数据中标为 3000 积分起、高门槛且非核心；本次未调用。
- Mock doctor 明确显示 `offline/mock`；Tushare 模式缺少 Token 时退出并声明不会回退到 Mock。
- 配置了本地 Token 时 doctor 仅报告 `configured_unverified`，不请求 provider，也不宣称权限通过。
- 权限拒绝、日志脱敏和本地依赖资源失败由单元测试覆盖。

## 解除阻塞所需条件

1. 在本地 secret manager 或运行环境配置 `TUSHARE_TOKEN`，不要把 Token 放入聊天、命令历史、Git 或测试报告。
2. 用户明确授权 P03-04 的最小真实调用范围。
3. 仅按权限矩阵中的顺序执行核心 API 极小日期查询；把脱敏后的状态码、API、日期、行数和时间写回矩阵。对于可选接口记录跳过理由；没有 3000 积分权限证据前不调用 `stock_st`。

在这些条件满足前，不开始依赖本票的 P04-01，也不将 P03 阶段标记通过。
