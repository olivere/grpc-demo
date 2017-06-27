package main

import (
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	etcdnaming "github.com/coreos/etcd/clientv3/naming"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/olivere/grpc/lb/healthz"
	"github.com/olivere/grpc/lb/static"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/naming"

	pb "github.com/olivere/grpc-demo/pb"
)

type Client struct {
	conn *grpc.ClientConn
	c    pb.ExampleClient

	addr         string
	healthchecks []string
	tls          bool
	serverName   string
	caFile       string
	limiter      *rate.Limiter
	maxRetries   uint
	etcdcli      *clientv3.Client
	balancer     grpc.Balancer
}

type ClientOption func(*Client)

func NewClient(options ...ClientOption) (*Client, error) {
	client := &Client{
		addr:         "localhost:10000",
		healthchecks: nil,
		tls:          false,
		serverName:   "",
		caFile:       "",
		limiter:      rate.NewLimiter(rate.Limit(1000), 10),
		maxRetries:   5,
		etcdcli:      nil,
	}
	for _, option := range options {
		option(client)
	}

	var opts []grpc.DialOption

	// Configure TLS
	if client.tls {
		cert, err := ioutil.ReadFile(client.caFile)
		if err != nil {
			return nil, errors.Wrap(err, "cannot read caFile")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(cert) {
			return nil, errors.New("failed to append certificate to pool")
		}
		var sn string
		if client.serverName != "" {
			sn = client.serverName
		} else {
			sn, _, err = net.SplitHostPort(client.addr)
			if err != nil {
				return nil, errors.Wrap(err, "cannot split address into host and port")
			}
		}
		creds := credentials.NewClientTLSFromCert(pool, sn)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// Retries
	retrycallopts := []grpcretry.CallOption{
		grpcretry.WithMax(client.maxRetries),
		grpcretry.WithCodes(codes.Unavailable, codes.ResourceExhausted),
	}
	opts = append(opts, grpc.WithStreamInterceptor(grpcretry.StreamClientInterceptor(retrycallopts...)))
	opts = append(opts, grpc.WithUnaryInterceptor(grpcretry.UnaryClientInterceptor(retrycallopts...)))

	// Monitoring via Prometheus
	opts = append(opts, grpc.WithUnaryInterceptor(grpcprom.UnaryClientInterceptor))
	opts = append(opts, grpc.WithStreamInterceptor(grpcprom.StreamClientInterceptor))

	// Load balancing and service discovery
	var err error
	var conn *grpc.ClientConn
	if client.etcdcli != nil {
		// Service name is "grpc-demo-example"... hard-coded. It must match the service-side.
		resolver := &etcdnaming.GRPCResolver{Client: client.etcdcli}
		balancer := grpc.RoundRobin(resolver)
		opts = append(opts, grpc.WithBalancer(balancer))
		// Block until we connect is necessary for etcd
		opts = append(opts, grpc.WithBlock()) // see https://github.com/coreos/etcd/issues/7821
		opts = append(opts, grpc.WithTimeout(10*time.Second))
		conn, err = grpc.Dial("grpc-demo-example", opts...)
		if err != nil {
			return nil, errors.Wrap(err, "cannot connect to etcd service")
		}
	} else if client.balancer != nil {
		opts = append(opts, grpc.WithBalancer(client.balancer))
		conn, err = grpc.Dial(client.addr, opts...)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot connect to %s", client.addr)
		}
	} else {
		addr := client.addr

		if len(client.healthchecks) > 0 {
			// Static resolver with health checks
			r, err := client.healthzResolver()
			if err != nil {
				return nil, err
			}
			b := grpc.RoundRobin(r)
			opts = append(opts, grpc.WithBalancer(b))
		} else {
			// Static resolver
			addrs := strings.Split(client.addr, ",")
			if len(addrs) > 1 {
				addr = ""
				r := static.NewResolver(addrs...)
				b := grpc.RoundRobin(r)
				opts = append(opts, grpc.WithBalancer(b))
			}
		}

		// Connect
		conn, err = grpc.Dial(addr, opts...)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot connect to %s", client.addr)
		}
	}
	client.conn = conn

	client.c = pb.NewExampleClient(client.conn)

	return client, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) healthzResolver() (naming.Resolver, error) {
	if len(c.healthchecks) == 0 {
		return nil, errors.New("no healthcheck URLs specified")
	}
	addrs := strings.Split(c.addr, ",")
	if want, have := len(addrs), len(c.healthchecks); want != have {
		return nil, errors.Errorf("there must be a healthcheck URL for every gRPC endpoint; "+
			"you passed %d gRPC endpoints but have %d healthcheck URLs", want, have)
	}

	var endpoints []healthz.Endpoint
	for i, addr := range addrs {
		healthcheckURL := c.healthchecks[i]
		if _, err := url.Parse(healthcheckURL); err != nil {
			return nil, errors.Wrapf(err, "invalid URL: %s", healthcheckURL)
		}
		endpoints = append(endpoints, healthz.Endpoint{
			Addr:     addr,
			CheckURL: healthcheckURL,
		})
	}

	r, err := healthz.NewResolver(healthz.SetEndpoints(endpoints...))
	if err != nil {
		return nil, errors.Wrap(err, "error creating healthz resolver")
	}
	return r, nil
}

func SetAddr(addr string) ClientOption {
	return func(client *Client) {
		client.addr = addr
	}
}

func SetHealthcheckURL(urls ...string) ClientOption {
	return func(client *Client) {
		client.healthchecks = urls
	}
}

func SetTLS(tls bool) ClientOption {
	return func(client *Client) {
		client.tls = tls
	}
}

func SetServerName(serverName string) ClientOption {
	return func(client *Client) {
		client.serverName = serverName
	}
}

func SetCAFile(caFile string) ClientOption {
	return func(client *Client) {
		client.caFile = caFile
	}
}

func SetRateLimiter(limiter *rate.Limiter) ClientOption {
	return func(client *Client) {
		client.limiter = limiter
	}
}

func SetMaxRetries(maxRetries uint) ClientOption {
	return func(client *Client) {
		client.maxRetries = maxRetries
	}
}

func SetBalancer(balancer grpc.Balancer) ClientOption {
	return func(client *Client) {
		client.balancer = balancer
	}
}

// SetEtcdClient sets the etcd client to use for service discovery.
// If it is non-nil, it means we use etcd.
func SetEtcdClient(etcdcli *clientv3.Client) ClientOption {
	return func(client *Client) {
		client.etcdcli = etcdcli
	}
}

// -- Client functions --

func (c *Client) Hello(ctx context.Context, in *pb.HelloRequest, opts ...grpc.CallOption) (*pb.HelloResponse, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.c.Hello(ctx, in, opts...)
}

func (c *Client) Ticker(ctx context.Context, in *pb.TickerRequest, opts ...grpc.CallOption) (pb.Example_TickerClient, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.c.Ticker(ctx, in, opts...)
}
