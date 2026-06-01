package delivery

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"mangatyapi/internal/catalog/domain"
	"mangatyapi/internal/catalog/usecase"
	"mangatyapi/pkg/cloudinary"
	"mangatyapi/pkg/middleware"
	"mangatyapi/pkg/response"
)

type CatalogHandler struct {
	catalogUC  *usecase.UseCase
	cloudinary *cloudinary.Service
}

func NewCatalogHandler(catalogUC *usecase.UseCase, cldService *cloudinary.Service) *CatalogHandler {
	return &CatalogHandler{
		catalogUC:  catalogUC,
		cloudinary: cldService,
	}
}

// ListCategories - Público, obtener todas las categorías
func (h *CatalogHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catalogUC.ListCategories()
	if err != nil {
		response.InternalServerError(w, "Error obteniendo categorías")
		return
	}

	response.OK(w, "Categorías obtenidas exitosamente", categories)
}

// GetComicsByCategory - Filtrar cómics por múltiples categorías
func (h *CatalogHandler) GetComicsByCategory(w http.ResponseWriter, r *http.Request) {
	categoryIDsStr := r.URL.Query()["category_id"] // Obtener todos los valores
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Parsear múltiples category_id
	var categoryIDs []int
	for _, idStr := range categoryIDsStr {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			response.BadRequest(w, "category_id inválido: "+idStr)
			return
		}
		categoryIDs = append(categoryIDs, id)
	}

	// Si también se pasa category_ids como string separado por comas
	categoryIDsComma := r.URL.Query().Get("category_ids")
	if categoryIDsComma != "" {
		for _, idStr := range strings.Split(categoryIDsComma, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err != nil {
				response.BadRequest(w, "category_ids inválido: "+idStr)
				return
			}
			categoryIDs = append(categoryIDs, id)
		}
	}

	page := 1
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	limit := 20
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 20
		}
	}

	comics, total, err := h.catalogUC.GetComicsByCategory(categoryIDs, page, limit)
	if err != nil {
		response.InternalServerError(w, "Error obteniendo cómics: "+err.Error())
		return
	}

	totalPages := (total + limit - 1) / limit
	hasMore := page < totalPages

	meta := &response.Meta{
		Page:       page,
		PerPage:    limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasMore:    hasMore,
	}

	response.SuccessWithMeta(w, http.StatusOK, "Cómics obtenidos exitosamente", comics, meta)
}




// CreateComic - Solo autores y administradores
// CreateComic - Crea cómic con opción de subir portada en la misma petición
func (h *CatalogHandler) CreateComic(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if !middleware.HasRole(r.Context(), "autor") && !middleware.HasRole(r.Context(), "administrador") {
		response.Error(w, http.StatusForbidden, "Solo los autores pueden crear cómics. Usa /user/become-author para convertirte en autor.")
		return
	}

	// Detectar si es multipart (con imagen) o JSON
	contentType := r.Header.Get("Content-Type")

	var req domain.CreateComicRequest
	var coverFile multipart.File
	var coverHeader *multipart.FileHeader

	if strings.Contains(contentType, "multipart/form-data") {
		// ===== MODO MULTIPART: JSON + Imagen =====
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			response.BadRequest(w, "Error procesando formulario: "+err.Error())
			return
		}

		// Obtener datos JSON del campo "data"
		dataStr := r.FormValue("data")
		if dataStr == "" {
			response.BadRequest(w, "Campo 'data' requerido con los datos del cómic en JSON")
			return
		}

		if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
			response.BadRequest(w, "JSON inválido en campo 'data': "+err.Error())
			return
		}

		// Obtener archivo de portada (opcional)
		coverFile, coverHeader, _ = r.FormFile("cover")
		if coverFile != nil {
			defer coverFile.Close()
		}

	} else {
		// ===== MODO JSON: Sin imagen =====
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.BadRequest(w, "Datos inválidos: "+err.Error())
			return
		}
	}

	// Validar categorías
	if len(req.CategoryIDs) == 0 {
		response.BadRequest(w, "Debe seleccionar al menos una categoría")
		return
	}

	// Crear el cómic en la BD
	resp, err := h.catalogUC.CreateComic(userID, req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	result := map[string]interface{}{
		"comic_id": resp.ComicID,
	}

	// Subir portada si se proporcionó y Cloudinary está disponible
	if coverFile != nil && h.cloudinary != nil {
		coverImage, err := h.cloudinary.UploadComicCover(coverFile, coverHeader, resp.ComicID)
		if err != nil {
			// La portada falló pero el cómic ya se creó
			result["cover_warning"] = "Cómic creado pero la portada falló: " + err.Error()
		} else {
			result["cover"] = coverImage

			// ✅ Guardar referencia de la portada en la BD
			cover := &domain.ComicCover{
				ComicID:            resp.ComicID,
				CloudinaryPublicID: fmt.Sprintf("comics/%s/cover", resp.ComicID),
				ImageURL:           coverImage.Original,
				SecureURL:          coverImage.Original,
				ThumbnailURL:       coverImage.Thumbnail,
				MediumURL:          coverImage.Medium,
				HighURL:            coverImage.High,
			}

			// Usar el usecase para guardar (maneja errores sin romper)
			if saveErr := h.catalogUC.SaveComicCover(cover); saveErr != nil {
				log.Printf("⚠️ Error guardando portada en BD para cómic %s: %v", resp.ComicID, saveErr)
			}
		}
	}

	response.Created(w, "Cómic registrado exitosamente en el catálogo", result)
}

// GetComic - Todos los usuarios autenticados
func (h *CatalogHandler) GetComic(w http.ResponseWriter, r *http.Request) {
	comicID := extractPathParam(r.URL.Path, "/api/v1/comics/")
	if comicID == "" {
		response.BadRequest(w, "ID de cómic no proporcionado")
		return
	}

	resp, err := h.catalogUC.GetComic(comicID)
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.OK(w, "Cómic encontrado", resp)
}

// CreateChapter - Solo autores y administradores
// CreateChapter - Crea capítulo con opción de subir páginas en la misma petición
func (h *CatalogHandler) CreateChapter(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if !middleware.HasRole(r.Context(), "autor") && !middleware.HasRole(r.Context(), "administrador") {
		response.Error(w, http.StatusForbidden, "Solo los autores pueden añadir capítulos")
		return
	}

	comicID := extractPathParam(r.URL.Path, "/api/v1/comics/")
	comicID = strings.Split(comicID, "/chapters")[0]

	// Detectar el tipo de contenido
	contentType := r.Header.Get("Content-Type")

	var req domain.CreateChapterRequest
	var pageFiles []*multipart.FileHeader

	if strings.Contains(contentType, "multipart/form-data") {
		// ===== MODO MULTIPART: JSON + Páginas =====
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			response.BadRequest(w, "Error procesando formulario: "+err.Error())
			return
		}

		// Obtener datos JSON del campo "data"
		dataStr := r.FormValue("data")
		if dataStr == "" {
			response.BadRequest(w, "Campo 'data' requerido con los datos del capítulo en JSON")
			return
		}

		if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
			response.BadRequest(w, "JSON inválido en campo 'data': "+err.Error())
			return
		}

		// Obtener archivos de páginas (opcional)
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			pageFiles = r.MultipartForm.File["pages"]
		}

	} else {
		// ===== MODO JSON: Sin páginas =====
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.BadRequest(w, "Datos inválidos: "+err.Error())
			return
		}
	}

	// Validaciones
	if req.ChapterNumber <= 0 {
		response.BadRequest(w, "El número de capítulo debe ser mayor a cero")
		return
	}
	if req.PriceTycoins < 0 {
		response.BadRequest(w, "El precio no puede ser negativo")
		return
	}

	// Crear el capítulo en la BD
	resp, err := h.catalogUC.CreateChapter(userID, comicID, req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	result := map[string]interface{}{
		"chapter_id":     resp.ChapterID,
		"chapter_number": req.ChapterNumber,
		"message":        resp.Message,
	}

	// Subir páginas si se proporcionaron y Cloudinary está disponible
	if len(pageFiles) > 0 && h.cloudinary != nil {
		pages, err := h.cloudinary.UploadChapterPages(pageFiles, comicID, req.ChapterNumber)
		if err != nil {
			result["pages_warning"] = "Capítulo creado pero las páginas fallaron: " + err.Error()
		} else {
			// Guardar páginas en la BD
			var domainPages []domain.ChapterPageDetail
			for _, p := range pages {
				domainPages = append(domainPages, domain.ChapterPageDetail{
					ChapterID:  resp.ChapterID,
					PageNumber: p.PageNumber,
					ImageURL:   p.ImageURL,
					SecureURL:  p.SecureURL,
					Width:      p.Width,
					Height:     p.Height,
				})
			}

			if saveErr := h.catalogUC.SaveChapterPages(resp.ChapterID, pages); saveErr != nil {
				log.Printf("⚠️ Error guardando páginas en BD: %v", saveErr)
			}

			result["pages"] = domainPages
			result["total_pages"] = len(pages)
		}
	}

	response.Created(w, resp.Message, result)
}

// ListChapters - Todos los usuarios autenticados
func (h *CatalogHandler) ListChapters(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	comicID := extractPathParam(r.URL.Path, "/api/v1/comics/")
	comicID = strings.Split(comicID, "/chapters")[0]

	resp, err := h.catalogUC.ListChapters(comicID, userID)
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.OK(w, "Capítulos listados exitosamente", resp)
}



func (h *CatalogHandler) ReadChapter(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	chapterID := extractPathParam(r.URL.Path, "/api/v1/chapters/")
	chapterID = strings.Split(chapterID, "/read")[0]

	resp, err := h.catalogUC.ReadChapter(userID, chapterID)
	if err != nil {
		if strings.Contains(err.Error(), "INSUFFICIENT_PERMISSIONS") {
			response.Forbidden(w, err.Error())
			return
		}
		response.NotFound(w, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Pages-Count", fmt.Sprintf("%d", resp.PagesCount))

	response.OK(w, "Acceso concedido", resp)
}

// GetRecommendations - Todos los usuarios autenticados
func (h *CatalogHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	resp, err := h.catalogUC.GetRecommendations(userID)
	if err != nil {
		response.InternalServerError(w, err.Error())
		return
	}

	response.OK(w, "Recomendaciones generadas", resp)
}

// UploadComicCover - Solo autores y administradores
func (h *CatalogHandler) UploadComicCover(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if userID == "" {
		response.Unauthorized(w, "Usuario no autenticado")
		return
	}

	if !middleware.HasRole(r.Context(), "autor") && !middleware.HasRole(r.Context(), "administrador") {
		response.Forbidden(w, "Solo los autores pueden subir portadas")
		return
	}

	if h.cloudinary == nil {
		response.Error(w, http.StatusServiceUnavailable, "Servicio de imágenes no disponible")
		return
	}

	comicID := extractPathParam(r.URL.Path, "/api/v1/comics/")
	comicID = strings.Split(comicID, "/cover")[0]

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.BadRequest(w, "Error procesando formulario")
		return
	}

	file, header, err := r.FormFile("cover")
	if err != nil {
		response.BadRequest(w, "Archivo de portada no proporcionado")
		return
	}
	defer file.Close()

	// Subir a Cloudinary
	coverImage, err := h.cloudinary.UploadComicCover(file, header, comicID)
	if err != nil {
		response.InternalServerError(w, "Error subiendo portada: "+err.Error())
		return
	}

	cover := &domain.ComicCover{
		ComicID:            comicID,
		CloudinaryPublicID: fmt.Sprintf("comics/%s/cover", comicID),
		ImageURL:           coverImage.Original,
		SecureURL:          coverImage.Original,
		ThumbnailURL:       coverImage.Thumbnail,
		MediumURL:          coverImage.Medium,
		HighURL:            coverImage.High,
	}

	if err := h.catalogUC.SaveComicCover(cover); err != nil {
		log.Printf("⚠️ Error guardando portada en BD: %v", err)
		// No fallamos la petición, la imagen ya está en Cloudinary
	}

	response.Created(w, "Portada subida exitosamente", coverImage)
}


// DeleteComic - Solo administradores
func (h *CatalogHandler) DeleteComic(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if userID == "" {
		response.Unauthorized(w, "Usuario no autenticado")
		return
	}

	if !middleware.HasRole(r.Context(), "administrador") {
		response.Forbidden(w, "Solo los administradores pueden eliminar cómics")
		return
	}

	comicID := extractPathParam(r.URL.Path, "/api/v1/comics/")

	if h.cloudinary != nil {
		if err := h.cloudinary.DeleteComic(comicID); err != nil {
			response.InternalServerError(w, "Error eliminando imágenes: "+err.Error())
			return
		}
	}

	response.OK(w, "Cómic eliminado exitosamente", nil)
}

func extractPathParam(path, prefix string) string {
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (h *CatalogHandler) UploadChapterPages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "Usuario no autenticado")
		return
	}

	if !middleware.HasRole(r.Context(), "autor") && !middleware.HasRole(r.Context(), "administrador") {
		response.Forbidden(w, "Solo los autores pueden subir páginas")
		return
	}

	if h.cloudinary == nil {
		response.Error(w, http.StatusServiceUnavailable, "Servicio de imágenes no disponible")
		return
	}

	// Extraer comicID y chapterNumber
	path := r.URL.Path
	trimmed := strings.TrimPrefix(path, "/api/v1/comics/")
	parts := strings.Split(trimmed, "/chapters/")
	if len(parts) != 2 {
		response.BadRequest(w, "URL inválida")
		return
	}
	comicID := parts[0]
	chapterParts := strings.Split(parts[1], "/pages")
	chapterNumber, err := strconv.Atoi(chapterParts[0])
	if err != nil {
		response.BadRequest(w, "Número de capítulo inválido")
		return
	}

	// Parsear formulario
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		response.BadRequest(w, "Error procesando formulario")
		return
	}

	files := r.MultipartForm.File["pages"]
	if len(files) == 0 {
		response.BadRequest(w, "No se proporcionaron páginas")
		return
	}

	// Subir a Cloudinary
	pages, err := h.cloudinary.UploadChapterPages(files, comicID, chapterNumber)
	if err != nil {
		response.InternalServerError(w, "Error subiendo páginas: "+err.Error())
		return
	}

	// Obtener el chapterID real desde la BD
	chapter, err := h.catalogUC.GetChapterByNumber(comicID, chapterNumber)
	if err != nil {
		response.InternalServerError(w, "Error buscando capítulo: "+err.Error())
		return
	}

	// ✅ Guardar páginas en la BD
	if err := h.catalogUC.SaveChapterPages(chapter.ID, pages); err != nil {
		response.InternalServerError(w, "Error guardando páginas: "+err.Error())
		return
	}

	response.Created(w, fmt.Sprintf("%d páginas subidas y guardadas exitosamente", len(pages)), map[string]interface{}{
		"comic_id":       comicID,
		"chapter_id":     chapter.ID,
		"chapter_number": chapterNumber,
		"pages":          pages,
		"total_pages":    len(pages),
	})
}

