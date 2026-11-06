package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func TestSignalWaitsForApplyResult(t *testing.T) {
	m, _ := readyModel(t, false)
	m.busy = true
	next, cmd := m.Update(shutdownMsg{})
	m = next.(model)
	if cmd != nil || !m.quitAfter {
		t.Fatal("signal interrupted apply")
	}
	next, cmd = m.Update(resultMsg{action: "apply", err: errors.New("recovery backups at /tmp/example")})
	m = next.(model)
	if cmd == nil || !strings.Contains(m.err, "recovery backups") {
		t.Fatal("signal lost completed apply diagnostics")
	}
}

type blockingBackend struct {
	fakeBackend
	entered, release chan struct{}
}

func (b *blockingBackend) Apply(context.Context, install.Preview, install.Confirmations) (install.Status, error) {
	close(b.entered)
	<-b.release
	return install.Status{}, nil
}

func TestShutdownTracksActiveTransaction(t *testing.T) {
	b := &blockingBackend{entered: make(chan struct{}), release: make(chan struct{})}
	tracked := &trackedBackend{backend: b}
	applied, closed := make(chan error, 1), make(chan struct{})
	go func() {
		_, err := tracked.Apply(context.Background(), install.Preview{}, install.Confirmations{})
		applied <- err
	}()
	<-b.entered
	go func() { tracked.close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("shutdown abandoned active apply")
	default:
	}
	close(b.release)
	if err := <-applied; err != nil {
		t.Fatal(err)
	}
	<-closed
	if _, err := tracked.Apply(context.Background(), install.Preview{}, install.Confirmations{}); err == nil {
		t.Fatal("apply began after shutdown")
	}
}
