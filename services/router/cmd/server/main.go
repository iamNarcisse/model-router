package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	config "llm-router/services/router/internal/config"
	embedding "llm-router/services/router/internal/embedding"
	router "llm-router/services/router/internal/router"
	server "llm-router/services/router/internal/server"
	pb "llm-router/services/router/pkg/pb"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize embedding client
	embedClient, err := embedding.NewClient(cfg.Embedding.Address, cfg.Embedding.Timeout)
	if err != nil {
		log.Fatalf("Failed to create embedding client: %v", err)
	}
	defer embedClient.Close()

	// Initialize router
	rtr := router.New(embedClient, &cfg.Qdrant, &cfg.Routing)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	routerServer := server.NewRouterServer(rtr)
	pb.RegisterRouterServiceServer(grpcServer, routerServer)
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Create gRPC-gateway mux
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = pb.RegisterRouterServiceHandlerFromEndpoint(
		ctx,
		gwMux,
		fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort),
		opts,
	)
	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	// Create main HTTP mux combining gateway and health endpoints
	httpMux := http.NewServeMux()
	httpMux.Handle("/v1/", gwMux)                        // gRPC-gateway routes
	httpMux.Handle("/health", server.NewHealthHandler(rtr)) // Legacy health endpoint
	httpMux.Handle("/ready", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	// Start HTTP server (gateway + health)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: httpMux,
	}

	go func() {
		log.Printf("HTTP server (gRPC-gateway + health) listening on :%d", cfg.Server.HTTPPort)
		log.Printf("  - REST API: http://localhost:%d/v1/*", cfg.Server.HTTPPort)
		log.Printf("  - Health:   http://localhost:%d/health", cfg.Server.HTTPPort)
		log.Printf("  - OpenAPI:  docs/api/router.swagger.json")
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	grpcServer.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	log.Println("Server stopped")
}
