package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type RelayBalanceHandler struct {
	svc *service.RelayBalanceService
}

func NewRelayBalanceHandler(svc *service.RelayBalanceService) *RelayBalanceHandler {
	return &RelayBalanceHandler{svc: svc}
}

type relayBalanceStationRequest struct {
	Name           string `json:"name" binding:"required"`
	BaseURL        string `json:"base_url" binding:"required"`
	Script         string `json:"script" binding:"required"`
	PackageJSON    string `json:"package_json"`
	CronExpression string `json:"cron_expression"`
	Enabled        bool   `json:"enabled"`
}

func (h *RelayBalanceHandler) ListStations(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := service.RelayBalanceListParams{Page: page, PageSize: pageSize, Search: strings.TrimSpace(c.Query("search"))}
	if v := strings.TrimSpace(c.Query("enabled")); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			response.BadRequest(c, "Invalid enabled")
			return
		}
		params.Enabled = &enabled
	}
	items, total, err := h.svc.ListStations(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *RelayBalanceHandler) GetStation(c *gin.Context) {
	id, ok := parseRelayBalanceID(c)
	if !ok {
		return
	}
	station, err := h.svc.GetStation(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, station)
}

func (h *RelayBalanceHandler) CreateStation(c *gin.Context) {
	var req relayBalanceStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	station, err := h.svc.CreateStation(c.Request.Context(), relayBalanceInputFromRequest(req, getAdminIDFromContext(c)))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, station)
}

func (h *RelayBalanceHandler) UpdateStation(c *gin.Context) {
	id, ok := parseRelayBalanceID(c)
	if !ok {
		return
	}
	var req relayBalanceStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	station, err := h.svc.UpdateStation(c.Request.Context(), id, relayBalanceInputFromRequest(req, getAdminIDFromContext(c)))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, station)
}

func (h *RelayBalanceHandler) DeleteStation(c *gin.Context) {
	id, ok := parseRelayBalanceID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteStation(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Relay balance station deleted successfully"})
}

func (h *RelayBalanceHandler) RunNow(c *gin.Context) {
	id, ok := parseRelayBalanceID(c)
	if !ok {
		return
	}
	run, err := h.svc.RunNow(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, run)
}

func (h *RelayBalanceHandler) ListRuns(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := service.RelayBalanceRunListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      strings.TrimSpace(c.Query("status")),
		SortOrder:   strings.TrimSpace(c.DefaultQuery("sort_order", "desc")),
		Granularity: strings.TrimSpace(c.Query("granularity")),
	}
	if v := strings.TrimSpace(c.Query("station_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid station_id")
			return
		}
		params.StationID = id
	}
	if v := strings.TrimSpace(c.Query("started_from")); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "Invalid started_from")
			return
		}
		params.StartedFrom = &parsed
	}
	if v := strings.TrimSpace(c.Query("started_to")); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "Invalid started_to")
			return
		}
		params.StartedTo = &parsed
	}
	runs, total, err := h.svc.ListRuns(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, runs, total, page, pageSize)
}

func (h *RelayBalanceHandler) GetTrend(c *gin.Context) {
	params := service.RelayBalanceTrendParams{
		Granularity: strings.TrimSpace(c.Query("granularity")),
	}
	if v := strings.TrimSpace(c.Query("started_from")); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "Invalid started_from")
			return
		}
		params.StartedFrom = &parsed
	}
	if v := strings.TrimSpace(c.Query("started_to")); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "Invalid started_to")
			return
		}
		params.StartedTo = &parsed
	}
	trend, err := h.svc.GetTrend(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, trend)
}

func (h *RelayBalanceHandler) GetTotalBalance(c *gin.Context) {
	total, err := h.svc.GetTotalBalance(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, total)
}

func relayBalanceInputFromRequest(req relayBalanceStationRequest, adminID int64) service.RelayBalanceStationInput {
	return service.RelayBalanceStationInput{Name: req.Name, BaseURL: req.BaseURL, Script: req.Script, PackageJSON: req.PackageJSON, CronExpression: req.CronExpression, Enabled: req.Enabled, CreatedBy: adminID}
}

func parseRelayBalanceID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid relay balance station ID")
		return 0, false
	}
	return id, true
}
