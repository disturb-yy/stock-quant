# shared/types 规约

- 仅放跨领域的通用值类型；不要加入具体领域模型或依赖 SQL、HTTP、provider。
- `TradingDate` 是无时区的民用日期，规范格式为 `YYYY-MM-DD`，有效年份 `0001`–`9999`；转成 Tushare 日期为 `YYYYMMDD`，不得经 UTC 时间戳隐式平移。
- 零值不是有效交易日；写入数据库或发往 Tushare 前必须使用会返回 error 的格式化方法。
- 从时间点提取交易日必须显式传入 location。业务日期不得由 `time.Now()` 推导。
- `Decimal` 保存规范化的精确十进制文本；JSON 输出为数字，SQL 使用参数化文本并从文本扫描。不要通过 `float64` 中转持久化值。
- `Decimal.Fits` 必须在写入前验证 DECIMAL 精度与 scale，不能让 MySQL 静默舍入。
- `Float64Checked` 是显式计算边界；禁止将其结果反向当成原始持久化值。
- `AmountYuan` 单位为人民币元，包装精确 `Decimal`；只有显式调用 `Float64Checked` 才可转为计算用 `float64`。
