package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	heartbeatPath = "/heartbeat"
	probePath     = "/probe"
)

type response struct {
	Status string `json:"status"`
}

// RegisterRoutes registers process-level health endpoints on the HTTP router.
func RegisterRoutes(router *gin.Engine) {
	router.GET(heartbeatPath, heartbeatHandler)
	router.GET(probePath, probeHandler)
}

func heartbeatHandler(context *gin.Context) {
	context.JSON(http.StatusOK, response{Status: "ok"})
}

func probeHandler(context *gin.Context) {
	context.JSON(http.StatusOK, response{Status: "ok"})
}
