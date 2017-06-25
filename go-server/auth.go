package main

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type contextKey uint

const (
	userKey contextKey = iota
)

// authenticate takes the user from the gRPC metadata and
// adds it into the context values, if available. Otherwise
// an error with gRPC code Unauthenticated is returned.
func authenticate(ctx context.Context) (context.Context, error) {
	user, ok := extractUserFromMD(ctx)
	if !ok {
		return ctx, grpc.Errorf(codes.Unauthenticated, "request is not authenticated")
	}
	return context.WithValue(ctx, userKey, user), nil
}

// getUser returns the user previously added via authenticate.
func getUser(ctx context.Context) (string, bool) {
	if user, ok := ctx.Value(userKey).(string); ok && user != "" {
		return user, true
	}
	return "", false
}

// extractUserFromMD extracts the user from gRPC metadata.
func extractUserFromMD(ctx context.Context) (string, bool) {
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
