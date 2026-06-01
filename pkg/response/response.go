package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// StandardResponse es el formato uniforme para todas las respuestas de la API
// Diseñado para ser fácilmente parseable desde Flutter/Dart
type StandardResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// Meta contiene información de paginación para scroll infinito en Flutter
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
}

// ErrorDetail para errores de validación campo por campo
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Success envía una respuesta exitosa genérica
func Success(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := StandardResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, `{"success":false,"message":"Error interno del servidor"}`, http.StatusInternalServerError)
	}
}

// SuccessWithMeta envía respuesta exitosa con metadatos de paginación
func SuccessWithMeta(w http.ResponseWriter, statusCode int, message string, data interface{}, meta *Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := StandardResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, `{"success":false,"message":"Error interno del servidor"}`, http.StatusInternalServerError)
	}
}

// Error envía una respuesta de error simple
func Error(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := StandardResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, `{"success":false,"message":"Error interno del servidor"}`, http.StatusInternalServerError)
	}
}

// ValidationError envía errores de validación detallados
func ValidationError(w http.ResponseWriter, message string, errors []ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	
	resp := StandardResponse{
		Success:   false,
		Message:   message,
		Errors:    errors,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, `{"success":false,"message":"Error interno del servidor"}`, http.StatusInternalServerError)
	}
}

// Created envía respuesta 201 para recursos creados exitosamente
func Created(w http.ResponseWriter, message string, data interface{}) {
	Success(w, http.StatusCreated, message, data)
}

// OK envía respuesta 200 exitosa
func OK(w http.ResponseWriter, message string, data interface{}) {
	Success(w, http.StatusOK, message, data)
}

// NoContent envía respuesta 204 sin contenido
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// PaginatedOK envía respuesta paginada exitosa
func PaginatedOK(w http.ResponseWriter, message string, data interface{}, meta *Meta) {
	SuccessWithMeta(w, http.StatusOK, message, data, meta)
}

// Unauthorized envía error 401
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "No autorizado"
	}
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden envía error 403
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Acceso prohibido"
	}
	Error(w, http.StatusForbidden, message)
}

// NotFound envía error 404
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Recurso no encontrado"
	}
	Error(w, http.StatusNotFound, message)
}

// InternalServerError envía error 500
func InternalServerError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Error interno del servidor"
	}
	Error(w, http.StatusInternalServerError, message)
}

// BadRequest envía error 400
func BadRequest(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Solicitud inválida"
	}
	Error(w, http.StatusBadRequest, message)
}

// TooManyRequests envía error 429
func TooManyRequests(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Demasiadas solicitudes. Intente de nuevo más tarde"
	}
	Error(w, http.StatusTooManyRequests, message)
}