package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func TestProviderListsAndIndependentEdits(t *testing.T) {
	m, _ := readyModel(t, false)
	m.catalog, m.page = selectionCatalog(), 1
	m.draft.Providers = []string{"codex", "claude"}
	m.draft.ByProvider = map[string]install.ProviderSettings{
		"codex":  {Requested: []string{"ticket-to-plan"}},
		"claude": {Requested: []string{"commit-message"}},
	}
	m.loadProvider()
	m.saved = clone(m.draft)
	m.status.Observed = map[string][]string{"codex": {"ticket-to-plan", "plan-review"}, "claude": {"commit-message"}}
	m.expanded["productivity"] = true
	m.expanded["scoping"] = true
	lines := strings.Join(m.selectionLines(), "\n")
	for _, label := range []string{"Codex", "Claude Code", "ticket-to-plan", "commit-message (installed)"} {
		if !strings.Contains(lines, label) {
			t.Fatalf("provider lists missing %q: %s", label, lines)
		}
	}
	for i, r := range m.skillRows() {
		if r.provider == "claude" && r.name == "commit-message" {
			m.cursor = i
			break
		}
	}
	m, _ = press(m, "enter")
	if m.providerView("claude").selected()["commit-message"] != "" {
		t.Fatal("Claude row did not change")
	}
	if m.providerView("codex").selected()["plan-review"] != "required" {
		t.Fatal("Claude edit changed Codex helpers")
	}
	m.page, m.cursor = 2, len(m.storageRows())+2
	m, _ = press(m, "down")
	m, _ = press(m, "enter")
	if !m.providerView("claude").guidanceEnabled() || m.providerView("codex").guidanceEnabled() {
		t.Fatal("guidance crossed providers")
	}
	if !m.dirty() {
		t.Fatal("provider edits were not tracked")
	}
	m.unsaved, m.target = true, -1
	m, _ = press(m, "d")
	if m.dirty() || m.providerView("claude").guidanceEnabled() {
		t.Fatal("discard did not restore both lists")
	}
}

func TestInstalledListsStaySeparateFromDraft(t *testing.T) {
	m, _ := backendFixture(t)
	m.draft.Providers = []string{"codex", "claude"}
	m.draft.ByProvider = map[string]install.ProviderSettings{
		"codex":  {Requested: []string{"ticket-to-plan"}, Concise: true, FullTemplate: true},
		"claude": {Requested: []string{"commit-message"}},
	}
	m.loadProvider()
	m = saveFixture(t, m)
	overview := func() string {
		return strings.SplitN(strings.Join(m.overviewLines(), "\n"), "\nSource", 2)[0]
	}
	before := overview()
	codex, claude, ok := strings.Cut(before, "Claude Code")
	if !ok {
		t.Fatal("installed Claude summary missing", before)
	}
	for id, summary := range map[string]string{"codex": codex, "claude": claude} {
		if !strings.Contains(summary, fmt.Sprintf("Installed skills: %d", len(m.status.Observed[id]))) {
			t.Fatal("installed skill count missing", before)
		}
		for _, name := range m.status.Observed[id] {
			if strings.Contains(before, "\n  "+name+"\n") {
				t.Fatalf("overview lists installed skill %q: %s", name, before)
			}
		}
	}
	for _, label := range []string{"Guidance", "guidance", "Concise replies", "Markdown output instructions"} {
		if strings.Contains(before, label) {
			t.Fatalf("overview includes guidance section %q: %s", label, before)
		}
	}
	m.page = 2
	settings := strings.Join(m.settingsLines(), "\n")
	for _, guide := range m.status.Guidance {
		for _, path := range []string{guide.Path, guide.CodingPath, guide.DocumentationPath, guide.ConcisePath, guide.OutputPath} {
			if path == "" {
				continue
			}
			if strings.Contains(before, path) || !strings.Contains(settings, path) {
				t.Fatalf("guidance path should appear only in settings: %s", path)
			}
		}
	}
	if strings.Contains(before, "Target:") || strings.Contains(before, "Follow coding standards") {
		t.Fatal("overview includes guidance details", before)
	}
	m.page, m.cursor = 2, len(m.storageRows())
	m.cursor = len(m.storageRows()) + 3
	m, _ = press(m, "enter")
	if overview() != before {
		t.Fatal("draft changed installed state")
	}
	m = saveFixture(t, m)
	if !m.status.Guidance["claude"].FullTemplate || !m.status.Guidance["codex"].Concise {
		t.Fatal("applied guidance not visible")
	}
}

func TestOverviewShowsOnlyObservedProviders(t *testing.T) {
	for _, tc := range []struct {
		name     string
		skills   []string
		guidance string
		visible  bool
	}{
		{name: "absent"},
		{name: "skills", skills: []string{"commit-message"}, visible: true},
		{name: "guidance", guidance: "Installed", visible: true},
		{name: "guidance conflict", guidance: "Needs review", visible: true},
		{name: "no guidance", guidance: "Not installed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := readyModel(t, true)
			m.page = managementPage
			m.saved.Providers = []string{"codex", "claude"}
			m.draft.Providers = []string{"codex", "claude"}
			m.status.Observed = map[string][]string{"codex": {"commit-message"}, "claude": tc.skills}
			m.status.Guidance = map[string]install.GuidanceStatus{"claude": {State: tc.guidance}}
			lines := strings.Join(m.viewLines(), "\n")
			if strings.Contains(lines, "Claude Code") != tc.visible {
				t.Fatal(lines)
			}
			if strings.Contains(strings.Split(lines, "\n")[2], "claude") != tc.visible {
				t.Fatal("provider summary differs", lines)
			}
			if !strings.Contains(lines, "Codex") {
				t.Fatal("installed Codex hidden", lines)
			}
		})
	}
}
