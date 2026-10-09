# factor/ports 规约

- `FactorRunner` 是请求原始因子的应用边界；v1 参数和结果采用已冻结的 `contracts` DTO。
- 具体 Python 进程、stdin/stdout、超时和重试由后续 infrastructure adapter 负责。
- 领域因子计算保持纯函数，不能依赖本 package 的 runner 实现。
