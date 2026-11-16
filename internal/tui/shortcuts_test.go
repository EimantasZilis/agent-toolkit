package tui

import "testing"

func TestBackShortcuts(t *testing.T) {
	for _, key := range []string{"backspace", "esc"} {
		for _, tc := range []struct {
			name       string
			page, want int
			wizard     bool
			installed  bool
			quit       bool
		}{
			{"settings", 3, -1, false, true, false},
			{"configure menu", -1, managementPage, false, true, false},
			{"management menu", managementPage, managementPage, false, true, true},
			{"setup menu", -1, -1, false, false, true},
			{"wizard page", 3, 2, true, false, false},
			{"wizard start", 0, 0, true, false, true},
		} {
			t.Run(key+"/"+tc.name, func(t *testing.T) {
				m, b := readyModel(t, tc.wizard)
				m.page, m.status.Installed = tc.page, tc.installed
				m, cmd := press(m, key)
				if m.page != tc.want || (cmd != nil) != tc.quit || m.unsaved || len(b.calls) != 0 {
					t.Fatalf("page=%d quit=%t unsaved=%t calls=%v", m.page, cmd != nil, m.unsaved, b.calls)
				}
			})
		}
	}
}

func TestEscapeCancelsWithoutLosingDraft(t *testing.T) {
	for _, state := range []string{"unsaved", "conflict", "path", "busy"} {
		t.Run(state, func(t *testing.T) {
			m, b := readyModel(t, false)
			m.draft.Concise = true
			switch state {
			case "unsaved":
				m.unsaved, m.depart = true, true
			case "conflict":
				m.conflictReview, m.err = true, "blocked"
			case "path":
				m.editingPath = true
				m.originalPath, m.outputPath, m.draft.Output = "/original", "/edited", "/edited"
			case "busy":
				m.busy = true
			}
			m, cmd := press(m, "esc")
			if cmd != nil || !m.draft.Concise || m.unsaved || m.depart || m.conflictReview || m.editingPath || len(b.calls) != 0 {
				t.Fatal("escape did not cancel safely while preserving the draft")
			}
			if state == "path" && (m.outputPath != "/original" || m.draft.Output != "/original") {
				t.Fatal("escape did not restore the original path")
			}
			if state == "busy" && !m.busy {
				t.Fatal("escape interrupted an active operation")
			}
		})
	}
}
