# gRPC Demo

This is a small experiment of working with gRPC, in various
languages, under different configurations etc.

# Configurations

## Monitoring with Prometheus

You can monitor the go-server with Prometheus. It pulls the
metrics from the `/metrics` endpoint of the server.

```
$ prometheus -config.file=etc/prometheus.yml
```

## Load balancing with etcd

When a server starts up, it registers itself with etcd.
To use this, you need to have etcd started.

On macOS with Homebrew. Also make sure to always use v3 of the API with `etcdctl`:

```
$ brew install etcd
$ brew services start etcd
$ export ETCDCTL_API=3
```

Then start e.g. two servers like this. Notice that you might need to increase
the rate limits by allowing a large number of qps and bursts, like 1000 qps and
20 bursts:

```
$ cd go-server
$ go build
$ ./go-server -disco=etcd -addr=:0 -qps=1000 -burst=20 >& server1.log &
$ head server1.log
@time=2017-06-26T10:15:33.665292995+02:00 caller=main.go:181 msg="Server started" addr=:56241 disco=etcd
$ ./go-server -disco=etcd -addr=:0 -qps=1000 -burst=20 >& server2.log &
$ head server2.log
@time=2017-06-26T10:15:47.55199946+02:00 caller=main.go:181 msg="Server started" addr=:56255 disco=etcd
```

Now, run a client and make usage of etcd for client-side load-balancing as well,
running a larger number of RPC requests in parallel:

```
$ cd go-client
$ go build
$ ./go-client hello -disco=etcd -parallel=50
```

Tail the server logs to see that both servers are requested in round-robin mode.

Watch how both servers are registered in etcd (the `grpc-demo-example` is the name
of the service--hardcoded in both client and server as of now):

```
$ etcdctl get --prefix grpc-demo-example
grpc-demo-example/:56449
{"Op":0,"Addr":":56449","Metadata":null}
grpc-demo-example/:56456
{"Op":0,"Addr":":56456","Metadata":null}
```

And if you sometimes need to remote those keys (or one of them) manually, just do:

```
$ etcdctl del grpc-demo-example/:56449
1
$ etcdctl del --prefix grpc-demo-example
1
```

### Cleanup etcd

```
$ killall go-server
$ killall go-client
$ etcdcli del --prefix grpc-demo-service
```


# License

MIT
