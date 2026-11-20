package tui

import (
	"strings"
	"testing"
)

func TestErrorDetailsCanScrollWithoutChangingDraft(t *testing.T) {
	m, b := readyModel(t, false)
	m.draft.Concise = true
	m.err = strings.Repeat("long error with path /tmp/example\n", 80) + "last diagnostic"
	m, cmd := press(m, "e")
	if cmd != nil || !m.errorReview || !strings.Contains(m.View().Content, "Operation error") {
		t.Fatal("missing full error view")
	}
	m, _ = press(m, "pgdown")
	if !m.manualScroll || m.scroll == 0 {
		t.Fatal("error cannot scroll")
	}
	m, cmd = press(m, "esc")
	if cmd != nil || m.errorReview || !m.draft.Concise || len(b.calls) != 0 {
		t.Fatal("error view changed draft or installation")
	}
}

func TestConflictIncludesCauseAndRecovery(t *testing.T) {
	m, _ := readyModel(t, false)
	next, _ := m.result(resultMsg{action: "preview", response: response{Conflicts: []operation{{Provider: "codex", Name: "example", Path: "/tmp/example", Reason: "Changed permissions", Recovery: "Restore matching backup"}}}})
	m = next.(model)
	for _, want := range []string{"Codex / example", "/tmp/example", "Changed permissions", "Restore matching backup"} {
		if !strings.Contains(strings.Join(m.viewLines(), "\n"), want) {
			t.Fatal("missing diagnostic", want)
		}
	}
}
