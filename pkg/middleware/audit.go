package middleware

import (
	"net/http"
	"time"

	"mangatyapi/pkg/security"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func AuditMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// No auditar endpoints de documentación
			if r.URL.Path == "/docs" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			
			next.ServeHTTP(rw, r)
			
			// Crear log de auditoría
			auditLog := security.AuditLog{
				UserID:   GetUserID(r.Context()),
				IP:       r.RemoteAddr,
				Action:   r.Method,
				Endpoint: r.URL.Path,
				Method:   r.Method,
				Status:   rw.status,
				Details:  time.Since(start).String(),
			}
			
			security.LogAudit(auditLog)
		})
	}
}