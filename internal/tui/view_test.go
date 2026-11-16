package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestQuitPromptOverlaysScreenAndCancels(t *testing.T) {
	m, b := readyModel(t, false)
	m.width, m.height = 100, 30
	m.catalog, m.page = selectionCatalog(), 1
	m.draft.Concise = true
	before := m.View().Content
	m, cmd := press(m, "q")
	if cmd != nil || !m.unsaved {
		t.Fatal("quit did not open the unsaved dialog")
	}
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	boxTop, boxBottom := -1, -1
	for i, line := range lines {
		if strings.HasPrefix(ansi.Cut(line, 14, 86), "╭") {
			boxTop = i
		}
		if strings.HasPrefix(ansi.Cut(line, 14, 86), "╰") {
			boxBottom = i
		}
	}
	if boxTop < 1 || boxBottom <= boxTop || boxTop != (m.height-(boxBottom-boxTop+1))/2 {
		t.Fatal("quit dialog is not centered in a separate bordered box")
	}
	if !strings.Contains(lines[1], "agent-toolkit") || !strings.Contains(strings.Join(lines, "\n"), "Unsaved changes") {
		t.Fatal("dialog or underlying screen is missing")
	}
	for _, key := range []string{"down", "enter", "pgdown"} {
		m, cmd = press(m, key)
		if cmd != nil || !m.unsaved {
			t.Fatal("underlying controls handled input while dialog was open")
		}
	}
	m, cmd = press(m, "backspace")
	if cmd != nil || m.unsaved || m.View().Content != before || len(b.calls) != 0 {
		t.Fatal("cancel did not restore the unchanged screen without I/O")
	}
}

func TestQuitPromptFitsSmallTerminals(t *testing.T) {
	m, _ := readyModel(t, false)
	m.unsaved = true
	for _, size := range [][2]int{{80, 24}, {30, 8}, {12, 6}, {5, 6}, {4, 5}, {1, 1}} {
		m.width, m.height = size[0], size[1]
		lines := strings.Split(m.View().Content, "\n")
		if len(lines) > m.height {
			t.Fatalf("dialog exceeds terminal height at %v", size)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatalf("dialog exceeds terminal width at %v", size)
			}
		}
	}
}
