package tui

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func finishDialogs(t *testing.T, m model) model {
	t.Helper()
	for len(m.dialogs) > 0 {
		next, cmd := press(m, "y")
		m = next
		if cmd != nil {
			m, _ = complete(m, cmd)
		}
	}
	if m.err != "" {
		t.Fatal(m.err)
	}
	return m
}

func TestManagementRefreshAndUninstall(t *testing.T) {
	for _, providers := range [][]string{{"codex"}, {"claude"}, {"codex", "claude"}} {
		t.Run(strings.Join(providers, "+"), func(t *testing.T) {
			m, root := backendFixture(t)
			m.draft.Providers = providers
			m.draft.Requested = []string{"commit-message"}
			m.draft.Concise, m.draft.FullTemplate = true, true
			m = saveFixture(t, m)
			if m.page != managementPage {
				t.Fatal("setup did not open management")
			}
			for _, label := range []string{"Configure", "Refresh installation", "Uninstall"} {
				if !strings.Contains(strings.Join(m.viewLines(), "\n"), label) {
					t.Fatal("missing action: " + label)
				}
			}
			saved := clone(m.saved)
			before, err := m.backend.Status(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			m.cursor = 1
			m, cmd := press(m, "enter")
			m, _ = complete(m, cmd)
			if m.applyRequest.Intent != install.Refresh || len(m.dialogs) == 0 {
				t.Fatal("refresh did not preview")
			}
			m = finishDialogs(t, m)
			if !reflect.DeepEqual(m.saved, saved) || !strings.Contains(m.notice, "Start a new agent session") {
				t.Fatal("refresh changed preferences or omitted restart message")
			}
			after, err := m.backend.Status(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if before.Revision != after.Revision {
				t.Fatal("unchanged refresh changed installation")
			}
			m.cursor = 2
			m, cmd = press(m, "enter")
			m, _ = complete(m, cmd)
			if len(m.dialogs) != 1 || !strings.Contains(m.dialogs[0].text, "forget saved settings") {
				t.Fatal("missing explicit final uninstall prompt")
			}
			m, cmd = press(m, "enter")
			if cmd != nil {
				t.Fatal("default No started uninstall")
			}
			assertFile(t, filepath.Join(root, "toolkit/config/settings.env"), true)
			m, cmd = press(m, "enter")
			m, _ = complete(m, cmd)
			m = finishDialogs(t, m)
			if !m.wizard || m.status.Installed || m.page != 0 {
				t.Fatal("uninstall did not return to fresh setup")
			}
			assertFile(t, filepath.Join(root, "toolkit/config/settings.env"), false)
			assertFile(t, filepath.Join(root, "toolkit/config/inventory.json"), false)
		})
	}
}

func TestRefreshConflictShowsPathAndPreservesChoices(t *testing.T) {
	m, root := backendFixture(t)
	m.draft.Providers = []string{"codex"}
	m.draft.Requested = []string{"commit-message"}
	m = saveFixture(t, m)
	before := clone(m.draft)
	path := filepath.Join(root, "agents/skills/agent-toolkit/commit-message/SKILL.md")
	if err := os.WriteFile(path, []byte("modified by user\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m.cursor = 1
	m, cmd := press(m, "enter")
	m, _ = complete(m, cmd)
	if len(m.dialogs) != 0 || !strings.Contains(m.err, filepath.Dir(path)) || !reflect.DeepEqual(m.draft, before) {
		t.Fatalf("conflict lost choices or hid its path: dialogs=%v error=%q draft=%+v before=%+v", m.dialogs, m.err, m.draft, before)
	}
}

func TestSetupCanForgetUnavailableSavedSkill(t *testing.T) {
	m, root := backendFixture(t)
	m.draft.Providers = []string{"codex"}
	m.draft.Requested = []string{"commit-message"}
	m = saveFixture(t, m)
	path := filepath.Join(root, "toolkit/config/settings.env")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "REQUESTED_SKILLS=commit-message", "REQUESTED_SKILLS=commit-message unavailable-skill", 1))
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	m, _ = complete(m, m.Init())
	visible := false
	for _, skill := range m.catalog.Skills {
		if skill.Name == "unavailable-skill" {
			visible = true
		}
	}
	if !visible {
		t.Fatal("unavailable saved choice cannot be reviewed")
	}
	m.toggleSkill("unavailable-skill")
	m = saveFixture(t, m)
	if strings.Contains(strings.Join(m.saved.Requested, " "), "unavailable") {
		t.Fatal("unavailable selection was retained")
	}
}

func TestFullGuidanceToggleIsSeparate(t *testing.T) {
	m, _ := readyModel(t, false)
	m.page = 3
	m, _ = press(m, "space")
	m, _ = press(m, "down")
	m, _ = press(m, "space")
	if !m.draft.Concise || !m.draft.FullTemplate {
		t.Fatal("settings did not enable independently")
	}
	m, _ = press(m, "space")
	if !m.draft.Concise || m.draft.FullTemplate {
		t.Fatal("full guidance changed concise loading")
	}
}
