// Package main is the entry point for the MCP service.
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

	"go.uber.org/zap"

	_ "github.com/lib/pq"

	"github.com/antigravity/go-agent-service/internal/appregistry"
	"github.com/antigravity/go-agent-service/internal/config"
	"github.com/antigravity/go-agent-service/internal/keystore"
	"github.com/antigravity/go-agent-service/internal/mcp"
	"github.com/antigravity/go-agent-service/internal/nucleus"
	"github.com/antigravity/go-agent-service/internal/tools"
)

type nucleusToolAdapter struct {
	tool *tools.NucleusSearchTool
}

func (a nucleusToolAdapter) Definition() mcp.ToolDefinition {
	if a.tool == nil {
		return mcp.ToolDefinition{}
	}
	def := a.tool.Definition()
	actions := make([]mcp.ActionDefinition, 0, len(def.Actions))
	for _, action := range def.Actions {
		actions = append(actions, mcp.ActionDefinition{
			Name:         action.Name,
			Description:  action.Description,
			InputSchema:  action.InputSchema,
			OutputSchema: action.OutputSchema,
		})
	}
	return mcp.ToolDefinition{
		Name:        def.Name,
		Description: def.Description,
		Actions:     actions,
	}
}

func (a nucleusToolAdapter) Execute(ctx context.Context, params map[string]any) (*mcp.Result, error) {
	if a.tool == nil {
		return &mcp.Result{Success: false, Message: "nucleus tool not available"}, nil
	}
	res, err := a.tool.Execute(ctx, params)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return &mcp.Result{Success: false, Message: "empty result"}, nil
	}
	return &mcp.Result{
		Success: res.Success,
		Data:    res.Data,
		Message: res.Message,
	}, nil
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg, err := config.Load()
	if err != nil {
		sugar.Fatalf("Failed to load config: %v", err)
	}

	port := getEnvInt("MCP_PORT", 9100)
	keystoreURL := getEnv("KEYSTORE_URL", "http://localhost:9200")

	sugar.Infow("Starting MCP Server",
		"port", port,
		"ucl_url", cfg.Nucleus.UCLURL,
		"nucleus_api", cfg.Nucleus.APIURL,
		"keystore_url", keystoreURL,
	)

	// Remote keystore client (optional)
	var keyStore keystore.Store
	if keystoreURL != "" {
		keyStore = keystore.NewRemoteStore(keystoreURL, sugar)
	}

	var uclServer *mcp.Server

	// Nucleus tool for brain search
	nucleusClient := nucleus.NewClientWithConfig(nucleus.ClientConfig{
		APIURL:   cfg.Nucleus.APIURL,
		Username: cfg.Nucleus.Username,
		Password: cfg.Nucleus.Password,
		TenantID: cfg.Nucleus.TenantID,
	}, sugar)
	nucleusTool := tools.NewNucleusSearchTool(nucleusClient)

	// App registry (optional)
	var resolver *appregistry.Resolver
	if cfg.PostgresURL != "" {
		db, err := sql.Open("postgres", cfg.PostgresURL)
		if err != nil {
			sugar.Warnw("Failed to connect to Postgres for app registry", "error", err)
		} else {
			resolver = &appregistry.Resolver{
				Registry: appregistry.NewPostgresStore(db),
				Nucleus:  nucleusClient,
				KeyStore: keyStore,
			}
			defer db.Close()
		}
	}

	uclServer = mcp.NewServer(cfg.Nucleus.UCLURL, keyStore, resolver, sugar)
	if err := uclServer.Connect(context.Background()); err != nil {
		sugar.Warnw("Failed to connect to UCL", "error", err)
	}

	service := mcp.NewService(uclServer, nucleusToolAdapter{tool: nucleusTool}, sugar)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: service.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		sugar.Infof("MCP server listening on :%d", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("MCP server failed: %v", err)
		}
	}()

	<-ctx.Done()
	sugar.Info("Shutting down MCP server")
	_ = httpServer.Close()
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
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
