package main

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCommandProcess(t *testing.T) {
	if os.Getenv("AGENT_TOOLKIT_TEST_COMMAND") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"agent-toolkit"}, os.Args[i+1:]...)
			break
		}
	}
	flag.CommandLine = flag.NewFlagSet("agent-toolkit", flag.ExitOnError)
	main()
	os.Exit(0)
}

func TestSourceCommandWithoutTerminal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    string
		success bool
	}{
		{"version", []string{"--version"}, "agent-toolkit development", true},
		{"startup", nil, "requires an interactive terminal", false},
		{"unknown", []string{"configure"}, "unexpected arguments", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			args := append([]string{"-test.run=TestCommandProcess", "--", "--root", root}, tc.args...)
			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = append(os.Environ(), "AGENT_TOOLKIT_TEST_COMMAND=1")
			data, err := cmd.CombinedOutput()
			if (err == nil) != tc.success || !strings.Contains(string(data), tc.want) {
				t.Fatalf("command: %s %v", data, err)
			}
			if strings.Contains(string(data), "protocol") {
				t.Fatal("obsolete protocol reporting")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("command wrote before startup: %v %v", entries, err)
			}
		})
	}
}
