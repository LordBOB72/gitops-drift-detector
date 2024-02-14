package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/yourorg/gitops-drift-detector/internal/alert"
	"github.com/yourorg/gitops-drift-detector/internal/api"
	"github.com/yourorg/gitops-drift-detector/internal/audit"
	"github.com/yourorg/gitops-drift-detector/internal/cluster"
	"github.com/yourorg/gitops-drift-detector/internal/db"
	"github.com/yourorg/gitops-drift-detector/internal/drift"
	"github.com/yourorg/gitops-drift-detector/internal/git"
	"github.com/yourorg/gitops-drift-detector/internal/reconcile"
	"github.com/yourorg/gitops-drift-detector/internal/webhook"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg := loadConfig()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	if err := db.Migrate(pool); err != nil {
		log.Fatal("migration failed", zap.Error(err))
	}

	clusterMgr := cluster.NewManager(log)
	gitPoller := git.NewPoller(cfg.GitPollInterval, log)
	auditLog := audit.NewLogger(pool, log)
	alerter := alert.NewAlerter(cfg.WebhookURL, log)
	driftEngine := drift.NewEngine(clusterMgr, gitPoller, auditLog, alerter, log)
	reconciler := reconcile.NewReconciler(clusterMgr, auditLog, log)

	// kick off background polling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go driftEngine.Run(ctx)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	h := api.NewHandler(clusterMgr, gitPoller, driftEngine, reconciler, auditLog, alerter, log)
	h.Register(r)

	wh := webhook.NewHandler(driftEngine, log)
	r.POST("/webhooks/git", wh.HandleGitPush)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("server listening", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Info("server stopped")
}

type config struct {
	Port            string
	DatabaseURL     string
	GitPollInterval time.Duration
	WebhookURL      string
}

func loadConfig() config {
	pollInterval, _ := time.ParseDuration(getEnv("GIT_POLL_INTERVAL", "30s"))
	return config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://drift:drift@localhost:5432/drift?sslmode=disable"),
		GitPollInterval: pollInterval,
		WebhookURL:      getEnv("ALERT_WEBHOOK_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
