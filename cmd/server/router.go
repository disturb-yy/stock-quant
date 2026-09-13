package main

import (
	"log/slog"
	"net/http"

	"example.com/stock-ddd/internal/health"
	"example.com/stock-ddd/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	defaultHTTPAddress = ":8357"
	apiV1Prefix        = "/api/v1"
)

func newRouter(applicationLogger *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(logger.GinMiddleware(applicationLogger), gin.Recovery())
	health.RegisterRoutes(router.Group(apiV1Prefix))
	return router
}

func newHTTPServer(applicationLogger *slog.Logger) *http.Server {
	return &http.Server{
		Addr:    defaultHTTPAddress,
		Handler: newRouter(applicationLogger),
	}
}
