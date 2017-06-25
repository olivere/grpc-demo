package main

import (
	"fmt"

	"github.com/go-kit/kit/log"
)

// grpcLogger wraps log messages from gRPC and redirects them to logger.
type grpcLogger struct {
	log.Logger
}

// Print wraps a logging message from gRPC and redirects it into Go-kit's logger.
func (l grpcLogger) Print(v ...interface{}) {
	l.Logger.Log("component", "grpc", "msg", fmt.Sprintf("%v", v...))
}

// Printf wraps a logging message from gRPC and redirects it into Go-kit's logger.
func (l grpcLogger) Printf(format string, v ...interface{}) {
	l.Logger.Log("component", "grpc", "msg", fmt.Sprintf(format, v...))
}

// Println wraps a logging message from gRPC and redirects it into Go-kit's logger.
func (l grpcLogger) Println(v ...interface{}) {
	l.Logger.Log("component", "grpc", "msg", fmt.Sprintf("%v", v...))
}

// Fatal wraps a logging message from gRPC and redirects it into Go-kit's logger.
func (l grpcLogger) Fatal(v ...interface{}) {
	l.Logger.Log("component", "grpc", "err", fmt.Sprintf("%v", v...))
}

// Fatalf wraps a logging message from gRPC and redirects it into Go-kit's logger.
func (l grpcLogger) Fatalf(format string, v ...interface{}) {
	l.Logger.Log("component", "grpc", "err", fmt.Sprintf(format, v...))
}

// Fatalln wraps a logging message from gRPC and redirects it into Go-kit's logger.
func (l grpcLogger) Fatalln(v ...interface{}) {
	l.Logger.Log("component", "grpc", "err", fmt.Sprintf("%v", v...))
}
