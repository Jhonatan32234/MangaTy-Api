package middleware

import "net/http"

func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevenir MIME sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			
			// Prevenir clickjacking
			w.Header().Set("X-Frame-Options", "DENY")
			
			// Habilitar XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			
			// Content Security Policy
			w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https://res.cloudinary.com; style-src 'self' 'unsafe-inline' https://unpkg.com; script-src 'self' 'unsafe-inline' https://unpkg.com")
			
			// Referrer Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			
			// HSTS (solo en producción con HTTPS)
			// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			
			// Cache control para datos sensibles
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			w.Header().Set("Pragma", "no-cache")
			
			// Remover headers que revelan tecnología
			w.Header().Del("X-Powered-By")
			w.Header().Del("Server")
			
			next.ServeHTTP(w, r)
		})
	}
}