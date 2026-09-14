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
		"/api/v1/health":       healthPath(),
		"/api/v1/openapi.json": openAPIPath(),
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
			"Response":          responseSchema(),
			"ErrorResponse":     errorSchema(),
			"PaginationRequest": paginationRequestSchema(),
			"PaginationMeta":    paginationMetaSchema(),
			"PaginatedResponse": paginatedResponseSchema(),
			"HealthResponse":    healthSchema(),
			"DemoCounts":        demoCountsSchema(),
			"DemoSampleStock":   demoSampleStockSchema(),
			"DemoStatus":        demoStatusSchema(),
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
		"required": []string{"instruments", "daily_bars", "financial_metrics"},
		"properties": map[string]any{
			"instruments":       map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"daily_bars":        map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"financial_metrics": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
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
			"seed_version": map[string]any{"type": "string", "example": "fnd-003-demo-v1"},
			"as_of":        map[string]any{"type": "string", "format": "date", "nullable": true, "example": "2024-06-28"},
			"counts":       map[string]any{"$ref": "#/components/schemas/DemoCounts"},
			"sample_stocks": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/components/schemas/DemoSampleStock"},
			},
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
