package main

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

func getUser(ctx context.Context) (string, bool) {
	return getUserFromMetadata(ctx)
}

func getUserFromMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md["user"]
	if len(values) != 1 || values[0] == "" {
		return "", false
	}
	return values[0], true
}

func getUserFromContext(ctx context.Context) (string, bool) {
	if user, ok := ctx.Value("user").(string); ok && user != "" {
		return user, true
	}
	return "", false
}
