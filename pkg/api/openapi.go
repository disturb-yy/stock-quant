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
		"/api/v1/health":                            healthPath(),
		"/api/v1/markets/overview":                  marketOverviewPath(),
		"/api/v1/markets/sectors":                   marketSectorsPath(),
		"/api/v1/markets/signals":                   marketSignalsPath(),
		"/api/v1/markets/rankings":                  marketRankingsPath(),
		"/api/v1/stocks/{symbol}":                   stockOverviewPath(),
		"/api/v1/stocks/{symbol}/bars":              stockBarsPath(),
		"/api/v1/stocks/{symbol}/financials":        stockFinancialsPath(),
		"/api/v1/stocks/{symbol}/valuation":         stockValuationPath(),
		"/api/v1/screeners":                         screenersPath(),
		"/api/v1/screeners/{id}":                    screenerByIDPath(),
		"/api/v1/screeners/run":                     screenerRunPath(),
		"/api/v1/stock-pools":                       stockPoolsPath(),
		"/api/v1/stock-pools/{id}":                  stockPoolByIDPath(),
		"/api/v1/stock-pools/{id}/summary":          stockPoolSummaryPath(),
		"/api/v1/stock-pools/{id}/members":          stockPoolMembersPath(),
		"/api/v1/stock-pools/{id}/members/{symbol}": stockPoolMemberBySymbolPath(),
		"/api/v1/research":                          researchPath(),
		"/api/v1/research/{id}":                     researchByIDPath(),
		"/api/v1/openapi.json":                      openAPIPath(),
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

func stockBarsPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getStockBars",
			"summary":     "读取股票研究型日线",
			"parameters": []any{
				map[string]any{
					"name": "symbol", "in": "path", "required": true,
					"description": "直接使用 Markets API 返回的 code，不做前端转换。",
					"schema":      map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
				},
				map[string]any{
					"name": "timeframe", "in": "query", "required": false,
					"schema": map[string]any{"type": "string", "enum": []string{"1d"}, "default": "1d"},
				},
				map[string]any{
					"name": "adjust", "in": "query", "required": false,
					"schema": map[string]any{"type": "string", "enum": []string{"none", "qfq", "hfq"}, "default": "none"},
				},
				map[string]any{
					"name": "range", "in": "query", "required": false,
					"description": "与 from/to 互斥；省略时使用最近 120 个交易日。",
					"schema":      map[string]any{"type": "string", "enum": []string{"20d", "60d", "120d", "all"}, "default": "120d"},
				},
				map[string]any{
					"name": "from", "in": "query", "required": false,
					"schema": map[string]any{"type": "string", "format": "date", "example": "2024-06-03"},
				},
				map[string]any{
					"name": "to", "in": "query", "required": false,
					"schema": map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
				},
				map[string]any{
					"name": "benchmark", "in": "query", "required": false,
					"schema": map[string]any{"type": "string", "enum": []string{"000300.SH"}, "example": "000300.SH"},
				},
			},
			"responses": map[string]any{
				"200": stockBarsResponse(),
				"400": errorResponse("股票行情参数无效"),
				"404": errorResponse("股票不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("股票行情数据不可用"),
			},
		},
	}
}

func stockFinancialsPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getStockFinancials",
			"summary":     "读取股票财务摘要、趋势与简化报表",
			"parameters": []any{
				map[string]any{
					"name": "symbol", "in": "path", "required": true,
					"description": "直接使用 Markets API 返回的 code，不做前端转换。",
					"schema":      map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
				},
				map[string]any{
					"name": "period", "in": "query", "required": false,
					"schema": map[string]any{"type": "string", "enum": []string{"annual", "quarterly"}, "default": "annual"},
				},
				map[string]any{
					"name": "range", "in": "query", "required": false,
					"description": "年度最多返回 3/5 个财年；季度返回所选年数内最多 12/20 个报告期。",
					"schema":      map[string]any{"type": "string", "enum": []string{"3y", "5y"}, "default": "5y"},
				},
			},
			"responses": map[string]any{
				"200": stockFinancialsResponse(),
				"400": errorResponse("股票财务参数无效"),
				"404": errorResponse("股票不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("股票财务数据暂不可用"),
			},
		},
	}
}

func stockValuationPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getStockValuation",
			"summary":     "读取股票估值、历史分位与同业中位数",
			"parameters": []any{
				map[string]any{
					"name": "symbol", "in": "path", "required": true,
					"description": "直接使用 Markets API 返回的 code，不做前端转换。",
					"schema":      map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
				},
				map[string]any{
					"name": "range", "in": "query", "required": false,
					"description": "以目标股票最新可用估值观测为锚点，不使用服务机器当前日期。",
					"schema":      map[string]any{"type": "string", "enum": []string{"3y", "5y"}, "default": "5y"},
				},
			},
			"responses": map[string]any{
				"200": stockValuationResponse(),
				"400": errorResponse("股票估值参数无效"),
				"404": errorResponse("股票不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("股票估值数据暂不可用"),
			},
		},
	}
}

func screenerRunPath() map[string]any {
	return map[string]any{
		"post": map[string]any{
			"operationId": "runScreener",
			"summary":     "执行结构化量化选股",
			"requestBody": map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/ScreenerRunRequest"},
						"example": map[string]any{"spec": map[string]any{
							"universe_id": "cn_a_share_active",
							"filters":     []map[string]any{{"field_id": "valuation.pe_ttm", "operator": "lte", "value": "15"}},
							"ranking":     map[string]any{"field_id": "technical.volume", "direction": "desc"},
							"top_n":       20,
						}},
					},
				},
			},
			"responses": map[string]any{
				"200": jsonReferenceResponse("量化选股临时执行结果", "#/components/schemas/ScreenerRunResponse"),
				"400": errorResponse("选股条件或规格无效"),
				"404": errorResponse("选股 Universe 不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("选股数据或执行快照不可用"),
			},
		},
	}
}

func screenersPath() map[string]any {
	return map[string]any{
		"post": map[string]any{
			"operationId": "createScreener",
			"summary":     "保存规范化的量化选股方案",
			"requestBody": screenerRequestBody("ScreenerCreateRequest", map[string]any{
				"name": "低估值方案", "description": "仅保存条件，不保存执行结果。",
				"spec": map[string]any{
					"universe_id": "cn_a_share_active", "filters": []map[string]any{{"field_id": "valuation.pe_ttm", "operator": "lte", "value": "15"}},
					"ranking": map[string]any{"field_id": "technical.volume", "direction": "desc"}, "top_n": 20,
				},
			}),
			"responses": savedScreenerResponses("保存方案", "Screener"),
		},
		"get": map[string]any{
			"operationId": "listScreeners",
			"summary":     "按最近更新时间分页读取保存方案",
			"parameters":  screenerPaginationParameters(),
			"responses":   savedScreenerListResponses(),
		},
	}
}

func screenerByIDPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getScreener",
			"summary":     "读取当前保存方案",
			"parameters":  []any{screenerIDParameter()},
			"responses":   savedScreenerResponses("保存方案", "Screener"),
		},
		"put": map[string]any{
			"operationId": "updateScreener",
			"summary":     "按版本号更新保存方案",
			"parameters":  []any{screenerIDParameter()},
			"requestBody": screenerRequestBody("ScreenerUpdateRequest", map[string]any{
				"name": "更新后的方案", "description": nil,
				"spec": map[string]any{
					"universe_id": "cn_a_share_active", "filters": []map[string]any{},
					"ranking": map[string]any{"field_id": "technical.close", "direction": "desc"}, "top_n": 10,
				}, "version": 1,
			}),
			"responses": map[string]any{
				"200": jsonReferenceResponse("更新后的保存方案", "#/components/schemas/Screener"),
				"400": errorResponse("保存方案参数无效"),
				"404": errorResponse("保存方案不存在"),
				"405": errorResponse("请求方法不被允许"),
				"409": errorResponse("保存方案版本已过期"),
				"503": errorResponse("保存方案数据不可用"),
			},
		},
	}
}

func screenerRequestBody(schema string, example map[string]any) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema":  map[string]any{"$ref": "#/components/schemas/" + schema},
				"example": example,
			},
		},
	}
}

func screenerIDParameter() map[string]any {
	return map[string]any{
		"name": "id", "in": "path", "required": true,
		"schema": map[string]any{"type": "integer", "format": "int64", "minimum": 1, "example": 1},
	}
}

func screenerPaginationParameters() []any {
	return []any{
		map[string]any{"name": "page", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": DefaultPage, "default": DefaultPage}},
		map[string]any{"name": "page_size", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize, "default": DefaultPageSize}},
	}
}

func savedScreenerResponses(description, schema string) map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse(description, "#/components/schemas/"+schema),
		"400": errorResponse("保存方案参数无效"),
		"404": errorResponse("保存方案不存在"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("保存方案数据不可用"),
	}
}

func savedScreenerListResponses() map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse("保存方案列表", "#/components/schemas/ScreenerListResponse"),
		"400": errorResponse("分页参数无效"),
		"404": errorResponse("请求的资源不存在"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("保存方案数据不可用"),
	}
}

func stockPoolsPath() map[string]any {
	return map[string]any{
		"post": map[string]any{
			"operationId": "createStockPool",
			"summary":     "创建来源固定为 manual 的手工股票池",
			"requestBody": stockPoolRequestBody(),
			"responses":   stockPoolResponses("新建股票池"),
		},
		"get": map[string]any{
			"operationId": "listStockPools",
			"summary":     "按名称搜索并按最近更新时间稳定分页读取股票池",
			"description": "仅搜索 name；q 按字面包含匹配。固定排序为 updated_at DESC、id DESC，不支持客户端指定排序字段。",
			"parameters":  stockPoolListParameters(),
			"responses":   stockPoolListResponses(),
		},
	}
}

func stockPoolByIDPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getStockPool",
			"summary":     "读取股票池概览",
			"parameters":  []any{stockPoolIDParameter()},
			"responses":   stockPoolResponses("股票池概览"),
		},
	}
}

func stockPoolSummaryPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getStockPoolSummary",
			"summary":     "读取股票池来源与基础画像摘要",
			"description": "来源只读取真实持久化事实；行业、PE、ROE 分别公开 availability、as_of、provenance 和不可用原因。",
			"parameters":  []any{stockPoolIDParameter()},
			"responses":   stockPoolSummaryResponses(),
		},
	}
}

func stockPoolMembersPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "listStockPoolMembers",
			"summary":     "按原始股票代码升序分页读取股票池成员",
			"description": "成员 symbol 直接复用 instruments.code/Markets code，不按名称、排名或本地映射转换。",
			"parameters":  append([]any{stockPoolIDParameter()}, stockPoolMemberPaginationParameters()...),
			"responses":   stockPoolMemberListResponses(),
		},
		"post": map[string]any{
			"operationId": "addStockPoolMember",
			"summary":     "向股票池添加一只真实股票",
			"parameters":  []any{stockPoolIDParameter()},
			"requestBody": stockPoolMemberRequestBody(),
			"responses":   stockPoolMemberAddResponses(),
		},
	}
}

func stockPoolMemberBySymbolPath() map[string]any {
	return map[string]any{
		"delete": map[string]any{
			"operationId": "deleteStockPoolMember",
			"summary":     "从股票池删除一只真实股票",
			"parameters":  []any{stockPoolIDParameter(), stockPoolMemberSymbolParameter()},
			"responses":   stockPoolMemberDeleteResponses(),
		},
	}
}

func stockPoolRequestBody() map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema":  map[string]any{"$ref": "#/components/schemas/StockPoolCreateRequest"},
				"example": map[string]any{"name": "红利观察", "description": "仅供长期观察。"},
			},
		},
	}
}

func stockPoolListParameters() []any {
	return []any{
		map[string]any{"name": "q", "in": "query", "required": false, "description": "按 name 字面包含搜索，首尾空白会忽略。", "schema": map[string]any{"type": "string", "maxLength": 100}},
		map[string]any{"name": "page", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": DefaultPage, "default": DefaultPage}},
		map[string]any{"name": "page_size", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize, "default": DefaultPageSize}},
	}
}

func stockPoolMemberPaginationParameters() []any {
	return []any{
		map[string]any{"name": "page", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": DefaultPage, "default": DefaultPage}},
		map[string]any{"name": "page_size", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize, "default": DefaultPageSize}},
	}
}

func stockPoolIDParameter() map[string]any {
	return map[string]any{
		"name": "id", "in": "path", "required": true,
		"schema": map[string]any{"type": "integer", "format": "int64", "minimum": 1, "example": 1},
	}
}

func stockPoolMemberSymbolParameter() map[string]any {
	return map[string]any{
		"name": "symbol", "in": "path", "required": true,
		"description": "直接使用 Markets API 返回的 code，不做名称、排名或本地映射转换。",
		"schema":      map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
	}
}

func stockPoolResponses(description string) map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse(description, "#/components/schemas/StockPool"),
		"400": errorResponse("股票池参数无效"),
		"404": errorResponse("股票池不存在"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("股票池数据不可用"),
	}
}

func stockPoolSummaryResponses() map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse("股票池来源与基础画像摘要", "#/components/schemas/StockPoolSummary"),
		"400": errorResponse("股票池参数无效"),
		"404": errorResponse("股票池不存在"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("股票池画像数据不可用"),
	}
}

func stockPoolListResponses() map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse("股票池列表", "#/components/schemas/StockPoolListResponse"),
		"400": errorResponse("搜索或分页参数无效"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("股票池数据不可用"),
	}
}

func stockPoolMemberRequestBody() map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema":  map[string]any{"$ref": "#/components/schemas/StockPoolMemberAddRequest"},
				"example": map[string]any{"symbol": "000001.SZ"},
			},
		},
	}
}

func stockPoolMemberListResponses() map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse("股票池成员列表", "#/components/schemas/StockPoolMemberListResponse"),
		"400": errorResponse("股票池成员分页或身份参数无效"),
		"404": errorResponse("股票池不存在"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("股票池成员数据不可用"),
	}
}

func stockPoolMemberAddResponses() map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse("添加股票池成员结果", "#/components/schemas/StockPoolMemberAddResponse"),
		"400": errorResponse("股票池成员参数无效"),
		"404": errorResponse("股票池或股票不存在"),
		"405": errorResponse("请求方法不被允许"),
		"409": errorResponse("股票已在股票池中"),
		"503": errorResponse("股票池成员数据不可用"),
	}
}

func stockPoolMemberDeleteResponses() map[string]any {
	return map[string]any{
		"200": jsonReferenceResponse("删除股票池成员结果", "#/components/schemas/StockPoolMemberDeleteResponse"),
		"400": errorResponse("股票池成员参数无效"),
		"404": errorResponse("股票池或成员不存在"),
		"405": errorResponse("请求方法不被允许"),
		"503": errorResponse("股票池成员数据不可用"),
	}
}

func stockValuationResponse() map[string]any {
	return map[string]any{
		"description": "股票估值、历史分位与同业中位数",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/StockValuation"},
				"example": map[string]any{
					"symbol": "000001.SZ", "name": "平安银行", "requested_range": "5y",
					"effective_range": map[string]any{"from": "2020-06-26", "to": "2024-06-28"}, "as_of": "2024-06-28",
					"metrics": map[string]any{
						"pe_ttm": map[string]any{"current": map[string]any{"value": "7.40", "as_of": "2024-06-28", "basis": "ttm"}, "history": []map[string]any{{"as_of": "2020-06-26", "value": "4.20"}}, "percentile": map[string]any{"value": "100.00", "sample_size": 5, "range_from": "2020-06-26", "range_to": "2024-06-28", "method": "inclusive_rank"}, "position": "high"},
						"pb":     map[string]any{"current": map[string]any{"value": "0.52", "as_of": "2024-06-28", "basis": "latest_daily_basic"}, "history": []map[string]any{}, "percentile": map[string]any{"value": nil, "sample_size": 0, "range_from": nil, "range_to": nil, "method": "inclusive_rank"}, "position": nil},
						"ps_ttm": map[string]any{"current": map[string]any{"value": "1.40", "as_of": "2024-06-28", "basis": "ttm"}, "history": []map[string]any{}, "percentile": map[string]any{"value": nil, "sample_size": 0, "range_from": nil, "range_to": nil, "method": "inclusive_rank"}, "position": nil},
					},
					"industry_comparisons": []map[string]any{{"industry": map[string]any{"code": "BANK", "name": "银行"}, "as_of": "2024-06-28", "metrics": map[string]any{"pe_ttm": map[string]any{"value": "6.00", "sample_size": 3}, "pb": map[string]any{"value": "0.70", "sample_size": 3}, "ps_ttm": map[string]any{"value": "1.20", "sample_size": 3}}}},
					"source":               map[string]any{"mode": "demo", "provider": "mysql-demo-fixture", "seed_version": "fnd-003-demo-v8", "as_of": "2024-06-28"},
				},
			},
		},
	}
}

func stockFinancialsResponse() map[string]any {
	return map[string]any{
		"description": "股票财务摘要、趋势与简化报表",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/StockFinancials"},
				"example": map[string]any{
					"symbol": "000001.SZ", "name": "平安银行", "period": "annual", "requested_range": "5y",
					"effective_range":    map[string]any{"from": "2019-12-31", "to": "2023-12-31"},
					"reporting_currency": "CNY", "amount_unit": "CNY", "latest_report_date": "2023-12-31",
					"summary": map[string]any{"period_end": "2023-12-31", "published_at": "2024-04-30", "revenue": "1600.00", "revenue_yoy_pct": "11.11", "net_profit": "272.00", "net_profit_yoy_pct": "14.48", "gross_margin_pct": "41.00", "roe_pct": "23.78", "operating_cash_flow": "345.00", "free_cash_flow": "240.00", "debt_to_asset_pct": "45.00", "current_ratio": "2.53"},
					"reports": []map[string]any{{"period_end": "2023-12-31", "fiscal_year": 2023, "fiscal_quarter": nil, "published_at": "2024-04-30", "income": map[string]any{"revenue": "1600.00", "gross_profit": "656.00", "operating_profit": "421.60", "net_profit": "272.00"}, "balance": map[string]any{"cash_and_equivalents": "283.50", "accounts_receivable": "202.50", "inventory": "243.00", "current_assets": "810.00", "current_liabilities": "320.00", "total_assets": "2080.00", "total_liabilities": "936.00", "total_equity": "1144.00"}, "cash_flow": map[string]any{"operating_cash_flow": "345.00", "capital_expenditure": "105.00", "investing_cash_flow": "-75.90", "financing_cash_flow": "-34.50", "net_cash_change": "234.60"}, "indicators": map[string]any{"revenue_yoy_pct": "11.11", "net_profit_yoy_pct": "14.48", "gross_margin_pct": "41.00", "roe_pct": "23.78", "free_cash_flow": "240.00", "debt_to_asset_pct": "45.00", "current_ratio": "2.53"}}},
					"source":  map[string]any{"mode": "demo", "provider": "mysql-demo-fixture", "seed_version": "fnd-003-demo-v8", "as_of": "2024-06-28"},
				},
			},
		},
	}
}

func stockBarsResponse() map[string]any {
	return map[string]any{
		"description": "股票研究型日线",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/StockBars"},
				"example": map[string]any{
					"symbol": "000001.SZ", "name": "平安银行", "timeframe": "1d", "adjust": "none",
					"effective_range": map[string]any{"from": "2024-06-27", "to": "2024-06-28"},
					"bars":            []map[string]any{{"trade_date": "2024-06-27", "open": "10.12", "high": "10.28", "low": "10.05", "close": "10.22", "volume": int64(78210000), "ma5": nil, "ma20": nil}},
					"benchmark":       nil,
					"source":          map[string]any{"mode": "demo", "provider": "mysql-demo-fixture", "seed_version": "fnd-003-demo-v8"},
				},
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
			"Response":                       responseSchema(),
			"ErrorResponse":                  errorSchema(),
			"PaginationRequest":              paginationRequestSchema(),
			"PaginationMeta":                 paginationMetaSchema(),
			"PaginatedResponse":              paginatedResponseSchema(),
			"HealthResponse":                 healthSchema(),
			"DemoCounts":                     demoCountsSchema(),
			"DemoSampleStock":                demoSampleStockSchema(),
			"DemoStatus":                     demoStatusSchema(),
			"MarketOverview":                 marketOverviewSchema(),
			"MarketSectors":                  marketSectorsSchema(),
			"MarketSector":                   marketSectorSchema(),
			"SectorLeader":                   sectorLeaderSchema(),
			"MarketDataSource":               marketDataSourceSchema(),
			"MarketIndex":                    marketIndexSchema(),
			"MarketBreadth":                  marketBreadthSchema(),
			"MarketTurnover":                 marketTurnoverSchema(),
			"MarketSignals":                  marketSignalsSchema(),
			"SignalParameters":               signalParametersSchema(),
			"SignalResult":                   signalResultSchema(),
			"MarketRankings":                 marketRankingsSchema(),
			"MarketRanking":                  marketRankingSchema(),
			"StockOverview":                  stockOverviewSchema(),
			"StockQuote":                     stockQuoteSchema(),
			"StockMetrics":                   stockMetricsSchema(),
			"StockMetric":                    stockMetricSchema(),
			"StockSparkline":                 stockSparklineSchema(),
			"StockSparklinePoint":            stockSparklinePointSchema(),
			"StockBars":                      stockBarsSchema(),
			"StockBar":                       stockBarSchema(),
			"StockEffectiveRange":            stockEffectiveRangeSchema(),
			"StockBenchmark":                 stockBenchmarkSchema(),
			"StockBenchmarkPoint":            stockBenchmarkPointSchema(),
			"StockFinancials":                stockFinancialsSchema(),
			"StockFinancialSummary":          stockFinancialSummarySchema(),
			"StockFinancialReport":           stockFinancialReportSchema(),
			"StockFinancialIncome":           stockFinancialIncomeSchema(),
			"StockFinancialBalance":          stockFinancialBalanceSchema(),
			"StockFinancialCashFlow":         stockFinancialCashFlowSchema(),
			"StockFinancialIndicators":       stockFinancialIndicatorsSchema(),
			"StockFinancialSource":           stockFinancialSourceSchema(),
			"StockValuation":                 stockValuationSchema(),
			"StockValuationMetrics":          stockValuationMetricsSchema(),
			"StockValuationMetric":           stockValuationMetricSchema(),
			"StockValuationCurrent":          stockValuationCurrentSchema(),
			"StockValuationPoint":            stockValuationPointSchema(),
			"StockValuationPercentile":       stockValuationPercentileSchema(),
			"StockIndustry":                  stockIndustrySchema(),
			"StockIndustryComparison":        stockIndustryComparisonSchema(),
			"StockIndustryComparisonMetrics": stockIndustryComparisonMetricsSchema(),
			"StockIndustryMetric":            stockIndustryMetricSchema(),
			"StockValuationSource":           stockValuationSourceSchema(),
			"ScreenerRunRequest":             screenerRunRequestSchema(),
			"ScreenerCreateRequest":          screenerCreateRequestSchema(),
			"ScreenerUpdateRequest":          screenerUpdateRequestSchema(),
			"Screener":                       screenerSchema(),
			"ScreenerListResponse":           screenerListResponseSchema(),
			"ScreenerSpec":                   screenerSpecSchema(),
			"ScreenerFilter":                 screenerFilterSchema(),
			"ScreenerRanking":                screenerRankingSchema(),
			"ScreenerRunResponse":            screenerRunResponseSchema(),
			"ScreenerSnapshot":               screenerSnapshotSchema(),
			"ScreenerUniverse":               screenerUniverseSchema(),
			"ScreenerResult":                 screenerResultSchema(),
			"ScreenerRankingResult":          screenerRankingResultSchema(),
			"ScreenerFieldResult":            screenerFieldResultSchema(),
			"ScreenerSource":                 screenerSourceSchema(),
			"StockPoolCreateRequest":         stockPoolCreateRequestSchema(),
			"StockPool":                      stockPoolSchema(),
			"StockPoolSummary":               stockPoolSummarySchema(),
			"StockPoolSummarySource":         stockPoolSummarySourceSchema(),
			"StockPoolIndustrySummary":       stockPoolIndustrySummarySchema(),
			"StockPoolIndustryBucket":        stockPoolIndustryBucketSchema(),
			"StockPoolMetricSummary":         stockPoolMetricSummarySchema(),
			"StockPoolListResponse":          stockPoolListResponseSchema(),
			"StockPoolMember":                stockPoolMemberSchema(),
			"StockPoolMemberAddRequest":      stockPoolMemberAddRequestSchema(),
			"StockPoolMemberListResponse":    stockPoolMemberListResponseSchema(),
			"StockPoolMemberAddResponse":     stockPoolMemberAddResponseSchema(),
			"StockPoolMemberDeleteResponse":  stockPoolMemberDeleteResponseSchema(),
			"ResearchCreateRequest":          researchCreateRequestSchema(),
			"ResearchProject":                researchProjectSchema(),
			"ResearchListResponse":           researchListResponseSchema(),
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

func stockBarsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"symbol", "name", "timeframe", "adjust", "effective_range", "bars", "benchmark", "source"},
		"properties": map[string]any{
			"symbol":          map[string]any{"type": "string", "example": "000001.SZ"},
			"name":            map[string]any{"type": "string", "example": "平安银行"},
			"timeframe":       map[string]any{"type": "string", "enum": []string{"1d"}},
			"adjust":          map[string]any{"type": "string", "enum": []string{"none", "qfq", "hfq"}},
			"effective_range": map[string]any{"$ref": "#/components/schemas/StockEffectiveRange"},
			"bars":            map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockBar"}},
			"benchmark":       map[string]any{"$ref": "#/components/schemas/StockBenchmark", "nullable": true},
			"source":          map[string]any{"$ref": "#/components/schemas/MarketDataSource"},
		},
	}
}

func stockBarSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"trade_date", "open", "high", "low", "close", "volume", "ma5", "ma20"},
		"properties": map[string]any{
			"trade_date": map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"open":       map[string]any{"type": "string", "example": "10.22"},
			"high":       map[string]any{"type": "string", "example": "10.36"},
			"low":        map[string]any{"type": "string", "example": "10.18"},
			"close":      map[string]any{"type": "string", "example": "10.31"},
			"volume":     map[string]any{"type": "integer", "format": "int64", "minimum": 0, "example": int64(81540000)},
			"ma5":        map[string]any{"type": "string", "nullable": true, "example": "10.08"},
			"ma20":       map[string]any{"type": "string", "nullable": true, "example": "9.96"},
		},
	}
}

func stockEffectiveRangeSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"from", "to"},
		"properties": map[string]any{
			"from": map[string]any{"type": "string", "format": "date", "nullable": true},
			"to":   map[string]any{"type": "string", "format": "date", "nullable": true},
		},
	}
}

func stockBenchmarkSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "points"},
		"properties": map[string]any{
			"code":   map[string]any{"type": "string", "enum": []string{"000300.SH"}},
			"name":   map[string]any{"type": "string", "example": "沪深300"},
			"points": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockBenchmarkPoint"}},
		},
	}
}

func stockBenchmarkPointSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"trade_date", "close", "stock_return_pct", "benchmark_return_pct", "relative_return_pct"},
		"properties": map[string]any{
			"trade_date":           map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"close":                map[string]any{"type": "string", "example": "3401.76"},
			"stock_return_pct":     map[string]any{"type": "string", "example": "8.72"},
			"benchmark_return_pct": map[string]any{"type": "string", "example": "4.10"},
			"relative_return_pct":  map[string]any{"type": "string", "example": "4.62"},
		},
	}
}

func stockFinancialsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"symbol", "name", "period", "requested_range", "effective_range", "reporting_currency", "amount_unit", "latest_report_date", "summary", "reports", "source"},
		"properties": map[string]any{
			"symbol":             map[string]any{"type": "string", "example": "000001.SZ"},
			"name":               map[string]any{"type": "string", "example": "平安银行"},
			"period":             map[string]any{"type": "string", "enum": []string{"annual", "quarterly"}},
			"requested_range":    map[string]any{"type": "string", "enum": []string{"3y", "5y"}},
			"effective_range":    map[string]any{"$ref": "#/components/schemas/StockEffectiveRange"},
			"reporting_currency": map[string]any{"type": "string", "enum": []string{"CNY"}},
			"amount_unit":        map[string]any{"type": "string", "enum": []string{"CNY"}, "description": "金额字段的统一单位。"},
			"latest_report_date": map[string]any{"type": "string", "format": "date", "nullable": true},
			"summary":            map[string]any{"$ref": "#/components/schemas/StockFinancialSummary", "nullable": true},
			"reports":            map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockFinancialReport"}},
			"source":             map[string]any{"$ref": "#/components/schemas/StockFinancialSource"},
		},
	}
}

func stockValuationSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"symbol", "name", "requested_range", "effective_range", "as_of", "metrics", "industry_comparisons", "source"},
		"properties": map[string]any{
			"symbol":               map[string]any{"type": "string", "example": "000001.SZ"},
			"name":                 map[string]any{"type": "string", "example": "平安银行"},
			"requested_range":      map[string]any{"type": "string", "enum": []string{"3y", "5y"}},
			"effective_range":      map[string]any{"$ref": "#/components/schemas/StockEffectiveRange"},
			"as_of":                map[string]any{"type": "string", "format": "date", "nullable": true},
			"metrics":              map[string]any{"$ref": "#/components/schemas/StockValuationMetrics"},
			"industry_comparisons": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockIndustryComparison"}},
			"source":               map[string]any{"$ref": "#/components/schemas/StockValuationSource"},
		},
	}
}

func stockValuationMetricsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"pe_ttm", "pb", "ps_ttm"},
		"properties": map[string]any{
			"pe_ttm": map[string]any{"$ref": "#/components/schemas/StockValuationMetric"},
			"pb":     map[string]any{"$ref": "#/components/schemas/StockValuationMetric"},
			"ps_ttm": map[string]any{"$ref": "#/components/schemas/StockValuationMetric"},
		},
	}
}

func stockValuationMetricSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"current", "history", "percentile", "position"},
		"properties": map[string]any{
			"current":    map[string]any{"$ref": "#/components/schemas/StockValuationCurrent"},
			"history":    map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockValuationPoint"}},
			"percentile": map[string]any{"$ref": "#/components/schemas/StockValuationPercentile"},
			"position":   map[string]any{"type": "string", "nullable": true, "enum": []string{"low", "middle", "high"}},
		},
	}
}

func stockValuationCurrentSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"value", "as_of", "basis"},
		"properties": map[string]any{
			"value": map[string]any{"type": "string", "nullable": true, "example": "7.40"},
			"as_of": map[string]any{"type": "string", "format": "date", "nullable": true, "example": "2024-06-28"},
			"basis": map[string]any{"type": "string", "nullable": true, "enum": []string{"ttm", "latest_daily_basic"}},
		},
	}
}

func stockValuationPointSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"as_of", "value"},
		"properties": map[string]any{
			"as_of": map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
			"value": map[string]any{"type": "string", "example": "7.40"},
		},
	}
}

func stockValuationPercentileSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"value", "sample_size", "range_from", "range_to", "method"},
		"properties": map[string]any{
			"value":       map[string]any{"type": "string", "nullable": true, "example": "66.67"},
			"sample_size": map[string]any{"type": "integer", "minimum": 0},
			"range_from":  map[string]any{"type": "string", "format": "date", "nullable": true},
			"range_to":    map[string]any{"type": "string", "format": "date", "nullable": true},
			"method":      map[string]any{"type": "string", "enum": []string{"inclusive_rank"}},
		},
	}
}

func stockIndustryComparisonSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"industry", "as_of", "metrics"},
		"properties": map[string]any{
			"industry": map[string]any{"$ref": "#/components/schemas/StockIndustry"},
			"as_of":    map[string]any{"type": "string", "format": "date", "nullable": true},
			"metrics":  map[string]any{"$ref": "#/components/schemas/StockIndustryComparisonMetrics"},
		},
	}
}

func stockIndustrySchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name"},
		"properties": map[string]any{
			"code": map[string]any{"type": "string", "example": "BANK"},
			"name": map[string]any{"type": "string", "example": "银行"},
		},
	}
}

func stockIndustryComparisonMetricsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"pe_ttm", "pb", "ps_ttm"},
		"properties": map[string]any{
			"pe_ttm": map[string]any{"$ref": "#/components/schemas/StockIndustryMetric"},
			"pb":     map[string]any{"$ref": "#/components/schemas/StockIndustryMetric"},
			"ps_ttm": map[string]any{"$ref": "#/components/schemas/StockIndustryMetric"},
		},
	}
}

func stockIndustryMetricSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"value", "sample_size"},
		"properties": map[string]any{
			"value":       map[string]any{"type": "string", "nullable": true, "example": "6.00"},
			"sample_size": map[string]any{"type": "integer", "minimum": 0},
		},
	}
}

func stockValuationSourceSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"mode", "provider", "seed_version", "as_of"},
		"properties": map[string]any{
			"mode":         map[string]any{"type": "string", "enum": []string{"demo", "real", "fallback"}},
			"provider":     map[string]any{"type": "string", "enum": []string{"mysql-demo-fixture", "tushare", "external-real-provider", "local-fixture-fallback"}},
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v8"},
			"as_of":        map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
		},
	}
}

func stockFinancialSummarySchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"period_end", "published_at", "revenue", "revenue_yoy_pct", "net_profit", "net_profit_yoy_pct", "gross_margin_pct", "roe_pct", "operating_cash_flow", "free_cash_flow", "debt_to_asset_pct", "current_ratio"},
		"properties": map[string]any{
			"period_end":          map[string]any{"type": "string", "format": "date"},
			"published_at":        map[string]any{"type": "string", "format": "date", "nullable": true},
			"revenue":             financialNullableAmount("收入金额，单位为 CNY。"),
			"revenue_yoy_pct":     financialNullablePercent("仅与同口径上一财年或上年同期比较。"),
			"net_profit":          financialNullableAmount("归母净利润金额，单位为 CNY。"),
			"net_profit_yoy_pct":  financialNullablePercent("仅与同口径上一财年或上年同期比较。"),
			"gross_margin_pct":    financialNullablePercent("毛利率百分比。"),
			"roe_pct":             financialNullablePercent("ROE 百分比，按归母净利润除以期末归属权益。"),
			"operating_cash_flow": financialNullableAmount("经营现金流金额，单位为 CNY。"),
			"free_cash_flow":      financialNullableAmount("自由现金流金额，口径为经营现金流减资本开支。"),
			"debt_to_asset_pct":   financialNullablePercent("资产负债率百分比。"),
			"current_ratio":       financialNullableRatio("流动比率倍数。"),
		},
	}
}

func stockFinancialReportSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"period_end", "fiscal_year", "fiscal_quarter", "published_at", "income", "balance", "cash_flow", "indicators"},
		"properties": map[string]any{
			"period_end":     map[string]any{"type": "string", "format": "date"},
			"fiscal_year":    map[string]any{"type": "integer", "minimum": 1},
			"fiscal_quarter": map[string]any{"type": "string", "enum": []string{"Q1", "Q2", "Q3", "Q4"}, "nullable": true},
			"published_at":   map[string]any{"type": "string", "format": "date", "nullable": true},
			"income":         map[string]any{"$ref": "#/components/schemas/StockFinancialIncome"},
			"balance":        map[string]any{"$ref": "#/components/schemas/StockFinancialBalance"},
			"cash_flow":      map[string]any{"$ref": "#/components/schemas/StockFinancialCashFlow"},
			"indicators":     map[string]any{"$ref": "#/components/schemas/StockFinancialIndicators"},
		},
	}
}

func stockFinancialIncomeSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"revenue", "gross_profit", "operating_profit", "net_profit"},
		"properties": map[string]any{
			"revenue":          financialNullableAmount("营业收入金额，单位为 CNY。"),
			"gross_profit":     financialNullableAmount("毛利润金额，单位为 CNY。"),
			"operating_profit": financialNullableAmount("营业利润金额，单位为 CNY。"),
			"net_profit":       financialNullableAmount("归母净利润金额，单位为 CNY。"),
		},
	}
}

func stockFinancialBalanceSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"cash_and_equivalents", "accounts_receivable", "inventory", "current_assets", "current_liabilities", "total_assets", "total_liabilities", "total_equity"},
		"properties": map[string]any{
			"cash_and_equivalents": financialNullableAmount("现金及现金等价物金额，单位为 CNY。"),
			"accounts_receivable":  financialNullableAmount("应收账款金额，单位为 CNY。"),
			"inventory":            financialNullableAmount("存货金额，单位为 CNY。"),
			"current_assets":       financialNullableAmount("流动资产金额，单位为 CNY。"),
			"current_liabilities":  financialNullableAmount("流动负债金额，单位为 CNY。"),
			"total_assets":         financialNullableAmount("资产总额，单位为 CNY。"),
			"total_liabilities":    financialNullableAmount("负债总额，单位为 CNY。"),
			"total_equity":         financialNullableAmount("权益总额，单位为 CNY。"),
		},
	}
}

func stockFinancialCashFlowSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"operating_cash_flow", "capital_expenditure", "investing_cash_flow", "financing_cash_flow", "net_cash_change"},
		"properties": map[string]any{
			"operating_cash_flow": financialNullableAmount("经营现金流金额，单位为 CNY。"),
			"capital_expenditure": financialNullableAmount("资本开支现金流出额的非负绝对值，单位为 CNY。"),
			"investing_cash_flow": financialNullableAmount("投资现金流金额，单位为 CNY。"),
			"financing_cash_flow": financialNullableAmount("筹资现金流金额，单位为 CNY。"),
			"net_cash_change":     financialNullableAmount("现金净增加额，单位为 CNY。"),
		},
	}
}

func stockFinancialIndicatorsSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"revenue_yoy_pct", "net_profit_yoy_pct", "gross_margin_pct", "roe_pct", "free_cash_flow", "debt_to_asset_pct", "current_ratio"},
		"properties": map[string]any{
			"revenue_yoy_pct":    financialNullablePercent("收入同比百分比，仅比较同口径期间。"),
			"net_profit_yoy_pct": financialNullablePercent("净利润同比百分比，仅比较同口径期间。"),
			"gross_margin_pct":   financialNullablePercent("毛利率百分比。"),
			"roe_pct":            financialNullablePercent("ROE 百分比，按归母净利润除以期末归属权益。"),
			"free_cash_flow":     financialNullableAmount("自由现金流金额，口径为经营现金流减资本开支。"),
			"debt_to_asset_pct":  financialNullablePercent("资产负债率百分比。"),
			"current_ratio":      financialNullableRatio("流动比率倍数。"),
		},
	}
}

func stockFinancialSourceSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"mode", "provider", "seed_version", "as_of"},
		"properties": map[string]any{
			"mode":         map[string]any{"type": "string", "enum": []string{"demo", "real", "fallback"}},
			"provider":     map[string]any{"type": "string", "enum": []string{"mysql-demo-fixture", "tushare", "external-real-provider", "local-fixture-fallback"}},
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v8"},
			"as_of":        map[string]any{"type": "string", "format": "date", "example": "2024-06-28"},
		},
	}
}

func financialNullableAmount(description string) map[string]any {
	return map[string]any{"type": "string", "nullable": true, "description": description, "example": "1600.00"}
}

func financialNullablePercent(description string) map[string]any {
	return map[string]any{"type": "string", "nullable": true, "description": description, "example": "41.00"}
}

func financialNullableRatio(description string) map[string]any {
	return map[string]any{"type": "string", "nullable": true, "description": description, "example": "2.53"}
}

func screenerRunRequestSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"spec"}, "properties": map[string]any{"spec": map[string]any{"$ref": "#/components/schemas/ScreenerSpec"}}}
}

func screenerCreateRequestSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"name", "spec"},
		"properties": map[string]any{
			"name":        map[string]any{"type": "string", "minLength": 1, "maxLength": 100},
			"description": map[string]any{"type": "string", "maxLength": 500, "nullable": true},
			"spec":        map[string]any{"$ref": "#/components/schemas/ScreenerSpec"},
		},
	}
}

func screenerUpdateRequestSchema() map[string]any {
	schema := screenerCreateRequestSchema()
	schema["required"] = []string{"name", "description", "spec", "version"}
	properties := schema["properties"].(map[string]any)
	properties["version"] = map[string]any{"type": "integer", "format": "int64", "minimum": 1}
	return schema
}

func screenerSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"id", "name", "description", "spec", "version", "created_at", "updated_at"},
		"properties": map[string]any{
			"id":          map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"name":        map[string]any{"type": "string", "example": "低估值方案"},
			"description": map[string]any{"type": "string", "nullable": true, "example": "仅保存条件，不保存执行结果。"},
			"spec":        map[string]any{"$ref": "#/components/schemas/ScreenerSpec"},
			"version":     map[string]any{"type": "integer", "format": "int64", "minimum": 1, "example": 1},
			"created_at":  map[string]any{"type": "string", "format": "date-time"},
			"updated_at":  map[string]any{"type": "string", "format": "date-time"},
		},
	}
}

func screenerListResponseSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"data", "pagination"},
		"properties": map[string]any{
			"data":       map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Screener"}},
			"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
		},
	}
}

func researchPath() map[string]any {
	return map[string]any{
		"post": map[string]any{
			"operationId": "createResearchProject",
			"summary":     "创建 Research 项目",
			"requestBody": researchRequestBody(),
			"responses": map[string]any{
				"200": jsonReferenceResponse("新建 Research 项目", "#/components/schemas/ResearchProject"),
				"400": errorResponse("Research 项目参数无效"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("Research 数据不可用"),
			},
		},
		"get": map[string]any{
			"operationId": "listResearchProjects",
			"summary":     "按最近更新时间分页读取最近 Research 项目",
			"description": "固定按 updated_at DESC、id DESC 排序，不支持客户端指定排序字段。空列表仍返回 200 和 data 数组。",
			"parameters":  researchPaginationParameters(),
			"responses": map[string]any{
				"200": jsonReferenceResponse("最近 Research 项目列表", "#/components/schemas/ResearchListResponse"),
				"400": errorResponse("分页参数无效"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("Research 数据不可用"),
			},
		},
	}
}

func researchByIDPath() map[string]any {
	return map[string]any{
		"get": map[string]any{
			"operationId": "getResearchProject",
			"summary":     "读取 Research 项目基础元数据",
			"parameters":  []any{researchIDParameter()},
			"responses": map[string]any{
				"200": jsonReferenceResponse("Research 项目", "#/components/schemas/ResearchProject"),
				"400": errorResponse("Research 项目 ID 无效"),
				"404": errorResponse("Research 项目不存在"),
				"405": errorResponse("请求方法不被允许"),
				"503": errorResponse("Research 数据不可用"),
			},
		},
	}
}

func researchRequestBody() map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema":  map[string]any{"$ref": "#/components/schemas/ResearchCreateRequest"},
				"example": map[string]any{"name": "平安银行估值研究", "description": "记录估值与行业判断。"},
			},
		},
	}
}

func researchIDParameter() map[string]any {
	return map[string]any{
		"name": "id", "in": "path", "required": true,
		"schema": map[string]any{"type": "integer", "format": "int64", "minimum": 1, "example": 1},
	}
}

func researchPaginationParameters() []any {
	return []any{
		map[string]any{"name": "page", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": DefaultPage, "default": DefaultPage}},
		map[string]any{"name": "page_size", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize, "default": DefaultPageSize}},
	}
}

func researchCreateRequestSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"name"}, "additionalProperties": false,
		"description": "仅允许名称和可空描述；id、created_at、updated_at 由服务端生成。名称和描述会去除首尾空白。",
		"properties": map[string]any{
			"name":        map[string]any{"type": "string", "minLength": 1, "maxLength": 100},
			"description": map[string]any{"type": "string", "maxLength": 500, "nullable": true},
		},
	}
}

func researchProjectSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"id", "name", "description", "created_at", "updated_at"},
		"properties": map[string]any{
			"id":          map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"name":        map[string]any{"type": "string", "example": "平安银行估值研究"},
			"description": map[string]any{"type": "string", "nullable": true, "example": "记录估值与行业判断。"},
			"created_at":  map[string]any{"type": "string", "format": "date-time"},
			"updated_at":  map[string]any{"type": "string", "format": "date-time"},
		},
	}
}

func researchListResponseSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"data", "pagination"},
		"properties": map[string]any{
			"data":       map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/ResearchProject"}},
			"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
		},
	}
}

func stockPoolCreateRequestSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"name"}, "additionalProperties": false,
		"description": "仅允许名称和可空描述；id、source、member_count、created_at、updated_at 均由服务端生成。名称和描述会去除首尾空白。",
		"properties": map[string]any{
			"name":        map[string]any{"type": "string", "minLength": 1, "maxLength": 100},
			"description": map[string]any{"type": "string", "maxLength": 500, "nullable": true},
		},
	}
}

func stockPoolSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"id", "name", "description", "source", "member_count", "created_at", "updated_at"},
		"properties": map[string]any{
			"id":           map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"name":         map[string]any{"type": "string", "example": "红利观察"},
			"description":  map[string]any{"type": "string", "nullable": true, "example": "仅供长期观察。"},
			"source":       map[string]any{"type": "string", "enum": []string{"manual"}},
			"member_count": map[string]any{"type": "integer", "format": "int64", "minimum": 0, "description": "由 t_stock_pool_member 持久化关系实时计算。"},
			"created_at":   map[string]any{"type": "string", "format": "date-time"},
			"updated_at":   map[string]any{"type": "string", "format": "date-time"},
		},
	}
}

func stockPoolSummarySchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"id", "name", "description", "source", "member_count", "created_at", "updated_at", "industry", "pe", "roe"},
		"properties": map[string]any{
			"id":           map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"name":         map[string]any{"type": "string"},
			"description":  map[string]any{"type": "string", "nullable": true},
			"source":       map[string]any{"$ref": "#/components/schemas/StockPoolSummarySource"},
			"member_count": map[string]any{"type": "integer", "format": "int64", "minimum": 0, "description": "由真实成员关系计算。"},
			"created_at":   map[string]any{"type": "string", "format": "date-time"},
			"updated_at":   map[string]any{"type": "string", "format": "date-time"},
			"industry":     map[string]any{"$ref": "#/components/schemas/StockPoolIndustrySummary"},
			"pe":           map[string]any{"$ref": "#/components/schemas/StockPoolMetricSummary"},
			"roe":          map[string]any{"$ref": "#/components/schemas/StockPoolMetricSummary"},
		},
	}
}

func stockPoolSummarySourceSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"type", "reference", "created_at"},
		"properties": map[string]any{
			"type":       map[string]any{"type": "string", "enum": []string{"manual", "screener"}, "description": "screener 仅在 SCR-003 已真实写入来源 metadata 时出现。"},
			"reference":  map[string]any{"type": "string", "nullable": true, "description": "已持久化且可安全公开的来源引用；手工非 Seed Pool 可为空。"},
			"created_at": map[string]any{"type": "string", "format": "date-time"},
		},
	}
}

func stockPoolIndustrySummarySchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"availability", "distribution", "as_of", "provenance", "unavailable_reason"},
		"properties": map[string]any{
			"availability":       map[string]any{"type": "string", "enum": []string{"available", "empty", "unavailable"}},
			"distribution":       map[string]any{"type": "array", "nullable": true, "items": map[string]any{"$ref": "#/components/schemas/StockPoolIndustryBucket"}},
			"as_of":              map[string]any{"type": "string", "format": "date", "nullable": true, "description": "行业关系当前没有独立日期字段时为 null。"},
			"provenance":         map[string]any{"type": "string", "nullable": true, "example": "sector_memberships"},
			"unavailable_reason": map[string]any{"type": "string", "nullable": true},
		},
	}
}

func stockPoolIndustryBucketSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"code", "name", "member_count"},
		"properties": map[string]any{
			"code":         map[string]any{"type": "string"},
			"name":         map[string]any{"type": "string"},
			"member_count": map[string]any{"type": "integer", "format": "int64", "minimum": 1},
		},
	}
}

func stockPoolMetricSummarySchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"availability", "value", "sample_size", "as_of", "basis", "provenance", "unavailable_reason"},
		"properties": map[string]any{
			"availability":       map[string]any{"type": "string", "enum": []string{"available", "empty", "unavailable"}},
			"value":              map[string]any{"type": "string", "nullable": true, "description": "有效成员的算术平均值，保持十进制字符串精度。"},
			"sample_size":        map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"as_of":              map[string]any{"type": "string", "format": "date", "nullable": true},
			"basis":              map[string]any{"type": "string", "nullable": true},
			"provenance":         map[string]any{"type": "string", "nullable": true, "example": "financial_metrics"},
			"unavailable_reason": map[string]any{"type": "string", "nullable": true},
		},
	}
}

func stockPoolListResponseSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"data", "pagination"},
		"properties": map[string]any{
			"data":       map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockPool"}},
			"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
		},
	}
}

func stockPoolMemberSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"symbol", "name"},
		"description": "真实股票身份；symbol 直接复用 Markets 返回的 code。",
		"properties": map[string]any{
			"symbol": map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
			"name":   map[string]any{"type": "string", "example": "平安银行"},
		},
	}
}

func stockPoolMemberAddRequestSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"symbol"}, "additionalProperties": false,
		"description": "仅允许提交 Markets 返回的原始股票 code。",
		"properties": map[string]any{
			"symbol": map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
		},
	}
}

func stockPoolMemberListResponseSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"data", "pagination"},
		"properties": map[string]any{
			"data":       map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/StockPoolMember"}},
			"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
		},
	}
}

func stockPoolMemberAddResponseSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"member", "member_count"},
		"properties": map[string]any{
			"member":       map[string]any{"$ref": "#/components/schemas/StockPoolMember"},
			"member_count": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
		},
	}
}

func stockPoolMemberDeleteResponseSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"symbol", "member_count"},
		"properties": map[string]any{
			"symbol":       map[string]any{"type": "string", "pattern": "^[A-Za-z0-9]{1,16}\\.[A-Za-z]{2,8}$", "example": "000001.SZ"},
			"member_count": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
		},
	}
}

func screenerSpecSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"required":    []string{"universe_id", "filters", "ranking", "top_n"},
		"description": "仅允许已发布的 canonical field_id、平面 AND 条件和单一排序字段；因子字段在 FAC-001 读模型交付前不开放。",
		"properties": map[string]any{
			"universe_id": map[string]any{"type": "string", "enum": []string{"cn_a_share_active"}, "example": "cn_a_share_active"},
			"filters":     map[string]any{"type": "array", "maxItems": 20, "items": map[string]any{"$ref": "#/components/schemas/ScreenerFilter"}},
			"ranking":     map[string]any{"$ref": "#/components/schemas/ScreenerRanking"},
			"top_n":       map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "example": 20},
		},
		"x-field-registry": screenerFieldRegistry(),
	}
}

func screenerFilterSchema() map[string]any {
	return map[string]any{
		"type": "object", "required": []string{"field_id", "operator", "value"},
		"properties": map[string]any{
			"field_id": map[string]any{"type": "string", "enum": screenerFieldIDs()},
			"operator": map[string]any{"type": "string", "enum": []string{"eq", "neq", "gt", "gte", "lt", "lte", "between"}},
			"value": map[string]any{"oneOf": []any{
				map[string]any{"type": "string", "description": "十进制字符串，保持 API 精度。"},
				map[string]any{"type": "array", "minItems": 2, "maxItems": 2, "items": map[string]any{"type": "string"}, "description": "between 的闭区间下限和上限。"},
			}},
		},
	}
}

func screenerRankingSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"field_id", "direction"}, "description": "排序值相同时固定按 Markets code（symbol）升序稳定排序。", "properties": map[string]any{
		"field_id":  map[string]any{"type": "string", "enum": screenerFieldIDs(), "description": "只能引用 registry 中 sortable=true 的字段。"},
		"direction": map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
	}}
}

func screenerRunResponseSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"spec", "snapshot", "universe", "matched_count", "returned_count", "results", "source"}, "properties": map[string]any{
		"spec":           map[string]any{"$ref": "#/components/schemas/ScreenerSpec"},
		"snapshot":       map[string]any{"$ref": "#/components/schemas/ScreenerSnapshot"},
		"universe":       map[string]any{"$ref": "#/components/schemas/ScreenerUniverse"},
		"matched_count":  map[string]any{"type": "integer", "minimum": 0},
		"returned_count": map[string]any{"type": "integer", "minimum": 0},
		"results":        map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/ScreenerResult"}},
		"source":         map[string]any{"$ref": "#/components/schemas/ScreenerSource"},
	}}
}

func screenerSnapshotSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"as_of", "field_as_of", "definition_versions"}, "properties": map[string]any{
		"as_of":               map[string]any{"type": "string", "format": "date"},
		"field_as_of":         map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string", "format": "date"}},
		"definition_versions": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
	}}
}

func screenerUniverseSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"id", "name", "eligible_count"}, "properties": map[string]any{
		"id":             map[string]any{"type": "string", "example": "cn_a_share_active"},
		"name":           map[string]any{"type": "string", "example": "A 股在市股票"},
		"eligible_count": map[string]any{"type": "integer", "minimum": 0},
	}}
}

func screenerResultSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"symbol", "name", "industries", "rank", "ranking", "fields"}, "properties": map[string]any{
		"symbol":     map[string]any{"type": "string", "description": "直接复用 Markets 返回的 code。", "example": "000001.SZ"},
		"name":       map[string]any{"type": "string", "example": "平安银行"},
		"industries": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"rank":       map[string]any{"type": "integer", "minimum": 1},
		"ranking":    map[string]any{"$ref": "#/components/schemas/ScreenerRankingResult"},
		"fields":     map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/ScreenerFieldResult"}},
	}}
}

func screenerRankingResultSchema() map[string]any {
	return screenerFieldResultSchema()
}

func screenerFieldResultSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"field_id", "label", "value", "unit", "basis", "as_of", "unavailable_reason"}, "properties": map[string]any{
		"field_id":           map[string]any{"type": "string", "enum": screenerFieldIDs()},
		"label":              map[string]any{"type": "string"},
		"value":              map[string]any{"type": "string", "nullable": true},
		"unit":               map[string]any{"type": "string"},
		"basis":              map[string]any{"type": "string", "nullable": true},
		"as_of":              map[string]any{"type": "string", "format": "date", "nullable": true},
		"unavailable_reason": map[string]any{"type": "string", "nullable": true},
	}}
}

func screenerSourceSchema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"mode", "provider", "seed_version", "as_of"}, "properties": map[string]any{
		"mode":         map[string]any{"type": "string", "enum": []string{"demo", "real", "fallback"}},
		"provider":     map[string]any{"type": "string"},
		"seed_version": map[string]any{"type": "string"},
		"as_of":        map[string]any{"type": "string", "format": "date"},
	}}
}

func screenerFieldIDs() []string {
	return []string{"market.market_cap", "market.turnover_rate", "technical.close", "technical.volume", "technical.turnover_amount", "valuation.pe_ttm", "valuation.pb", "valuation.ps_ttm", "fundamental.revenue", "fundamental.net_profit", "fundamental.total_assets", "fundamental.total_liabilities", "fundamental.total_equity", "fundamental.operating_cash_flow", "fundamental.roe_pct"}
}

func screenerFieldRegistry() []map[string]any {
	operators := []string{"eq", "neq", "gt", "gte", "lt", "lte", "between"}
	result := make([]map[string]any, 0, len(screenerFieldIDs()))
	for _, id := range screenerFieldIDs() {
		category, label, unit := "", "", ""
		switch id {
		case "market.market_cap":
			category, label, unit = "Market", "总市值", "CNY"
		case "market.turnover_rate":
			category, label, unit = "Market", "换手率", "%"
		case "technical.close":
			category, label, unit = "Technical", "收盘价", "CNY"
		case "technical.volume":
			category, label, unit = "Technical", "成交量", "股"
		case "technical.turnover_amount":
			category, label, unit = "Technical", "成交额", "CNY"
		case "valuation.pe_ttm":
			category, label, unit = "Valuation", "市盈率 TTM", "倍"
		case "valuation.pb":
			category, label, unit = "Valuation", "市净率", "倍"
		case "valuation.ps_ttm":
			category, label, unit = "Valuation", "市销率 TTM", "倍"
		case "fundamental.revenue":
			category, label, unit = "Fundamental", "营业收入", "CNY"
		case "fundamental.net_profit":
			category, label, unit = "Fundamental", "净利润", "CNY"
		case "fundamental.total_assets":
			category, label, unit = "Fundamental", "总资产", "CNY"
		case "fundamental.total_liabilities":
			category, label, unit = "Fundamental", "总负债", "CNY"
		case "fundamental.total_equity":
			category, label, unit = "Fundamental", "股东权益", "CNY"
		case "fundamental.operating_cash_flow":
			category, label, unit = "Fundamental", "经营现金流", "CNY"
		case "fundamental.roe_pct":
			category, label, unit = "Fundamental", "ROE", "%"
		}
		result = append(result, map[string]any{"field_id": id, "category": category, "label": label, "unit": unit, "value_type": "decimal", "operators": operators, "sortable": true})
	}
	return result
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
					string(CodeConflict),
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
		"required": []string{"instruments", "daily_bars", "daily_basics", "financial_metrics", "financial_reports", "valuation_snapshots", "index_snapshots"},
		"properties": map[string]any{
			"instruments":         map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"daily_bars":          map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"daily_basics":        map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"financial_metrics":   map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"financial_reports":   map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"valuation_snapshots": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"index_snapshots":     map[string]any{"type": "integer", "format": "int64", "minimum": 0},
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
				"enum": []string{"mysql-demo-fixture", "tushare", "external-real-provider", "local-fixture-fallback"},
			},
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v8"},
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
				"enum": []string{"mysql-demo-fixture", "tushare", "external-real-provider", "local-fixture-fallback"},
			},
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v8"},
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
