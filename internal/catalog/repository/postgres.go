package repository

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"mangatyapi/internal/catalog/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateComic(comic *domain.Comic) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insertar el cómic
	query := `INSERT INTO comics (author_id, title, synopsis) VALUES ($1, $2, $3) RETURNING id, published_at`
	err = tx.QueryRow(query, comic.AuthorID, comic.Title, comic.Synopsis).Scan(&comic.ID, &comic.PublishedAt)
	if err != nil {
		return err
	}

	// 2. Insertar las categorías en la tabla intermedia
	for _, catID := range comic.CategoryIDs {
		_, err = tx.Exec(`INSERT INTO comic_categories (comic_id, category_id) VALUES ($1, $2)`, comic.ID, catID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}



func (r *PostgresRepository) ListCategories() ([]domain.Category, error) {
	query := `SELECT id, name, description FROM categories ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error listando categorías: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var cat domain.Category
		var desc sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &desc); err != nil {
			return nil, fmt.Errorf("error escaneando categoría: %w", err)
		}
		cat.Description = desc.String
		categories = append(categories, cat)
	}
	return categories, nil
}

// GetComicsByCategory obtiene cómics filtrados por categoría con paginación
func (r *PostgresRepository) GetComicsByCategory(categoryIDs []int, page, limit int) ([]domain.GetComicResponse, int, error) {
	offset := (page - 1) * limit

	baseFrom := `
		FROM comics c
		JOIN users u ON c.author_id = u.id
		LEFT JOIN comic_categories cc ON c.id = cc.comic_id
		LEFT JOIN categories cat ON cc.category_id = cat.id
	`

	var whereClause string
	var countArgs []interface{}
	var queryArgs []interface{}

	if len(categoryIDs) > 0 {
		placeholders := make([]string, len(categoryIDs))
		for i, id := range categoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			countArgs = append(countArgs, id)
			queryArgs = append(queryArgs, id)
		}
		whereClause = fmt.Sprintf("WHERE cc.category_id IN (%s)", strings.Join(placeholders, ", "))
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT c.id) %s %s", baseFrom, whereClause)

	var total int
	var err error
	if len(countArgs) > 0 {
		err = r.db.QueryRow(countQuery, countArgs...).Scan(&total)
	} else {
		err = r.db.QueryRow(countQuery).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("error contando cómics: %w", err)
	}

	// Si no hay resultados, retornar vacío
	if total == 0 {
		return []domain.GetComicResponse{}, 0, nil
	}

	// Main query
	limitPlaceholder := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, limit, offset)

	query := fmt.Sprintf(`
		SELECT c.id, c.title, u.username, c.synopsis,
			   COALESCE(string_agg(DISTINCT cat.name, ', ' ORDER BY cat.name), '') as category_names,
			   (SELECT COUNT(*) FROM chapters WHERE comic_id = c.id) as chapters_count,
			   c.published_at
		%s
		%s
		GROUP BY c.id, u.username
		ORDER BY c.published_at DESC
		LIMIT %s OFFSET %s
	`, baseFrom, whereClause, limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error obteniendo cómics: %w", err)
	}
	defer rows.Close()

	var comics []domain.GetComicResponse
	for rows.Next() {
		var comic domain.GetComicResponse
		err := rows.Scan(
			&comic.ComicID, &comic.Title, &comic.AuthorName,
			&comic.Synopsis, &comic.CategoryName, &comic.ChaptersCount,
			&comic.PublishedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error escaneando cómic: %w", err)
		}

		// Obtener portada desde la BD si existe
		cover, err := r.GetComicCover(comic.ComicID)
		if err == nil && cover != nil {
			comic.CoverURL = cover.SecureURL
			comic.CoverThumbnail = cover.ThumbnailURL
		}

		comics = append(comics, comic)
	}

	return comics, total, nil
}

func (r *PostgresRepository) GetComicByID(comicID string) (*domain.GetComicResponse, error) {
	comic := &domain.GetComicResponse{}
	query := `
		SELECT c.id, c.title, u.username, c.synopsis, 
			   COALESCE(string_agg(DISTINCT cat.name, ', ' ORDER BY cat.name), '') as category_names,
			   (SELECT COUNT(*) FROM chapters WHERE comic_id = c.id) as chapters_count, 
			   c.published_at
		FROM comics c
		JOIN users u ON c.author_id = u.id
		LEFT JOIN comic_categories cc ON c.id = cc.comic_id
		LEFT JOIN categories cat ON cc.category_id = cat.id
		WHERE c.id = $1
		GROUP BY c.id, u.username
	`
	err := r.db.QueryRow(query, comicID).Scan(
		&comic.ComicID, &comic.Title, &comic.AuthorName,
		&comic.Synopsis, &comic.CategoryName, &comic.ChaptersCount,
		&comic.PublishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cómic no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error buscando cómic: %w", err)
	}

	// Obtener portada desde la BD si existe
	cover, err := r.GetComicCover(comicID)
	if err == nil && cover != nil {
		comic.CoverURL = cover.SecureURL
		comic.CoverThumbnail = cover.ThumbnailURL
	}

	// Obtener las categorías como array
	categories, err := r.GetComicCategories(comicID)
	if err == nil && len(categories) > 0 {
		// Ya tienes CategoryName como string, aquí puedes agregar más info si necesitas
	}

	return comic, nil
}

// GetComicCover obtiene la portada de un cómic desde la BD
func (r *PostgresRepository) GetComicCover(comicID string) (*domain.ComicCover, error) {
	cover := &domain.ComicCover{}
	query := `
		SELECT id, comic_id, cloudinary_public_id, image_url, secure_url,
			   COALESCE(thumbnail_url, ''), COALESCE(medium_url, ''), COALESCE(high_url, ''),
			   width, height, format, bytes, created_at
		FROM comic_covers
		WHERE comic_id = $1
	`
	err := r.db.QueryRow(query, comicID).Scan(
		&cover.ID, &cover.ComicID, &cover.CloudinaryPublicID,
		&cover.ImageURL, &cover.SecureURL,
		&cover.ThumbnailURL, &cover.MediumURL, &cover.HighURL,
		&cover.Width, &cover.Height, &cover.Format, &cover.Bytes, &cover.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No tiene portada, no es error
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo portada: %w", err)
	}
	return cover, nil
}

// SaveComicCover guarda o actualiza la portada de un cómic
func (r *PostgresRepository) SaveComicCover(cover *domain.ComicCover) error {
	query := `
		INSERT INTO comic_covers (comic_id, cloudinary_public_id, image_url, secure_url, 
			thumbnail_url, medium_url, high_url, width, height, format, bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (comic_id) 
		DO UPDATE SET 
			cloudinary_public_id = EXCLUDED.cloudinary_public_id,
			image_url = EXCLUDED.image_url,
			secure_url = EXCLUDED.secure_url,
			thumbnail_url = EXCLUDED.thumbnail_url,
			medium_url = EXCLUDED.medium_url,
			high_url = EXCLUDED.high_url,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			format = EXCLUDED.format,
			bytes = EXCLUDED.bytes,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query,
		cover.ComicID, cover.CloudinaryPublicID, cover.ImageURL, cover.SecureURL,
		cover.ThumbnailURL, cover.MediumURL, cover.HighURL,
		cover.Width, cover.Height, cover.Format, cover.Bytes,
	)
	if err != nil {
		return fmt.Errorf("error guardando portada: %w", err)
	}
	return nil
}
// Helper para extraer public_id de una URL de Cloudinary
func extractPublicIDFromURL(url string) string {
	parts := strings.Split(url, "/upload/")
	if len(parts) < 2 {
		return url
	}
	// Quitar versión vXXXXXXXXX/
	re := regexp.MustCompile(`^v\d+/`)
	return re.ReplaceAllString(parts[1], "")
}

// GetComicCategories obtiene las categorías de un cómic específico
func (r *PostgresRepository) GetComicCategories(comicID string) ([]domain.Category, error) {
	query := `
		SELECT cat.id, cat.name, COALESCE(cat.description, '')
		FROM categories cat
		JOIN comic_categories cc ON cat.id = cc.category_id
		WHERE cc.comic_id = $1
		ORDER BY cat.name ASC
	`

	rows, err := r.db.Query(query, comicID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo categorías del cómic: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var cat domain.Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description); err != nil {
			return nil, fmt.Errorf("error escaneando categoría: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, nil
}


func (r *PostgresRepository) CreateChapter(chapter *domain.Chapter) error {
    query := `
        INSERT INTO chapters (comic_id, chapter_number, title, price_tycoins)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at
    `
    return r.db.QueryRow(query, chapter.ComicID, chapter.ChapterNumber,
        chapter.Title, chapter.PriceTycoins).
        Scan(&chapter.ID, &chapter.CreatedAt)
}

func (r *PostgresRepository) GetChapterByID(chapterID string) (*domain.Chapter, error) {
    chapter := &domain.Chapter{}
    query := `
        SELECT id, comic_id, chapter_number, title, price_tycoins
        FROM chapters WHERE id = $1
    `
    err := r.db.QueryRow(query, chapterID).Scan(
        &chapter.ID, &chapter.ComicID, &chapter.ChapterNumber,
        &chapter.Title, &chapter.PriceTycoins,
    )
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("capítulo no encontrado")
    }
    return chapter, err
}

func (r *PostgresRepository) ListChapters(comicID, userID string) ([]domain.Chapter, error) {
	query := `
		SELECT ch.id, ch.comic_id, ch.chapter_number, ch.title, 
			   ch.price_tycoins,
			   CASE 
				   WHEN ch.price_tycoins = 0 THEN true
				   WHEN uc.user_id IS NOT NULL THEN true
				   ELSE false
			   END as is_unlocked
		FROM chapters ch
		LEFT JOIN unlocked_chapters uc ON ch.id = uc.chapter_id AND uc.user_id = $2
		WHERE ch.comic_id = $1
		ORDER BY ch.chapter_number ASC
	`
	rows, err := r.db.Query(query, comicID, userID)
	if err != nil {
		return nil, fmt.Errorf("error listando capítulos: %w", err)
	}
	defer rows.Close()

	var chapters []domain.Chapter
	for rows.Next() {
		var ch domain.Chapter
		err := rows.Scan(&ch.ID, &ch.ComicID, &ch.ChapterNumber, &ch.Title,
			&ch.PriceTycoins, &ch.IsUnlocked)
		if err != nil {
			return nil, fmt.Errorf("error escaneando capítulo: %w", err)
		}
		chapters = append(chapters, ch)
	}
	return chapters, nil
}


func (r *PostgresRepository) IsChapterUnlocked(userID, chapterID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM unlocked_chapters 
			WHERE user_id = $1 AND chapter_id = $2
		)
	`
	err := r.db.QueryRow(query, userID, chapterID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error verificando desbloqueo: %w", err)
	}
	return exists, nil
}


// Nuevo: Obtener información del cómic para Cloudinary
func (r *PostgresRepository) GetComicChapterInfo(chapterID string) (comicID string, chapterNumber int, err error) {
	query := `
		SELECT comic_id, chapter_number
		FROM chapters
		WHERE id = $1
	`
	err = r.db.QueryRow(query, chapterID).Scan(&comicID, &chapterNumber)
	if err == sql.ErrNoRows {
		return "", 0, fmt.Errorf("capítulo no encontrado")
	}
	if err != nil {
		return "", 0, fmt.Errorf("error obteniendo info del capítulo: %w", err)
	}
	return comicID, chapterNumber, nil
}

// GetChapterByNumber busca un capítulo por comic_id y número
func (r *PostgresRepository) GetChapterByNumber(comicID string, chapterNumber int) (*domain.Chapter, error) {
	chapter := &domain.Chapter{}
	query := `
		SELECT id, comic_id, chapter_number, title, price_tycoins
		FROM chapters
		WHERE comic_id = $1 AND chapter_number = $2
	`
	err := r.db.QueryRow(query, comicID, chapterNumber).Scan(
		&chapter.ID, &chapter.ComicID, &chapter.ChapterNumber,
		&chapter.Title, &chapter.PriceTycoins,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("capítulo no encontrado")
	}
	return chapter, err
}

// SaveChapterPages guarda las páginas subidas a Cloudinary en la BD
func (r *PostgresRepository) SaveChapterPages(chapterID string, pages []domain.ChapterPageDetail) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	for _, page := range pages {
		// Extraer public_id de la URL
		publicID := extractPublicID(page.SecureURL)
		
		_, err := tx.Exec(`
			INSERT INTO chapter_pages (chapter_id, page_number, cloudinary_public_id, image_url, secure_url, width, height, bytes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (chapter_id, page_number) 
			DO UPDATE SET 
				cloudinary_public_id = EXCLUDED.cloudinary_public_id,
				image_url = EXCLUDED.image_url,
				secure_url = EXCLUDED.secure_url,
				width = EXCLUDED.width,
				height = EXCLUDED.height,
				bytes = EXCLUDED.bytes
		`, chapterID, page.PageNumber, publicID, page.ImageURL, page.SecureURL, page.Width, page.Height, page.Bytes)
		
		if err != nil {
			return fmt.Errorf("error guardando página %d: %w", page.PageNumber, err)
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetChapterPages(chapterID string) ([]domain.ChapterPageDetail, error) {
	query := `
		SELECT id, chapter_id, page_number, cloudinary_public_id, 
			   image_url, secure_url, width, height, format, bytes
		FROM chapter_pages
		WHERE chapter_id = $1
		ORDER BY page_number ASC
	`

	rows, err := r.db.Query(query, chapterID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo páginas: %w", err)
	}
	defer rows.Close()

	var pages []domain.ChapterPageDetail
	for rows.Next() {
		var p domain.ChapterPageDetail
		err := rows.Scan(&p.ID, &p.ChapterID, &p.PageNumber, &p.CloudinaryPublicID,
			&p.ImageURL, &p.SecureURL, &p.Width, &p.Height, &p.Format, &p.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error escaneando página: %w", err)
		}
		pages = append(pages, p)
	}

	return pages, nil
}

// GetChapterPageCount obtiene el número total de páginas
func (r *PostgresRepository) GetChapterPageCount(chapterID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM chapter_pages WHERE chapter_id = $1`, chapterID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error contando páginas: %w", err)
	}
	return count, nil
}

// DeleteChapterPages elimina todas las páginas de un capítulo
func (r *PostgresRepository) DeleteChapterPages(chapterID string) error {
	_, err := r.db.Exec(`DELETE FROM chapter_pages WHERE chapter_id = $1`, chapterID)
	return err
}

// extractPublicID extrae el public_id de una URL de Cloudinary
func extractPublicID(url string) string {
	// Ejemplo: https://res.cloudinary.com/dpkbywksr/image/upload/v123/mangaty/comics/.../page_001.jpg
	// Retorna: mangaty/comics/.../page_001
	parts := strings.Split(url, "/upload/")
	if len(parts) < 2 {
		return url
	}
	
	// Quitar versión (v123) y extensión
	uploadPart := parts[1]
	// Remover el prefijo de versión vXXXXXXXXX/
	re := regexp.MustCompile(`^v\d+/`)
	uploadPart = re.ReplaceAllString(uploadPart, "")
	// Remover extensión
	uploadPart = strings.TrimSuffix(uploadPart, filepath.Ext(uploadPart))
	
	return uploadPart
}