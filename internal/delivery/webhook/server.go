package webhook

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// Server represents the webhook HTTP server
type Server struct {
	router *chi.Mux
	logger *zerolog.Logger
	port   string
}

// NewServer creates a new webhook server
func NewServer(handler *Handler, logger *zerolog.Logger, port string) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Post("/webhook", handler.HandleWebhook)

	return &Server{
		router: r,
		logger: logger,
		port:   port,
	}
}

// Start starts the webhook server
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":" + s.port,
		Handler: s.router,
	}

	// Start server in goroutine
	go func() {
		s.logger.Info().Str("port", s.port).Msg("Webhook server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal().Err(err).Msg("Webhook server failed")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	s.logger.Info().Msg("Shutting down webhook server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

// loggerMiddleware creates a middleware for request logging
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
				Str("request_id", middleware.GetReqID(r.Context())).
				Msg("HTTP request")
		})
	}
}
