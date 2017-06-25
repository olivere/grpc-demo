package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/olivere/grpc-demo/pb"
)

// helloCommand executes the Hello RPC.
type helloCommand struct {
	addr       string
	tls        bool
	serverName string
	caFile     string
}

func init() {
	RegisterCommand("hello", func(flags *flag.FlagSet) Command {
		cmd := new(helloCommand)
		flags.StringVar(&cmd.addr, "addr", ":10000", "Host and port to bind to")
		flags.BoolVar(&cmd.tls, "tls", false, "Enable TLS")
		flags.StringVar(&cmd.serverName, "serverName", "", "Server to check the certificate")
		flags.StringVar(&cmd.caFile, "caFile", "", "Certificate file in e.g. PEM format")
		return cmd
	})
}

func (cmd *helloCommand) Describe() string {
	return "Run the Hello RPC call."
}

func (cmd *helloCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s hello [-addr=...] [-tls=...] [-serverName=...] [-cert=...] [-key=...]\n", os.Args[0])
}

/*
func (cmd *helloCommand) Examples() []string {
	return []string{}
}
*/

func (cmd *helloCommand) Run(args []string) error {
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
