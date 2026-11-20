package tui

import (
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func TestRefreshProtectsPendingDraft(t *testing.T) {
	for _, key := range []string{"enter", "esc", "n", "d"} {
		t.Run(key, func(t *testing.T) {
			m, b := readyModel(t, false)
			m.page, m.cursor = managementPage, 1
			m.draft.Concise = true
			m.storeProvider()
			next, cmd := m.startAction(install.Refresh)
			m = next.(model)
			if cmd != nil || !m.refreshPending || len(b.calls) != 0 {
				t.Fatal("refresh bypassed draft decision")
			}
			m, cmd = press(m, key)
			if key == "d" {
				if cmd == nil || m.dirty() {
					t.Fatal("discard did not prepare saved settings")
				}
				m, _ = complete(m, cmd)
				if strings.Join(b.calls, ",") != "preview" {
					t.Fatal(b.calls)
				}
			} else if cmd != nil || !m.draft.Concise || m.cursor != 1 || m.refreshPending {
				t.Fatal("cancel lost draft or focus")
			}
		})
	}
}

func TestNoticeKeepsUnsavedIndicator(t *testing.T) {
	m, _ := readyModel(t, false)
	m.draft.Concise, m.notice = true, "Selection staged"
	if !strings.Contains(m.View().Content, "Unsaved configuration") {
		t.Fatal("notice hid unsaved state")
	}
}
