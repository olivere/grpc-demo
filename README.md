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

On macOS with Homebrew:

```
$ brew install etcd
$ brew services start etcd
$ export ETCD_VERSION=3
$ ...
```

# License

MIT
