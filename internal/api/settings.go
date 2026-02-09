package api

import (
	"context"
	"net/http"

	"github.com/itsmrshow/bulwark/internal/notify"
)

type settingsResponse struct {
	Notifications notify.Settings `json:"notifications"`
	Locked        notify.Settings `json:"locked,omitempty"`
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.notify == nil {
			writeJSON(w, http.StatusOK, settingsResponse{Notifications: notify.Defaults()})
			return
		}
		settings := s.notify.Settings()
		locked := s.notify.EnvLocked()
		if locked.DiscordWebhook != "" {
			settings.DiscordWebhook = "ENV:configured"
		}
		if locked.SlackWebhook != "" {
			settings.SlackWebhook = "ENV:configured"
		}
		writeJSON(w, http.StatusOK, settingsResponse{Notifications: settings, Locked: locked})
	case http.MethodPut:
		s.requireWrite(http.HandlerFunc(s.handleSettingsUpdate)).ServeHTTP(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
	}
}

func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var payload settingsResponse
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if s.notify == nil {
		writeError(w, http.StatusServiceUnavailable, "settings unavailable", "notifications manager not initialized")
		return
	}

	if err := s.notify.Update(context.Background(), payload.Notifications); err != nil {
		writeError(w, http.StatusBadRequest, "invalid settings", err.Error())
		return
	}

	s.notify.Reload(context.Background())

	settings := s.notify.Settings()
	locked := s.notify.EnvLocked()
	if locked.DiscordWebhook != "" {
		settings.DiscordWebhook = "ENV:configured"
	}
	if locked.SlackWebhook != "" {
		settings.SlackWebhook = "ENV:configured"
	}
	writeJSON(w, http.StatusOK, settingsResponse{Notifications: settings, Locked: locked})
}

func (s *Server) handleNotificationsTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	if s.notify == nil {
		writeError(w, http.StatusServiceUnavailable, "notifications unavailable", "notifications manager not initialized")
		return
	}

	if err := s.notify.Test(r.Context()); err != nil {
		writeError(w, http.StatusBadRequest, "notification failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
