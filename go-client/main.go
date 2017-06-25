package main

//go:generate protoc -I ../pb/ ../pb/example.proto --go_out=plugins=grpc:../pb

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

var (
	modeCommand = make(map[string]Command)
	modeFlags   = make(map[string]*flag.FlagSet)
)

// ErrUsage is returned when an unknown command is called.
var ErrUsage = UsageError("invalid command")

// UsageError is used to indicate a problem with invoking the executable,
// e.g. invalid parameters.
type UsageError string

func (e UsageError) Error() string {
	return fmt.Sprintf("Usage error: %s", string(e))
}

// Exit will exit the program and set the given exit code.
func Exit(code int) {
	os.Exit(code)
}

// Command represents a registered installment of the program.
type Command interface {
	Usage()
	Run(args []string) error
}

// describer can be implemented by commands to print a description.
type describer interface {
	Describe() string
}

// exampler can be implemented by commands to print an example call.
type exampler interface {
	Examples() []string
}

// RegisterCommand registers a command to run for a given mode.
func RegisterCommand(mode string, makeCmd func(Flags *flag.FlagSet) Command) {
	if _, dup := modeCommand[mode]; dup {
		log.Fatalf("duplicate command %q registered", mode)
	}
	flags := flag.NewFlagSet(mode+"options", flag.ContinueOnError)
	flags.Usage = func() {}

	modeFlags[mode] = flags
	modeCommand[mode] = makeCmd(flags)
}

// Errorf prints to os.Stderr.
func Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func hasFlags(flags *flag.FlagSet) bool {
	any := false
	flags.VisitAll(func(*flag.Flag) {
		any = true
	})
	return any
}

func usage(msg string) {
	cmdName := filepath.Base(os.Args[0])
	if msg != "" {
		Errorf("Error: %v\n", msg)
	}
	Errorf(`
Usage: ` + cmdName + ` <mode> [commandopts] [commandargs]

Modes:

`)

	var modes []string
	for mode, cmd := range modeCommand {
		if _, ok := cmd.(describer); ok {
			modes = append(modes, mode)
		}
	}
	sort.Strings(modes)
	for _, mode := range modes {
		cmd, _ := modeCommand[mode]
		if des, ok := cmd.(describer); ok {
			Errorf("  %-25s %s\n", mode, des.Describe())
		}
	}

	Errorf("\nExamples:\n")
	for mode, cmd := range modeCommand {
		if ex, ok := cmd.(exampler); ok {
			exs := ex.Examples()
			if len(exs) > 0 {
				Errorf("\n")
			}
			for _, example := range exs {
				Errorf("  %s %s %s\n", cmdName, mode, example)
			}
		}
	}

	Errorf(`
For mode-specific help:

  ` + cmdName + ` help <mode>
`)
	//flag.PrintDefaults()
	Exit(1)
}

func help(mode string) {
	cmdName := os.Args[0]
	cmd, ok := modeCommand[mode]
	if !ok {
		usage(fmt.Sprintf("Unknown mode %q", mode))
	}
	cmdFlags := modeFlags[mode]
	cmdFlags.SetOutput(os.Stderr)
	if des, ok := cmd.(describer); ok {
		Errorf("%s\n", des.Describe())
	}
	Errorf("\n")
	cmd.Usage()
	if hasFlags(cmdFlags) {
		cmdFlags.PrintDefaults()
	}
	if ex, ok := cmd.(exampler); ok {
		Errorf("\nExamples:\n")
		for _, example := range ex.Examples() {
			Errorf("  %s %s %s\n", cmdName, mode, example)
		}
	}
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage("No mode given.")
	}

	mode := args[0]
	cmd, ok := modeCommand[mode]
	if !ok {
		usage(fmt.Sprintf("Unknown mode %q", mode))
	}

	cmdFlags := modeFlags[mode]
	cmdFlags.SetOutput(os.Stderr)
	err := cmdFlags.Parse(args[1:])
	if err != nil {
		err = ErrUsage
	} else {
		err = cmd.Run(cmdFlags.Args())
	}
	if e, isUsageErr := err.(UsageError); isUsageErr {
		Errorf("%s\n", e)
		cmd.Usage()
		Errorf("\nGlobal options:\n")
		flag.PrintDefaults()

		if hasFlags(cmdFlags) {
			Errorf("\nMode-specific options for mode %q:\n", mode)
			cmdFlags.PrintDefaults()
		}
		Exit(1)
	}

	if err != nil {
		Errorf("Error: %v\n", err)
		Exit(2)
	}
}
