package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	authpkg "github.com/Kocoro-lab/Shannon/go/orchestrator/internal/auth"
	"github.com/Kocoro-lab/Shannon/go/orchestrator/internal/daemon"
	"go.uber.org/zap"
)

// WebhookTriggerHandler handles external webhook triggers to agent daemons.
type WebhookTriggerHandler struct {
	hub    *daemon.Hub
	logger *zap.Logger
}

func NewWebhookTriggerHandler(hub *daemon.Hub, logger *zap.Logger) *WebhookTriggerHandler {
	return &WebhookTriggerHandler{hub: hub, logger: logger}
}

type webhookTriggerRequest struct {
	Event  string          `json:"event"`
	Source string          `json:"source"`
	Data   json.RawMessage `json:"data"`
}

// HandleTrigger processes POST /api/v1/agents/{agent_name}/trigger
func (wth *WebhookTriggerHandler) HandleTrigger(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := r.Context().Value(authpkg.UserContextKey).(*authpkg.UserContext)
	if !ok || userCtx == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	agentName := r.PathValue("agent_name")
	if agentName == "" {
		http.Error(w, `{"error":"agent_name required"}`, http.StatusBadRequest)
		return
	}

	var req webhookTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	source := req.Source
	if source == "" {
		source = "unknown"
	}

	// Build text representation of the webhook event
	text := fmt.Sprintf("[webhook:%s] event=%s", source, req.Event)
	if len(req.Data) > 0 {
		text += "\n" + string(req.Data)
	}

	sender := "webhook:" + source

	payload := daemon.MessagePayload{
		Channel:   "webhook",
		AgentName: agentName,
		Source:    "webhook",
		Sender:    sender,
		Text:      text,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	meta := daemon.ClaimMetadata{
		ChannelType: "webhook",
		AgentName:   agentName,
	}

	if err := wth.hub.Dispatch(r.Context(), userCtx.TenantID.String(), userCtx.UserID.String(), payload, meta); err != nil {
		if err == daemon.ErrNoDaemonConnected {
			wth.logger.Warn("webhook trigger: no daemon connected",
				zap.String("agent_name", agentName),
				zap.String("source", source),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"agent is currently offline"}`))
			return
		}
		wth.logger.Error("webhook trigger dispatch failed",
			zap.String("agent_name", agentName),
			zap.Error(err),
		)
		http.Error(w, `{"error":"dispatch failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"dispatched"}`))
}
