package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestOverviewOffersWorkflowsBeforeDetails(t *testing.T) {
	m, b := readyModel(t, false)
	m.page = managementPage
	m.status.Observed = map[string][]string{"codex": {"commit-message"}, "claude": {"commit-message", "ticket-to-plan"}}
	lines := strings.Join(m.overviewLines(), "\n")
	if !strings.Contains(lines, "2 unique · 3 installations") {
		t.Fatal("skill counts conflate identities and installations")
	}
	if strings.Index(lines, "Explore workflows") > strings.Index(lines, "Storage") {
		t.Fatal("paths precede useful actions")
	}
	for _, size := range [][2]int{{40, 12}, {80, 24}, {120, 40}} {
		m.width, m.height = size[0], size[1]
		content := m.View().Content
		if !strings.Contains(content, "Explore workflows") {
			t.Fatal("workflow entry hidden", size)
		}
		for _, line := range strings.Split(content, "\n") {
			if ansi.StringWidth(line) > m.width {
				t.Fatal("overview overflow")
			}
		}
	}
	m, cmd := press(m, "enter")
	if cmd != nil || m.page != workflowsPage || len(b.calls) != 0 {
		t.Fatal("overview did not open workflows without writes")
	}
}

func TestStarterExplainsBenefitAndCustomization(t *testing.T) {
	m, _ := readyModel(t, true)
	text := strings.Join(m.selectionLines(), "\n")
	for _, want := range []string{"Starter selection", "concise-output", "commit-message", "Workflows"} {
		if !strings.Contains(text, want) {
			t.Fatal("starter explanation missing", want)
		}
	}
}
