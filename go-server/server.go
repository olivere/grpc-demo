package main

//go:generate protoc -I ../pb/ ../pb/example.proto --go_out=plugins=grpc:../pb

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/olivere/grpc-demo/pb"
)

type Server struct {
	log.Logger
}

func NewServer(logger log.Logger) *Server {
	return &Server{
		Logger: log.With(logger, "component", "server"),
	}
}

func (s *Server) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	user, ok := getUser(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "client didn't pass a user")
	}
	s.Log("method", "Hello", "user", user)

	d, ok := ctx.Deadline()
	if !ok {
		return nil, status.Error(codes.InvalidArgument,
			"no deadline/timeout specified in request")
	}
	timeout := d.Sub(time.Now())
	if timeout < 5*time.Second || timeout >= 30*time.Second {
		return nil, status.Errorf(codes.InvalidArgument,
			"deadline must be 5-30 seconds in future; was: %v", timeout)
	}

	var gender string
	switch req.Gender {
	case pb.Gender_MALE:
		gender = "male person"
	case pb.Gender_FEMALE:
		gender = "female person"
	default:
		gender = "person of an unknown gender"
	}
	msg := fmt.Sprintf("%s: Hello %s, you are a %d year old %s.",
		time.Now().Format(time.RFC3339),
		req.Name,
		req.Age,
		gender,
	)
	return &pb.HelloResponse{
		Message: msg,
	}, nil
}

func (s *Server) Ticker(req *pb.TickerRequest, stream pb.Example_TickerServer) error {
	ctx := stream.Context()

	user, ok := getUser(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "client didn't pass a user")
	}
	s.Log("method", "Ticker", "user", user)

	loc, err := time.LoadLocation(req.Timezone)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid timezone")
	}
	ticker := time.NewTicker(time.Duration(req.Interval) * time.Nanosecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := stream.Send(&pb.TickerResponse{
				Tick: time.Now().In(loc).Format(time.RFC3339),
			})
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
