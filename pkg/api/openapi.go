package api

// OpenAPIDocument 返回服务实际输出的 API v1 OpenAPI 契约。
// 使用 map 构造可以保持该契约包不依赖 Gin，同时让新增业务接口按相同 schema 扩展。
func OpenAPIDocument(includeDevelopment ...bool) map[string]any {
	return map[string]any{
		"openapi":    "3.0.3",
		"info":       apiInfo(),
		"paths":      apiPaths(developmentPathEnabled(includeDevelopment)),
		"components": apiComponents(),
	}
}

func developmentPathEnabled(values []bool) bool {
	return len(values) == 0 || values[0]
}

func apiInfo() map[string]any {
	return map[string]any{
		"title":       "Stock Quant API",
		"version":     "v1",
		"description": "股票量化服务 API v1 契约",
	}
}

func apiPaths(includeDevelopment bool) map[string]any {
	paths := map[string]any{
		"/api/v1/health":           healthPath(),
		"/api/v1/markets/overview": marketOverviewPath(),
		"/api/v1/markets/sectors":  marketSectorsPath(),
		"/api/v1/markets/signals":  marketSignalsPath(),
		"/api/v1/markets/rankings": marketRankingsPath(),
		"/api/v1/stocks/{symbol}":  stockOverviewPath(),
		"/api/v1/openapi.json":     openAPIPath(),
	}
	if includeDevelopment {
		paths["/api/v1/dev/demo-status"] = demoStatusPath()
	}
	return paths
}

func healthPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getHealth",
			"summary":     "服务健康检查",
			"responses": map[string]any{
				"200": healthResponse(),
				"404": errorResponse("请求的资源不存在"),
				"405": errorResponse("请求方法不被允许"),
				"500": errorResponse("服务内部错误"),
			},
		},
	}
}

func openAPIPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getOpenAPI",
			"summary":     "读取 API v1 OpenAPI 契约",
			"responses": map[string]any{
				"200": map[string]any{
					"description": "OpenAPI 文档",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"type": "object"},
						},
					},
				},
				"404": errorResponse("请求的资源不存在"),
				"405": errorResponse("请求方法不被允许"),
				"500": errorResponse("服务内部错误"),
			},
		},
	}
}

func demoStatusPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getDevDemoStatus",
			"summary":     "读取开发环境演示数据状态",
			"responses": map[string]any{
				"200": jsonReferenceResponse("演示数据状态", "#/components/schemas/DemoStatus"),
				"404": errorResponse("开发接口仅在开发环境注册"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("演示数据存储不可用"),
			},
		},
	}
}

func marketOverviewPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getMarketOverview",
			"summary":     "读取市场概览",
			"responses": map[string]any{
				"200": jsonReferenceResponse("市场概览", "#/components/schemas/MarketOverview"),
				"404": errorResponse("请求的资源不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("市场概览数据不可用"),
			},
		},
	}
}

func marketSectorsPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getMarketSectors",
			"summary":     "读取行业表现",
			"responses": map[string]any{
				"200": jsonReferenceResponse("行业表现", "#/components/schemas/MarketSectors"),
				"404": errorResponse("请求的资源不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("行业数据不可用"),
			},
		},
	}
}

func marketSignalsPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getMarketSignals",
			"summary":     "按最新交易日扫描市场信号",
			"parameters": []any{
				map[string]any{
					"name": "type", "in": "query", "required": true,
					"schema": map[string]any{"type": "string", "enum": []string{"volume_surge", "breakout", "new_high", "strong"}},
				},
				map[string]any{
					"name": "params", "in": "query", "required": false,
					"description": "URL 编码的 JSON 参数对象；不同信号只允许使用其适用字段，省略时使用默认值。",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema":  map[string]any{"$ref": "#/components/schemas/SignalParameters"},
							"example": map[string]any{"window": 20, "multiple": 1.5},
						},
					},
				},
			},
			"responses": map[string]any{
				"200": jsonReferenceResponse("市场信号扫描结果", "#/components/schemas/MarketSignals"),
				"400": errorResponse("信号类型或参数无效"),
				"404": errorResponse("请求的资源不存在"),
				"405": errorResponse("请求方法不被允许"),
				"422": errorResponse("历史行情不足"),
				"503": errorResponse("市场信号数据不可用"),
			},
		},
	}
}

func marketRankingsPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getMarketRankings",
			"summary":     "读取股票排行榜",
			"parameters": []any{
				map[string]any{
					"name": "metric", "in": "query", "required": true,
					"schema": map[string]any{"type": "string", "enum": []string{"gain", "loss", "turnover_amount", "turnover_rate"}},
				},
				map[string]any{
					"name": "page", "in": "query", "required": false,
					"schema": map[string]any{"type": "integer", "minimum": DefaultPage, "default": DefaultPage},
				},
				map[string]any{
					"name": "page_size", "in": "query", "required": false,
					"schema": map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize, "default": DefaultPageSize},
				},
			},
			"responses": map[string]any{
				"200": jsonReferenceResponse("股票排行榜", "#/components/schemas/MarketRankings"),
				"400": errorResponse("排行指标或分页参数无效"),
				"404": errorResponse("请求的资源不存在"),
				"405": errorResponse("请求方法不被允许"),
				"422": errorResponse("历史行情不足"),
				"503": errorResponse("股票排行数据不可用"),
			},
		},
	}
}

func stockOverviewPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getStockOverview",
			"summary":     "读取股票详情概览",
			"parameters": []any{
				map[string]any{
					"name": "symbol", "in": "path", "required": true,
					"description": "直接使用 Markets API 返回的 code，不做前端转换。",
					"schema":      map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
				},
			},
			"responses": map[string]any{
				"200": jsonReferenceResponse("股票详情概览", "#/components/schemas/StockOverview"),
				"400": errorResponse("股票代码参数无效"),
				"404": errorResponse("股票不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("股票详情数据不可用"),
			},
		},
	}
}

func jsonReferenceResponse(description, reference string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": reference},
			},
		},
	}
}

func apiComponents() map[string]any {
	return map[string]any{
		"schemas": map[string]any{
			"Response":            responseSchema(),
			"ErrorResponse":       errorSchema(),
			"PaginationRequest":   paginationRequestSchema(),
			"PaginationMeta":      paginationMetaSchema(),
			"PaginatedResponse":   paginatedResponseSchema(),
			"HealthResponse":      healthSchema(),
			"DemoCounts":          demoCountsSchema(),
			"DemoSampleStock":     demoSampleStockSchema(),
			"DemoStatus":          demoStatusSchema(),
			"MarketOverview":      marketOverviewSchema(),
			"MarketSectors":       marketSectorsSchema(),
			"MarketSector":        marketSectorSchema(),
			"SectorLeader":        sectorLeaderSchema(),
			"MarketDataSource":    marketDataSourceSchema(),
			"MarketIndex":         marketIndexSchema(),
			"MarketBreadth":       marketBreadthSchema(),
			"MarketTurnover":      marketTurnoverSchema(),
			"MarketSignals":       marketSignalsSchema(),
			"SignalParameters":    signalParametersSchema(),
			"SignalResult":        signalResultSchema(),
			"MarketRankings":      marketRankingsSchema(),
			"MarketRanking":       marketRankingSchema(),
			"StockOverview":       stockOverviewSchema(),
			"StockQuote":          stockQuoteSchema(),
			"StockMetrics":        stockMetricsSchema(),
			"StockMetric":         stockMetricSchema(),
			"StockSparkline":      stockSparklineSchema(),
			"StockSparklinePoint": stockSparklinePointSchema(),
		},
	}
}

func stockOverviewSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"symbol", "name", "industry", "quote", "metrics", "sparkline"},
		"properties": map[string]any{
			"symbol":    map[string]any{"type": "string", "example": "000001.SZ"},
			"name":      map[string]any{"type": "string", "example": "平安银行"},
			"industry":  map[string]any{"type": "string", "example": "银行"},
			"quote":     map[string]any{"$ref": "#/components/schemas/StockQuote"},
			"metrics":   map[string]any{"$ref": "#/components/schemas/StockMetrics"},
			"sparkline": map[string]any{"$ref": "#/components/schemas/StockSparkline"},
		},
	}
}

func stockQuoteSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"last", "change", "change_pct", "as_of"},
		"properties": map[string]any{
			"last":       map[string]any{"type": "string", "example": "10.31"},
			"change":     map[string]any{"type": "string", "example": "0.09"},
			"change_pct": map[string]any{"type": "string", "example": "0.88"},
			"as_of":      map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
		},
	}
}

func stockMetricsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"market_cap", "pe_ttm", "pb", "roe"},
		"properties": map[string]any{
			"market_cap": map[string]any{"$ref": "#/components/schemas/StockMetric"},
			"pe_ttm":     map[string]any{"$ref": "#/components/schemas/StockMetric"},
			"pb":         map[string]any{"$ref": "#/components/schemas/StockMetric"},
			"roe":        map[string]any{"$ref": "#/components/schemas/StockMetric"},
		},
	}
}

func stockMetricSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"value", "as_of", "basis"},
		"properties": map[string]any{
			"value": map[string]any{"type": "string", "nullable": true, "example": "5.82"},
			"as_of": map[string]any{"type": "string", "format": "date", "nullable": true, "example": "2024-06-28"},
			"basis": map[string]any{"type": "string", "nullable": true, "enum": []string{"latest_daily_basic", "ttm", "latest_report"}},
		},
	}
}

func stockSparklineSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"period", "points"},
		"properties": map[string]any{
			"period": map[string]any{"type": "string", "enum": []string{"20d"}, "example": "20d"},
			"points": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockSparklinePoint"}, "example": []map[string]string{{"trade_date": "2024-06-03", "open": "9.84", "high": "9.91", "low": "9.79", "close": "9.88"}}},
		},
	}
}

func stockSparklinePointSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"trade_date", "open", "high", "low", "close"},
		"properties": map[string]any{
			"trade_date": map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"open":       map[string]any{"type": "string", "example": "10.20"},
			"high":       map[string]any{"type": "string", "example": "10.40"},
			"low":        map[string]any{"type": "string", "example": "10.10"},
			"close":      map[string]any{"type": "string", "example": "10.31"},
		},
	}
}

func responseSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"data"},
		"properties": map[string]any{
			"data": map[string]any{},
		},
	}
}

func errorSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "message"},
		"properties": map[string]any{
			"code": map[string]any{
				"type": "string",
				"enum": []string{
					string(CodeNotFound),
					string(CodeMethodNotAllowed),
					string(CodeValidation),
					string(CodeInvalidPagination),
					string(CodeInsufficientHistory),
					string(CodeInternal),
					string(CodeDependencyUnavailable),
				},
			},
			"message": map[string]any{"type": "string"},
			"details": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
	}
}

func paginationRequestSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"page": map[string]any{
				"type":    "integer",
				"minimum": DefaultPage,
				"default": DefaultPage,
			},
			"page_size": map[string]any{
				"type":    "integer",
				"minimum": 1,
				"maximum": MaxPageSize,
				"default": DefaultPageSize,
			},
		},
	}
}

func paginationMetaSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"page", "page_size", "total", "total_pages"},
		"properties": map[string]any{
			"page":        map[string]any{"type": "integer", "minimum": DefaultPage},
			"page_size":   map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize},
			"total":       map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"total_pages": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
		},
	}
}

func paginatedResponseSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"data", "pagination"},
		"properties": map[string]any{
			"data":       map[string]any{"type": "array", "items": map[string]any{}},
			"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
		},
	}
}

func healthSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"status"},
		"properties": map[string]any{
			"status": map[string]any{"type": "string", "example": "ok"},
		},
	}
}

func demoCountsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"instruments", "daily_bars", "daily_basics", "financial_metrics", "index_snapshots"},
		"properties": map[string]any{
			"instruments":       map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"daily_bars":        map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"daily_basics":      map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"financial_metrics": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"index_snapshots":   map[string]any{"type": "integer", "format": "int64", "minimum": 0},
		},
	}
}

func demoSampleStockSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "exchange", "status"},
		"properties": map[string]any{
			"code":     map[string]any{"type": "string", "example": "600519.SH"},
			"name":     map[string]any{"type": "string", "example": "贵州茅台"},
			"exchange": map[string]any{"type": "string", "enum": []string{"SSE", "SZSE"}},
			"status":   map[string]any{"type": "string", "enum": []string{"active"}},
		},
	}
}

func demoStatusSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"mode", "provider", "seed_version", "as_of", "counts", "sample_stocks"},
		"properties": map[string]any{
			"mode": map[string]any{
				"type": "string",
				"enum": []string{"demo", "real", "fallback"},
			},
			"provider": map[string]any{
				"type": "string",
				"enum": []string{"mysql-demo-fixture", "external-real-provider", "local-fixture-fallback"},
			},
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v5"},
			"as_of":        map[string]any{"type": "string", "format": "date", "nullable": true, "example": "2024-06-28"},
			"counts":       map[string]any{"$ref": "#/components/schemas/DemoCounts"},
			"sample_stocks": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/components/schemas/DemoSampleStock"},
			},
		},
	}
}

func marketOverviewSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"as_of", "observed_at", "source", "indices", "breadth", "turnover"},
		"properties": map[string]any{
			"as_of":       map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"observed_at": map[string]any{"type": "string", "format": "date-time", "example": "2024-06-28T07:00:00Z"},
			"source":      map[string]any{"$ref": "#/components/schemas/MarketDataSource"},
			"indices": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/components/schemas/MarketIndex"},
			},
			"breadth":  map[string]any{"$ref": "#/components/schemas/MarketBreadth"},
			"turnover": map[string]any{"$ref": "#/components/schemas/MarketTurnover"},
		},
	}
}

func marketSectorsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"as_of", "source", "sectors"},
		"properties": map[string]any{
			"as_of":   map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"source":  map[string]any{"$ref": "#/components/schemas/MarketDataSource"},
			"sectors": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/MarketSector"}},
		},
	}
}

func marketSectorSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "change_percent", "component_count", "leader"},
		"properties": map[string]any{
			"code":            map[string]any{"type": "string", "example": "BANK"},
			"name":            map[string]any{"type": "string", "example": "银行"},
			"change_percent":  map[string]any{"type": "string", "example": "0.88"},
			"component_count": map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"leader":          map[string]any{"$ref": "#/components/schemas/SectorLeader"},
		},
	}
}

func sectorLeaderSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "change_percent"},
		"properties": map[string]any{
			"code":           map[string]any{"type": "string", "example": "000001.SZ"},
			"name":           map[string]any{"type": "string", "example": "平安银行"},
			"change_percent": map[string]any{"type": "string", "example": "0.88"},
		},
	}
}

func marketDataSourceSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"mode", "provider", "seed_version"},
		"properties": map[string]any{
			"mode": map[string]any{
				"type": "string",
				"enum": []string{"demo", "real", "fallback"},
			},
			"provider": map[string]any{
				"type": "string",
				"enum": []string{"mysql-demo-fixture", "external-real-provider", "local-fixture-fallback"},
			},
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v5"},
		},
	}
}

func marketIndexSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "close", "change", "change_percent"},
		"properties": map[string]any{
			"code":           map[string]any{"type": "string", "example": "000001.SH"},
			"name":           map[string]any{"type": "string", "example": "上证指数"},
			"close":          map[string]any{"type": "string", "example": "2994.73"},
			"change":         map[string]any{"type": "string", "example": "-3.89"},
			"change_percent": map[string]any{"type": "string", "example": "-0.13"},
		},
	}
}

func marketBreadthSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"advancing", "declining", "unchanged"},
		"properties": map[string]any{
			"advancing": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"declining": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"unchanged": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
		},
	}
}

func marketTurnoverSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"amount", "currency"},
		"properties": map[string]any{
			"amount":   map[string]any{"type": "string", "example": "12002494200.00"},
			"currency": map[string]any{"type": "string", "enum": []string{"CNY"}},
		},
	}
}

func signalParametersSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"window": map[string]any{
				"type": "integer", "enum": []int{20, 60, 120}, "default": 20,
				"description": "放量、突破、新高和强势均使用的历史交易日窗口。",
			},
			"multiple": map[string]any{
				"type": "number", "enum": []float64{1.5, 2}, "default": 1.5,
				"description": "仅放量信号使用的成交量倍数。",
			},
			"top_percent": map[string]any{
				"type": "integer", "enum": []int{10, 20}, "default": 10,
				"description": "仅强势信号使用的收益率排名比例。",
			},
		},
	}
}

func marketSignalsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"type", "params", "as_of", "source", "signals"},
		"properties": map[string]any{
			"type":    map[string]any{"type": "string", "enum": []string{"volume_surge", "breakout", "new_high", "strong"}},
			"params":  map[string]any{"$ref": "#/components/schemas/SignalParameters"},
			"as_of":   map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"source":  map[string]any{"$ref": "#/components/schemas/MarketDataSource"},
			"signals": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/SignalResult"}},
		},
	}
}

func signalResultSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "signal"},
		"properties": map[string]any{
			"code":   map[string]any{"type": "string", "example": "300750.SZ"},
			"name":   map[string]any{"type": "string", "example": "宁德时代"},
			"signal": map[string]any{"type": "string", "enum": []string{"volume_surge", "breakout", "new_high", "strong"}},
		},
	}
}

func marketRankingsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"metric", "as_of", "source", "data", "pagination"},
		"properties": map[string]any{
			"metric":     map[string]any{"type": "string", "enum": []string{"gain", "loss", "turnover_amount", "turnover_rate"}},
			"as_of":      map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"source":     map[string]any{"$ref": "#/components/schemas/MarketDataSource"},
			"data":       map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/MarketRanking"}},
			"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
		},
	}
}

func marketRankingSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"rank", "code", "name", "value", "close", "change", "change_percent", "turnover_amount", "turnover_rate"},
		"properties": map[string]any{
			"rank":            map[string]any{"type": "integer", "minimum": 1, "example": 1},
			"code":            map[string]any{"type": "string", "example": "300750.SZ"},
			"name":            map[string]any{"type": "string", "example": "宁德时代"},
			"value":           map[string]any{"type": "string", "description": "当前 metric 对应的排行值", "example": "21.42"},
			"close":           map[string]any{"type": "string", "example": "190.12"},
			"change":          map[string]any{"type": "string", "example": "2.67"},
			"change_percent":  map[string]any{"type": "string", "example": "1.42"},
			"turnover_amount": map[string]any{"type": "string", "example": "7633021600.00"},
			"turnover_rate":   map[string]any{"type": "string", "example": "3.42"},
		},
	}
}

func healthResponse() map[string]any {
	return map[string]any{
		"description": "服务正常",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema":  map[string]any{"$ref": "#/components/schemas/HealthResponse"},
				"example": map[string]any{"status": "ok"},
			},
		},
	}
}

func errorResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/ErrorResponse"},
			},
		},
	}
}
