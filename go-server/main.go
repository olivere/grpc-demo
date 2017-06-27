package main

//go:generate protoc -I ../pb/ ../pb/example.proto --go_out=plugins=grpc:../pb

import (
	"context"
	tlspkg "crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/coreos/etcd/clientv3"
	etcdnaming "github.com/coreos/etcd/clientv3/naming"
	"github.com/go-kit/kit/log"
	grpcmw "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcopentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/olivere/randport"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soheilhy/cmux"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/naming"

	"github.com/gorilla/mux"
	"github.com/olivere/grpc-demo/go-server/health"
	pb "github.com/olivere/grpc-demo/pb"
)

const (
	// serviceName is the name under which the server is registered in client-side load balancers.
	serviceName = "grpc-demo-example"
)

var (
	_ = grpcmw.ChainStreamServer
	_ = grpcauth.StreamServerInterceptor
	_ = grpcopentracing.StreamServerInterceptor
	_ = grpcprom.Register
	_ = cmux.Any
)

func envString(name, defaults string) string {
	v := os.Getenv(name)
	if v != "" {
		return v
	}
	return defaults
}

func main() {
	var (
		disco    = flag.String("disco", envString("DISCO", ""), "Service discovery mechanism (blank or etcd)")
		addr     = flag.String("addr", envString("ADDR", "localhost:10000"), "Host and port to bind to")
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

	// Configure host and port
	host, portStr, err := net.SplitHostPort(*addr)
	if err != nil {
		logger.Log("msg", "Cannot split host and port", "addr", *addr, "err", err)
		os.Exit(1)
	}
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = randport.Get()
		*addr = net.JoinHostPort(host, strconv.Itoa(port))
	}

	// Create server
	srv := NewServer(logger)

	// Create listener
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Log("msg", "Listen failed", "err", err)
		os.Exit(1)
	}

	// Service discovery mechanism
	switch *disco {
	case "etcd":
		etcdcli, err := clientv3.NewFromURL("http://localhost:2379")
		if err != nil {
			logger.Log("msg", "Cannot connect to etcd", "err", err)
			os.Exit(1)
		}
		// Register in etcd
		resolver := &etcdnaming.GRPCResolver{Client: etcdcli}
		err = resolver.Update(context.Background(), serviceName, naming.Update{Op: naming.Add, Addr: *addr})
		if err != nil {
			logger.Log("msg", "Cannot register service in etcd", "service", serviceName, "addr", *addr, "err", err)
			os.Exit(1)
		}
		// Unregister when done
		defer resolver.Update(context.Background(), serviceName, naming.Update{Op: naming.Delete, Addr: *addr})
	}

	// Server options
	var pool *x509.CertPool
	var cert tlspkg.Certificate
	var opts []grpc.ServerOption
	if *tls {
		var err error
		cert, err = tlspkg.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			logger.Log("msg", "Cannot load certificate", "err", err)
			os.Exit(1)
		}

		// Create pool to trust
		caCert, err := ioutil.ReadFile(*certFile)
		if err != nil {
			logger.Log("msg", "Cannot load certificate", "err", err)
			os.Exit(1)
		}
		pool = x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)

		// We don't need the instruct gRPC to do TLS because we are using cmux to proxy TLS
		// creds := credentials.NewServerTLSFromCert(&cert)
		// opts = append(opts, grpc.Creds(creds))
	}

	tap := NewTapHandler(
		NewMetrics(),
		rate.Limit(*qps),
		*burst,
	)

	// Common options
	// opts = append(opts, grpc.MaxRecvMsgSize(1<<20)) // 1MB
	opts = append(opts, grpc.InTapHandle(tap.Handle))

	// gRPC middleware
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

	// Multiplex connections
	//
	// We have two modes of operating. When TLS is enabled, we serve both gRPC and
	// HTTP over TLS, i.e. Prometheus metrics are only available via https://.../metrics.
	//
	// When TLS is disabled, we are serving both gRPC and HTTP unencrypted.
	//
	// Notice that we could change this via recursive multiplexing in cmux:
	// https://godoc.org/github.com/soheilhy/cmux#ex-package--RecursiveCmux
	// That would allow us to serve e.g. HTTP over TLS as well as unencrypted.
	if *tls {
		tlscfg := &tlspkg.Config{
			Certificates: []tlspkg.Certificate{cert},
			RootCAs:      pool,
		}
		lis = tlspkg.NewListener(lis, tlscfg)
	}
	tcpmux := cmux.New(lis)
	httplis := tcpmux.Match(cmux.HTTP1Fast())
	// grpclis := tcpmux.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	// httplis := tcpmux.Match(cmux.Any())
	grpclis := tcpmux.Match(cmux.Any())
	// _ = httplis
	// _ = grpclis

	errc := make(chan error, 1)

	// gRPC listener
	go func() {
		err := grpcServer.Serve(grpclis)
		if err != cmux.ErrListenerClosed {
			errc <- err
		} else {
			errc <- nil
		}
	}()

	// HTTP listener
	go func() {
		r := mux.NewRouter()

		// Health endpoints
		r.HandleFunc("/healthz", health.HealthzHandler)
		r.HandleFunc("/healthz/status", health.ToggleHealthzStatusHandler)
		r.HandleFunc("/readiness", health.ReadinessHandler)
		r.HandleFunc("/readiness/status", health.ToggleHealthzStatusHandler)
		defer func() {
			health.SetHealtzStatus(http.StatusServiceUnavailable)
			health.SetReadinessStatus(http.StatusServiceUnavailable)
		}()
		r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Log("msg", "unmatched HTTP request", "url", r.RequestURI)
		})

		// Metrics endpoints
		r.Handle("/metrics", prometheus.Handler())

		httpsrv := &http.Server{
			Addr:    *addr,
			Handler: r,
		}
		err := httpsrv.Serve(httplis)
		if err != cmux.ErrListenerClosed {
			errc <- err
		} else {
			errc <- nil
		}
	}()

	// Start multiplexer
	go func() { errc <- tcpmux.Serve() }()

	// Wait for Ctrl+C and other signals
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		errc <- nil
	}()

	// Log all settings for debugging purposes
	logger.Log(
		"msg", "Server started",
		"addr", *addr,
		"disco", *disco,
		"tls", *tls,
		"certFile", *certFile,
		"keyFile", *keyFile,
		"qps", *qps,
		"burst", *burst,
	)
	defer logger.Log("msg", "Server stopped")

	// Wait for completion
	if err := <-errc; err != nil {
		logger.Log("msg", "Exit with failure", "err", err)
	}
}
