// Command agent-toolkit runs the managed installation TUI from a source clone.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/EimantasZilis/agent-toolkit/internal/tui"
)

var version = "development"

func main() {
	root := flag.String("root", ".", "toolkit source repository")
	showVersion := flag.Bool("version", false, "print application version")
	flag.Parse()
	if flag.NArg() > 0 && flag.Arg(0) == "native" {
		if err := runNative(*root, flag.Args()[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments; run without a subcommand to open the TUI")
		os.Exit(2)
	}
	if *showVersion {
		fmt.Printf("agent-toolkit %s\n", version)
		return
	}
	if err := tui.Run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
