package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/achmichael/pribadi-go/internal/auth/infrastructure"
)

func RegisterAuthRoutes(r chi.Router, h *AuthHandler, tokenSvc *infrastructure.TokenService) {
	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)

		// Public
		r.Post("/register", h.Register)
		r.Post("/verify-otp", h.VerifyOTP) // Impl similar to Register
		r.Post("/refresh-token", h.RefreshToken)

		// Rate limited
		r.Group(func(r chi.Router) {
			r.Use(RateLimitMiddleware)
			r.Post("/login", h.Login)
			r.Post("/resend-otp", h.ResendOTP)
			r.Post("/forgot-password", h.ForgotPassword)
			r.Post("/verify-reset-otp", h.VerifyResetOTP)
			r.Post("/reset-password", h.ResetPassword)
		})

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(tokenSvc))
			r.Post("/logout", h.Logout)
			r.Get("/me", h.Me)
		})
	})
}
