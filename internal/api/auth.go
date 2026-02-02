package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Session storage (in-memory for now)
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time // sessionID -> expiry
}

func newSessionStore() *sessionStore {
	store := &sessionStore{
		sessions: make(map[string]time.Time),
	}
	// Cleanup expired sessions every hour
	go store.cleanup()
	return store
}

func (ss *sessionStore) create() string {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Generate random session ID
	b := make([]byte, 32)
	rand.Read(b)
	sessionID := hex.EncodeToString(b)

	// Sessions expire after 24 hours
	ss.sessions[sessionID] = time.Now().Add(24 * time.Hour)
	return sessionID
}

func (ss *sessionStore) validate(sessionID string) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	expiry, exists := ss.sessions[sessionID]
	if !exists {
		return false
	}

	return time.Now().Before(expiry)
}

func (ss *sessionStore) delete(sessionID string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, sessionID)
}

func (ss *sessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ss.mu.Lock()
		now := time.Now()
		for id, expiry := range ss.sessions {
			if now.After(expiry) {
				delete(ss.sessions, id)
			}
		}
		ss.mu.Unlock()
	}
}

func (s *Server) requireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.ReadOnly {
			writeError(w, http.StatusForbidden, "read-only mode", "Set BULWARK_UI_READONLY=false to enable writes")
			return
		}

		if s.writeLimiter != nil && !s.writeLimiter.Allow() {
			writeError(w, http.StatusTooManyRequests, "rate limited", "Too many write requests")
			return
		}

		if s.cfg.WebToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for valid session cookie first
		if cookie, err := r.Cookie("bulwark_session"); err == nil {
			if s.sessions.validate(cookie.Value) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to Bearer token (for API clients)
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" || token != s.cfg.WebToken {
			writeError(w, http.StatusUnauthorized, "invalid token", "Login required or provide a valid Bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) methodOnly(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
			return
		}
		next(w, r)
	}
}
