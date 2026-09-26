package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const path = "/health"

type response struct {
	Status string `json:"status"`
}

// RegisterRoutes 将进程存活检查留在 Interface 层，避免 Composition Root 持有 HTTP 响应细节。
func RegisterRoutes(router *gin.RouterGroup) {
	router.GET(path, handler)
}

func handler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, response{Status: "ok"})
}
