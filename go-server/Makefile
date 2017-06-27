PKG=github.com/olivere/grpc-demo/go-server

default: build

build:
	go build

generate:
	go generate

start:
	./go-server -addr=localhost:10000

startall:
	./go-server -addr=localhost:10000 >& log/server1.log &
	./go-server -addr=localhost:10001 >& log/server2.log &
	./go-server -addr=localhost:10002 >& log/server3.log &

startalltls:
	./go-server -addr=grpc-demo.go:10000 -tls -cert=../etc/grpc-demo.go.pem -key=../etc/grpc-demo.go.key >& log/server1.log &
	./go-server -addr=grpc-demo.go:10001 -tls -cert=../etc/grpc-demo.go.pem -key=../etc/grpc-demo.go.key >& log/server2.log &
	./go-server -addr=grpc-demo.go:10002 -tls -cert=../etc/grpc-demo.go.pem -key=../etc/grpc-demo.go.key >& log/server3.log &

killall:
	killall go-server
