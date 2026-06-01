package middleware

import (
	"context"
	"net/http"
	"strings"

	"mangatyapi/pkg/security"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
	EmailKey  contextKey = "email"
	RolesKey  contextKey = "roles"
)

type AuthMiddleware struct{}

func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"success":false,"message":"Token no proporcionado"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"success":false,"message":"Formato de token inválido"}`, http.StatusUnauthorized)
			return
		}

		claims, err := security.ValidateJWT(parts[1])
		if err != nil {
			http.Error(w, `{"success":false,"message":"Token inválido o expirado"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)
		ctx = context.WithValue(ctx, RolesKey, claims.Roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetUserRoles(ctx context.Context) []string {
	if roles, ok := ctx.Value(RolesKey).([]string); ok {
		return roles
	}
	return []string{}
}

func HasRole(ctx context.Context, role string) bool {
	roles := GetUserRoles(ctx)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// Nuevo: Middleware para requerir roles específicos
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := GetUserRoles(r.Context())
			
			for _, required := range roles {
				for _, userRole := range userRoles {
					if userRole == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			
			http.Error(w, `{"success":false,"message":"No tienes los permisos necesarios"}`, http.StatusForbidden)
		})
	}
}