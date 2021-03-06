package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	pb "github.com/olivere/grpc-demo/pb"
)

// tickerCommand executes the streaming Ticker RPC.
type tickerCommand struct {
	disco       string
	addr        string
	healthcheck string
	tls         bool
	serverName  string
	caFile      string
	interval    time.Duration
	timezone    string
	qps         float64
	burst       int
	maxRetries  uint
	parallel    int
	forever     time.Duration
}

func init() {
	RegisterCommand("ticker", func(flags *flag.FlagSet) Command {
		cmd := new(tickerCommand)
		flags.StringVar(&cmd.disco, "disco", envString("DISCO", ""), "Service discovery mechanism (blank or etcd)")
		flags.StringVar(&cmd.addr, "addr", ":10000", "Server address")
		flags.StringVar(&cmd.healthcheck, "healthcheck", "", "Comma-separated list of healthchecks for each gRPC endpoint")
		flags.BoolVar(&cmd.tls, "tls", false, "Enable TLS")
		flags.StringVar(&cmd.serverName, "serverName", "", "Server to check the certificate")
		flags.StringVar(&cmd.caFile, "caFile", "", "Certificate file in e.g. PEM format")
		flags.DurationVar(&cmd.interval, "interval", 1*time.Second, "Time interval between ticker responses")
		flags.StringVar(&cmd.timezone, "tz", time.Local.String(), "Timezone to pass to ticker")
		flags.Float64Var(&cmd.qps, "qps", 0.0, "Rate limit for queries of seconds")
		flags.IntVar(&cmd.burst, "burst", 0, "Rate limiter bursts")
		flags.UintVar(&cmd.maxRetries, "retries", 5, "Number of retries when hitting rate limits")
		flags.IntVar(&cmd.parallel, "parallel", 1, "Number of requests to send in parallel (e.g. to test rate limiting)")
		flags.DurationVar(&cmd.forever, "t", -1, "Repeat the requests forever")
		return cmd
	})
}

func (cmd *tickerCommand) Describe() string {
	return "Run the streaming Ticker RPC call."
}

func (cmd *tickerCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s ticker [-disco=...] [-addr=...] [-tls=...] [-serverName=...] [-cert=...] [-key=...] [--ticker=...]\n", os.Args[0])
}

func (cmd *tickerCommand) Examples() []string {
	return []string{
		fmt.Sprintf("%s ticker -addr=localhost:10000 -interval=5s", os.Args[0]),
		fmt.Sprintf("%s ticker -interval=5s -tz=Europe/London", os.Args[0]),
		fmt.Sprintf("%s ticker -disco=etcd", os.Args[0]),
	}
}

func (cmd *tickerCommand) Run(args []string) error {
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

		g, ctx := errgroup.WithContext(ctx)

		for i := 0; i < cmd.parallel; i++ {
			g.Go(func() error {
				req := &pb.TickerRequest{
					Timezone: cmd.timezone,
					Interval: cmd.interval.Nanoseconds(),
				}
				stream, err := client.Ticker(ctx, req)
				if err != nil {
					return errors.Wrap(err, "initiate stream")
				}
				for {
					res, err := stream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						return errors.Wrap(err, "unexpected stream error")
					}
					fmt.Println(res.Tick)
				}
				return nil
			})
		}

		err := g.Wait()
		if cmd.forever.Seconds() > 0 {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			time.Sleep(cmd.forever)
			continue
		}
		return err
	}
}
