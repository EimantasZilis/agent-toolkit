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
	case "build", "snapshot":
		flags := flag.NewFlagSet("native "+args[0], flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		version := flags.String("version", "", "release version X.Y.Z")
		output := flags.String("output", "", "new destination")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *version == "" || *output == "" || flags.NArg() != 0 {
			return fmt.Errorf("native %s requires --version V --output DIR", args[0])
		}
		if args[0] == "snapshot" {
			return native.Snapshot(root, *version, *output)
		}
		return native.Build(root, *version, *output)
	case "validate", "validate-snapshot":
		if len(args) != 2 {
			return fmt.Errorf("native %s requires DIR", args[0])
		}
		if args[0] == "validate-snapshot" {
			return native.ValidateSnapshot(args[1])
		}
		return native.Validate(args[1])
	default:
		return fmt.Errorf("unknown native command: %s", args[0])
	}
}
