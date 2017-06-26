package main

import (
	"flag"
	"fmt"
	"os"
)

// helpCommand prints help for a mode.
type helpCommand struct {
	debug bool
}

func init() {
	RegisterCommand("help", func(flags *flag.FlagSet) Command {
		cmd := new(helpCommand)
		return cmd
	})
}

func (c *helpCommand) Describe() string {
	return "Print help about a mode."
}

func (c *helpCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s help <mode>\n", os.Args[0])
}

func (c *helpCommand) Examples() []string {
	return []string{}
}

func (c *helpCommand) Run(args []string) error {
	if len(args) != 1 {
		return ErrUsage
	}

	help(args[0])

	return nil
}
