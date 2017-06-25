package main

//go:generate protoc -I ../pb/ ../pb/example.proto --go_out=plugins=grpc:../pb

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/olivere/grpc-demo/pb"
)

func main() {
	var (
		addr     = flag.String("addr", ":10000", "Host and port to bind to")
		tls      = flag.Bool("tls", false, "Enabled TLS")
		certFile = flag.String("cert", "", "Certificate file")
		keyFile  = flag.String("key", "", "Key file")
	)
	flag.Parse()

	logger := log.NewLogfmtLogger(os.Stdout)
	logger = log.With(logger, "@time", log.DefaultTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)

	srv := NewServer(logger)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Log("msg", "Listen failed", "err", err)
		os.Exit(1)
	}
	var opts []grpc.ServerOption
	if *tls {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			logger.Log("msg", "Cannot create TLS credentials", "err", err)
			os.Exit(1)
		}
		opts = append(opts, grpc.Creds(creds))
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterExampleServer(grpcServer, srv)

	errc := make(chan error, 1)
	go func() {
		logger.Log("msg", "Server started")
		errc <- grpcServer.Serve(lis)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		errc <- nil
	}()

	if err := <-errc; err != nil {
		logger.Log("msg", "Exit with failure", "err", err)
	}
}

type Server struct {
	log.Logger
}

func NewServer(logger log.Logger) *Server {
	return &Server{
		Logger: log.With(logger, "component", "server"),
	}
}

func (s *Server) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	s.Log("msg", "received request", "name", req.Name)

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
	logger := log.With(s, "method", "Ticker")

	loc, err := time.LoadLocation(req.Timezone)
	if err != nil {
		logger.Log("err", err)
		return err
	}
	ticker := time.NewTicker(time.Duration(req.Interval) * time.Nanosecond)
	defer ticker.Stop()

	ctx := stream.Context()
	for {
		select {
		case <-ticker.C:
			err := stream.Send(&pb.TickerResponse{
				Tick: time.Now().In(loc).Format(time.RFC3339),
			})
			if err != nil {
				logger.Log("err", err)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
