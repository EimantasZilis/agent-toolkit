package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func storageFocus(t *testing.T, m model, id, kind string) model {
	t.Helper()
	for i, r := range m.storageRows() {
		if r.provider == id && r.kind == kind {
			m.cursor = i
			return m
		}
	}
	t.Fatalf("missing storage row: %s %s", id, kind)
	return m
}

func storagePath(t *testing.T, m model, id, path string) model {
	t.Helper()
	m.page = 2
	m = storageFocus(t, m, id, "external")
	m, _ = press(m, "enter")
	m = storageFocus(t, m, id, "path")
	m, _ = press(m, "enter")
	m, _ = press(m, "ctrl+u")
	next, _ := m.Update(tea.PasteMsg{Content: path})
	m = next.(model)
	m, _ = press(m, "enter")
	return m
}

func TestSharedStorageEditingAndDirectoryConfirmation(t *testing.T) {
	m, root := backendFixture(t)
	m.draft.Providers = []string{"codex", "claude"}
	m.draft.Requested = []string{"ticket-to-plan"}
	path := filepath.Join(root, "artifacts")
	m = storagePath(t, m, "", path)
	m = storageFocus(t, m, "", "split")
	m, _ = press(m, "enter")
	lines := strings.Join(m.settingsLines(), "\n")
	if !strings.Contains(lines, path+"/<repo>/<skill>/<branch>/<LLM>/") {
		t.Fatal(lines)
	}
	m = saveFixture(t, m)
	for _, id := range m.saved.Providers {
		if m.saved.ForProvider(id).Output != path {
			t.Fatal("root differs", id)
		}
	}
	if !m.saved.SeparateByProvider || m.dirty() {
		t.Fatal("shared settings not saved")
	}
	assertFile(t, path, true)
	m.page = 2
	m = storageFocus(t, m, "", "local")
	m, _ = press(m, "enter")
	if !m.dirty() {
		t.Fatal("shared root change not detected")
	}
	m.unsaved, m.target = true, -1
	m, _ = press(m, "d")
	if m.dirty() || m.draft.Output != path {
		t.Fatal("discard lost shared root")
	}
	m.draft.Providers = []string{"claude"}
	m.loadProvider()
	if !m.draft.SeparateByProvider || m.draft.Output != path {
		t.Fatal("provider count changed storage")
	}
}

func TestSharedBlankStorageBlocksSave(t *testing.T) {
	m, _ := readyModel(t, false)
	m.draft.Providers = []string{"codex", "claude"}
	m = storagePath(t, m, "", "")
	next, cmd := m.startSave()
	m = next.(model)
	if cmd != nil || !strings.Contains(m.err, "absolute") {
		t.Fatal("blank storage accepted", m.err)
	}
}

func TestConflictingLegacyStorageRequiresChoice(t *testing.T) {
	m, _ := readyModel(t, false)
	m.draft = settings{Providers: []string{"codex", "claude"}, ByProvider: map[string]install.ProviderSettings{
		"codex":  {Requested: []string{"commit-message"}, Output: t.TempDir()},
		"claude": {Requested: []string{"commit-message"}, Output: t.TempDir()},
	}}
	m.loadProvider()
	next, cmd := m.startSave()
	m = next.(model)
	if cmd != nil || !strings.Contains(m.err, "Choose one shared") {
		t.Fatal("conflicting roots accepted", m.err)
	}
	m.page = 2
	m = storageFocus(t, m, "", "local")
	m, _ = press(m, "enter")
	if !m.draft.SharedStorage || m.draft.Output != "" {
		t.Fatal("shared location unresolved")
	}
}

func TestDisappearedSharedOutputRequiresNewPreview(t *testing.T) {
	m, _ := readyModel(t, false)
	m.draft.Providers = []string{"codex", "claude"}
	path := filepath.Join(t.TempDir(), "artifacts")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	m = storagePath(t, m, "", path)
	next, cmd := m.startSave()
	m = next.(model)
	m, _ = complete(m, cmd)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	next, cmd = m.apply()
	m = next.(model)
	m, _ = complete(m, cmd)
	if !strings.Contains(m.err, "preview again") {
		t.Fatal("created an unreviewed root", m.err)
	}
	assertFile(t, path, false)
}

func TestStorageLayoutVisibleAfterSeparationToggle(t *testing.T) {
	m, _ := readyModel(t, false)
	m.width, m.height = 70, 20
	m = storagePath(t, m, "", "/tmp/artifacts/longer-output-directory")
	m = storageFocus(t, m, "", "split")
	m, _ = press(m, "enter")
	visible := m.View().Content
	for _, text := range []string{"Separate by LLM", "Layout:", "<LLM>/"} {
		if !strings.Contains(visible, text) {
			t.Fatalf("storage preview missing %q: %s", text, visible)
		}
	}
}
