# backtest/ports 规约

- 接口只描述回测领域存储能力，不包含 SQL、HTTP 或引擎实现。
- 回测键包含策略、日期区间、配置、模式和不可变 snapshot hash。
- 同键返回已有运行；成功等终态不可覆盖；失败重试必须使用新运行键。
- 所有存储操作接收 `context.Context`。
