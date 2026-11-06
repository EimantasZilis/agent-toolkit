package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func previewFor(t *testing.T, s *Service, request Request) Preview {
	t.Helper()
	p, err := s.Preview(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", p.Conflicts)
	}
	return p
}

func applyFor(t *testing.T, s *Service, p Preview) Status {
	t.Helper()
	status, err := s.Apply(context.Background(), p, Confirmations{Removals: true})
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func TestApplyIsStableAndUninstallPreservesUserContent(t *testing.T) {
	s := testService(t)
	_, guide := s.destinations("codex")
	write(t, guide, "Hello\n")
	settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"ticket-to-plan", "python-standards"}, Concise: true, FullTemplate: true}
	request := Request{Intent: Configure, Desired: settings}
	status := applyFor(t, s, previewFor(t, s, request))
	if !status.Installed || !contains(status.Observed["codex"], "plan-review") || !contains(status.Observed["claude"], "python-standards") {
		t.Fatal(status)
	}
	before := status.Revision
	status = applyFor(t, s, previewFor(t, s, request))
	if before != status.Revision {
		t.Fatal("repeated apply changed installation")
	}
	p := previewFor(t, s, Request{Intent: Refresh})
	for _, op := range p.Operations {
		if op.Action != "keep" {
			t.Fatalf("unchanged refresh = %+v", op)
		}
	}
	root, _ := s.destinations("codex")
	write(t, filepath.Join(root, "user-skill/SKILL.md"), "User skill\n")
	status = applyFor(t, s, previewFor(t, s, Request{Intent: Uninstall}))
	if status.Installed {
		t.Fatal("uninstall recreated state")
	}
	data, err := os.ReadFile(guide)
	if err != nil || strings.TrimSpace(string(data)) != "Hello" {
		t.Fatalf("user guidance changed: %s %v", data, err)
	}
	if !exists(filepath.Join(root, "user-skill/SKILL.md")) {
		t.Fatal("removed user skill")
	}
	if exists(filepath.Join(s.Root, "config/settings.env")) || exists(filepath.Join(s.Root, "config/inventory.json")) {
		t.Fatal("uninstall retained saved state")
	}
}

func TestStaleForgedAndUnconfirmedApply(t *testing.T) {
	for _, change := range []string{"source", "destination", "forged", "unconfirmed", "unsafe-source"} {
		t.Run(change, func(t *testing.T) {
			s := testService(t)
			intent := Configure
			if change == "unconfirmed" {
				intent = Uninstall
			}
			p := previewFor(t, s, Request{Intent: intent, Desired: Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}})
			switch change {
			case "source":
				write(t, filepath.Join(s.Root, "internal/new.go"), "changed")
			case "destination":
				root, _ := s.destinations("codex")
				write(t, filepath.Join(root, "user-file"), "user")
			case "forged":
				p.Operations[0].Path = filepath.Join(t.TempDir(), "forged")
			case "unsafe-source":
				if err := os.Symlink("/missing", filepath.Join(s.Root, ".agents/unsafe")); err != nil {
					t.Fatal(err)
				}
				p = previewFor(t, s, p.Request)
			}
			if _, err := s.Apply(context.Background(), p, Confirmations{}); err == nil {
				t.Fatal("accepted invalid apply")
			}
			if exists(filepath.Join(s.Root, "config/settings.env")) {
				t.Fatal("rejected apply wrote state")
			}
		})
	}
}

func TestTransactionRollbackAcrossProvidersAndState(t *testing.T) {
	for _, point := range []string{"after-provider:codex", "after-provider:claude", "after-settings", "after-inventory"} {
		t.Run(point, func(t *testing.T) {
			s := testService(t)
			base := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"commit-message"}, Concise: true}
			applyFor(t, s, previewFor(t, s, Request{Intent: Configure, Desired: base}))
			before, err := s.revision()
			if err != nil {
				t.Fatal(err)
			}
			base.Requested = []string{"python-standards"}
			base.FullTemplate = true
			base.Output = filepath.Join(t.TempDir(), "new output/deep")
			p := previewFor(t, s, Request{Intent: Configure, Desired: base})
			s.fail = func(at string) error {
				if at == point {
					return errors.New("injected failure")
				}
				return nil
			}
			if _, err := s.Apply(context.Background(), p, Confirmations{Removals: true, CreateOutput: true}); err == nil {
				t.Fatal("expected failure")
			}
			after, err := s.revision()
			if err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatal("rollback changed managed data")
			}
			if exists(base.Output) {
				t.Fatal("rollback retained new empty output folder")
			}
			if exists(filepath.Join(s.Root, "config/apply.lock")) {
				t.Fatal("lock survived failure")
			}
		})
	}
}

func TestCancellationRollsBack(t *testing.T) {
	s := testService(t)
	p := previewFor(t, s, Request{Intent: Configure, Desired: Settings{Providers: []string{"codex", "claude"}, Requested: []string{"commit-message"}}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.fail = func(at string) error {
		if at == "after-provider:codex" {
			cancel()
		}
		return nil
	}
	if _, err := s.Apply(ctx, p, Confirmations{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
	root, _ := s.destinations("codex")
	if exists(filepath.Join(root, "commit-message")) || exists(filepath.Join(s.Root, "config/settings.env")) {
		t.Fatal("canceled apply survived")
	}
}

func TestConcurrentApplyLocksBeforeMutation(t *testing.T) {
	s := testService(t)
	p := previewFor(t, s, Request{Intent: Configure, Desired: Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}})
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	s.fail = func(at string) error {
		if at == "after-provider:codex" {
			close(entered)
			<-release
		}
		return nil
	}
	go func() { _, err := s.Apply(context.Background(), p, Confirmations{}); done <- err }()
	<-entered
	other, err := New(s.Root, s.Homes)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	_, err = other.Apply(context.Background(), p, Confirmations{})
	close(release)
	var coded *Error
	if !errors.As(err, &coded) || coded.Code != "locked" {
		t.Errorf("concurrent apply = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func recoveryRoot(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("missing failure")
	}
	_, tail, ok := strings.Cut(err.Error(), "recovery backups retained at ")
	if !ok {
		t.Fatalf("missing recovery diagnostic: %v", err)
	}
	root, _, _ := strings.Cut(tail, ": ")
	if !strings.HasPrefix(root, filepath.Join(os.TempDir(), "agent-toolkit-install-")) {
		t.Fatalf("unexpected recovery path: %s", root)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	return root
}

func TestPopulatedOutputIsPreservedOnRollback(t *testing.T) {
	s := testService(t)
	output := filepath.Join(t.TempDir(), "output")
	p := previewFor(t, s, Request{Intent: Configure, Desired: Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}, Output: output}})
	s.fail = func(at string) error {
		if at == "after-provider:codex" {
			if err := os.WriteFile(filepath.Join(output, "user-artifact"), []byte("keep"), 0644); err != nil {
				return err
			}
			return errors.New("injected failure")
		}
		return nil
	}
	_, err := s.Apply(context.Background(), p, Confirmations{CreateOutput: true})
	recoveryRoot(t, err)
	if !exists(filepath.Join(output, "user-artifact")) {
		t.Fatal("rollback removed a user artifact")
	}
}

func TestFailedRecoveryRetainsOriginalBackups(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}
	applyFor(t, s, previewFor(t, s, Request{Intent: Configure, Desired: settings}))
	before, err := os.ReadFile(filepath.Join(s.Root, "config/settings.env"))
	if err != nil {
		t.Fatal(err)
	}
	settings.Requested = []string{"pr-comment"}
	p := previewFor(t, s, Request{Intent: Configure, Desired: settings})
	s.fail = func(at string) error {
		if at == "after-settings" || at == "rollback:"+filepath.Join(s.Root, "config/settings.env") {
			return errors.New("injected failure")
		}
		return nil
	}
	_, err = s.Apply(context.Background(), p, Confirmations{Removals: true})
	root := recoveryRoot(t, err)
	data, err := os.ReadFile(filepath.Join(root, "0"))
	if err != nil || string(data) != string(before) {
		t.Fatalf("lost original settings backup: %s %v", data, err)
	}
}

func TestStateEncodingKeepsLegacyProviderRows(t *testing.T) {
	data, err := inventoryBytes([]byte("settings"), map[string][]string{"codex": {"one", "two"}, "claude": {}})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []string{`  "schema_version": 1,`, `    "codex": ["one", "two"]`, `    "claude": []`} {
		if !strings.Contains(string(data), row) {
			t.Fatalf("legacy row missing %q: %s", row, data)
		}
	}
}
