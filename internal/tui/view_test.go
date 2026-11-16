package tui

import (
	"fmt"
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

func TestWorkspaceTabsPreserveDraftAndFocus(t *testing.T) {
	m, b := readyModel(t, false)
	m.page, m.cursor = 1, 2
	m.draft.Concise = true
	m, cmd := press(m, "tab")
	if cmd != nil || m.page != 2 || m.unsaved {
		t.Fatal("tab prompted or performed I/O")
	}
	m, _ = press(m, "shift+tab")
	if m.page != 1 || m.cursor != 2 || !m.draft.Concise || len(b.calls) != 0 {
		t.Fatal("tab lost draft or focus")
	}
	m, _ = press(m, "5")
	if m.page != 3 {
		t.Fatal("direct tab shortcut failed")
	}
}

func TestWorkspaceFitsAndKeepsActionsVisible(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 35}, {40, 12}} {
		m, _ := readyModel(t, false)
		m.width, m.height, m.page = size[0], size[1], 1
		m.catalog = selectionCatalog()
		for _, skill := range m.catalog.Skills {
			m.expanded[skill.Category] = true
		}
		m.cursor = len(m.skillRows()) - 1
		for _, review := range []bool{false, true} {
			if review {
				m.dialogs = []dialog{{"Apply the reviewed changes? [y/N]"}}
				m.reviewDetails = true
			}
			content := ansi.Strip(m.View().Content)
			rows := strings.Split(content, "\n")
			if len(rows) > m.height {
				t.Fatalf("height overflow at %v", size)
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > m.width {
					t.Fatalf("width overflow at %v: %s", size, row)
				}
			}
			if !strings.Contains(content, "agent-toolkit") {
				t.Fatal("header disappeared")
			}
			if review && !strings.Contains(content, "[y/N]") {
				t.Fatal("confirmation disappeared")
			}
		}
	}
}

func TestOverviewWrapsPathsAndKeepsFocusedActionVisible(t *testing.T) {
	for _, width := range []int{24, 40, 80, 240} {
		m, _ := readyModel(t, false)
		m.width, m.height, m.page = width, 32, managementPage
		m.root = "/checkout/" + strings.Repeat("long-directory/", 20) + "\x1b[31m"
		m.saved.Output = "/artifacts/" + strings.Repeat("output/", 20)
		for cursor, action := range m.managementActions() {
			m.cursor = cursor
			content := m.View().Content
			rows := strings.Split(content, "\n")
			if len(rows) > m.height {
				t.Fatalf("overview exceeds height at width %d", width)
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > width {
					t.Fatalf("overview exceeds width %d: %q", width, row)
				}
			}
			if !strings.Contains(ansi.Strip(strings.Join(m.bodyRows(width-4), "")), "> "+action) {
				t.Fatalf("focused action %q truncated at width %d", action, width)
			}
			if !strings.Contains(ansi.Strip(content), "> "+ansi.Truncate(action, width-6, "")) {
				t.Fatalf("focused action %q hidden at width %d", action, width)
			}
		}
		body := ansi.Strip(strings.Join(m.bodyRows(width-4), "\n"))
		if !strings.Contains(strings.ReplaceAll(body, "\n", ""), `\x1b[31m`) {
			t.Fatal("source path control sequence was not shown as escaped text")
		}
	}
}

func TestSharedPresentationFitsEveryScreen(t *testing.T) {
	for _, size := range [][2]int{{24, 12}, {80, 24}, {120, 35}} {
		for _, screen := range []string{"providers", "skills", "storage", "guidance", "setup", "help", "summary", "details", "conflict", "unsaved", "loading"} {
			t.Run(fmt.Sprintf("%s/%dx%d", screen, size[0], size[1]), func(t *testing.T) {
				m, _ := readyModel(t, false)
				m.width, m.height = size[0], size[1]
				m.catalog = selectionCatalog()
				m.globalOutput, m.outputPath = true, "/tmp/artifacts/日本語/long-output-directory"
				switch screen {
				case "skills":
					m.page = 1
				case "storage":
					m.page, m.cursor, m.editingPath = 2, 2, true
				case "guidance":
					m.page = 3
				case "setup":
					m.page, m.wizard = 4, true
				case "help":
					m.help = true
				case "summary", "details":
					m.dialogs = []dialog{{"Apply changes? [y/N]"}}
					m.reviewDetails = screen == "details"
				case "conflict":
					m.conflictReview, m.err = true, "Conflicting path: /tmp/managed-skill"
				case "unsaved":
					m.unsaved = true
				case "loading":
					m.loaded = false
				}
				content := m.View().Content
				if len(strings.Split(content, "\n")) > m.height {
					t.Fatal("screen exceeds terminal height")
				}
				for _, row := range strings.Split(content, "\n") {
					if ansi.StringWidth(row) > m.width {
						t.Fatalf("screen exceeds terminal width: %q", row)
					}
				}
				if screen == "providers" || screen == "skills" || screen == "storage" || screen == "guidance" {
					body := m.bodyRows(size[0] - 4)
					if !strings.Contains(ansi.Strip(body[0]), "─") || body[0] == ansi.Strip(body[0]) {
						t.Fatal("page heading lacks shared divider and emphasis")
					}
				}
			})
		}
	}
}
