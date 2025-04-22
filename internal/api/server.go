package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	apiv1 "stahl/api/gen/go"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// APIServer represents the API server
type APIServer struct {
	grpcServer *grpc.Server
	httpServer *http.Server
	grpcPort   int
	httpPort   int
	service    *ConfigurationService
}

// NewAPIServer creates a new instance of APIServer
func NewAPIServer(service *ConfigurationService, grpcPort, httpPort int) *APIServer {
	return &APIServer{
		service:  service,
		grpcPort: grpcPort,
		httpPort: httpPort,
	}
}

// Start launches gRPC and HTTP servers
func (s *APIServer) Start(ctx context.Context) error {
	// Start gRPC server
	errCh := make(chan error, 1)
	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.grpcPort))
		if err != nil {
			errCh <- fmt.Errorf("failed to listen: %v", err)
			return
		}

		s.grpcServer = grpc.NewServer()
		// Register service
		apiv1.RegisterConfigurationServiceServer(s.grpcServer, s.service)

		log.Printf("gRPC server is listening on port %d", s.grpcPort)
		if err := s.grpcServer.Serve(listener); err != nil {
			errCh <- fmt.Errorf("failed to serve gRPC: %v", err)
		}
	}()

	// Start HTTP server with gRPC-Gateway
	go func() {
		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

		// Register HTTP handlers
		err := apiv1.RegisterConfigurationServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.grpcPort), opts)
		if err != nil {
			errCh <- fmt.Errorf("failed to register handler: %v", err)
			return
		}

		s.httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.httpPort),
			Handler: mux,
		}

		log.Printf("HTTP server is listening on port %d", s.httpPort)
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to serve HTTP: %v", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.Stop()
		return ctx.Err()
	}
}

// Stop shuts down the servers
func (s *APIServer) Stop() {
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}
