# shared/types 导航

| 文件 | 职责 |
|---|---|
| `trading_date.go` | `TradingDate` 日期互转及数据库/Tushare 日期格式化 |
| `decimal.go` | `Decimal` 精确十进制解析、规范化、JSON、DECIMAL 精度校验、精确乘整数和显式 float64 转换；`AmountYuan` 精确元金额 |
| `trading_date_test.go` | 日期与精确十进制值规则、JSON文本往返和 float64 精度边界测试 |
