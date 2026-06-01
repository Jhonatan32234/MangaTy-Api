package domain

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	PasswordHash string `json:"-"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Token     string `json:"token,omitempty"`
	ExpiresIn string `json:"expires_in,omitempty"`
}

// Nuevo: Solicitud para cambiar roles
type RoleChangeRequest struct {
	Action string `json:"action" validate:"required,oneof=add remove"` // add o remove
	Role   string `json:"role" validate:"required,oneof=autor administrador"`
}

// Nuevo: Perfil con roles
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}