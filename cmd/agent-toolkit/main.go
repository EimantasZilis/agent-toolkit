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
	showVersion := flag.Bool("version", false, "print application and backend protocol versions")
	flag.Parse()
	if *showVersion {
		fmt.Printf("agent-toolkit %s; managed protocol 1\n", version)
		return
	}
	if err := tui.Run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
