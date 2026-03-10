package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxBodyBytes = 4 << 20 // 4 MB

type contextKey int

const requestIDKey contextKey = iota

// withRequestID injects a random hex request ID into the context and response headers.
// Useful for correlating log entries with specific requests.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("%016x", rand.Uint64())
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestID returns the request ID stored in the context, or empty string.
func requestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// postOnly wraps a handler to reject anything that is not POST.
func postOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// decodeBody reads and JSON-decodes the request body into v.
// Limited to maxBodyBytes to prevent unbounded memory usage.
func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(v)
}

// respondJSON writes v as JSON with Content-Type application/json.
func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("respondJSON encode", "err", err)
	}
}

// respondError writes a JSON {"error": "..."} response.
// HTTP status is derived from the gRPC status code embedded in err, if any.
func respondError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(grpcCodeToHTTPStatus(err))
	resp := map[string]string{"error": err.Error()}
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		slog.Error("respondError encode", "err", encErr)
	}
}

// grpcCodeToHTTPStatus maps a gRPC error to an appropriate HTTP status code.
// Non-gRPC errors always map to 500.
func grpcCodeToHTTPStatus(err error) int {
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch st.Code() {
	case codes.OK:
		return http.StatusOK
	case codes.NotFound:
		return http.StatusNotFound
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
