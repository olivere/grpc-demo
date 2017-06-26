package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/coreos/etcd/clientv3"
	pb "github.com/olivere/grpc-demo/pb"
)

// helloCommand executes the Hello RPC.
type helloCommand struct {
	disco      string
	addr       string
	tls        bool
	serverName string
	caFile     string
	timeout    time.Duration
	qps        float64
	burst      int
	maxRetries uint
	parallel   int
}

func init() {
	RegisterCommand("hello", func(flags *flag.FlagSet) Command {
		cmd := new(helloCommand)
		flags.StringVar(&cmd.disco, "disco", envString("DISCO", ""), "Service discovery mechanism (blank or etcd)")
		flags.StringVar(&cmd.addr, "addr", ":10000", "Host and port to bind to")
		flags.BoolVar(&cmd.tls, "tls", false, "Enable TLS")
		flags.StringVar(&cmd.serverName, "serverName", "", "Server to check the certificate")
		flags.StringVar(&cmd.caFile, "caFile", "", "Certificate file in e.g. PEM format")
		flags.DurationVar(&cmd.timeout, "timeout", 10*time.Second, "Timeout for call")
		flags.Float64Var(&cmd.qps, "qps", 0.0, "Rate limit for queries of seconds")
		flags.IntVar(&cmd.burst, "burst", 0, "Rate limiter bursts")
		flags.UintVar(&cmd.maxRetries, "retries", 5, "Number of retries when hitting rate limits")
		flags.IntVar(&cmd.parallel, "parallel", 1, "Number of requests to send in parallel (e.g. to test rate limiting)")
		return cmd
	})
}

func (cmd *helloCommand) Describe() string {
	return "Run the Hello RPC call."
}

func (cmd *helloCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s hello [-disco=...] [-addr=...] [-tls=...] [-serverName=...] [-cert=...] [-key=...]\n", os.Args[0])
}

func (cmd *helloCommand) Examples() []string {
	return []string{
		fmt.Sprintf("%s hello", os.Args[0]),
		fmt.Sprintf("%s hello -addr=localhost:10000", os.Args[0]),
		fmt.Sprintf("%s hello -disco=etcd", os.Args[0]),
	}
}

func (cmd *helloCommand) Run(args []string) error {
	options := []ClientOption{
		SetAddr(cmd.addr),
		SetTLS(cmd.tls),
		SetServerName(cmd.serverName),
		SetCAFile(cmd.caFile),
		SetMaxRetries(cmd.maxRetries),
	}
	if cmd.qps > 0 && cmd.burst > 0 {
		limiter := rate.NewLimiter(rate.Limit(cmd.qps), cmd.burst)
		options = append(options, SetRateLimiter(limiter))
	}
	switch cmd.disco {
	case "etcd":
		etcdcli, err := clientv3.NewFromURL("http://localhost:2379")
		if err != nil {
			return err
		}
		options = append(options, SetEtcdClient(etcdcli))
	}
	client, err := NewClient(options...)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	// Set user
	ctx = setUser(ctx, uuid.New().String())
	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, cmd.timeout)
	defer cancel()

	if cmd.parallel <= 0 {
		cmd.parallel = 1
	}

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < cmd.parallel; i++ {
		g.Go(func() error {
			req := &pb.HelloRequest{
				Name:   names[rand.Intn(len(names))],
				Age:    int32(20 + rand.Intn(20)),
				Nanos:  time.Now().UnixNano(),
				Gender: randomGender(),
			}
			res, err := client.Hello(ctx, req)
			if err != nil {
				return err
			}
			fmt.Println(res.Message)
			return nil
		})
	}

	return g.Wait()
}

func randomGender() pb.Gender {
	switch rand.Int() % 3 {
	case 0:
		return pb.Gender_MALE
	case 1:
		return pb.Gender_FEMALE
	default:
		return pb.Gender_UNSPECIFIED
	}
}

var (
	names = []string{
		"Oliver",
		"Sandra",
		"George",
		"Peter",
		"Holger",
		"Annika",
		"Fred",
		"John",
		"Olaf",
		"Ulrike",
		"Norbert",
		"Georgio",
		"Andreas",
		"Zoe",
		"Benny",
		"Charles",
		"Fanny",
		"Didier",
		"Shay",
		"Colin",
	}
)
