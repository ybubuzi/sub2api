package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Uptime handles site-wide availability chart data for administrators.
// GET /api/v1/admin/usage/uptime?window=1h|6h&dimension=all|model|group
func (h *UsageHandler) Uptime(c *gin.Context) {
	query := service.UptimeChartQuery{
		Scope:     service.UptimeScopeAdmin,
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
