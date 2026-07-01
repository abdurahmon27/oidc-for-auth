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

	"github.com/user/auth-service/internal/auth"
	"github.com/user/auth-service/internal/config"
	db "github.com/user/auth-service/internal/database"
	"github.com/user/auth-service/internal/handler"
	"github.com/user/auth-service/internal/middleware"
	"github.com/user/auth-service/internal/service"
	"github.com/user/auth-service/internal/telegram"
	"github.com/user/auth-service/internal/token"
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
	cookieCfg := token.CookieConfig{
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

	// Provider registry
	registry := auth.NewRegistry()
	backendURL := "http://localhost:" + cfg.ServerPort

	if cfg.GoogleClientID != "" {
		p, err := auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, backendURL+"/auth/google/callback")
		if err != nil {
			log.Printf("warning: google provider init failed: %v", err)
		} else {
			registry.Register(p)
		}
	}

	if cfg.MicrosoftClientID != "" {
		p, err := auth.NewMicrosoftProvider(cfg.MicrosoftClientID, cfg.MicrosoftClientSecret, cfg.MicrosoftTenant, backendURL+"/auth/microsoft/callback")
		if err != nil {
			log.Printf("warning: microsoft provider init failed: %v", err)
		} else {
			registry.Register(p)
		}
	}

	if cfg.FacebookClientID != "" {
		registry.Register(auth.NewFacebookProvider(cfg.FacebookClientID, cfg.FacebookClientSecret, backendURL+"/auth/facebook/callback"))
	}

	if cfg.GitHubClientID != "" {
		registry.Register(auth.NewGitHubProvider(cfg.GitHubClientID, cfg.GitHubClientSecret, backendURL+"/auth/github/callback"))
	}

	// Handlers
	oauthHandler := handler.NewOAuthHandler(registry, userService, queries, cfg.JWTSecret, cookieCfg, cfg.FrontendURL)
	telegramHandler := handler.NewTelegramHandler(telegramBot, userService, queries, cfg.JWTSecret, cookieCfg, cfg.FrontendURL)
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

	// OAuth routes
	r.Get("/auth/{provider}/login", oauthHandler.Login)
	r.Get("/auth/{provider}/callback", oauthHandler.Callback)

	// Telegram routes (bot-delivered OTP)
	r.Post("/auth/telegram/start", telegramHandler.Start)
	r.Post("/auth/telegram/verify", telegramHandler.Verify)

	// Session routes
	r.Post("/auth/refresh", sessionHandler.Refresh)
	r.With(middleware.CSRF).Post("/auth/logout", sessionHandler.Logout)

	// Protected routes
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
		log.Printf("server starting on :%s", cfg.ServerPort)
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
