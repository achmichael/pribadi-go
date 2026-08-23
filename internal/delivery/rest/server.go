package rest

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/achmichael/pribadi-go/internal/usecase"
)

// RateLimiter tracks requests per user
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]int
	resetAt  time.Time
}

func newRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]int),
		resetAt:  time.Now().Add(time.Minute),
	}
}

func (rl *RateLimiter) allow(userID string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if time.Now().After(rl.resetAt) {
		rl.requests = make(map[string]int)
		rl.resetAt = time.Now().Add(time.Minute)
	}

	rl.requests[userID]++
	return rl.requests[userID] <= limit
}

// Server represents the REST API server for the dashboard
type Server struct {
	router         *chi.Mux
	logger         *zerolog.Logger
	port           string
	service        usecase.DashboardService
	webChatService usecase.WebChatService
	jwtKey         []byte
	rateLimiter    *RateLimiter
}

func (s *Server) Router() *chi.Mux {
    return s.router
}

// NewServer creates a new dashboard REST API server
func NewServer(service usecase.DashboardService, webChatService usecase.WebChatService, jwtSecret string, logger *zerolog.Logger, port string) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS config - Tightened for production
	r.Use(cors.Handler(cors.Options{
		// Only allow specific origins in production, fallback to localhost for dev
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8090"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	srv := &Server{
		router:         r,
		logger:         logger,
		port:           port,
		service:        service,
		webChatService: webChatService,
		jwtKey:         []byte(jwtSecret),
		rateLimiter:    newRateLimiter(),
	}

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/auth/login", srv.handleLogin)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(srv.authMiddleware)

			// Config
			r.Get("/config", srv.handleGetConfigAll)
			r.Get("/config/{key}", srv.handleGetConfigByKey)
			r.Put("/config/{key}", srv.handleUpdateConfig)

			// Entities
			r.Get("/entities/schemas", srv.handleListEntitySchemas)
			r.Post("/entities/schemas", srv.handleCreateEntitySchema)
			r.Get("/entities/schemas/{id}", srv.handleGetEntitySchema)
			r.Put("/entities/schemas/{id}", srv.handleUpdateEntitySchema)
			r.Delete("/entities/schemas/{id}", srv.handleDeleteEntitySchema)

			r.Get("/entities/schemas/{schema_id}/records", srv.handleListEntityRecords)
			r.Post("/entities/schemas/{schema_id}/records", srv.handleCreateEntityRecord)
			r.Get("/entities/records/{id}", srv.handleGetEntityRecord)
			r.Put("/entities/records/{id}", srv.handleUpdateEntityRecord)
			r.Delete("/entities/records/{id}", srv.handleDeleteEntityRecord)

			// Cron Jobs
			r.Get("/cron", srv.handleListCronJobs)
			r.Post("/cron", srv.handleCreateCronJob)
			r.Put("/cron/{id}", srv.handleUpdateCronJob)
			r.Delete("/cron/{id}", srv.handleDeleteCronJob)

			// Stocks
			r.Get("/stocks", srv.handleListStocks)
			r.Post("/stocks", srv.handleAddStock)
			r.Put("/stocks/{id}", srv.handleUpdateStock)
			r.Delete("/stocks/{id}", srv.handleDeleteStock)

			// Chat (Web UI)
			r.Group(func(r chi.Router) {
				r.Get("/chat/sessions", srv.handleGetSessions)
				r.Post("/chat/sessions", srv.handleCreateSession)
				r.Put("/chat/sessions/{id}", srv.handleUpdateSession)
				r.Delete("/chat/sessions/{id}", srv.handleDeleteSession)
				r.Get("/chat/sessions/{id}/history", srv.handleGetSessionHistory)
				r.Put("/chat/settings", srv.handleChatSettings)
				
				// Rate limited chat endpoints
				r.Group(func(r chi.Router) {
					r.Use(srv.chatRateLimitMiddleware)
					r.Post("/chat/stream", srv.handleChatStream)
					r.Post("/chat/upload", srv.handleFileUpload)
				})
			})
		})
	})

	// Serve static files for Dashboard SPA
	srv.serveSPA(r, "dashboard/dist")

	return srv
}

func (s *Server) serveSPA(r chi.Router, publicDir string) {
	fs := http.FileServer(http.Dir(publicDir))

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		fullPath := filepath.Join(publicDir, filepath.Clean(path))
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.ServeFile(w, req, filepath.Join(publicDir, "index.html"))
			return
		}

		fs.ServeHTTP(w, req)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return s.jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Add user ID to context if needed
		userID := claims["user_id"].(string)
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// chatRateLimitMiddleware limits chat requests per user
func (s *Server) chatRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(string)
		if !s.rateLimiter.allow(userID, 30) { // 30 messages per minute
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggerMiddleware(logger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("duration", time.Since(start)).
				Msg("API Request")
		})
	}
}

// Start starts the REST API server
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":" + s.port,
		Handler: s.router,
	}

	go func() {
		s.logger.Info().Str("port", s.port).Msg("REST API server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal().Err(err).Msg("REST API server failed")
		}
	}()

	<-ctx.Done()

	s.logger.Info().Msg("Shutting down REST API server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
