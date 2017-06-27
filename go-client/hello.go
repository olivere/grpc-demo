package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"strings"

	"github.com/coreos/etcd/clientv3"
	pb "github.com/olivere/grpc-demo/pb"
)

// helloCommand executes the Hello RPC.
type helloCommand struct {
	disco       string
	addr        string
	healthcheck string
	tls         bool
	serverName  string
	caFile      string
	timeout     time.Duration
	qps         float64
	burst       int
	maxRetries  uint
	parallel    int
	forever     time.Duration
}

func init() {
	RegisterCommand("hello", func(flags *flag.FlagSet) Command {
		cmd := new(helloCommand)
		flags.StringVar(&cmd.disco, "disco", envString("DISCO", ""), "Service discovery mechanism (blank or etcd)")
		flags.StringVar(&cmd.addr, "addr", ":10000", "Host and port to bind to")
		flags.StringVar(&cmd.healthcheck, "healthcheck", "", "Comma-separated list of healthchecks for each gRPC endpoint")
		flags.BoolVar(&cmd.tls, "tls", false, "Enable TLS")
		flags.StringVar(&cmd.serverName, "serverName", "", "Server to check the certificate")
		flags.StringVar(&cmd.caFile, "caFile", "", "Certificate file in e.g. PEM format")
		flags.DurationVar(&cmd.timeout, "timeout", 10*time.Second, "Timeout for call")
		flags.Float64Var(&cmd.qps, "qps", 0.0, "Rate limit for queries of seconds")
		flags.IntVar(&cmd.burst, "burst", 0, "Rate limiter bursts")
		flags.UintVar(&cmd.maxRetries, "retries", 5, "Number of retries when hitting rate limits")
		flags.IntVar(&cmd.parallel, "parallel", 1, "Number of requests to send in parallel (e.g. to test rate limiting)")
		flags.DurationVar(&cmd.forever, "t", -1, "Repeat the requests forever")
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
		fmt.Sprintf("%s hello -addr=localhost:10000,localhost:10001 -healthcheck=http://localhost:10000/healthz,http://localhost:10001/healthz", os.Args[0]),
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
	healhcheckURLs := strings.Split(cmd.healthcheck, ",")
	if len(healhcheckURLs) > 1 {
		options = append(options, SetHealthcheckURL(healhcheckURLs...))
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

	if cmd.parallel <= 0 {
		cmd.parallel = 1
	}

	for {
		ctx := context.Background()
		// Set user
		ctx = setUser(ctx, uuid.New().String())
		// Add timeout
		ctx, cancel := context.WithTimeout(ctx, cmd.timeout)
		defer cancel()

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
					return errors.Wrap(err, "cannot execute Hello request")
				}
				fmt.Println(res.Message)
				return nil
			})
		}

		err := g.Wait()
		if cmd.forever.Seconds() > 0 {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			cancel()
			time.Sleep(cmd.forever)
			continue
		}
		return err
	}
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
