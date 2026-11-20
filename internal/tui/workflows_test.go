package tui

import (
	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func workflowModel(t *testing.T) (model, *fakeBackend) {
	t.Helper()
	m, b := readyModel(t, false)
	c, err := catalog.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	m.catalog, m.page = catalogResponse(c), workflowsPage
	return m, b
}

func TestWorkflowStagesMissingSkillsForChosenAgent(t *testing.T) {
	m, b := workflowModel(t)
	m.draft.Providers = []string{"codex", "claude"}
	m.draft.Requested = []string{"commit-message"}
	m.storeProvider()
	before := clone(m.draft.ForProvider("codex"))
	m.cursor = len(workflows) + 1
	m, cmd := press(m, "enter")
	if cmd != nil || len(b.calls) != 0 {
		t.Fatal("workflow applied installation")
	}
	if !reflect.DeepEqual(before, m.draft.ForProvider("codex")) {
		t.Fatal("changed other agent")
	}
	if !slices.Contains(m.draft.Requested, "commit-message") || m.selected()["plan-review"] == "" || m.selected()["ticket-to-plan"] == "" {
		t.Fatal("lost existing selection or dependency")
	}
	if !strings.Contains(strings.Join(m.workflowLines(), "\n"), "in draft") {
		t.Fatal("missing coverage")
	}
	m, cmd = press(m, "s")
	if cmd == nil {
		t.Fatal("cannot review staged workflow")
	}
	m, _ = complete(m, cmd)
	m, cmd = press(m, "enter")
	if cmd != nil || !m.dirty() || strings.Join(b.calls, ",") != "preview" {
		t.Fatal("cancel lost draft or applied")
	}
}

func TestWorkflowNavigationAndCatalog(t *testing.T) {
	m, _ := workflowModel(t)
	for i := range workflows {
		m.cursor, m.workflowIndex = i, i
		for _, width := range []int{40, 80, 120} {
			m.width = width
			if !strings.Contains(m.View().Content, "Workflows") {
				t.Fatal("workflow view missing")
			}
		}
		for _, name := range m.workflowSkills() {
			if !slices.ContainsFunc(m.catalog.Skills, func(s skill) bool { return s.Name == name }) {
				t.Fatal("unknown workflow skill", name)
			}
		}
		if !strings.Contains(strings.Join(m.workflowLines(), "\n"), "Prompt: Use ") {
			t.Fatal("missing prompt")
		}
	}
	m.draft.Providers = nil
	m.cursor = len(workflows)
	m, cmd := press(m, "enter")
	if cmd != nil || m.page != 0 {
		t.Fatal("no-agent action did not open Agents")
	}
	m, _ = press(m, "5")
	if m.page != workflowsPage {
		t.Fatal("fifth tab unavailable")
	}
	m, _ = press(m, "esc")
	if m.page != managementPage {
		t.Fatal("workflow escape did not return home")
	}
}

func TestChoosingWorkflowMovesDirectlyToAgentActions(t *testing.T) {
	m, _ := workflowModel(t)
	m.cursor = 0
	m, _ = press(m, "enter")
	if m.cursor != len(workflows) || m.workflowIndex != 0 {
		t.Fatal("task choice did not reach its actions")
	}
	m, _ = press(m, "enter")
	if m.selected()["ticket-to-plan"] == "" || m.selected()["pr-description"] != "" {
		t.Fatal("navigation changed chosen workflow")
	}
}
