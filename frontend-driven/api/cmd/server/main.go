package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"github.com/user/auth-service-fd/internal/config"
	db "github.com/user/auth-service-fd/internal/database"
	"github.com/user/auth-service-fd/internal/handler"
	"github.com/user/auth-service-fd/internal/middleware"
	"github.com/user/auth-service-fd/internal/service"
	"github.com/user/auth-service-fd/internal/telegram"
	"github.com/user/auth-service-fd/internal/validate"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	// Database
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	cookieCfg := handler.CookieConfig{
		Domain: cfg.CookieDomain,
		Secure: cfg.CookieSecure,
	}

	// Services
	userService := service.NewUserService(queries)

	// Telegram bot: fetches its @username and polls for /start in the background.
	botCtx, botCancel := context.WithCancel(context.Background())
	defer botCancel()
	telegramBot := telegram.NewBot(cfg.TelegramAPIToken)
	telegramBot.Start(botCtx)

	// Validator registry
	registry := validate.NewRegistry()

	if cfg.GoogleClientID != "" {
		v, err := validate.NewGoogleValidator(cfg.GoogleClientID)
		if err != nil {
			log.Printf("warning: google validator init failed: %v", err)
		} else {
			registry.Register(v)
		}
	}

	if cfg.MicrosoftClientID != "" {
		v, err := validate.NewMicrosoftValidator(cfg.MicrosoftClientID, cfg.MicrosoftTenant)
		if err != nil {
			log.Printf("warning: microsoft validator init failed: %v", err)
		} else {
			registry.Register(v)
		}
	}

	if cfg.FacebookClientID != "" && cfg.FacebookClientSecret != "" {
		registry.Register(validate.NewFacebookValidator(cfg.FacebookClientID, cfg.FacebookClientSecret))
	}

	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		registry.Register(validate.NewGitHubValidator(cfg.GitHubClientID, cfg.GitHubClientSecret))
	}

	// Handlers
	authHandler := handler.NewAuthenticateHandler(registry, userService, queries, cfg.JWTSecret, cookieCfg)
	telegramHandler := handler.NewTelegramHandler(telegramBot, userService, queries, cfg.JWTSecret, cookieCfg)
	sessionHandler := handler.NewSessionHandler(queries, cfg.JWTSecret, cookieCfg)
	meHandler := handler.NewMeHandler(queries)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(corsMiddleware(cfg.FrontendURL))

	rl := middleware.NewRateLimiter(rate.Limit(10), 20)
	r.Use(rl.Middleware)

	// Auth endpoints
	r.Post("/auth/token", authHandler.Authenticate)
	r.Post("/auth/refresh", sessionHandler.Refresh)
	r.Post("/auth/logout", sessionHandler.Logout)

	// Telegram (bot-delivered OTP)
	r.Post("/auth/telegram/start", telegramHandler.Start)
	r.Post("/auth/telegram/verify", telegramHandler.Verify)

	// Protected
	r.With(middleware.Auth(cfg.JWTSecret)).Get("/me", meHandler.GetMe)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("frontend-driven API starting on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("server stopped")
}

func corsMiddleware(frontendURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
