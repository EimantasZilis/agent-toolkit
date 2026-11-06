//go:build darwin || linux

package testutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestTerminalChild(t *testing.T) {
	mode := os.Getenv("TOOLKIT_TERMINAL_CHILD")
	if mode == "" {
		return
	}
	size, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	if err != nil || size.Row != 40 || size.Col != 120 {
		fmt.Fprintln(os.Stderr, "wrong terminal size", size, err)
		os.Exit(1)
	}
	if mode == "immediate" {
		fmt.Fprintln(os.Stderr, "compiler diagnostic")
		os.Exit(2)
	}
	fmt.Println("terminal ready")
	var input [16]byte
	if _, err := os.Stdin.Read(input[:]); err != nil {
		os.Exit(1)
	}
	if mode == "success" {
		os.Exit(0)
	}
	os.Exit(2)
}

func terminalCommand(mode string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminalChild$")
	cmd.Env = Environment(map[string]string{"TOOLKIT_TERMINAL_CHILD": mode})
	return cmd
}

func TestTerminalStartupAndQuit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sent := false
	output, err := RunTerminal(ctx, terminalCommand("success"), func(output string) string {
		if !sent && strings.Contains(output, "terminal ready") {
			sent = true
			return "q\n"
		}
		return ""
	})
	if err != nil || !sent {
		t.Fatalf("terminal startup/quit: %v\n%s", err, output)
	}
}

func TestTerminalCancellationReapsProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := terminalCommand("wait")
	output, err := RunTerminal(ctx, cmd, func(output string) string {
		if strings.Contains(output, "terminal ready") {
			cancel()
		}
		return ""
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v\n%s", err, output)
	}
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() && cmd.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatal("child was not reaped after cancellation")
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("child survived cancellation: %v", err)
	}
}

func TestTerminalPreservesFailureDiagnostics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sent := false
	output, err := RunTerminal(ctx, terminalCommand("failure"), func(output string) string {
		if !sent && strings.Contains(output, "terminal ready") {
			sent = true
			return "q\n"
		}
		return ""
	})
	if err == nil || !strings.Contains(output, "terminal ready") {
		t.Fatalf("lost terminal failure: %v %q", err, output)
	}
}

func TestTerminalDrainsImmediateFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	output, err := RunTerminal(ctx, terminalCommand("immediate"), func(string) string { return "" })
	if err == nil || !strings.Contains(output, "compiler diagnostic") {
		t.Fatalf("lost immediate failure diagnostic: %v %q", err, output)
	}
}
