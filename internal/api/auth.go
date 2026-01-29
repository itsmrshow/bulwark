package api

import "net/http"

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
			writeError(w, http.StatusUnauthorized, "missing token", "BULWARK_WEB_TOKEN is not configured")
			return
		}

		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" || token != s.cfg.WebToken {
			writeError(w, http.StatusUnauthorized, "invalid token", "Provide a valid Bearer token")
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
