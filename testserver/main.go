package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"google.golang.org/genproto/googleapis/rpc/errdetails"

	pb "pulsar-grpc/testserver/hello"
)

type server struct {
	pb.UnimplementedEchoServiceServer
}

func (s *server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoRequest, error) {
	return req, nil
}

func (s *server) FailWithError(ctx context.Context, req *pb.ErrorRequest) (*pb.ErrorResponse, error) {
	c := codes.Code(req.Code)
	st, _ := status.New(c, req.Message).WithDetails(
		&errdetails.LocalizedMessage{
			Locale:  "ru",
			Message: "Произошла ошибка при обработке запроса",
		},
		&errdetails.ErrorInfo{
			Reason: "TEST_ERROR",
			Domain: "testserver.local",
			Metadata: map[string]string{
				"service": "EchoService",
				"method":  "FailWithError",
			},
		},
	)
	return nil, st.Err()
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterEchoServiceServer(s, &server{})
	reflection.Register(s)
	log.Println("Test gRPC server on :50051")
	log.Fatal(s.Serve(lis))
}
