# shared/types 导航

| 文件 | 职责 |
|---|---|
| `trading_date.go` | `TradingDate` 在规范日期、MySQL `DATE` 文本和 Tushare 日期之间转换并拒绝零值输出；`AmountYuan` 有限值校验 |
| `trading_date_test.go` | 日期格式、日历有效性、零值、显式时区、金额有限值测试 |
