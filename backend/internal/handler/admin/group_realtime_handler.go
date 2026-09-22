package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *OpsHandler) GetGroupRealtime(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	snapshot, err := h.opsService.GetGroupRealtime(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, snapshot)
}

func (h *OpsHandler) GetGroupRealtimeHistory(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service unavailable")
		return
	}
	minutes, err := strconv.Atoi(c.DefaultQuery("window_minutes", "60"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid window_minutes")
		return
	}
	var groupID *int64
	if raw, exists := c.GetQuery("group_id"); exists {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 0 {
			response.Error(c, http.StatusBadRequest, "Invalid group_id")
			return
		}
		groupID = &id
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	result, err := h.opsService.GetGroupRealtimeHistory(ctx, minutes, groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
