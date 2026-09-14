package api

// OpenAPIDocument 返回服务实际输出的 API v1 OpenAPI 契约。
// 使用 map 构造可以保持该契约包不依赖 Gin，同时让新增业务接口按相同 schema 扩展。
func OpenAPIDocument() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Stock Quant API",
			"version":     "v1",
			"description": "股票量化服务 API v1 契约",
		},
		"paths": map[string]any{
			"/api/v1/health": map[string]any{
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
			},
			"/api/v1/openapi.json": map[string]any{
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
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"Response": map[string]any{
					"type":     "object",
					"required": []string{"data"},
					"properties": map[string]any{
						"data": map[string]any{},
					},
				},
				"ErrorResponse": map[string]any{
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
							},
						},
						"message": map[string]any{"type": "string"},
						"details": map[string]any{
							"type":                 "object",
							"additionalProperties": true,
						},
					},
				},
				"PaginationRequest": map[string]any{
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
				},
				"PaginationMeta": map[string]any{
					"type":     "object",
					"required": []string{"page", "page_size", "total", "total_pages"},
					"properties": map[string]any{
						"page":        map[string]any{"type": "integer", "minimum": DefaultPage},
						"page_size":   map[string]any{"type": "integer", "minimum": 1, "maximum": MaxPageSize},
						"total":       map[string]any{"type": "integer", "format": "int64", "minimum": 0},
						"total_pages": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
					},
				},
				"PaginatedResponse": map[string]any{
					"type":     "object",
					"required": []string{"data", "pagination"},
					"properties": map[string]any{
						"data":       map[string]any{"type": "array", "items": map[string]any{}},
						"pagination": map[string]any{"$ref": "#/components/schemas/PaginationMeta"},
					},
				},
				"HealthResponse": map[string]any{
					"type":     "object",
					"required": []string{"status"},
					"properties": map[string]any{
						"status": map[string]any{"type": "string", "example": "ok"},
					},
				},
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
