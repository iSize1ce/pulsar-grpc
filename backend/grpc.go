package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jhump/protoreflect/v2/grpcdynamic"
	"github.com/jhump/protoreflect/v2/grpcreflect"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// grpcSession holds everything needed for a single gRPC reflection interaction:
// the connection, the reflection client, a type resolver, and a dynamic stub for invoking RPCs.
type grpcSession struct {
	conn     *grpc.ClientConn
	client   *grpcreflect.Client
	resolver protoresolve.Resolver
	stub     *grpcdynamic.Stub
}

// Close releases the reflection client and closes the underlying connection.
func (s *grpcSession) Close() {
	s.client.Reset()
	if err := s.conn.Close(); err != nil {
		slog.Warn("grpc conn close", "err", err)
	}
}

// openSession connects to a gRPC server and sets up reflection.
func openSession(ctx context.Context, addr string) (*grpcSession, error) {
	//nolint:staticcheck // grpc.Dial is deprecated but grpc.NewClient changes resolver semantics
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %q: %w", addr, err)
	}

	rc := grpcreflect.NewClientAuto(ctx, conn)
	return &grpcSession{
		conn:     conn,
		client:   rc,
		resolver: rc.AsResolver(),
		stub:     grpcdynamic.NewStub(conn),
	}, nil
}

// findServiceDesc resolves a service descriptor by its fully-qualified name
// using gRPC reflection.
func findServiceDesc(resolver protoresolve.Resolver, name string) (protoreflect.ServiceDescriptor, error) {
	desc, err := resolver.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("service %q not found: %w", name, err)
	}
	sd, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("%q is not a service (got %T)", name, desc)
	}
	return sd, nil
}

// findMethodDesc finds a method descriptor by name within a service.
// Iterates through all methods because the protobuf API doesn't provide direct name lookup.
func findMethodDesc(sd protoreflect.ServiceDescriptor, name string) (protoreflect.MethodDescriptor, error) {
	methods := sd.Methods()
	for i := 0; i < methods.Len(); i++ {
		md := methods.Get(i)
		if string(md.Name()) == name {
			return md, nil
		}
	}
	return nil, fmt.Errorf("method %q not found in service %q", name, sd.FullName())
}
