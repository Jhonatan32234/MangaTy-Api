package usecase

import (
	"fmt"

	"mangatyapi/internal/catalog/domain"
	"mangatyapi/pkg/cloudinary"
)

type Repository interface {
	CreateComic(comic *domain.Comic) error
	GetComicByID(comicID string) (*domain.GetComicResponse, error)
	GetComicsByCategory(categoryIDs []int, page, limit int) ([]domain.GetComicResponse, int, error)
	GetComicCategories(comicID string) ([]domain.Category, error)
	ListCategories() ([]domain.Category, error)
	CreateChapter(chapter *domain.Chapter) error
	ListChapters(comicID, userID string) ([]domain.Chapter, error)
	GetChapterByID(chapterID string) (*domain.Chapter, error)
	IsChapterUnlocked(userID, chapterID string) (bool, error)
	GetChapterPages(chapterID string) ([]domain.ChapterPageDetail, error) // ✅ Cambiado
	GetComicChapterInfo(chapterID string) (string, int, error)
	GetChapterPageCount(chapterID string) (int, error)
	SaveChapterPages(chapterID string, pages []domain.ChapterPageDetail) error
	GetChapterByNumber(comicID string, chapterNumber int) (*domain.Chapter, error)
	SaveComicCover(cover *domain.ComicCover) error
}

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) CreateComic(authorID string, req domain.CreateComicRequest) (*domain.CreateComicResponse, error) {
	comic := &domain.Comic{
		AuthorID:    authorID,
		Title:       req.Title,
		Synopsis:    req.Synopsis,
		CategoryIDs: req.CategoryIDs,
	}

	if err := uc.repo.CreateComic(comic); err != nil {
		return nil, err
	}

	return &domain.CreateComicResponse{
		Status:  "success",
		Message: "Cómic registrado exitosamente en el catálogo",
		ComicID: comic.ID,
	}, nil
}

func (uc *UseCase) SaveComicCover(cover *domain.ComicCover) error {
	return uc.repo.SaveComicCover(cover)
}

func (uc *UseCase) ListCategories() ([]domain.Category, error) {
	return uc.repo.ListCategories()
}

func (uc *UseCase) GetComic(comicID string) (*domain.GetComicResponse, error) {
	return uc.repo.GetComicByID(comicID)
}

// GetComicsByCategory ahora recibe []int
func (uc *UseCase) GetComicsByCategory(categoryIDs []int, page, limit int) ([]domain.GetComicResponse, int, error) {
	return uc.repo.GetComicsByCategory(categoryIDs, page, limit)
}

func (uc *UseCase) GetComicCategories(comicID string) ([]domain.Category, error) {
	return uc.repo.GetComicCategories(comicID)
}

func (uc *UseCase) CreateChapter(authorID, comicID string, req domain.CreateChapterRequest) (*domain.CreateChapterResponse, error) {
	if req.ChapterNumber <= 0 {
		return nil, fmt.Errorf("el número de capítulo debe ser mayor a cero")
	}
	if req.PriceTycoins < 0 {
		return nil, fmt.Errorf("el precio no puede ser negativo")
	}

	chapter := &domain.Chapter{
		ComicID:       comicID,
		ChapterNumber: req.ChapterNumber,
		Title:         req.Title,
		PriceTycoins:  req.PriceTycoins,
	}

	if err := uc.repo.CreateChapter(chapter); err != nil {
		return nil, err
	}

	return &domain.CreateChapterResponse{
		Status:    "success",
		Message:   fmt.Sprintf("Capítulo número %d añadido correctamente", req.ChapterNumber),
		ChapterID: chapter.ID,
	}, nil
}

func (uc *UseCase) ListChapters(comicID, userID string) (*domain.ListChaptersResponse, error) {
	chapters, err := uc.repo.ListChapters(comicID, userID)
	if err != nil {
		return nil, err
	}

	return &domain.ListChaptersResponse{
		ComicID:       comicID,
		TotalChapters: len(chapters),
		Chapters:      chapters,
	}, nil
}

func (uc *UseCase) ReadChapter(userID, chapterID string) (*domain.ReadChapterResponse, error) {
	chapter, err := uc.repo.GetChapterByID(chapterID)
	if err != nil {
		return nil, err
	}

	// Verificar desbloqueo para capítulos de pago
	if chapter.PriceTycoins > 0 {
		unlocked, err := uc.repo.IsChapterUnlocked(userID, chapterID)
		if err != nil {
			return nil, err
		}
		if !unlocked {
			return nil, fmt.Errorf("INSUFFICIENT_PERMISSIONS: Este capítulo requiere una transacción de TyCoins")
		}
	}

	// Obtener páginas reales desde la BD
	pages, err := uc.repo.GetChapterPages(chapterID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo páginas: %w", err)
	}

	return &domain.ReadChapterResponse{
		Status:     "granted",
		ChapterID:  chapterID,
		Title:      chapter.Title,
		PagesCount: len(pages),
		Pages:      pages,  // ✅ pages es []ChapterPageDetail
	}, nil
}

func (uc *UseCase) SaveChapterPages(chapterID string, cloudinaryPages []cloudinary.ChapterPage) error {
	// Convertir de cloudinary.ChapterPage a domain.ChapterPageDetail
	var domainPages []domain.ChapterPageDetail
	for _, p := range cloudinaryPages {
		domainPages = append(domainPages, domain.ChapterPageDetail{
			ChapterID:  chapterID,
			PageNumber: p.PageNumber,
			ImageURL:   p.ImageURL,
			SecureURL:  p.SecureURL,
			Width:      p.Width,
			Height:     p.Height,
		})
	}
	
	// Llamar al repositorio con el tipo correcto
	return uc.repo.SaveChapterPages(chapterID, domainPages)
}

func (uc *UseCase) GetChapterByNumber(comicID string, chapterNumber int) (*domain.Chapter, error) {
	return uc.repo.GetChapterByNumber(comicID, chapterNumber)
}

func (uc *UseCase) GetRecommendations(userID string) (*domain.RecommendationsResponse, error) {
	recommendations := []domain.Recommendation{
		{
			ComicID:         "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			Title:           "El Último Alquimista",
			Author:          "ArteGlobal",
			MatchPercentage: 94.5,
		},
	}

	return &domain.RecommendationsResponse{
		ItemsCount:      len(recommendations),
		Recommendations: recommendations,
	}, nil
}

func (uc *UseCase) GetChapterInfo(chapterID string) (comicID string, chapterNumber int, err error) {
	return uc.repo.GetComicChapterInfo(chapterID)
}