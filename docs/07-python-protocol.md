# 07 — Go ↔ Python 协议 v1（只传原始因子）

## 设计决定
- 初期以 `os/exec` subprocess 调用 Python worker（无守护服务/外部网络），`stdin` 一份 JSON 请求，`stdout` **恰好一份 JSON 响应**，`stderr` 独立日志。严禁把 debug print 写 stdout。
- 生产大数据不从 JSON 传 5000×61 根 K 线：平台生成不可变/只读 `snapshot_ref` 指向受控目录的文件；后续可换 Parquet，最初 JSONL 可用。`contracts/strategy-request.schema.json` 用 `data_ref` 表示路径/哈希；适配器使用白名单本地路径，绝不能接受 URL 或 `../` 穿越。
- `schema_version=v1`，`mode=raw_factors`，Python 输出股票的四项原始因子及质量错误，不计算最终百分位和权重；Go 唯一负责过滤、横截面百分位、综合分、Top N。
- `request_id`, `strategy_id`, `strategy_version`, `as_of`, `snapshot_hash`, `config_hash` 全链路复用；返回数据顺序不依赖 Go map iteration。

## 请求示例
见 `contracts/examples/request.json` 和 `contracts/strategy-request.schema.json`。

## 响应示例
见 `contracts/examples/result.json` 和 `contracts/strategy-result.schema.json`。

## Runner 接口与异常
```go
type Runner interface {
    Compute(ctx context.Context, req FactorRequest) (FactorResult, error)
}
// GoRunner 和 PythonRunner 必须接受同一请求并返回语义等价结果。
```
超时（例如 120 秒）/取消：杀整个进程组（Linux/WSL），回收子进程与临时文件，返回 `TIMEOUT`/`CANCELLED`；非零退出记录 stderr（长度封顶）；输出过长拒绝；无响应或格式错误归 `INVALID_WORKER_RESPONSE`；schema_version 不匹配 fail fast。

## 执行与安全
- worker 由**已注册、受信任**的 Python 路径执行；**不可**从 HTTP 用户传入任意脚本路径或 Python 代码。
- 使用固定解释器路径：`python/.venv/bin/python`；隔离依赖并锁版本；设置超时、并发数 1/2、CPU/内存/文件大小限制（容器/OS）。
- 禁止 worker 默认网络请求，尤其不能直接访问 Tushare 和 DB；无法完全 OS sandbox 时，只允许用户自己维护的本地可信代码，声明残留风险。
- 结果检查：ts_code 来自输入集合、无重复、无缺项或按策略明确容忍缺项、因子 finite、snapshot hash 一致。

## Go/Python 一致性与容差
- 公式遵循 `docs/03-strategy-spec.md`；每个固定价格序列计算误差目标：动量/活跃度 <=1e-10，年化波动 <=1e-10（浮点实际可用绝对+相对容差）。
- 股票池统一定义、独立 data quality gate；算法版本号统一。
- 单元测试必须覆盖复权分割、收入金额、20/40/61 日边界、NaN、负数、顺序、超时、坏 JSON、schema 不兼容、空结果、重复代码。

## 升级路径
`subprocess -> gRPC Python Worker` 不改变领域 `Runner`/数据字段，仅新增 adapter；不能在 v1 为了未来额外引入 gRPC 服务。
