package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/EimantasZilis/agent-toolkit/internal/native"
)

func runNative(root string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected native build or validate")
	}
	switch args[0] {
	case "build":
		flags := flag.NewFlagSet("native build", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		version := flags.String("version", "", "release version X.Y.Z")
		output := flags.String("output", "", "new destination")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *version == "" || *output == "" || flags.NArg() != 0 {
			return fmt.Errorf("native build requires --version V --output DIR")
		}
		return native.Build(root, *version, *output)
	case "validate":
		if len(args) != 2 {
			return fmt.Errorf("native validate requires DIR")
		}
		return native.Validate(args[1])
	default:
		return fmt.Errorf("unknown native command: %s", args[0])
	}
}
