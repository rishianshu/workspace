// Package main is the entry point for the keystore service.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/antigravity/go-agent-service/internal/config"
	"github.com/antigravity/go-agent-service/internal/keystore"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg, err := config.Load()
	if err != nil {
		sugar.Fatalf("Failed to load config: %v", err)
	}

	port := getEnvInt("KEYSTORE_PORT", 9200)

	db, err := sql.Open("postgres", cfg.KeyStore.DatabaseURL)
	if err != nil {
		sugar.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	store := keystore.NewPostgresStore(db)
	server := keystore.NewHTTPServer(store, sugar)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: server.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		sugar.Infof("Keystore server listening on :%d", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("Keystore server failed: %v", err)
		}
	}()

	<-ctx.Done()
	sugar.Info("Shutting down keystore server")
	_ = httpServer.Close()
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return num
}
