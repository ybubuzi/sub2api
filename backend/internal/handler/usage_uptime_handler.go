package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Uptime handles request availability chart data for the current user.
// GET /api/v1/usage/uptime?window=1h|6h&dimension=all|api_key|model
func (h *UsageHandler) Uptime(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	query := service.UptimeChartQuery{
		Scope:     service.UptimeScopeUser,
		UserID:    &subject.UserID,
		Window:    c.Query("window"),
		Dimension: service.UptimeDimension(c.Query("dimension")),
	}
	result, err := h.uptimeService.GetChart(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
