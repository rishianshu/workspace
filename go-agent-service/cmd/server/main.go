// Package main is the entry point for the Go Agent Service
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/antigravity/go-agent-service/internal/config"
	"github.com/antigravity/go-agent-service/internal/server"
	"github.com/antigravity/go-agent-service/internal/ucl"
	"github.com/antigravity/go-agent-service/internal/workflow"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		sugar.Fatalf("Failed to load config: %v", err)
	}

	sugar.Infow("Starting Go Agent Service",
		"grpc_port", cfg.GRPCPort,
		"http_port", cfg.GRPCPort+1,
		"nucleus_url", cfg.NucleusURL,
	)

	// Create agent server (shared between gRPC and HTTP)
	agentServer := server.NewAgentServer(cfg, sugar)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	server.RegisterAgentServiceServer(grpcServer, agentServer)
	reflection.Register(grpcServer)

	// Create HTTP handler for rust-gateway compatibility
	httpHandler := server.NewHTTPHandler(agentServer, sugar)
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/chat", httpHandler.HandleChat)
	httpMux.HandleFunc("/workflows", httpHandler.HandleListWorkflows)
	httpMux.HandleFunc("/workflows/create", httpHandler.HandleCreateWorkflow)
	httpMux.HandleFunc("/workflows/cancel", httpHandler.HandleCancelWorkflow)
	httpMux.HandleFunc("/tools", httpHandler.HandleListTools)
	httpMux.HandleFunc("/tools/execute", httpHandler.HandleExecuteTool)
	httpMux.HandleFunc("/action", httpHandler.HandleExecuteAction)
	httpMux.HandleFunc("/brain/search", httpHandler.HandleBrainSearch)
	httpMux.HandleFunc("/projects", httpHandler.HandleListProjects)
	httpMux.HandleFunc("/apps/instances", httpHandler.HandleAppInstances)
	httpMux.HandleFunc("/apps/users", httpHandler.HandleUserApps)
	httpMux.HandleFunc("/apps/projects", httpHandler.HandleProjectApps)
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		sugar.Fatalf("Failed to listen gRPC: %v", err)
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start gRPC in goroutine
	go func() {
		sugar.Infof("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			sugar.Fatalf("gRPC serve failed: %v", err)
		}
	}()

	// Start Temporal Worker
	go func() {
		// Create a separate Temporal client for the worker
		// Note: In production, we might want to share the client or connection
		c, err := client.Dial(client.Options{
			HostPort: cfg.TemporalHost,
			Logger:   nil, // TODO: Use zap adapter
		})
		if err != nil {
			sugar.Warnw("Failed to create Temporal client for worker", "error", err)
			return
		}
		defer c.Close()

		w := worker.New(c, "agent-task-queue", worker.Options{})

		// Register Workflows
		w.RegisterWorkflow(workflow.DynamicWorkflow)

		// Register Activities
		// Need a UCL executor. For now, we use a fresh StubRegistry or reusing AgentServer's registry would be better
		// But AgentServer exposes it? No.
		// Let's create a new StubRegistry for the worker
		uclRegistry := ucl.NewStubToolRegistry(sugar)
		activities := workflow.NewActivities(uclRegistry, sugar)

		w.RegisterActivity(activities.CallUCLActivity)
		w.RegisterActivity(activities.CallLLMActivity)
		w.RegisterActivity(activities.RequestApprovalActivity)

		sugar.Info("Starting Temporal Worker")
		if err := w.Run(worker.InterruptCh()); err != nil {
			sugar.Fatalf("Worker failed: %v", err)
		}
	}()

	// Start HTTP in goroutine
	httpPort := cfg.GRPCPort + 1
	go func() {
		sugar.Infof("HTTP server listening on :%d", httpPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), httpMux); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("HTTP serve failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	sugar.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
}
