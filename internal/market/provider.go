package market

import "strings"

// ProviderMode 是前端可依赖的 Provider 工作模式。
type ProviderMode string

const (
	// ModeDemo 表示使用版本化的 MySQL 演示 fixture。
	ModeDemo ProviderMode = "demo"
	// ModeReal 为真实外部 Provider 预留；当前切片没有实现真实 Provider。
	ModeReal ProviderMode = "real"
	// ModeFallback 表示回退到本地 fixture，或本地数据尚未准备好。
	ModeFallback ProviderMode = "fallback"
)

const (
	DemoProviderName     = "mysql-demo-fixture"
	RealProviderName     = "external-real-provider"
	FallbackProviderName = "local-fixture-fallback"
)

// ProviderSelection 是本地 Provider 解析结果。
type ProviderSelection struct {
	Mode     ProviderMode
	Provider string
}

// SelectProvider 解析请求的 Provider。当前不把未实现的真实 Provider 标记为 real。
func SelectProvider(requested string, fixtureReady bool) ProviderSelection {
	if strings.EqualFold(strings.TrimSpace(requested), string(ModeReal)) {
		return ProviderSelection{Mode: ModeFallback, Provider: FallbackProviderName}
	}
	if fixtureReady {
		return ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName}
	}
	return ProviderSelection{Mode: ModeFallback, Provider: FallbackProviderName}
}
