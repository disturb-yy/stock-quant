# 09 — Dashboard 低保真与数据映射

先只做 5 页；前端基线：顶部导航、克制的数据研究终端、深色中性画布、14px 字体；涨红跌绿、表格数字等宽，选股数据优先，避免营销式装饰。采用 React+TS+Vite。

```text
┌────────────────────────────────────────────────────────────────────────┐
│ Stock Quant    今日选股 | 策略 | 回测 | 数据质量 | 任务       设置       │
├────────────────────────────────────────────────────────────────────────┤
│ 交易日期 [YYYY-MM-DD]  策略 [momentum_v1]  [开始选股]  状态 SUCCESS    │
│ 股票池 4820  数据完整 4500  过滤 4250  合格 250  输出 Top 20         │
├──────────┬────────┬──────┬────────┬────────┬───────┬──────────────────┤
│ code     │ name   │ score│ M60    │ M20    │ 活跃度 │ 低波动           │
├──────────┼────────┼──────┼────────┼────────┼───────┼──────────────────┤
│ ...                                                                  │
└────────────────────────────────────────────────────────────────────────┘
```

## 页面 / API 映射
| 页面 | API | 必需交互/状态 |
|---|---|---|
| 每日选股 | `POST /screen-runs`; `GET /screen-runs/{id}` `/results` | loading/empty/success/failed/blocked, 重试, 因子明细展开 |
| 策略 | `GET /strategies`, `/strategies/{id}` | 版本只读查看，修改配置先作为新版本提交 |
| 回测 | `POST /backtest-runs`, `GET /backtest-runs/{id}`, `/equity` | 收益/回撤/基准/明细, 展示 research-only 标记 |
| 数据质量 | `GET /data-quality?date=...` | missing/unknown ST/修订批次/拒绝运行理由 |
| 任务 | `GET /sync-jobs/{id}` 和 run 状态接口 | PENDING/RUNNING/SUCCESS/FAILED/BLOCKED, 错误说明 |

## 前端结构建议
`web/src/pages/{Screening,Strategies,Backtest,DataQuality,Jobs}`, `components/{DataTable,Metric,RunStatus,EmptyState}`, `api/`, `types/`，统一 React Query 等数据请求方案可在 P11 确认。页面不得直接调用 Tushare。

## 验收
- API fixture 下 5 页正常渲染；空、加载、失败、阻断四状态分别截图或 DOM 断言；无假数据显示。
- 非交易日不能提交有效选股运行；用户能看到具体阻断原因。
- 移动端表格可横向滚动；键盘焦点可见。
