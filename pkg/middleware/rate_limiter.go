package middleware

import (
	"net/http"
	"time"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func RateLimiter() func(http.Handler) http.Handler {
	// 100 peticiones por minuto por IP
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			limiterCtx, err := instance.Get(r.Context(), ip)
			if err != nil {
				http.Error(w, `{"success":false,"message":"Error interno"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", string(rune(limiterCtx.Remaining)))
			w.Header().Set("X-RateLimit-Reset", string(rune(limiterCtx.Reset)))

			if limiterCtx.Reached {
				http.Error(w, `{"success":false,"message":"Demasiadas peticiones. Intente más tarde."}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// StrictRateLimiter para endpoints críticos (login, register)
func StrictRateLimiter() func(http.Handler) http.Handler {
	// 5 peticiones por minuto
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  5,
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			limiterCtx, err := instance.Get(r.Context(), ip)
			if err != nil {
				http.Error(w, `{"success":false,"message":"Error interno"}`, http.StatusInternalServerError)
				return
			}

			if limiterCtx.Reached {
				http.Error(w, `{"success":false,"message":"Demasiados intentos. Espere un minuto."}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}