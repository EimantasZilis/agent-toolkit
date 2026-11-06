//go:build darwin || linux

package testutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
)

// RunTerminal runs a command in a 120x40 controlling terminal. Respond receives
// captured output and returns keystrokes. Cancellation kills the entire process
// group, closes the terminal, and joins both reader and waiter before returning.
func RunTerminal(ctx context.Context, cmd *exec.Cmd, respond func(string) string) (output string, err error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	master, slave, err := openTerminal()
	if err != nil {
		return "", err
	}
	defer master.Close()
	defer slave.Close()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	if err := slave.Close(); err != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return "", errors.Join(err, cmd.Wait())
	}

	events := make(chan string, 8)
	stop := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		defer close(events)
		buffer := make([]byte, 8192)
		for {
			n, err := master.Read(buffer)
			if n > 0 {
				select {
				case events <- string(buffer[:n]):
				case <-stop:
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	wait := make(chan error, 1)
	waiter := wait
	go func() { waiter <- cmd.Wait() }()
	waited := false
	var waitErr error
	defer func() {
		// Kill surviving descendants even if the direct child exited successfully.
		if killErr := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); killErr != nil && !errors.Is(killErr, syscall.ESRCH) {
			err = errors.Join(err, killErr)
		}
		close(stop)
		_ = master.Close() // Unblocks the owned reader; deferred close is best effort.
		if !waited {
			<-wait
		}
		<-readerDone
	}()

	for {
		select {
		case <-ctx.Done():
			return output, fmt.Errorf("terminal command: %w", ctx.Err())
		case part, ok := <-events:
			if !ok {
				events = nil
				if waited {
					return output, waitErr
				}
				continue
			}
			output += part
			if len(output) > 262144 {
				output = output[len(output)-262144:]
			}
			if waited {
				continue
			}
			if keys := respond(output); keys != "" {
				if _, err := master.Write([]byte(keys)); err != nil {
					return output, err
				}
			}
		case err := <-wait:
			waited = true
			waitErr = err
			wait = nil
			// Drain output before returning, including diagnostics from a fast
			// failure. Descendants cannot keep the slave terminal open forever.
			if killErr := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); killErr != nil && !errors.Is(killErr, syscall.ESRCH) {
				waitErr = errors.Join(waitErr, killErr)
			}
			if events == nil {
				return output, waitErr
			}
		}
	}
}
