package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func selectionCatalog() response {
	return response{
		Skills: []skill{
			{Name: "ticket-to-plan", Category: "scoping", Helpers: []string{"plan-review"}},
			{Name: "plan-review", Category: "review"},
			{Name: "commit-message", Category: "productivity"},
			{Name: "concise-output", Category: "productivity"},
		},
		Bundles: []bundle{{Name: "starter", Skills: []string{"commit-message", "concise-output"}}},
	}
}

func TestFreshProvidersAndStarter(t *testing.T) {
	for _, providers := range [][]string{{"codex"}, {"claude"}, {"codex", "claude"}} {
		t.Run(strings.Join(providers, "+"), func(t *testing.T) {
			b := &fakeBackend{}
			m := newModel(b, t.TempDir())
			next, _ := m.Update(loadedMsg{catalog: selectionCatalog()})
			m = next.(model)
			if !m.wizard || len(m.draft.Providers) != 0 || !slices.Contains(m.draft.Requested, "commit-message") {
				t.Fatal("incorrect fresh defaults")
			}
			blocked, cmd := press(m, "tab")
			if cmd != nil || blocked.page != 0 || blocked.err == "" {
				t.Fatal("empty provider selection continued")
			}
			blocked.page = 4
			blocked, cmd = press(blocked, "s")
			if cmd != nil || blocked.err == "" || len(b.calls) != 0 {
				t.Fatal("empty setup applied")
			}

			for _, id := range providers {
				m.cursor = slices.Index(providerIDs, id)
				m, _ = press(m, "space")
			}
			m, cmd = press(m, "tab")
			if cmd != nil || m.page != 1 || len(m.draft.Providers) != len(providers) {
				t.Fatal("explicit selection did not continue")
			}
		})
	}
}

func TestSavedAndImportedSelections(t *testing.T) {
	for _, installed := range []bool{true, false} {
		m := newModel(&fakeBackend{}, t.TempDir())
		saved := settings{Providers: []string{"claude", "codex"}, Requested: []string{"ticket-to-plan", "plan-review"}, FullTemplate: true}
		next, _ := m.Update(loadedMsg{catalog: selectionCatalog(), status: response{Desired: saved}, installed: installed})
		m = next.(model)
		if m.wizard || m.dirty() || !slices.Equal(m.draft.Providers, saved.Providers) || !slices.Equal(m.draft.Requested, saved.Requested) || !m.draft.FullTemplate {
			t.Fatal("saved/imported choices changed")
		}
	}
}

func TestBundlesChildrenAndRequiredHelpers(t *testing.T) {
	m, b := readyModel(t, false)
	m.catalog, m.page = selectionCatalog(), 1
	rows := m.skillRows()
	if len(rows) != 3 || rows[0].category != "productivity" {
		t.Fatal("categories not grouped")
	}
	m, _ = press(m, "right")
	if len(m.skillRows()) != 5 {
		t.Fatal("category did not expand")
	}
	m, _ = press(m, "space")
	if m.selected()["commit-message"] == "" || m.selected()["concise-output"] == "" {
		t.Fatal("bundle did not select children")
	}
	m, _ = press(m, "down")
	m, _ = press(m, "space")
	if m.selected()["commit-message"] != "" || strings.Contains(m.selectionLines()[3], "[x]") {
		t.Fatal("child deselection did not clear bundle checkbox")
	}

	m.toggleSkill("ticket-to-plan")
	if m.selected()["plan-review"] != "required" {
		t.Fatal("helper not selected automatically")
	}
	m.toggleSkill("plan-review")
	if m.selected()["plan-review"] != "required" || !strings.Contains(m.notice, "ticket-to-plan") {
		t.Fatal("required helper was removed or unexplained")
	}
	if len(b.calls) != 0 {
		t.Fatal("editing invoked backend")
	}

	m.status.Observed = map[string][]string{"codex": {"ticket-to-plan", "plan-review"}}
	m.toggleSkill("ticket-to-plan")
	if m.selected()["plan-review"] != "retained" {
		t.Fatal("parent removal lost installed helper")
	}
	m.toggleSkill("plan-review")
	if m.selected()["plan-review"] != "" || !slices.Contains(m.removed, "plan-review") {
		t.Fatal("unused helper cannot be explicitly removed")
	}
}

func TestToggleAllAndAutomaticConcise(t *testing.T) {
	m, _ := readyModel(t, false)
	m.catalog, m.page, m.draft.Concise = selectionCatalog(), 1, true
	m, _ = press(m, "a")
	if len(m.selected()) != 4 || m.selected()["plan-review"] != "required" {
		t.Fatal("toggle all missed dependency closure")
	}
	m.status.Observed = map[string][]string{"codex": {"plan-review", "concise-output"}}
	m, _ = press(m, "a")
	if m.selected()["plan-review"] != "retained" || m.selected()["concise-output"] != "automatic" || m.selected()["ticket-to-plan"] != "" {
		t.Fatal("toggle all removed retained/automatic helpers")
	}
	m.toggleSkill("concise-output")
	if !strings.Contains(m.notice, "Page 4") || m.selected()["concise-output"] != "automatic" {
		t.Fatal("automatic concise removal was not blocked")
	}
}

func TestExactRemovalConfirmations(t *testing.T) {
	m, _ := readyModel(t, false)
	m.saved.Providers = []string{"codex", "claude"}
	m.draft.Providers = nil
	m.preview.Confirmation = true
	dialogs := m.confirmations()
	want := "Are you sure you want to uninstall Codex, Claude Code skills and any global settings associated with it? [y/N]"
	if dialogs[0].text != want {
		t.Fatalf("provider prompt = %q", dialogs[0].text)
	}

	m.draft.Providers = []string{"codex"}
	m.preview.Operations = []operation{{Provider: "codex", Action: "remove", Name: "ticket-to-plan"}}
	dialogs = m.confirmations()
	if dialogs[1].text != "Are you sure you want to uninstall these skills?\nticket-to-plan\n[y/N]" {
		t.Fatal(dialogs)
	}
}

func backendFixture(t *testing.T) (model, string) {
	t.Helper()
	root := t.TempDir()
	source, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	toolkit := filepath.Join(root, "toolkit")
	if err := os.Mkdir(toolkit, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"scripts", ".agents", ".env.template"} {
		if data, err := exec.Command("cp", "-R", filepath.Join(source, name), toolkit).CombinedOutput(); err != nil {
			t.Fatalf("copy fixture: %v: %s", err, data)
		}
	}
	for _, key := range []string{"MARKDOWN_OUTPUT_PATH", "AGENT_TOOLKIT_FAIL_STATE", "AGENT_TOOLKIT_FAIL_PROVIDER", "AGENT_TOOLKIT_INTERRUPT_PROVIDER"} {
		t.Setenv(key, "")
	}
	t.Setenv("AGENT_TOOLKIT_HOME", toolkit)
	t.Setenv("AGENT_TOOLKIT_ENV_FILE", filepath.Join(root, "absent.env"))
	t.Setenv("AGENTS_HOME", filepath.Join(root, "agents"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("CLAUDE_HOME", filepath.Join(root, "claude"))
	m := newModel(shellBackend{toolkit}, toolkit)
	next, _ := m.Update(m.Init()())
	m = next.(model)
	if m.err != "" {
		t.Fatal(m.err)
	}
	return m, root
}

func saveFixture(t *testing.T, m model) model {
	t.Helper()
	next, cmd := m.startSave()
	m = next.(model)
	if cmd == nil {
		t.Fatalf("save did not start: %s", m.err)
	}
	m, _ = complete(m, cmd)
	if m.err != "" {
		t.Fatal(m.err)
	}
	for len(m.dialogs) > 0 {
		m, cmd = press(m, "y")
		if cmd != nil {
			m, _ = complete(m, cmd)
		}
	}
	if m.err != "" || m.busy {
		t.Fatalf("apply: %s", m.err)
	}
	return m
}

func assertFile(t *testing.T, path string, exists bool) {
	t.Helper()
	_, err := os.Stat(path)
	if exists && err != nil {
		t.Fatal(err)
	}
	if !exists && !os.IsNotExist(err) {
		t.Fatalf("path still exists or is inaccessible: %s: %v", path, err)
	}
}

func TestBackendSharedSelectionAndPreservation(t *testing.T) {
	m, root := backendFixture(t)
	m.draft.Providers = []string{"codex"}
	m.draft.Requested = []string{"ticket-to-plan"}
	m = saveFixture(t, m)
	codex := filepath.Join(root, "agents", "skills", "agent-toolkit")
	claude := filepath.Join(root, "claude", "skills")
	assertFile(t, filepath.Join(codex, "plan-review", "SKILL.md"), true)
	user := filepath.Join(root, "codex", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(user), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(user, []byte("User instructions\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m.draft.Providers = add(m.draft.Providers, "claude")
	m = saveFixture(t, m)
	assertFile(t, filepath.Join(claude, "ticket-to-plan", "SKILL.md"), true)
	m.toggleSkill("ticket-to-plan")
	m = saveFixture(t, m)
	for _, dir := range []string{codex, claude} {
		assertFile(t, filepath.Join(dir, "ticket-to-plan"), false)
		assertFile(t, filepath.Join(dir, "plan-review", "SKILL.md"), true)
	}
	m.toggleSkill("plan-review")
	m.draft.Providers = []string{"codex"}
	m = saveFixture(t, m)
	assertFile(t, filepath.Join(codex, "plan-review"), false)
	assertFile(t, filepath.Join(claude, "plan-review"), false)
	data, err := os.ReadFile(user)
	if err != nil || string(data) != "User instructions\n" {
		t.Fatalf("user guidance changed: %q, %v", data, err)
	}
	if m.dirty() {
		t.Fatal("actual applied state not accepted")
	}
}
