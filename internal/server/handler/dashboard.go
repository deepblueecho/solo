package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/solo-ai/solo/internal/server/service"
)

type DashboardHandler struct {
	svc *service.AgentRunService
}

func NewDashboardHandler(pool *pgxpool.Pool) *DashboardHandler {
	return &DashboardHandler{svc: service.NewAgentRunService(pool)}
}

func (h *DashboardHandler) Live(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	live, err := h.svc.GetDashboardLive(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load live dashboard")
		return
	}
	writeJSON(w, http.StatusOK, live)
}

func (h *DashboardHandler) Insight(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	days := 7
	if raw := r.URL.Query().Get("window_days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			days = parsed
		}
	}
	insight, err := h.svc.GetDashboardInsight(r.Context(), userID, time.Now().UTC().AddDate(0, 0, -days))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load dashboard insight")
		return
	}
	writeJSON(w, http.StatusOK, insight)
}
