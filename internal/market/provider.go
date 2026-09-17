package market

import "strings"

// ProviderMode 是前端可依赖的 Provider 工作模式。
type ProviderMode string

const (
	// ModeDemo 表示使用版本化的 MySQL 演示 fixture。
	ModeDemo ProviderMode = "demo"
	// ModeReal 表示数据已由真实外部 Provider 同步并落入读模型。
	ModeReal ProviderMode = "real"
	// ModeFallback 表示回退到本地 fixture，或本地数据尚未准备好。
	ModeFallback ProviderMode = "fallback"
)

const (
	DemoProviderName     = "mysql-demo-fixture"
	RealProviderName     = "tushare"
	FallbackProviderName = "local-fixture-fallback"

	DemoMetadataName    = "fnd-003-demo"
	TushareMetadataName = "tushare-real"
)

// ProviderSelection 是本地 Provider 解析结果。
type ProviderSelection struct {
	Mode         ProviderMode
	Provider     string
	MetadataName string
}

// SelectProvider 解析未启用真实 Provider 时的本地 Provider。
func SelectProvider(requested string, fixtureReady bool) ProviderSelection {
	return SelectProviderWithAvailability(requested, fixtureReady, false)
}

// SelectProviderWithAvailability 只有在真实同步已经成功时才返回 real。
func SelectProviderWithAvailability(requested string, fixtureReady, realReady bool) ProviderSelection {
	if strings.EqualFold(strings.TrimSpace(requested), string(ModeReal)) {
		if realReady {
			return ProviderSelection{Mode: ModeReal, Provider: RealProviderName, MetadataName: TushareMetadataName}
		}
		return ProviderSelection{Mode: ModeFallback, Provider: FallbackProviderName}
	}
	if strings.EqualFold(strings.TrimSpace(requested), RealProviderName) {
		if realReady {
			return ProviderSelection{Mode: ModeReal, Provider: RealProviderName, MetadataName: TushareMetadataName}
		}
		return ProviderSelection{Mode: ModeFallback, Provider: FallbackProviderName}
	}
	if fixtureReady {
		return ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName, MetadataName: DemoMetadataName}
	}
	return ProviderSelection{Mode: ModeFallback, Provider: FallbackProviderName}
}

// IsRealProviderRequested 判断是否需要先同步 Tushare 数据。
func IsRealProviderRequested(requested string) bool {
	requested = strings.TrimSpace(requested)
	return strings.EqualFold(requested, string(ModeReal)) || strings.EqualFold(requested, RealProviderName)
}
