package grpc

import (
	"testing"

	"google.golang.org/grpc/codes"
)

func TestGrpcCodeToHTTP(t *testing.T) {
	tests := []struct {
		name     string
		code     codes.Code
		expected int
	}{
		{"OK", codes.OK, 200},
		{"Canceled", codes.Canceled, 499},
		{"Unknown", codes.Unknown, 500},
		{"InvalidArgument", codes.InvalidArgument, 400},
		{"DeadlineExceeded", codes.DeadlineExceeded, 504},
		{"NotFound", codes.NotFound, 404},
		{"AlreadyExists", codes.AlreadyExists, 409},
		{"PermissionDenied", codes.PermissionDenied, 403},
		{"ResourceExhausted", codes.ResourceExhausted, 429},
		{"FailedPrecondition", codes.FailedPrecondition, 400},
		{"Aborted", codes.Aborted, 409},
		{"OutOfRange", codes.OutOfRange, 400},
		{"Unimplemented", codes.Unimplemented, 501},
		{"Internal", codes.Internal, 500},
		{"Unavailable", codes.Unavailable, 503},
		{"DataLoss", codes.DataLoss, 500},
		{"Unauthenticated", codes.Unauthenticated, 401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grpcCodeToHTTP(tt.code)
			if result != tt.expected {
				t.Errorf("grpcCodeToHTTP(%v) = %d; want %d", tt.code, result, tt.expected)
			}
		})
	}
}
