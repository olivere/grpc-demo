package main

import (
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

func setUser(parent context.Context, user string) context.Context {
	return setUserInMetadata(parent, user)
}

func setUserInMetadata(parent context.Context, user string) context.Context {
	md := metadata.Pairs("user", uuid.New().String())
	return metadata.NewOutgoingContext(parent, md)
}

func setUserInContext(parent context.Context, user string) context.Context {
	return context.WithValue(parent, "user", user)
}
