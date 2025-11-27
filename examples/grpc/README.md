# gRPC Airbrake Example

This example demonstrates how to integrate Airbrake error tracking and performance monitoring with a gRPC server using the `grpc` package from gobrake.

## Features

- **Error Tracking**: Automatically captures and reports gRPC errors to Airbrake
- **Performance Monitoring**: Tracks request timing and status codes for APM
- **Unary Interceptor**: Monitors standard request-response RPCs
- **Stream Interceptor**: Monitors streaming RPCs

## Usage

### 1. Initialize the Airbrake Notifier

```go
notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
    ProjectId:   123456,
    ProjectKey:  "your-project-key",
    Environment: "production",
})
defer notifier.Close()
```

### 2. Add Interceptors to Your gRPC Server

```go
import grpcbrake "github.com/airbrake/gobrake/v5/grpc"

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpcbrake.UnaryServerInterceptor(notifier),
    ),
    grpc.ChainStreamInterceptor(
        grpcbrake.StreamServerInterceptor(notifier),
    ),
)
```

### 3. Register Your Services and Start the Server

```go
pb.RegisterYourServiceServer(server, &yourServiceImpl{})

listener, err := net.Listen("tcp", ":50051")
if err != nil {
    log.Fatalf("failed to listen: %v", err)
}

if err := server.Serve(listener); err != nil {
    log.Fatalf("failed to serve: %v", err)
}
```

## What Gets Tracked

### Errors
- All gRPC errors are automatically sent to Airbrake
- Error context includes the full method path (e.g., `/package.Service/Method`)
- gRPC status codes are converted to HTTP equivalents for consistency

### Performance Metrics
- Request duration
- Method path
- Status code (converted from gRPC to HTTP status codes)
- All requests are tracked, including successful ones

## gRPC to HTTP Status Code Mapping

The interceptor maps gRPC status codes to HTTP status codes for APM:

- `OK` → 200
- `Canceled` → 499
- `InvalidArgument` → 400
- `DeadlineExceeded` → 504
- `NotFound` → 404
- `AlreadyExists` → 409
- `PermissionDenied` → 403
- `ResourceExhausted` → 429
- `Unauthenticated` → 401
- `Unimplemented` → 501
- `Internal` → 500
- `Unavailable` → 503
- And more...

## Running the Example

1. Replace the placeholder project ID and key with your actual Airbrake credentials
2. Generate your gRPC service from proto files
3. Register your service implementation
4. Run the server:

```bash
go run main.go
```

## Notes

- The interceptors work with any gRPC service definition
- Both unary and streaming RPCs are supported
- Errors are sent asynchronously to avoid blocking your service
- Performance metrics are collected for APM (Application Performance Monitoring)
