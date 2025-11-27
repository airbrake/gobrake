package grpc

import (
	"context"

	"github.com/airbrake/gobrake/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that sends route
// performance stats and errors to Airbrake.
func UnaryServerInterceptor(notifier *gobrake.Notifier) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, metric := gobrake.NewRouteMetric(ctx, "POST", info.FullMethod)

		metric.StatusCode = 200 // default to OK
		resp, err := handler(ctx, req)
		if err != nil {
			metric.StatusCode = 500 // defaults to Internal Server Error
			st, ok := status.FromError(err)
			if ok {
				metric.StatusCode = grpcCodeToHTTP(st.Code())
			}

			notice := notifier.Notice(err, nil, 3)
			notice.Context["component"] = "grpc"
			notice.Context["action"] = info.FullMethod
			notifier.SendNoticeAsync(notice)
		}

		_ = notifier.Routes.Notify(ctx, metric)

		return resp, err
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor that sends route
// performance stats and errors to Airbrake.
func StreamServerInterceptor(notifier *gobrake.Notifier) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		ctx, metric := gobrake.NewRouteMetric(ctx, "POST", info.FullMethod)

		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		metric.StatusCode = 200 // default to OK
		err := handler(srv, wrappedStream)
		if err != nil {
			metric.StatusCode = 500 // defaults to Internal Server Error
			st, ok := status.FromError(err)
			if ok {
				metric.StatusCode = grpcCodeToHTTP(st.Code())
			}

			notice := notifier.Notice(err, nil, 3)
			notice.Context["component"] = "grpc"
			notice.Context["action"] = info.FullMethod
			notifier.SendNoticeAsync(notice)
		}

		_ = notifier.Routes.Notify(ctx, metric)
		return err
	}
}

// wrappedServerStream wraps grpc.ServerStream to override context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// grpcCodeToHTTP converts gRPC status codes to HTTP status codes
// Based on: https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md
func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.Canceled:
		return 499 // Client Closed Request
	case codes.Unknown:
		return 500
	case codes.InvalidArgument:
		return 400
	case codes.DeadlineExceeded:
		return 504
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.PermissionDenied:
		return 403
	case codes.ResourceExhausted:
		return 429
	case codes.FailedPrecondition:
		return 400
	case codes.Aborted:
		return 409
	case codes.OutOfRange:
		return 400
	case codes.Unimplemented:
		return 501
	case codes.Internal:
		return 500
	case codes.Unavailable:
		return 503
	case codes.DataLoss:
		return 500
	case codes.Unauthenticated:
		return 401
	default:
		return 500
	}
}
