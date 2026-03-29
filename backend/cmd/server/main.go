package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/vdparikh/make-mcp/backend/internal/api"
	"github.com/vdparikh/make-mcp/backend/internal/auth"
	"github.com/vdparikh/make-mcp/backend/internal/config"
	"github.com/vdparikh/make-mcp/backend/internal/database"
	webauthnpkg "github.com/vdparikh/make-mcp/backend/internal/webauthn"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required (see env.example.sh)")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required (see env.example.sh)")
	}
	if err := auth.SetJWTSecret(jwtSecret); err != nil {
		log.Fatalf("auth: %v", err)
	}

	db, err := database.New(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.RunMigrations(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migrations completed")

	if err := db.SeedDemoData(ctx); err != nil {
		log.Printf("Warning: Failed to seed demo data: %v", err)
	}

	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(abortAwareRecovery())

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"Mcp-Session-Id",
			"mcp-session-id",
			"Last-Event-ID",
			"X-User-ID",
			"X-Organization-ID",
		},
		ExposedHeaders: []string{"Content-Disposition", "Mcp-Session-Id", "mcp-session-id"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	r.Use(func(c *gin.Context) {
		corsHandler.HandlerFunc(c.Writer, c.Request)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	wa, err := webauthnpkg.NewWebAuthn(cfg.WebAuthn.RPID, cfg.WebAuthn.RPOrigins)
	if err != nil {
		log.Fatalf("Failed to init WebAuthn: %v", err)
	}
	sessionStore := webauthnpkg.NewSessionStore(5 * time.Minute)

	handler, err := api.NewHandler(db, wa, sessionStore, cfg)
	if err != nil {
		log.Fatalf("Failed to create API handler: %v", err)
	}
	handler.RegisterRoutes(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "MCP Server Builder API",
			"version": "1.0.0",
			"docs":    "/api/health",
			// Not an MCP/SSE endpoint — remote testers (e.g. Cloudflare Playground) need the hosted path from Deploy.
			"mcp_hosted_url": "/api/users/<user_id>/<server_slug>",
		})
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	addr := cfg.ListenAddr()
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}

// abortAwareRecovery is like gin.Recovery but re-panics http.ErrAbortHandler.
// net/http/httputil.ReverseProxy (hosted MCP) panics ErrAbortHandler when the
// client disconnects or during SSE teardown; Gin's default recovery logs that as a bug.
func abortAwareRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					panic(err)
				}
				log.Printf("[Recovery] panic: %v\n%s", err, debug.Stack())
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
