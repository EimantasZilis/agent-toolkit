// Command agent-toolkit runs the managed installation TUI from a source clone.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/EimantasZilis/agent-toolkit/internal/tui"
)

func main() {
	root := flag.String("root", ".", "toolkit source repository")
	version := flag.Bool("version", false, "print application and backend protocol versions")
	flag.Parse()
	if *version {
		fmt.Println("agent-toolkit development; managed protocol 1")
		return
	}
	if err := tui.Run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
