PKG=github.com/olivere/grpc-demo/go-client

default: build

build:
	go build

generate:
	go generate

starthello:
	./go-client hello -addr=127.0.0.1:10000 -t=1s

starthellotls:
	./go-client hello -addr=grpc-demo.go:10000 -tls -caFile=../etc/grpc-demo.go.pem -t=1s

starthellohealthz:
	./go-client hello \
		-addr=127.0.0.1:10000,127.0.0.1:10001,127.0.0.1:10002 \
		-healthcheck=http://127.0.0.1:10000/healthz,http://127.0.0.1:10001/healthz,http://127.0.0.1:10002/healthz \
		-t=1s

starthellohealthztls:
	./go-client hello \
		-addr=grpc-demo.go:10000,grpc-demo.go:10001,grpc-demo.go:10002 \
		-healthcheck=https://grpc-demo.go:10000/healthz,https://grpc-demo.go:10001/healthz,https://grpc-demo.go:10002/healthz \
		-tls \
		-caFile=../etc/grpc-demo.go.pem \
		-serverName=grpc-demo.go \
		-t=1s
