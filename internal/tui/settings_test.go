package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestGlobalPathValidationDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "absent"), link); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0700); err != nil {
			t.Error(err)
		}
	})
	cases := []struct {
		name, path string
		valid      bool
		missing    bool
	}{
		{"blank", "", false, false},
		{"relative", "../outputs", false, false},
		{"newline", root + "/bad\npath", false, false},
		{"control", root + "/bad\x1bpath", false, false},
		{"file", file, false, false},
		{"file parent", filepath.Join(file, "child"), false, false},
		{"dangling link", link, false, false},
		{"existing", root, true, false},
		{"missing", filepath.Join(root, "new", "outputs"), true, true},
	}
	if os.Geteuid() != 0 {
		cases = append(cases, struct {
			name, path string
			valid      bool
			missing    bool
		}{"unwritable", filepath.Join(locked, "new"), false, false})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, b := readyModel(t, true)
			m.page, m.globalOutput, m.draft.Output = 2, true, tc.path
			missing, err := m.validateOutput()
			if (err == nil) != tc.valid || missing != tc.missing {
				t.Fatalf("missing=%t error=%v", missing, err)
			}
			next, cmd := press(m, "tab")
			if cmd != nil || (next.page == 3) != tc.valid || len(b.calls) != 0 {
				t.Fatal("path validation did not control continuation without backend I/O")
			}
		})
	}
	assertFile(t, filepath.Join(root, "new"), false)
}

func TestFreshOutputRecommendationAndSavedLocal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, installed := range []bool{false, true} {
		m := newModel(&fakeBackend{}, t.TempDir())
		next, _ := m.Update(loadedMsg{installed: installed})
		m = next.(model)
		if m.globalOutput == installed || (m.draft.Output != "") == installed {
			t.Fatal("fresh recommendation or saved repo-local preference lost")
		}
		if !filepath.IsAbs(m.outputPath) {
			t.Fatal("recommended path is not absolute")
		}
	}
}

func TestOutputEditingAndModes(t *testing.T) {
	m, b := readyModel(t, false)
	m.page = 2
	m.draft.Providers = []string{"codex", "claude"}
	m.syncOutput()
	m, _ = press(m, "down")
	m, _ = press(m, "space")
	if !m.globalOutput || m.draft.Output == "" {
		t.Fatal("global selection did not reveal the path")
	}
	m, _ = press(m, "down")
	m, _ = press(m, "enter")
	if !m.editingPath || m.page != 2 {
		t.Fatal("path did not enter editing")
	}
	m, _ = press(m, "ctrl+u")
	for _, text := range []string{"/tmp/", "s q space é"} {
		next, _ := m.Update(tea.KeyPressMsg{Text: text})
		m = next.(model)
	}
	next, _ := m.Update(tea.PasteMsg{Content: "/pasted"})
	m = next.(model)
	if m.outputPath != "/tmp/s q space é/pasted" || len(b.calls) != 0 {
		t.Fatal("printable path keys invoked shortcuts or were lost")
	}
	m, _ = press(m, "left")
	m, _ = press(m, "backspace")
	m, _ = press(m, "delete")
	if m.outputPath != "/tmp/s q space é/past" {
		t.Fatal(m.outputPath)
	}
	next, _ = m.Update(tea.PasteMsg{Content: "bad\npath"})
	m = next.(model)
	if m.err == "" || strings.Contains(m.outputPath, "\n") {
		t.Fatal("multiline paste accepted")
	}
	m, _ = press(m, "esc")
	if m.editingPath || m.outputPath != m.originalPath {
		t.Fatal("cancel did not restore the path")
	}
	guidance := strings.Join(m.settingsLines(), "\n")
	if !strings.Contains(guidance, "Codex, Claude Code") || !strings.Contains(guidance, "repo, branch, and skill") {
		t.Fatal("missing provider-specific output guidance")
	}
	m.cursor = 0
	m, _ = press(m, "space")
	if m.globalOutput || m.draft.Output != "" || strings.Contains(strings.Join(m.settingsLines(), "\n"), "Path:") {
		t.Fatal("repo-local did not hide and clear path")
	}
}

func TestCreationConfirmationAndCancellation(t *testing.T) {
	for _, key := range []string{"enter", "n", "esc"} {
		t.Run(key, func(t *testing.T) {
			m, b := readyModel(t, false)
			path := filepath.Join(t.TempDir(), "missing", "output")
			m.globalOutput, m.draft.Output = true, path
			m, cmd := press(m, "s")
			if cmd == nil || !m.createOutput || m.applyRequest.Desired.Output != path {
				t.Fatal("missing directory was not staged")
			}
			m, _ = complete(m, cmd)
			if !strings.Contains(m.dialogs[0].text, "Create the Markdown output directory?\n"+path) {
				t.Fatal("missing creation confirmation")
			}
			m, cmd = press(m, key)
			if cmd != nil || len(m.dialogs) != 0 || !m.dirty() || strings.Join(b.calls, ",") != "preview" {
				t.Fatal("declined creation applied or lost draft")
			}
			assertFile(t, path, false)
		})
	}
}

func TestBlankGlobalModeIsDirtyAndDiscardable(t *testing.T) {
	m, _ := readyModel(t, false)
	m.page, m.globalOutput, m.outputPath = 2, true, ""
	if !m.dirty() {
		t.Fatal("blank global mode lost its draft state")
	}
	m, _ = press(m, "esc")
	if !m.unsaved {
		t.Fatal("leaving a blank global mode skipped Save/Discard")
	}
	m, _ = press(m, "d")
	if m.globalOutput || m.dirty() || m.page != -1 {
		t.Fatal("discard did not restore repo-local mode")
	}
}

func TestDisappearedOutputRequiresNewConfirmation(t *testing.T) {
	m, b := readyModel(t, false)
	path := filepath.Join(t.TempDir(), "outputs")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	m.globalOutput, m.draft.Output = true, path
	m, cmd := press(m, "s")
	m, _ = complete(m, cmd)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	m, cmd = press(m, "y")
	m, _ = complete(m, cmd)
	if !strings.Contains(m.err, "preview again") || strings.Join(b.calls, ",") != "preview" || !m.dirty() {
		t.Fatal("created a directory without confirmation")
	}
}

func TestBackendOutputAndConcise(t *testing.T) {
	m, root := backendFixture(t)
	m.draft.Providers = []string{"codex", "claude"}
	m.draft.Requested = []string{"commit-message"}
	m = saveFixture(t, m)
	path := filepath.Join(root, "new outputs", "literal $(touch PWN) 'quoted'")
	m.page, m.globalOutput, m.outputPath, m.draft.Output = 2, true, path, path
	m.draft.Concise = true
	m, cmd := press(m, "s")
	m, _ = complete(m, cmd)
	assertFile(t, filepath.Dir(path), false)
	if m.err != "" || len(m.dialogs) < 2 {
		t.Fatalf("preview failed: %s", m.err)
	}
	m = saveFixture(t, m)
	assertFile(t, path, true)
	if m.dirty() || m.saved.Output != path || !m.saved.Concise {
		t.Fatal("actual backend result not reflected")
	}
	artifact := filepath.Join(path, "keep.md")
	if err := os.WriteFile(artifact, []byte("existing artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	guidance := []string{filepath.Join(root, "codex", "AGENTS.md"), filepath.Join(root, "claude", "CLAUDE.md")}
	for _, file := range guidance {
		data, err := os.ReadFile(file)
		if err != nil || !strings.Contains(string(data), path) || !strings.Contains(string(data), "agent-toolkit:concise-output:start") {
			t.Fatalf("missing output/concise guidance: %s: %s: %v", file, data, err)
		}
		if strings.Contains(string(data), "## Code readability") {
			t.Fatal("TUI enabled full template")
		}
	}
	m.page, m.cursor = 3, 0
	m, _ = press(m, "space")
	m = saveFixture(t, m)
	for _, file := range guidance {
		data, err := os.ReadFile(file)
		if err != nil || !strings.Contains(string(data), path) || strings.Contains(string(data), "agent-toolkit:concise-output:start") {
			t.Fatalf("concise disable changed output guidance: %s: %v", data, err)
		}
	}
	assertFile(t, filepath.Join(root, "agents", "skills", "agent-toolkit", "concise-output", "SKILL.md"), true)
	assertFile(t, filepath.Join(root, "claude", "skills", "concise-output", "SKILL.md"), true)
	m.page, m.cursor = 2, 0
	m, _ = press(m, "space")
	m = saveFixture(t, m)
	for _, file := range guidance {
		assertFile(t, file, false)
	}
	assertFile(t, artifact, true)
}

func TestFullTemplatePreferencePreserved(t *testing.T) {
	m, _ := readyModel(t, false)
	m.draft.FullTemplate, m.page = true, 3
	m, _ = press(m, "space")
	if !m.draft.FullTemplate || !m.draft.Concise || strings.Contains(strings.Join(m.settingsLines(), "\n"), "Full template") {
		t.Fatal("full-template preference was changed or exposed")
	}
	next, cmd := m.startSave()
	m = next.(model)
	if cmd == nil || !m.applyRequest.Desired.FullTemplate {
		t.Fatal("full template lost at typed boundary")
	}

}

func TestViewportResizeLongListsAndDialogs(t *testing.T) {
	m, _ := readyModel(t, false)
	m.catalog, m.page = selectionCatalog(), 1
	for _, s := range m.catalog.Skills {
		m.expanded[s.Category] = true
	}
	m.cursor = len(m.skillRows()) - 1
	for _, size := range [][2]int{{80, 24}, {30, 8}, {12, 6}, {5, 6}, {4, 24}, {80, 5}, {4, 5}, {1, 1}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(model)
		lines := strings.Split(m.View().Content, "\n")
		if len(lines) > size[1] {
			t.Fatalf("height overflow at %v: %d", size, len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("width overflow at %v: %q", size, line)
			}
		}
	}
	m.width, m.height = 30, 8
	m.dialogs = []dialog{{"Remove these skills?\n" + strings.Repeat("skill-name\n", 40) + "[y/N]"}}
	if !strings.Contains(m.View().Content, "[y/N]") {
		t.Fatal("default choice is offscreen")
	}
	before := m.View().Content
	m, _ = press(m, "pgup")
	if m.View().Content == before || !strings.Contains(m.View().Content, "pgup/pgdown") {
		t.Fatal("long dialog cannot be scrolled")
	}
}
