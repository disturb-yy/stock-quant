# 13 — 安全与隔离

## Tushare/数据库
- Token 仅通过 `TUSHARE_TOKEN`/运行机 secret manager 注入，禁止落入日志、异常、ZIP、配置示例与 Git。
- `MYSQL_DSN` 同理，数据库采用权限最小化账号。查询参数绑定，分页/排序只允许白名单。
- SQL migrations 单独授权；Web API 没有裸 SQL/路径任意调用功能。

## Python 子进程
- 只执行管理员登记、带版本与 hash 的受信任脚本；**不得** HTTP 接受脚本体、任意脚本路径或命令行自由拼接。
- `exec.CommandContext` 配合受控 Python 解释器路径；限制 CPU/内存/运行时间、stderr/stdout 最大长度，失败时清理子进程组；禁用不必要网络访问。
- 生产 snapshot_ref 解析为 sandbox 根目录下的普通文件并检查 canonical path；输入 schema 固定、JSON 限制大小；可转临时文件但不能让 Python 读其他任意路径。
- Python 依赖锁文件及环境版本必须纳入执行元数据；第三方量化库视为依赖，不是权限边界。

## API 与权限
- 第一版服务默认绑定 loopback；部署公网前必须加入认证、HTTPS、授权、访问限流和 CSRF/CORS 规则。
- 调用敏感操作（同步、运行、修改策略）记录 operator/trace id，不记录凭据。
- 防止用户请求任意 `data_ref` 和文件下载，不提供任意文件读取 API。

## 错误信息
对外返回可理解业务错误与 request_id，内部完整错误只写受控日志；日志从请求 JSON 中移除 `token/password/authorization` 字段。

## 安全验收
泄漏检测（grep token/DSN）、目录穿越尝试拒绝、坏 JSON/大 JSON 拒绝、进程超时停止、无登录公网暴露扫描、CI fixture 无真实凭据。
