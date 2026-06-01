package delivery

import (
	"encoding/json"
	"net/http"

	"mangatyapi/internal/wallet/domain"
	"mangatyapi/internal/wallet/usecase"
	"mangatyapi/pkg/middleware"
	"mangatyapi/pkg/response"
)

type WalletHandler struct {
	walletUC *usecase.UseCase
}

func NewWalletHandler(walletUC *usecase.UseCase) *WalletHandler {
	return &WalletHandler{walletUC: walletUC}
}

func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "Usuario no autenticado")
		return
	}

	resp, err := h.walletUC.GetBalance(userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.OK(w, "Saldo consultado exitosamente", resp)
}


func (h *WalletHandler) UnlockChapter(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "Usuario no autenticado")
		return
	}

	var req domain.UnlockChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Datos inválidos")
		return
	}

	if req.ChapterID == "" {
		response.BadRequest(w, "ID de capítulo requerido")
		return
	}

	resp, err := h.walletUC.UnlockChapter(userID, req.ChapterID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.OK(w, resp.Message, resp)
}
