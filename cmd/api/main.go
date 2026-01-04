package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/auth"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/config"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/database"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/handler"
	appMiddleware "github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/middleware"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/repository"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations in dev environment
	if cfg.Environment == "dev" {
		log.Println("Running database migrations...")
		if err := database.Migrate(cfg.DatabaseURL); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations completed successfully")
	}

	// Validate JWT secret is configured
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg)

	// Initialize Google Drive service (optional - only if credentials are configured)
	var gdriveService *storage.GDriveService
	if cfg.GDriveCredentialsPath != "" && cfg.GDriveTokenPath != "" && cfg.GDriveFolderID != "" {
		var err error
		gdriveService, err = storage.NewGDriveService(cfg.GDriveCredentialsPath, cfg.GDriveTokenPath, cfg.GDriveFolderID)
		if err != nil {
			log.Printf("Warning: Failed to initialize Google Drive service: %v", err)
			log.Println("Image uploads will be disabled")
		} else {
			log.Println("Google Drive service initialized successfully")
		}
	} else {
		log.Println("Google Drive credentials not configured, image uploads disabled")
	}

	// Initialize repository and handlers
	queries := repository.New(db)
	authHandler := handler.NewAuthHandler(queries, jwtManager)
	markerHandler := handler.NewMarkerHandler(queries, gdriveService, cfg.DeepLinkBaseURL)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			// Public routes
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(appMiddleware.JWTAuth(jwtManager))
				r.Get("/me", authHandler.GetMe)
				r.Post("/logout", authHandler.Logout)
			})
		})

		// Marker routes
		r.Route("/markers", func(r chi.Router) {
			// Public route - for QR code scanning
			r.Get("/code/{shortCode}", markerHandler.GetByShortCode)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(appMiddleware.JWTAuth(jwtManager))
				r.Get("/", markerHandler.List)
				r.Get("/paginated", markerHandler.ListPaginated)
				r.Post("/", markerHandler.Create)
				r.Get("/{id}", markerHandler.GetByID)
				r.Get("/{id}/qr", markerHandler.GenerateQR)
				r.Put("/{id}", markerHandler.Update)
				r.Delete("/{id}", markerHandler.Delete)
			})
		})
	})

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s (env: %s)", port, cfg.Environment)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
		os.Exit(1)
	}
}
