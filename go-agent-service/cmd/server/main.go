// Package main is the entry point for the Go Agent Service
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/antigravity/go-agent-service/internal/config"
	"github.com/antigravity/go-agent-service/internal/server"
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
		"port", cfg.GRPCPort,
		"nucleus_url", cfg.NucleusURL,
	)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	
	// Register agent service
	agentServer := server.NewAgentServer(cfg, sugar)
	server.RegisterAgentServiceServer(grpcServer, agentServer)
	
	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		sugar.Fatalf("Failed to listen: %v", err)
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		sugar.Infof("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			sugar.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	sugar.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
}
