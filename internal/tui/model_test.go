package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeBackend struct {
	calls []string
	fail  string
}

func (b *fakeBackend) run(action string, args []string) (response, error) {
	b.calls = append(b.calls, action)
	if b.fail == action {
		return response{}, errors.New("stale_revision: preview again; recovery /tmp/backup")
	}
	r := response{Protocol: 1, OK: true, Kind: action, Revision: "revision"}
	if action == "apply" {
		r.Desired = settings{Providers: []string{"codex"}, Concise: true}
	}
	return r, nil
}

func readyModel(t *testing.T, wizard bool) (model, *fakeBackend) {
	t.Helper()
	b := &fakeBackend{}
	m := newModel(b, t.TempDir())
	m.loaded, m.wizard, m.page = true, wizard, 0
	m.saved = settings{Providers: []string{"codex"}}
	m.draft = clone(m.saved)
	return m, b
}

func press(m model, key string) (model, tea.Cmd) {
	next, cmd := m.key(key)
	return next.(model), cmd
}

func complete(m model, cmd tea.Cmd) (model, tea.Cmd) {
	next, nextCmd := m.Update(cmd())
	return next.(model), nextCmd
}

func TestWizardStagesAndAppliesOnce(t *testing.T) {
	m, b := readyModel(t, true)
	m.draft.Concise = true
	for i := 0; i < 4; i++ {
		var cmd tea.Cmd
		m, cmd = press(m, "tab")
		if cmd != nil || m.unsaved {
			t.Fatal("wizard navigation performed I/O or prompted")
		}
	}
	m, cmd := press(m, "s")
	if !m.busy || cmd == nil {
		t.Fatal("save did not start async preview")
	}
	if _, duplicate := press(m, "s"); duplicate != nil {
		t.Fatal("duplicate save started")
	}
	m, _ = complete(m, cmd)
	if len(m.dialogs) == 0 {
		t.Fatal("missing final review")
	}
	m, cmd = press(m, "y")
	if !m.busy {
		t.Fatal("apply is not asynchronous")
	}
	if _, quit := press(m, "ctrl+c"); quit != nil {
		t.Fatal("quit interrupted apply")
	}
	m, _ = complete(m, cmd)
	if m.wizard || m.page != -1 || m.dirty() {
		t.Fatalf("setup not saved: %+v", m)
	}
	if strings.Join(b.calls, ",") != "preview,apply" {
		t.Fatal(b.calls)
	}
}

func TestDirtyDeparture(t *testing.T) {
	for _, key := range []string{"esc", "q", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			m, _ := readyModel(t, false)
			m.draft.Concise = true
			m, cmd := press(m, key)
			if !m.unsaved || cmd != nil || !strings.Contains(strings.ReplaceAll(m.View().Content, "\n", ""), unsavedPrompt) {
				t.Fatal("missing exact unsaved prompt")
			}
			canceled, cmd := press(m, "esc")
			if canceled.unsaved || !canceled.dirty() || cmd != nil {
				t.Fatal("cancel lost draft")
			}
			m, cmd = press(m, "d")
			if m.dirty() {
				t.Fatal("discard did not restore saved values")
			}
			if key == "esc" && m.page != -1 {
				t.Fatal("did not navigate back")
			}
			if key != "esc" && cmd == nil {
				t.Fatal("did not quit")
			}
		})
	}
}

func TestSaveBeforeDepartureAndRetry(t *testing.T) {
	m, b := readyModel(t, false)
	m.draft.Concise = true
	m, _ = press(m, "q")
	m, cmd := press(m, "s")
	b.fail = "preview"
	m, _ = complete(m, cmd)
	if !m.dirty() || !strings.Contains(m.err, "stale_revision") || m.busy {
		t.Fatal("preview failure lost draft")
	}
	b.fail = ""
	m, cmd = press(m, "s")
	m, _ = complete(m, cmd)
	m, _ = press(m, "enter")
	if !m.dirty() || len(m.dialogs) > 0 {
		t.Fatal("default No did not preserve draft")
	}
	m, cmd = press(m, "s")
	m, _ = complete(m, cmd)
	m, cmd = press(m, "y")
	b.fail = "apply"
	m, _ = complete(m, cmd)
	if !m.dirty() || !strings.Contains(m.err, "recovery /tmp/backup") {
		t.Fatal("apply failure not recoverable")
	}
	b.fail = ""
	m, _ = press(m, "q")
	m, cmd = press(m, "s")
	m, _ = complete(m, cmd)
	m, cmd = press(m, "y")
	m, cmd = complete(m, cmd)
	if m.dirty() || cmd == nil {
		t.Fatal("successful save did not finish pending quit")
	}
}

func TestProtocolFixtures(t *testing.T) {
	for _, name := range []string{"status-fresh", "preview-install"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "managed", name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		kind := strings.Split(name, "-")[0]
		if _, err := decodeResponse(data, kind); err != nil {
			t.Fatal(err)
		}
		if _, err := decodeResponse([]byte(strings.Replace(string(data), `"protocol_version": 1`, `"protocol_version": 9`, 1)), kind); err == nil {
			t.Fatal("accepted incompatible protocol")
		}
	}
	for _, code := range []string{"stale_revision", "unsupported_protocol"} {
		_, err := decodeResponse([]byte(`{"protocol_version":1,"ok":false,"error":{"code":"`+code+`","message":"retry"}}`), "preview")
		if err == nil || !strings.Contains(err.Error(), code) {
			t.Fatal("error code lost", err)
		}
	}
}

func TestWizardQuitSaveReturnsToReview(t *testing.T) {
	m, b := readyModel(t, true)
	m.draft.Concise = true
	m, _ = press(m, "q")
	m, cmd := press(m, "s")
	if cmd != nil || m.page != 4 || m.unsaved || !m.dirty() || len(b.calls) != 0 {
		t.Fatal("quitting wizard saved before final review")
	}
}

func TestRemovalConfirmationDeferredAndCanceled(t *testing.T) {
	for _, key := range []string{"enter", "n", "esc"} {
		t.Run(key, func(t *testing.T) {
			m, b := readyModel(t, false)
			m.draft.Concise = true
			if len(m.dialogs) != 0 {
				t.Fatal("confirmation shown while editing")
			}

			m, cmd := press(m, "s")
			msg := cmd().(resultMsg)
			msg.response.Confirmation = true
			next, _ := m.Update(msg)
			m = next.(model)
			if len(m.dialogs) != 2 {
				t.Fatal("missing deferred removal confirmation")
			}
			m, cmd = press(m, key)
			if cmd != nil || !m.dirty() || len(m.dialogs) != 0 || strings.Join(b.calls, ",") != "preview" {
				t.Fatal("declining removal changed saved state")
			}
		})
	}
}

func TestPreviewConflictPreservesDraft(t *testing.T) {
	m, b := readyModel(t, false)
	m.draft.Concise = true
	m, cmd := press(m, "s")
	msg := cmd().(resultMsg)
	msg.response.Conflicts = []operation{{Name: "user-owned"}}
	next, cmd := m.Update(msg)
	m = next.(model)
	if cmd != nil || !m.dirty() || m.err == "" || len(m.dialogs) > 0 || strings.Join(b.calls, ",") != "preview" {
		t.Fatal("conflicted preview allowed apply or lost draft")
	}
}
