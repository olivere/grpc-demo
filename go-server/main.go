package main

//go:generate protoc -I ../pb/ ../pb/example.proto --go_out=plugins=grpc:../pb

import (
	"flag"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	grpcmw "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcopentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soheilhy/cmux"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	pb "github.com/olivere/grpc-demo/pb"
)

func main() {
	var (
		addr     = flag.String("addr", ":10000", "Host and port to bind to")
		tls      = flag.Bool("tls", false, "Enabled TLS")
		certFile = flag.String("cert", "", "Certificate file")
		keyFile  = flag.String("key", "", "Key file")
		qps      = flag.Float64("qps", 5, "Queries per second in rate limiter")
		burst    = flag.Int("burst", 1, "Burst in rate limiter")
	)
	flag.Parse()

	logger := log.NewLogfmtLogger(os.Stdout)
	logger = log.With(logger, "@time", log.DefaultTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.NewStdlibAdapter(logger))
	grpclog.SetLogger(grpcLogger{logger})

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

	tap := NewTapHandler(
		NewMetrics(),
		rate.Limit(*qps),
		*burst,
	)

	// Common options
	opts = append(opts, grpc.MaxRecvMsgSize(1<<20)) // 1MB
	opts = append(opts, grpc.InTapHandle(tap.Handle))

	// Prometheus
	opts = append(opts, grpc.StreamInterceptor(grpcmw.ChainStreamServer(
		grpcprom.StreamServerInterceptor,
		grpcopentracing.StreamServerInterceptor(),
		grpcauth.StreamServerInterceptor(authenticate),
	)))
	opts = append(opts, grpc.UnaryInterceptor(grpcmw.ChainUnaryServer(
		grpcprom.UnaryServerInterceptor,
		grpcopentracing.UnaryServerInterceptor(),
		grpcauth.UnaryServerInterceptor(authenticate),
	)))

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterExampleServer(grpcServer, srv)
	grpcprom.Register(grpcServer)

	http.Handle("/metrics", prometheus.Handler())

	// Multiplex connections
	m := cmux.New(lis)
	grpclis := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httplis := m.Match(cmux.HTTP1Fast())

	errc := make(chan error, 1)
	go func() {
		errc <- grpcServer.Serve(grpclis)
	}()

	go func() {
		httpsrv := &http.Server{
			Addr: *addr,
		}
		errc <- httpsrv.Serve(httplis)
	}()

	go func() {
		errc <- m.Serve()
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		errc <- nil
	}()

	logger.Log("msg", "Server started")

	if err := <-errc; err != nil {
		logger.Log("msg", "Exit with failure", "err", err)
	}
}
