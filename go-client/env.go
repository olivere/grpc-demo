package main

import "os"

func envString(name, defaults string) string {
	v := os.Getenv(name)
	if v != "" {
		return v
	}
	return defaults
}
