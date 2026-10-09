# shared/types 规约

- 仅放跨领域的通用值类型；不要加入具体领域模型或依赖 SQL、HTTP、provider。
- `TradingDate` 是无时区的民用日期，规范格式为 `YYYY-MM-DD`，有效年份 `0001`–`9999`；转成 Tushare 日期为 `YYYYMMDD`，不得经 UTC 时间戳隐式平移。
- 从时间点提取交易日必须显式传入 location。业务日期不得由 `time.Now()` 推导。
- `AmountYuan` 单位为人民币元，只拒绝 NaN/Inf；不要在共享类型中擅自限制正负、舍入或小数位。
