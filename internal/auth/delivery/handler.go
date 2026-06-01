package delivery

import (
	"encoding/json"
	"net/http"

	"mangatyapi/internal/auth/domain"
	"mangatyapi/internal/auth/usecase"
	"mangatyapi/pkg/middleware"
	"mangatyapi/pkg/response"
)

type AuthHandler struct {
	authUC *usecase.UseCase
}

func NewAuthHandler(authUC *usecase.UseCase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Datos inválidos")
		return
	}

	resp, err := h.authUC.Register(req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	response.Created(w, resp.Message, map[string]interface{}{
		"user_id": resp.UserID,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Datos inválidos")
		return
	}

	resp, err := h.authUC.Login(req)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	response.OK(w, "Login exitoso", map[string]interface{}{
		"token":      resp.Token,
		"expires_in": resp.ExpiresIn,
	})
}

// Nuevo: Obtener perfil
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	
	profile, err := h.authUC.GetProfile(userID)
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.OK(w, "Perfil obtenido", profile)
}

// Nuevo: Admin asigna rol
func (h *AuthHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetUserID(r.Context())
	
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Datos inválidos")
		return
	}

	resp, err := h.authUC.AssignRole(adminID, req.UserID, req.Role)
	if err != nil {
		response.Forbidden(w, err.Error())
		return
	}

	response.OK(w, resp.Message, nil)
}