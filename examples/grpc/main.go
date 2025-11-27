package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/airbrake/gobrake/v5"
	grpcbrake "github.com/airbrake/gobrake/v5/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Example proto definitions
// In a real application, these would be generated from .proto files

type HelloRequest struct {
	Name string
}

type HelloResponse struct {
	Message string
}

type GreeterServer interface {
	SayHello(context.Context, *HelloRequest) (*HelloResponse, error)
}

// greeterServer implements the GreeterServer interface
type greeterServer struct{}

func (s *greeterServer) SayHello(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
	}

	// Simulate an error for demonstration
	if req.Name == "error" {
		return nil, status.Error(codes.Internal, "simulated internal error")
	}

	return &HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.Name),
	}, nil
}

func main() {
	// Initialize Airbrake notifier
	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   123456,  // Replace with your project ID
		ProjectKey:  "FIXME", // Replace with your project key
		Environment: "development",
	})
	defer notifier.Close()

	// Create a gRPC server with Airbrake interceptors
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcbrake.UnaryServerInterceptor(notifier),
		),
		grpc.ChainStreamInterceptor(
			grpcbrake.StreamServerInterceptor(notifier),
		),
	)

	// Register your service
	// In a real application, you would use generated RegisterXXXServer functions
	// RegisterGreeterServer(server, &greeterServer{})

	// Start the server
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("gRPC server listening on :50051")
	log.Println("Airbrake integration enabled")

	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
