package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	healthPath = "/health"
)

type response struct {
	Status string `json:"status"`
}

// RegisterRoutes registers the health endpoint on an API version group.
func RegisterRoutes(router *gin.RouterGroup) {
	router.GET(healthPath, healthHandler)
}

func healthHandler(context *gin.Context) {
	context.JSON(http.StatusOK, response{Status: "ok"})
}
