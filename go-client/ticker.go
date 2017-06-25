package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/olivere/grpc-demo/pb"
)

// tickerCommand executes the streaming Ticker RPC.
type tickerCommand struct {
	addr       string
	tls        bool
	serverName string
	caFile     string
	interval   time.Duration
}

func init() {
	RegisterCommand("ticker", func(flags *flag.FlagSet) Command {
		cmd := new(tickerCommand)
		flags.StringVar(&cmd.addr, "addr", ":10000", "Server address")
		flags.BoolVar(&cmd.tls, "tls", false, "Enable TLS")
		flags.StringVar(&cmd.serverName, "serverName", "", "Server to check the certificate")
		flags.StringVar(&cmd.caFile, "caFile", "", "Certificate file in e.g. PEM format")
		flags.DurationVar(&cmd.interval, "interval", 1*time.Second, "Time interval between ticker responses")
		return cmd
	})
}

func (cmd *tickerCommand) Describe() string {
	return "Run the streaming Ticker RPC call."
}

func (cmd *tickerCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s ticker [-addr=...] [-tls=...] [-serverName=...] [-cert=...] [-key=...] [--ticker=...]\n", os.Args[0])
}

func (cmd *tickerCommand) Examples() []string {
	return []string{
		fmt.Sprintf("%s ticker -addr=localhost:10000 -interval=5s", os.Args[0]),
	}
}

func (cmd *tickerCommand) Run(args []string) error {
	var opts []grpc.DialOption
	if cmd.tls {
		var sn string
		if cmd.serverName != "" {
			sn = cmd.serverName
		}
		var creds credentials.TransportCredentials
		if cmd.caFile != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(cmd.caFile, sn)
			if err != nil {
				return errors.Wrap(err, "cannot read TLS credentials")
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, sn)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cmd.addr, opts...)
	if err != nil {
		return errors.Wrapf(err, "cannot connect to %s", cmd.addr)
	}
	defer conn.Close()

	client := pb.NewExampleClient(conn)

	ctx := context.Background()
	req := &pb.TickerRequest{
		Interval: cmd.interval.Nanoseconds(),
	}
	stream, err := client.Ticker(ctx, req)
	if err != nil {
		return errors.Wrap(err, "cannot retrieve stream")
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "stream failed")
		}
		fmt.Println(res.Tick)
	}
	return nil
}
