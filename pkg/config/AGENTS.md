# AGENTS.md

## Scope

通用配置能力。

## Rule

只能放与具体业务领域无关的配置加载和解析。

禁止：

- 股票业务规则。
- 行情规则。
- 将某一领域 Service 放进本 package。
- 通过 config package 形成 Service Locator。
