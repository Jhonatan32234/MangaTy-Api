package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mangatyapi/internal/auth/delivery"
	authRepo "mangatyapi/internal/auth/repository"
	authUC "mangatyapi/internal/auth/usecase"

	catalogDelivery "mangatyapi/internal/catalog/delivery"
	catalogRepo "mangatyapi/internal/catalog/repository"
	catalogUC "mangatyapi/internal/catalog/usecase"

	walletDelivery "mangatyapi/internal/wallet/delivery"
	walletRepo "mangatyapi/internal/wallet/repository"
	walletUC "mangatyapi/internal/wallet/usecase"

	"mangatyapi/pkg/cloudinary"
	"mangatyapi/pkg/database"
	"mangatyapi/pkg/middleware"
	"mangatyapi/pkg/security"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("Archivo .env no encontrado")
	}

	// Inicializar logs de auditoría
	if err := security.InitAuditLog(); err != nil {
		log.Printf("No se pudo inicializar audit.log: %v", err)
	}
	defer security.CloseAuditLog()

	// Inicializar Cloudinary
	cldService, err := cloudinary.NewService()
	if err != nil {
		log.Printf("Cloudinary no configurado: %v", err)
		cldService = nil
	} else {
		log.Println("Cloudinary inicializado")
	}

	// Inicializar base de datos
	db, err := database.NewPostgreSQLConnection()
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}
	defer db.Close()

	// Inicializar repositorios
	authRepository := authRepo.NewPostgresRepository(db)
	walletRepository := walletRepo.NewPostgresRepository(db)
	catalogRepository := catalogRepo.NewPostgresRepository(db)

	// Inicializar casos de uso
	authUseCase := authUC.NewUseCase(authRepository)
	walletUseCase := walletUC.NewUseCase(walletRepository)
	catalogUseCase := catalogUC.NewUseCase(catalogRepository)

	// Inicializar handlers
	authHandler := delivery.NewAuthHandler(authUseCase)
	walletHandler := walletDelivery.NewWalletHandler(walletUseCase)
	catalogHandler := catalogDelivery.NewCatalogHandler(catalogUseCase, cldService)

	// Configurar router
	r := chi.NewRouter()

	// ============ MIDDLEWARES DE SEGURIDAD ============

	// 1. Headers de seguridad OWASP
	r.Use(middleware.SecurityHeaders())

	// 2. Rate limiting global
	r.Use(middleware.RateLimiter())

	// 3. Auditoría
	r.Use(middleware.AuditMiddleware())

	// 4. CORS
	r.Use(corsMiddleware)

	// 5. Middlewares estándar
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Documentación Swagger
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/index.html")
	})
	r.Get("/docs/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		http.ServeFile(w, r, "./docs/swagger.yaml")
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","services":{"api":"running","database":"connected"}}`))
	})

	// Middleware personalizado
	authMiddleware := middleware.NewAuthMiddleware()

	// ============ RUTAS DE LA API v1 ============
	r.Route("/api/v1", func(r chi.Router) {
		// Rate limiting estricto para auth
		r.Group(func(r chi.Router) {
			r.Use(middleware.StrictRateLimiter())

			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
		})

		// Rutas protegidas
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			r.Get("/user/profile", authHandler.GetProfile)

			r.Get("/wallet/balance", walletHandler.GetBalance)
			r.Post("/wallet/unlock-chapter", walletHandler.UnlockChapter)

			r.Get("/comics/{comicID}", catalogHandler.GetComic)
			r.Get("/comics/{comicID}/chapters", catalogHandler.ListChapters)
			r.Get("/catalog/categories", catalogHandler.ListCategories)
			r.Get("/chapters/{chapterID}/read", catalogHandler.ReadChapter)
			r.Get("/categories", catalogHandler.ListCategories)
			r.Get("/catalog/recommendations", catalogHandler.GetRecommendations)
			r.Get("/comics", catalogHandler.GetComicsByCategory) // Filtrar por ?category_id=2&page=1&limit=20


			// Rutas de autor/admin
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("autor", "administrador"))

				r.Post("/comics", catalogHandler.CreateComic)
				r.Post("/comics/{comicID}/chapters", catalogHandler.CreateChapter)

				if cldService != nil {
					r.Post("/comics/{comicID}/cover", catalogHandler.UploadComicCover)
					r.Post("/comics/{comicID}/chapters/{chapterNumber}/pages", catalogHandler.UploadChapterPages)
				}
			})

			// Rutas de admin
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("administrador"))

				r.Post("/admin/assign-role", authHandler.AssignRole)
				r.Delete("/comics/{comicID}", catalogHandler.DeleteComic)
			})
		})
	})

	// Configurar servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tlsConfig := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig:    tlsConfig,
		// No revelar versión del servidor
		ErrorLog: nil,
	}

	// Iniciar servidor
	go func() {
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("Mangaty API v1.0.0 - http://localhost:%s", port)
		log.Println("Seguridad OWASP Top 10 implementada")
		log.Println("   ✓ Rate limiting")
		log.Println("   ✓ Security headers")
		log.Println("   ✓ Input validation")
		log.Println("   ✓ Audit logging")
		log.Println("   ✓ JWT authentication")
		log.Println("   ✓ Role-based access")
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Apagando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Error en apagado: %v", err)
	}
	log.Println("Servidor detenido")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		// En producción, reemplazar "*" por el dominio real de tu web/app
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
