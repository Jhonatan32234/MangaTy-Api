package domain

import "time"

type Comic struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	Synopsis    string    `json:"synopsis"`
	CategoryIDs []int     `json:"category_ids"`
	CoverURL    string    `json:"cover_url,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Chapter struct {
	ID            string    `json:"id"`
	ComicID       string    `json:"comic_id"`
	ChapterNumber int       `json:"chapter_number"`
	Title         string    `json:"title"`
	PriceTycoins  int       `json:"price_tycoins"`
	IsUnlocked    bool      `json:"is_unlocked"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateComicRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=150"`
	Synopsis    string `json:"synopsis" validate:"required"`
	CategoryIDs []int  `json:"category_ids" validate:"required"`
}

type CreateComicResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	ComicID string `json:"comic_id"`
	Cover   *ComicImage `json:"cover,omitempty"` // Portada subida (opcional)
}

type ComicImage struct {
	Thumbnail string `json:"thumbnail"`
	Medium    string `json:"medium"`
	High      string `json:"high"`
	Original  string `json:"original"`
}

// ComicCover representa la portada guardada en BD
type ComicCover struct {
	ID                 string    `json:"id"`
	ComicID            string    `json:"comic_id"`
	CloudinaryPublicID string    `json:"cloudinary_public_id"`
	ImageURL           string    `json:"image_url"`
	SecureURL          string    `json:"secure_url"`
	ThumbnailURL       string    `json:"thumbnail_url,omitempty"`
	MediumURL          string    `json:"medium_url,omitempty"`
	HighURL            string    `json:"high_url,omitempty"`
	Width              int       `json:"width"`
	Height             int       `json:"height"`
	Format             string    `json:"format"`
	Bytes              int64     `json:"bytes"`
	CreatedAt          time.Time `json:"created_at"`
}


// GetComicResponse ahora incluye la portada
type GetComicResponse struct {
	ComicID       string    `json:"comic_id"`
	Title         string    `json:"title"`
	AuthorName    string    `json:"author_name"`
	Synopsis      string    `json:"synopsis"`
	CategoryName  string    `json:"category_name"`
	CoverURL      string    `json:"cover_url,omitempty"`
	CoverThumbnail string   `json:"cover_thumbnail,omitempty"`
	ChaptersCount int       `json:"chapters_count"`
	PublishedAt   time.Time `json:"published_at"`
}

type CreateChapterRequest struct {
	ChapterNumber int    `json:"chapter_number"`
	Title         string `json:"title"`
	PriceTycoins  int    `json:"price_tycoins"`
}

type CreateChapterResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	ChapterID string `json:"chapter_id"`
	Pages     []ChapterPageDetail `json:"pages,omitempty"`
}

type ListChaptersResponse struct {
	ComicID       string    `json:"comic_id"`
	TotalChapters int       `json:"total_chapters"`
	Chapters      []Chapter `json:"chapters"`
}

type Recommendation struct {
	ComicID         string  `json:"comic_id"`
	Title           string  `json:"title"`
	Author          string  `json:"author"`
	MatchPercentage float64 `json:"match_percentage"`
}

type RecommendationsResponse struct {
	ItemsCount      int              `json:"items_count"`
	Recommendations []Recommendation `json:"recommendations"`
}


type ReadChapterResponse struct {
	Status     string              `json:"status"`
	ChapterID  string              `json:"chapter_id"`
	Title      string              `json:"title"`
	PagesCount int                 `json:"pages_count"`
	Pages      []ChapterPageDetail `json:"pages"`  // ✅ Cambiado de []string
}

type ChapterPageDetail struct {
	ID                 string `json:"id"`
	ChapterID          string `json:"chapter_id"`
	PageNumber         int    `json:"page_number"`
	CloudinaryPublicID string `json:"cloudinary_public_id"`
	ImageURL           string `json:"image_url"`
	SecureURL          string `json:"secure_url"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	Format             string `json:"format"`
	Bytes              int64  `json:"bytes"`
}