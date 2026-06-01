package cloudinary

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Service struct {
	cld *cloudinary.Cloudinary
	ctx context.Context
}

type ComicImage struct {
	Thumbnail string `json:"thumbnail"`
	Medium    string `json:"medium"`
	High      string `json:"high"`
	Original  string `json:"original"`
}

type ChapterPage struct {
	PageNumber int    `json:"page_number"`
	ImageURL   string `json:"image_url"`
	SecureURL  string `json:"secure_url"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

func NewService() (*Service, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("faltan credenciales de Cloudinary")
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}

	return &Service{cld: cld, ctx: context.Background()}, nil
}

func (s *Service) UploadComicCover(file multipart.File, fileHeader *multipart.FileHeader, comicID string) (*ComicImage, error) {
	if !isValidImage(fileHeader.Filename) {
		return nil, fmt.Errorf("formato no soportado")
	}

	publicID := fmt.Sprintf("comics/%s/cover", comicID)

	uploadParams := uploader.UploadParams{
		PublicID:       publicID,
		Folder:         "mangaty/comics",
		Transformation: "c_fill,g_auto,q_auto:good",
		Eager:          "c_fill,g_auto,h_300,w_200|c_fill,g_auto,h_600,w_400|c_fill,g_auto,h_1200,w_800",
		Overwrite:      api.Bool(true),
	}

	result, err := s.cld.Upload.Upload(s.ctx, file, uploadParams)
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("https://res.cloudinary.com/%s", s.cld.Config.Cloud.CloudName)
	
	return &ComicImage{
		Original:  result.SecureURL,
		Thumbnail: fmt.Sprintf("%s/upload/c_fill,g_auto,h_300,w_200/%s.%s", baseURL, publicID, result.Format),
		Medium:    fmt.Sprintf("%s/upload/c_fill,g_auto,h_600,w_400/%s.%s", baseURL, publicID, result.Format),
		High:      fmt.Sprintf("%s/upload/c_fill,g_auto,h_1200,w_800/%s.%s", baseURL, publicID, result.Format),
	}, nil
}

func (s *Service) UploadChapterPages(files []*multipart.FileHeader, comicID string, chapterNumber int) ([]ChapterPage, error) {
	var pages []ChapterPage
	folder := fmt.Sprintf("comics/%s/chapters/ch_%d", comicID, chapterNumber)

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}

		publicID := fmt.Sprintf("%s/page_%03d", folder, i+1)

		uploadParams := uploader.UploadParams{
			PublicID:       publicID,
			Folder:         "mangaty",
			Transformation: "q_auto:good,f_auto",
			Overwrite:      api.Bool(true),
		}

		result, err := s.cld.Upload.Upload(s.ctx, file, uploadParams)
		file.Close()
		
		if err != nil {
			return nil, err
		}

		pages = append(pages, ChapterPage{
			PageNumber: i + 1,
			ImageURL:   result.URL,
			SecureURL:  result.SecureURL,
			Width:      result.Width,
			Height:     result.Height,
		})
	}

	return pages, nil
}

func (s *Service) DeleteComic(comicID string) error {
	// Eliminar recursos uno por uno usando prefijos conocidos
	coverPublicID := fmt.Sprintf("mangaty/comics/%s/cover", comicID)
	_, err := s.cld.Admin.DeleteAssets(s.ctx, admin.DeleteAssetsParams{
		PublicIDs: []string{coverPublicID},
	})
	return err
}

func isValidImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	return validExts[ext]
}

// En pkg/cloudinary/cloudinary.go

// GenerateChapterPageURLs genera URLs con extensión .jpg
func (s *Service) GenerateChapterPageURLs(comicID string, chapterNumber, totalPages int) []ChapterPage {
	var pages []ChapterPage
	baseURL := fmt.Sprintf("https://res.cloudinary.com/%s", s.cld.Config.Cloud.CloudName)
	folder := fmt.Sprintf("comics/%s/chapters/ch_%d", comicID, chapterNumber)

	for i := 1; i <= totalPages; i++ {
		publicID := fmt.Sprintf("%s/page_%03d", folder, i)
		page := ChapterPage{
			PageNumber: i,
			ImageURL:   fmt.Sprintf("%s/image/upload/q_auto:good,f_auto/%s.jpg", baseURL, publicID),
			SecureURL:  fmt.Sprintf("%s/image/upload/q_auto:good,f_auto/%s.jpg", baseURL, publicID),
		}
		pages = append(pages, page)
	}

	return pages
}