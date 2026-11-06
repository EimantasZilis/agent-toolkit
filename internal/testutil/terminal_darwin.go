package testutil

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openTerminal() (*os.File, *os.File, error) {
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, err
	}
	failed := true
	defer func() {
		if failed {
			_ = unix.Close(fd)
		}
	}()
	for _, request := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
		if err := unix.IoctlSetInt(fd, request, 0); err != nil {
			return nil, nil, fmt.Errorf("initialize PTY (ioctl %#x): %w", request, err)
		}
	}
	var name [128]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0]))); errno != 0 {
		return nil, nil, fmt.Errorf("get PTY slave name: %w", errno)
	}
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	if n == len(name) {
		return nil, nil, fmt.Errorf("PTY slave name is not NUL-terminated")
	}

	slave, err := os.OpenFile(string(name[:n]), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open PTY slave: %w", err)
	}
	// Darwin accepts the window-size ioctl on the opened slave terminal.
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 40, Col: 120}); err != nil {
		_ = slave.Close() // Best effort while preserving the ioctl error.
		return nil, nil, fmt.Errorf("size PTY slave: %w", err)
	}

	failed = false
	return os.NewFile(uintptr(fd), "pty-master"), slave, nil
}
